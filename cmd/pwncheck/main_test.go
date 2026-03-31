package main

import "testing"

func TestRootCommandIncludesVerify(t *testing.T) {
	cmd := newRootCommand(func(_ string) error {
		return nil
	})
	cmd.SetArgs([]string{"verify", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}
}
