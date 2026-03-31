package verify

import (
	"context"
	"testing"
	"time"

	"github.com/moolen/pwncheck/internal/config"
	"github.com/moolen/pwncheck/internal/provenance"
	"github.com/moolen/pwncheck/internal/state"
)

type fakeDeps struct {
	baseline   map[string]state.RepositoryState
	tags       []string
	digests    map[string]string
	provenance map[string]provenance.Result
	saved      map[string]state.RepositoryState
	now        time.Time
}

func (f *fakeDeps) ListTags(_ context.Context, _ string) ([]string, error) {
	return f.tags, nil
}

func (f *fakeDeps) ResolveDigest(_ context.Context, _ string, tag string) (string, error) {
	return f.digests[tag], nil
}

func (f *fakeDeps) Verify(_ context.Context, digest string) (provenance.Result, error) {
	return f.provenance[digest], nil
}

func (f *fakeDeps) LoadState(_ context.Context, repo config.RepositoryConfig) (state.RepositoryState, bool, error) {
	if f.baseline == nil {
		return state.RepositoryState{}, false, nil
	}
	st, ok := f.baseline[repo.Name]
	return st, ok, nil
}

func (f *fakeDeps) SaveState(_ context.Context, repo config.RepositoryConfig, st state.RepositoryState) error {
	if f.saved == nil {
		f.saved = make(map[string]state.RepositoryState)
	}
	f.saved[repo.Name] = st
	return nil
}

func (f *fakeDeps) Now() time.Time {
	if f.now.IsZero() {
		return time.Unix(100, 0).UTC()
	}
	return f.now
}

func testConfig() config.Config {
	return config.Config{
		Repositories: []config.RepositoryConfig{
			{
				Name:    "external-secrets",
				Package: "ghcr.io/external-secrets/external-secrets",
				Policy: config.ProvenancePolicy{
					Issuer:       "https://token.actions.githubusercontent.com",
					Repository:   "external-secrets/external-secrets",
					WorkflowPath: ".github/workflows/release.yaml",
					Ref:          "refs/heads/main",
				},
			},
		},
	}
}

func TestRunBootstrapsMissingBaseline(t *testing.T) {
	deps := &fakeDeps{
		tags:    []string{"v1.2.3"},
		digests: map[string]string{"v1.2.3": "sha256:abc"},
		provenance: map[string]provenance.Result{
			"sha256:abc": {
				ProvenanceDigest: "sha256:prov",
				SubjectDigest:    "sha256:abc",
				Identity: provenance.VerifiedIdentity{
					Issuer:       "https://token.actions.githubusercontent.com",
					Repository:   "external-secrets/external-secrets",
					WorkflowPath: ".github/workflows/release.yaml",
					Ref:          "refs/heads/main",
				},
			},
		},
	}

	result, err := Run(context.Background(), deps, testConfig())
	if err != nil {
		t.Fatalf("run verify: %v", err)
	}

	if !result.Updated {
		t.Fatal("expected bootstrap to update baseline")
	}

	if _, ok := deps.saved["external-secrets"]; !ok {
		t.Fatal("expected state to be saved")
	}
}

func TestRunFlagsDriftForExistingTagDigestChange(t *testing.T) {
	deps := &fakeDeps{
		baseline: map[string]state.RepositoryState{
			"external-secrets": {
				SchemaVersion: state.CurrentSchemaVersion,
				Repository:    "external-secrets",
				Tags: map[string]state.TagRecord{
					"v1.2.3": {Tag: "v1.2.3", ManifestDigest: "sha256:old", ProvenanceDigest: "sha256:prov"},
				},
			},
		},
		tags:    []string{"v1.2.3"},
		digests: map[string]string{"v1.2.3": "sha256:new"},
		provenance: map[string]provenance.Result{
			"sha256:new": {
				ProvenanceDigest: "sha256:prov2",
				SubjectDigest:    "sha256:new",
				Identity: provenance.VerifiedIdentity{
					Issuer:       "https://token.actions.githubusercontent.com",
					Repository:   "external-secrets/external-secrets",
					WorkflowPath: ".github/workflows/release.yaml",
					Ref:          "refs/heads/main",
				},
			},
		},
	}

	_, err := Run(context.Background(), deps, testConfig())
	if err == nil {
		t.Fatal("expected drift error")
	}
}

func TestRunAddsNewTagsWithoutFailing(t *testing.T) {
	deps := &fakeDeps{
		baseline: map[string]state.RepositoryState{
			"external-secrets": {
				SchemaVersion: state.CurrentSchemaVersion,
				Repository:    "external-secrets",
				Tags: map[string]state.TagRecord{
					"v1.2.3": {Tag: "v1.2.3", ManifestDigest: "sha256:abc", ProvenanceDigest: "sha256:prov"},
				},
			},
		},
		tags:    []string{"v1.2.3", "v1.2.4"},
		digests: map[string]string{"v1.2.3": "sha256:abc", "v1.2.4": "sha256:def"},
		provenance: map[string]provenance.Result{
			"sha256:abc": {
				ProvenanceDigest: "sha256:prov",
				SubjectDigest:    "sha256:abc",
				Identity: provenance.VerifiedIdentity{
					Issuer:       "https://token.actions.githubusercontent.com",
					Repository:   "external-secrets/external-secrets",
					WorkflowPath: ".github/workflows/release.yaml",
					Ref:          "refs/heads/main",
				},
			},
			"sha256:def": {
				ProvenanceDigest: "sha256:prov-new",
				SubjectDigest:    "sha256:def",
				Identity: provenance.VerifiedIdentity{
					Issuer:       "https://token.actions.githubusercontent.com",
					Repository:   "external-secrets/external-secrets",
					WorkflowPath: ".github/workflows/release.yaml",
					Ref:          "refs/heads/main",
				},
			},
		},
	}

	result, err := Run(context.Background(), deps, testConfig())
	if err != nil {
		t.Fatalf("run verify: %v", err)
	}

	if !result.Updated {
		t.Fatal("expected baseline update for new tag")
	}
}
