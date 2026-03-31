package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/moolen/pwncheck/internal/config"
	ghstate "github.com/moolen/pwncheck/internal/github"
	"github.com/moolen/pwncheck/internal/provenance"
	"github.com/moolen/pwncheck/internal/registry"
	"github.com/moolen/pwncheck/internal/state"
	verifypkg "github.com/moolen/pwncheck/internal/verify"
)

type runtimeDeps struct {
	registry *registry.Client
	verifier *provenance.CosignVerifier
	store    *ghstate.StateStore
	now      func() time.Time
}

func runVerify(ctx context.Context, configPath string) error {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return errors.New("GITHUB_TOKEN is required")
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	deps := &runtimeDeps{
		registry: registry.NewClient(),
		verifier: provenance.NewCosignVerifier(),
		store:    ghstate.NewStateStore(ghstate.NewRESTClient(token), cfg.Release),
		now:      func() time.Time { return time.Now().UTC() },
	}

	result, err := verifypkg.Run(ctx, deps, cfg)
	if err != nil {
		return err
	}

	if result.Updated {
		fmt.Fprintln(os.Stdout, "baseline updated")
		return nil
	}

	fmt.Fprintln(os.Stdout, "no changes detected")
	return nil
}

func (d *runtimeDeps) ListTags(ctx context.Context, repository string) ([]string, error) {
	return d.registry.ListTags(ctx, repository)
}

func (d *runtimeDeps) ResolveDigest(ctx context.Context, repository string, tag string) (string, error) {
	return d.registry.ResolveDigest(ctx, repository, tag)
}

func (d *runtimeDeps) Verify(ctx context.Context, imageRef string, policy provenance.Policy) (provenance.Result, error) {
	return d.verifier.Verify(ctx, imageRef, policy)
}

func (d *runtimeDeps) LoadState(ctx context.Context, repo config.RepositoryConfig) (state.RepositoryState, bool, error) {
	return d.store.LoadState(ctx, repo)
}

func (d *runtimeDeps) SaveState(ctx context.Context, repo config.RepositoryConfig, st state.RepositoryState) error {
	return d.store.SaveState(ctx, repo, st)
}

func (d *runtimeDeps) Now() time.Time {
	return d.now()
}
