package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	ghapi "github.com/google/go-github/v69/github"
	"golang.org/x/oauth2"

	"github.com/moolen/pwncheck/internal/config"
	"github.com/moolen/pwncheck/internal/state"
)

var ErrNotFound = errors.New("not found")

type Release struct {
	ID      int64
	TagName string
	Name    string
	Assets  []Asset
}

type Asset struct {
	ID   int64
	Name string
}

type Client interface {
	GetReleaseByTag(ctx context.Context, owner string, repo string, tag string) (Release, error)
	CreateRelease(ctx context.Context, owner string, repo string, release Release) (Release, error)
	DeleteAsset(ctx context.Context, owner string, repo string, assetID int64) error
	UploadAsset(ctx context.Context, owner string, repo string, releaseID int64, name string, reader io.Reader) error
}

type StateClient interface {
	Client
	DownloadAsset(ctx context.Context, owner string, repo string, assetID int64) ([]byte, error)
}

type RESTClient struct {
	client *ghapi.Client
}

func NewRESTClient(token string) *RESTClient {
	httpClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	))
	return &RESTClient{
		client: ghapi.NewClient(httpClient),
	}
}

func EnsureRelease(ctx context.Context, client Client, cfg config.ReleaseConfig) (Release, error) {
	release, err := client.GetReleaseByTag(ctx, cfg.Owner, cfg.Repo, cfg.Tag)
	if err == nil {
		return release, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Release{}, fmt.Errorf("get release %s: %w", cfg.Tag, err)
	}

	release, err = client.CreateRelease(ctx, cfg.Owner, cfg.Repo, Release{
		TagName: cfg.Tag,
		Name:    cfg.Name,
	})
	if err != nil {
		return Release{}, fmt.Errorf("create release %s: %w", cfg.Tag, err)
	}

	return release, nil
}

func UploadState(ctx context.Context, client Client, owner string, repo string, release Release, assetName string, data []byte) error {
	for _, asset := range release.Assets {
		if asset.Name != assetName {
			continue
		}
		if err := client.DeleteAsset(ctx, owner, repo, asset.ID); err != nil {
			return fmt.Errorf("delete existing asset %s: %w", assetName, err)
		}
	}

	if err := client.UploadAsset(ctx, owner, repo, release.ID, assetName, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("upload asset %s: %w", assetName, err)
	}

	return nil
}

func AssetNameForRepository(name string) string {
	return name + ".json"
}

type StateStore struct {
	client StateClient
	cfg    config.ReleaseConfig
}

func NewStateStore(client StateClient, cfg config.ReleaseConfig) *StateStore {
	return &StateStore{
		client: client,
		cfg:    cfg,
	}
}

func (c *RESTClient) GetReleaseByTag(ctx context.Context, owner string, repo string, tag string) (Release, error) {
	release, _, err := c.client.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	if err != nil {
		var rateErr *ghapi.ErrorResponse
		if errors.As(err, &rateErr) && rateErr.Response.StatusCode == http.StatusNotFound {
			return Release{}, ErrNotFound
		}
		return Release{}, err
	}

	return toRelease(release), nil
}

func (c *RESTClient) CreateRelease(ctx context.Context, owner string, repo string, release Release) (Release, error) {
	created, _, err := c.client.Repositories.CreateRelease(ctx, owner, repo, &ghapi.RepositoryRelease{
		TagName: ghapi.String(release.TagName),
		Name:    ghapi.String(release.Name),
		Body:    ghapi.String("State storage for pwncheck baselines"),
	})
	if err != nil {
		return Release{}, err
	}

	return toRelease(created), nil
}

func (c *RESTClient) DeleteAsset(ctx context.Context, owner string, repo string, assetID int64) error {
	_, err := c.client.Repositories.DeleteReleaseAsset(ctx, owner, repo, assetID)
	return err
}

func (c *RESTClient) UploadAsset(ctx context.Context, owner string, repo string, releaseID int64, name string, reader io.Reader) error {
	file, err := os.CreateTemp("", "pwncheck-asset-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}

	_, _, err = c.client.Repositories.UploadReleaseAsset(ctx, owner, repo, releaseID, &ghapi.UploadOptions{Name: name}, file)
	return err
}

func (c *RESTClient) DownloadAsset(ctx context.Context, owner string, repo string, assetID int64) ([]byte, error) {
	rc, redirectURL, err := c.client.Repositories.DownloadReleaseAsset(ctx, owner, repo, assetID, http.DefaultClient)
	if err != nil {
		return nil, err
	}

	if redirectURL != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, redirectURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}

	defer rc.Close()
	return io.ReadAll(rc)
}

func (s *StateStore) LoadState(ctx context.Context, repoCfg config.RepositoryConfig) (state.RepositoryState, bool, error) {
	release, err := EnsureRelease(ctx, s.client, s.cfg)
	if err != nil {
		return state.RepositoryState{}, false, err
	}

	assetName := AssetNameForRepository(repoCfg.Name)
	for _, asset := range release.Assets {
		if asset.Name != assetName {
			continue
		}
		data, err := s.client.DownloadAsset(ctx, s.cfg.Owner, s.cfg.Repo, asset.ID)
		if err != nil {
			return state.RepositoryState{}, false, fmt.Errorf("download asset %s: %w", assetName, err)
		}
		var repoState state.RepositoryState
		if err := json.Unmarshal(data, &repoState); err != nil {
			return state.RepositoryState{}, false, fmt.Errorf("decode asset %s: %w", assetName, err)
		}
		return repoState, true, nil
	}

	return state.RepositoryState{}, false, nil
}

func (s *StateStore) SaveState(ctx context.Context, repoCfg config.RepositoryConfig, repoState state.RepositoryState) error {
	release, err := EnsureRelease(ctx, s.client, s.cfg)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(repoState, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state for %s: %w", repoCfg.Name, err)
	}

	return UploadState(ctx, s.client, s.cfg.Owner, s.cfg.Repo, release, AssetNameForRepository(repoCfg.Name), data)
}

func toRelease(release *ghapi.RepositoryRelease) Release {
	result := Release{
		ID:      release.GetID(),
		TagName: release.GetTagName(),
		Name:    release.GetName(),
	}

	for _, asset := range release.Assets {
		result.Assets = append(result.Assets, Asset{
			ID:   asset.GetID(),
			Name: asset.GetName(),
		})
	}

	return result
}
