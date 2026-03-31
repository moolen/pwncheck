package registry

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type fakeClient struct {
	digests map[string]string
}

func (f fakeClient) ResolveDigest(_ context.Context, repository string, tag string) (string, error) {
	return f.digests[tag], nil
}

func TestFilterSemverTags(t *testing.T) {
	tags := []string{"latest", "v1.2.3", "1.2.4", "v1.2.3-rc.1", "0.3.0-dev-aws-sm", "main"}

	got := FilterSemverTags(tags)
	want := []string{"1.2.4", "v1.2.3"}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("tags mismatch (-want +got):\n%s", diff)
	}
}

func TestResolveTagReturnsManifestDigest(t *testing.T) {
	client := fakeClient{
		digests: map[string]string{"v1.2.3": "sha256:abc"},
	}

	digest, err := ResolveTag(context.Background(), client, "ghcr.io/external-secrets/external-secrets", "v1.2.3")
	if err != nil {
		t.Fatalf("resolve tag: %v", err)
	}

	if digest != "sha256:abc" {
		t.Fatalf("digest mismatch: got %q", digest)
	}
}
