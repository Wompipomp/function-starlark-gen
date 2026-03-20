// Package cmd provides the CLI commands for the starlark-gen tool.
//
// The root command initializes cobra and registers all subcommands.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// newRootCmd creates the root cobra command. Separated from Execute() for
// testability -- tests can create fresh command instances without side effects.
func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "starlark-gen",
		Short: "Generate typed Starlark schemas from OpenAPI specs",
		Long: `starlark-gen generates typed Starlark schema files from OpenAPI specifications.
Generated schemas provide construction-time validation for Kubernetes and
Crossplane resource definitions, catching typos, wrong types, and missing
required fields immediately instead of failing silently at apply time.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.AddCommand(newK8sCmd())

	return rootCmd
}

// Execute runs the root command. It exits with code 1 on error.
func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
