package github

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/moolen/pwncheck/internal/config"
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
