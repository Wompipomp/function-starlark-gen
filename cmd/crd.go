package cmd

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/wompipomp/starlark-gen/internal/emitter"
	"github.com/wompipomp/starlark-gen/internal/pipeline"
)

// newCrdCmd creates the crd subcommand.
func newCrdCmd() *cobra.Command {
	var (
		pkg     string
		output  string
		verbose bool
	)

	crdCmd := &cobra.Command{
		Use:   "crd <file> [files...]",
		Short: "Generate .star schemas from CRD YAML files",
		Long: `Generate typed Starlark schema files from Kubernetes CustomResourceDefinition YAML files.
Each CRD group and version produces a single .star file with schema() definitions.
Multiple CRD files can be provided and will be merged by group and version.
Supports both v1 and v1beta1 CRD formats, and multi-document YAML files.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := pipeline.CRDOptions{
				Paths:     args,
				Package:   pkg,
				OutputDir: output,
				Verbose:   verbose,
			}

			result, err := pipeline.RunCRD(opts)
			if err != nil {
				return err
			}

			// Print warnings to stderr.
			for _, warn := range result.Warnings {
				fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", warn)
			}

			if verbose {
				// Per-file listing sorted alphabetically.
				printCRDVerboseOutput(cmd, result)
			}

			// Summary line on stdout.
			fmt.Fprintln(cmd.OutOrStdout(), emitter.SummaryLine(result.FileCount, result.SchemaCount, result.OutputDir))

			return nil
		},
	}

	crdCmd.Flags().StringVarP(&pkg, "package", "p", "", "OCI package prefix for load() paths (required)")
	crdCmd.Flags().StringVarP(&output, "output", "o", "./out", "Output directory for generated .star files")
	crdCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show per-file listing instead of summary only")
	_ = crdCmd.MarkFlagRequired("package")

	return crdCmd
}

// printCRDVerboseOutput prints a per-file listing sorted alphabetically by file path.
// Format:
//
//	example.com/v1.star (5 schemas)
func printCRDVerboseOutput(cmd *cobra.Command, result *pipeline.CRDResult) {
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
