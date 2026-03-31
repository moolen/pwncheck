package verify

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/moolen/pwncheck/internal/config"
	"github.com/moolen/pwncheck/internal/provenance"
	"github.com/moolen/pwncheck/internal/registry"
	"github.com/moolen/pwncheck/internal/state"
)

type Dependencies interface {
	ListTags(ctx context.Context, repository string) ([]string, error)
	ResolveDigest(ctx context.Context, repository string, tag string) (string, error)
	Verify(ctx context.Context, digest string) (provenance.Result, error)
	LoadState(ctx context.Context, repo config.RepositoryConfig) (state.RepositoryState, bool, error)
	SaveState(ctx context.Context, repo config.RepositoryConfig, st state.RepositoryState) error
	Now() time.Time
}

type RunResult struct {
	Updated bool
}

func Run(ctx context.Context, deps Dependencies, cfg config.Config) (RunResult, error) {
	var result RunResult
	var driftErrors []string

	for _, repo := range cfg.Repositories {
		repoState, ok, err := deps.LoadState(ctx, repo)
		if err != nil {
			return RunResult{}, fmt.Errorf("load state for %s: %w", repo.Name, err)
		}
		if !ok {
			repoState = state.RepositoryState{
				SchemaVersion: state.CurrentSchemaVersion,
				Repository:    repo.Name,
				Tags:          make(map[string]state.TagRecord),
			}
		}
		if repoState.Tags == nil {
			repoState.Tags = make(map[string]state.TagRecord)
		}

		tags, err := deps.ListTags(ctx, repo.Package)
		if err != nil {
			return RunResult{}, fmt.Errorf("list tags for %s: %w", repo.Name, err)
		}

		for _, tag := range registry.FilterSemverTags(tags) {
			record, err := buildRecord(ctx, deps, repo, tag)
			if err != nil {
				return RunResult{}, fmt.Errorf("verify %s:%s: %w", repo.Package, tag, err)
			}

			baseline, exists := repoState.Tags[tag]
			if !exists {
				repoState.Tags[tag] = record
				repoState.UpdatedAt = deps.Now()
				result.Updated = true
				continue
			}

			compare := state.CompareTag(baseline, record)
			if compare.Drift {
				driftErrors = append(driftErrors, fmt.Sprintf("%s:%s %s", repo.Name, tag, strings.Join(compare.Reasons, ", ")))
			}
		}

		if result.Updated {
			if err := deps.SaveState(ctx, repo, repoState); err != nil {
				return RunResult{}, fmt.Errorf("save state for %s: %w", repo.Name, err)
			}
		}
	}

	if len(driftErrors) > 0 {
		return result, errors.New(strings.Join(driftErrors, "; "))
	}

	return result, nil
}

func buildRecord(ctx context.Context, deps Dependencies, repo config.RepositoryConfig, tag string) (state.TagRecord, error) {
	digest, err := deps.ResolveDigest(ctx, repo.Package, tag)
	if err != nil {
		return state.TagRecord{}, fmt.Errorf("resolve digest: %w", err)
	}

	provenanceResult, err := provenance.VerifyDigest(ctx, deps, digest)
	if err != nil {
		return state.TagRecord{}, err
	}

	policy := provenance.Policy{
		Issuer:       repo.Policy.Issuer,
		Repository:   repo.Policy.Repository,
		WorkflowPath: repo.Policy.WorkflowPath,
		Ref:          repo.Policy.Ref,
	}
	if err := policy.Match(provenanceResult.Identity); err != nil {
		return state.TagRecord{}, fmt.Errorf("policy mismatch: %w", err)
	}

	return state.TagRecord{
		Tag:              tag,
		ManifestDigest:   digest,
		ProvenanceDigest: provenanceResult.ProvenanceDigest,
		VerificationTime: deps.Now(),
		VerifiedIdentity: state.VerifiedIdentity{
			Issuer:       provenanceResult.Identity.Issuer,
			Repository:   provenanceResult.Identity.Repository,
			WorkflowPath: provenanceResult.Identity.WorkflowPath,
			Ref:          provenanceResult.Identity.Ref,
			Subject:      provenanceResult.Identity.Subject,
		},
	}, nil
}
