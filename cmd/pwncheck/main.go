package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

type VerifyRunner func(context.Context, string) error

func newRootCommand(run VerifyRunner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pwncheck",
		Short: "Verify container images against a stored provenance baseline",
	}

	cmd.AddCommand(newVerifyCommand(run))

	return cmd
}

func newVerifyCommand(run VerifyRunner) *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify GHCR tags against stored baseline state",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), configPath)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "config.yaml", "Path to config file")

	return cmd
}

func main() {
	cmd := newRootCommand(runVerify)

	cmd.SetContext(context.Background())

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
