package main

import (
	"bytes"
	"context"
	"os"
	"testing"
)

func TestRootCommandIncludesVerify(t *testing.T) {
	cmd := newRootCommand(func(_ context.Context, _ string) error {
		return nil
	})
	cmd.SetArgs([]string{"verify", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}
}

func TestVerifyCommandUsesProvidedConfigPath(t *testing.T) {
	var gotPath string

	cmd := newRootCommand(func(_ context.Context, path string) error {
		gotPath = path
		return nil
	})
	cmd.SetArgs([]string{"verify", "--config", "custom.yaml"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}

	if gotPath != "custom.yaml" {
		t.Fatalf("config path mismatch: got %q", gotPath)
	}
}

func TestVerifyWorkflowExists(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/verify.yaml")
	if err != nil {
		t.Fatalf("read workflow: %v", err)
	}

	if !bytes.Contains(data, []byte("go run ./cmd/pwncheck verify --config config.yaml")) {
		t.Fatal("expected workflow to execute the pwncheck verify command")
	}
}
