package registry

import (
	"context"
	"fmt"
	"regexp"
	"slices"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type DigestResolver interface {
	ResolveDigest(ctx context.Context, repository string, tag string) (string, error)
}

type Client struct{}

var stableSemverPattern = regexp.MustCompile(`^v?\d+\.\d+\.\d+$`)

func NewClient() *Client {
	return &Client{}
}

func FilterSemverTags(tags []string) []string {
	filtered := make([]string, 0, len(tags))
	for _, tag := range tags {
		if stableSemverPattern.MatchString(tag) {
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

func (c *Client) ListTags(ctx context.Context, repository string) ([]string, error) {
	repoRef, err := name.NewRepository(repository)
	if err != nil {
		return nil, fmt.Errorf("parse repository %q: %w", repository, err)
	}

	tags, err := remote.List(repoRef, remote.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("list tags for %s: %w", repository, err)
	}

	return tags, nil
}

func (c *Client) ResolveDigest(ctx context.Context, repository string, tag string) (string, error) {
	tagRef, err := name.NewTag(fmt.Sprintf("%s:%s", repository, tag))
	if err != nil {
		return "", fmt.Errorf("parse tag %s:%s: %w", repository, tag, err)
	}

	desc, err := remote.Head(tagRef, remote.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("head %s:%s: %w", repository, tag, err)
	}

	return desc.Digest.String(), nil
}
