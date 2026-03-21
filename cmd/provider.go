package cmd

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/wompipomp/starlark-gen/internal/emitter"
	"github.com/wompipomp/starlark-gen/internal/pipeline"
)

// newProviderCmd creates the provider subcommand.
func newProviderCmd() *cobra.Command {
	var (
		pkg     string
		output  string
		verbose bool
	)

	providerCmd := &cobra.Command{
		Use:   "provider <file> [files...]",
		Short: "Generate .star schemas from Crossplane provider CRDs",
		Long: `Generate typed Starlark schema files from Crossplane provider CRD YAML files.

Applies Crossplane-specific annotations: forProvider fields are marked as
"Reconcilable configuration" (continuously reconciled), initProvider fields as
"Write-once initialization" (set only at creation). Standard Crossplane fields
(providerConfigRef, deletionPolicy, managementPolicies, etc.) receive descriptive
documentation.

Status subtrees are fully excluded from generated schemas since provider status
fields are managed by the controller, not user-configurable.

CRDs without forProvider/initProvider structure are generated as plain CRDs with
a warning. Multiple CRD files can be provided and will be merged by group and version.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := pipeline.ProviderOptions{
				Paths:     args,
				Package:   pkg,
				OutputDir: output,
				Verbose:   verbose,
			}

			result, err := pipeline.RunProvider(opts)
			if err != nil {
				return err
			}

			// Print warnings to stderr.
			for _, warn := range result.Warnings {
				fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", warn)
			}

			if verbose {
				// Per-file listing sorted alphabetically.
				printProviderVerboseOutput(cmd, result)
			}

			// Summary line on stdout.
			fmt.Fprintln(cmd.OutOrStdout(), emitter.SummaryLine(result.FileCount, result.SchemaCount, result.OutputDir))

			return nil
		},
	}

	providerCmd.Flags().StringVarP(&pkg, "package", "p", "", "OCI package prefix for load() paths (required)")
	providerCmd.Flags().StringVarP(&output, "output", "o", "./out", "Output directory for generated .star files")
	providerCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show per-file listing instead of summary only")
	_ = providerCmd.MarkFlagRequired("package")

	return providerCmd
}

// printProviderVerboseOutput prints a per-file listing sorted alphabetically by file path.
// Format:
//
//	s3.aws.upbound.io/v1beta1.star (5 schemas)
func printProviderVerboseOutput(cmd *cobra.Command, result *pipeline.ProviderResult) {
	// Sort file paths alphabetically.
	paths := make([]string, 0, len(result.Files))
	for fp := range result.Files {
		paths = append(paths, fp)
	}
	sort.Strings(paths)

	for _, fp := range paths {
		content := result.Files[fp]
		schemaCount := bytes.Count(content, []byte(" = schema("))
		fmt.Fprintf(cmd.OutOrStdout(), "%s (%d schemas)\n", fp, schemaCount)
	}
}
