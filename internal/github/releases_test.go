package github

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/moolen/pwncheck/internal/config"
)

type fakeClient struct {
	release     Release
	hasRelease  bool
	deletedAsset bool
	uploaded    map[string]string
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		uploaded: make(map[string]string),
	}
}

func fakeReleaseWithAsset(name string) Release {
	return Release{
		ID:      1,
		TagName: "pwncheck-state",
		Assets: []Asset{
			{ID: 7, Name: name},
		},
	}
}

func (f *fakeClient) GetReleaseByTag(_ context.Context, owner string, repo string, tag string) (Release, error) {
	if !f.hasRelease {
		return Release{}, ErrNotFound
	}
	return f.release, nil
}

func (f *fakeClient) CreateRelease(_ context.Context, owner string, repo string, release Release) (Release, error) {
	f.hasRelease = true
	f.release = release
	f.release.ID = 1
	return f.release, nil
}

func (f *fakeClient) DeleteAsset(_ context.Context, owner string, repo string, assetID int64) error {
	f.deletedAsset = true
	return nil
}

func (f *fakeClient) UploadAsset(_ context.Context, owner string, repo string, releaseID int64, name string, reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	f.uploaded[name] = string(data)
	return nil
}

func TestEnsureReleaseCreatesMissingRelease(t *testing.T) {
	client := newFakeClient()

	release, err := EnsureRelease(context.Background(), client, config.ReleaseConfig{
		Owner: "moolen",
		Repo:  "pwncheck",
		Tag:   "pwncheck-state",
		Name:  "Pwncheck State",
	})
	if err != nil {
		t.Fatalf("ensure release: %v", err)
	}

	if release.TagName != "pwncheck-state" {
		t.Fatalf("tag mismatch: got %q", release.TagName)
	}
}

func TestUploadStateReplacesExistingAsset(t *testing.T) {
	client := newFakeClient()
	client.hasRelease = true
	release := fakeReleaseWithAsset("external-secrets.json")

	err := UploadState(context.Background(), client, "moolen", "pwncheck", release, "external-secrets.json", []byte(`{}`))
	if err != nil {
		t.Fatalf("upload state: %v", err)
	}

	if !client.deletedAsset {
		t.Fatal("expected existing asset to be deleted before upload")
	}

	if got := client.uploaded["external-secrets.json"]; strings.TrimSpace(got) != "{}" {
		t.Fatalf("uploaded data mismatch: got %q", got)
	}
}
