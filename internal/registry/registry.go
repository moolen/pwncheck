package registry

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/mod/semver"
)

type DigestResolver interface {
	ResolveDigest(ctx context.Context, repository string, tag string) (string, error)
}

func FilterSemverTags(tags []string) []string {
	filtered := make([]string, 0, len(tags))
	for _, tag := range tags {
		candidate := tag
		if !strings.HasPrefix(candidate, "v") {
			candidate = "v" + candidate
		}
		if semver.IsValid(candidate) {
			filtered = append(filtered, tag)
		}
	}

	slices.Sort(filtered)

	return filtered
}

func ResolveTag(ctx context.Context, client DigestResolver, repository string, tag string) (string, error) {
	digest, err := client.ResolveDigest(ctx, repository, tag)
	if err != nil {
		return "", fmt.Errorf("resolve %s:%s: %w", repository, tag, err)
	}
	if digest == "" {
		return "", fmt.Errorf("resolve %s:%s: empty digest", repository, tag)
	}

	return digest, nil
}
