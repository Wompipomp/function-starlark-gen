package cmd

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wompipomp/starlark-gen/internal/emitter"
	"github.com/wompipomp/starlark-gen/internal/pipeline"
)

// newK8sCmd creates the k8s subcommand.
func newK8sCmd() *cobra.Command {
	var (
		pkg     string
		output  string
		verbose bool
	)

	k8sCmd := &cobra.Command{
		Use:   "k8s <swagger.json>",
		Short: "Generate .star schemas from K8s swagger.json",
		Long: `Generate typed Starlark schema files from a Kubernetes swagger.json specification.
Each API group and version produces a single .star file with schema() definitions
for all resource types. Cross-file references use OCI short-form load() paths.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := pipeline.K8sOptions{
				SwaggerPath: args[0],
				Package:     pkg,
				OutputDir:   output,
				Verbose:     verbose,
			}

			result, err := pipeline.RunK8s(opts)
			if err != nil {
				return err
			}

			// Print warnings to stderr.
			for _, warn := range result.Warnings {
				fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", warn)
			}

			if verbose {
				// Per-file listing sorted alphabetically.
				printVerboseOutput(cmd, result)
			}

			// Summary line on stdout.
			fmt.Fprintln(cmd.OutOrStdout(), emitter.SummaryLine(result.FileCount, result.SchemaCount, result.OutputDir))

			return nil
		},
	}

	k8sCmd.Flags().StringVarP(&pkg, "package", "p", "", "OCI package prefix for load() paths (required)")
	k8sCmd.Flags().StringVarP(&output, "output", "o", "./out", "Output directory for generated .star files")
	k8sCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show per-file listing instead of summary only")
	_ = k8sCmd.MarkFlagRequired("package")

	return k8sCmd
}

// printVerboseOutput prints a per-file listing sorted alphabetically by file path.
// Format:
//
//	apps/v1.star (5 schemas)
//	core/v1.star (12 schemas)
//	meta/v1.star (3 schemas)
func printVerboseOutput(cmd *cobra.Command, result *pipeline.K8sResult) {
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

// schemaCountInContent counts the number of schema definitions in generated content.
func schemaCountInContent(content []byte) int {
	return strings.Count(string(content), " = schema(")
}
