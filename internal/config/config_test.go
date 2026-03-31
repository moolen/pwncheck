package config

import "testing"

func TestLoadConfigParsesRepositoryPolicy(t *testing.T) {
	cfg, err := Load("../../testdata/config/minimal.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if got, want := cfg.Repositories[0].Package, "ghcr.io/external-secrets/external-secrets"; got != want {
		t.Fatalf("package mismatch: got %q want %q", got, want)
	}

	if got, want := cfg.Repositories[0].Policy.Ref, "refs/heads/main"; got != want {
		t.Fatalf("ref mismatch: got %q want %q", got, want)
	}
}

func TestLoadConfigRejectsMissingPolicy(t *testing.T) {
	_, err := Load("../../testdata/config/invalid-missing-policy.yaml")
	if err == nil {
		t.Fatal("expected validation error")
	}
}
