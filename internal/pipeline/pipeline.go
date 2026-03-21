// Package pipeline orchestrates the end-to-end generation pipeline, wiring
// all stages (Loader, Resolver, Organizer, TypeGraph, Emitter) together.
//
// RunK8s is the entry point for Kubernetes swagger.json processing.
// RunCRD is the entry point for CRD YAML file processing.
// Both pipelines follow the same fail-fast pattern: Load -> Resolve ->
// Organize -> Sort -> ValidateDAG -> Emit -> Write.
package pipeline

import (
	"fmt"

	"github.com/wompipomp/starlark-gen/internal/emitter"
	"github.com/wompipomp/starlark-gen/internal/loader"
	"github.com/wompipomp/starlark-gen/internal/organizer"
	"github.com/wompipomp/starlark-gen/internal/resolver"
	"github.com/wompipomp/starlark-gen/internal/typegraph"
)

// K8sOptions holds the configuration for a K8s generation run.
type K8sOptions struct {
	// SwaggerPath is the path to the swagger.json file (positional arg).
	SwaggerPath string

	// Package is the OCI package prefix for generated load() paths
	// (e.g., "schemas-k8s:v1.31").
	Package string

	// OutputDir is the directory where generated .star files are written
	// (e.g., "./out").
	OutputDir string

	// Verbose enables per-file listing output instead of summary-only.
	Verbose bool
}

// K8sResult holds the output of a successful K8s generation run.
type K8sResult struct {
	// Files is the generated content keyed by file path.
	Files emitter.EmitResult

	// FileCount is the number of files written to disk.
	FileCount int

	// SchemaCount is the total number of schema definitions across all files.
	SchemaCount int

	// Warnings is the combined list of non-fatal warnings from all stages.
	Warnings []string

	// OutputDir is the directory where files were written.
	OutputDir string
}

// RunK8s executes the full K8s generation pipeline: Load -> Resolve -> Organize
// -> Sort -> ValidateDAG -> Emit -> Write.
//
// Each stage's error is wrapped with context for clear diagnostics. On first
// error, execution stops immediately with no partial output. Warnings from
// the resolver and organizer stages are collected into a single slice.
func RunK8s(opts K8sOptions) (*K8sResult, error) {
	// Stage 1: Load swagger.json.
	model, err := loader.LoadSwagger(opts.SwaggerPath)
	if err != nil {
		return nil, fmt.Errorf("loading swagger: %w", err)
	}

	// Stage 2: Resolve all definitions to TypeNodes.
	nodes, resolverWarnings := resolver.Resolve(model)

	// Initialize warnings with non-nil slice.
	warnings := make([]string, 0, len(resolverWarnings))
	warnings = append(warnings, resolverWarnings...)

	// Stage 3: Organize TypeNodes into files by API group/version.
	fileMap, orgWarnings, err := organizer.Organize(nodes, opts.Package)
	if err != nil {
		return nil, fmt.Errorf("organizing definitions: %w", err)
	}
	warnings = append(warnings, orgWarnings...)

	// Stage 4: Sort types within each file (topological order).
	for fp, types := range fileMap {
		sorted, err := typegraph.SortTypesInFile(types)
		if err != nil {
			return nil, fmt.Errorf("sorting types in %s: %w", fp, err)
		}
		fileMap[fp] = sorted
	}

	// Stage 5: Validate inter-file dependency DAG and get emission order.
	fileOrder, err := typegraph.ValidateLoadDAG(fileMap)
	if err != nil {
		return nil, fmt.Errorf("validating load DAG: %w", err)
	}

	// Stage 6: Generate Starlark code.
	result, err := emitter.Emit(fileMap, fileOrder, opts.Package)
	if err != nil {
		return nil, fmt.Errorf("emitting starlark: %w", err)
	}

	// Stage 7: Write files to disk.
	fileCount, schemaCount, err := emitter.WriteFiles(result, opts.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("writing files: %w", err)
	}

	return &K8sResult{
		Files:       result,
		FileCount:   fileCount,
		SchemaCount: schemaCount,
		Warnings:    warnings,
		OutputDir:   opts.OutputDir,
	}, nil
}

// CRDOptions holds the configuration for a CRD generation run.
type CRDOptions struct {
	// Paths is the list of CRD YAML file paths (positional args).
	Paths []string

	// Package is the OCI package prefix for generated load() paths.
	Package string

	// OutputDir is the directory where generated .star files are written.
	OutputDir string

	// Verbose enables per-file listing output.
	Verbose bool
}

// CRDResult holds the output of a successful CRD generation run.
type CRDResult struct {
	// Files is the generated content keyed by file path.
	Files emitter.EmitResult

	// FileCount is the number of files written to disk.
	FileCount int

	// SchemaCount is the total number of schema definitions across all files.
	SchemaCount int

	// Warnings is the combined list of non-fatal warnings from all stages.
	Warnings []string

	// OutputDir is the directory where files were written.
	OutputDir string
}

// RunCRD executes the full CRD generation pipeline: Load -> Resolve ->
// Organize -> Sort -> ValidateDAG -> Emit -> Write.
//
// Unlike RunK8s, the CRD pipeline skips the Organizer's DefinitionKeyToFilePath
// mapping because the CRD resolver pre-sets FilePath on each TypeNode from
// CRD metadata (group/version.star).
//
// Each stage's error is wrapped with context for clear diagnostics. On first
// error, execution stops immediately with no partial output.
func RunCRD(opts CRDOptions) (*CRDResult, error) {
	// Stage 1: Load CRD YAML files.
	crds, err := loader.LoadCRDs(opts.Paths)
	if err != nil {
		return nil, fmt.Errorf("loading CRDs: %w", err)
	}

	// Stage 2: Resolve CRDs to TypeNodes.
	nodes, resolverWarnings := resolver.ResolveCRDs(crds)

	// Initialize warnings with non-nil slice.
	warnings := make([]string, 0, len(resolverWarnings))
	warnings = append(warnings, resolverWarnings...)

	// Stage 3: Organize into FileMap. CRD TypeNodes already have FilePath
	// pre-set by the resolver, so we group by FilePath directly.
	fileMap := make(organizer.FileMap)
	for i := range nodes {
		node := &nodes[i]
		fileMap[node.FilePath] = append(fileMap[node.FilePath], node)
	}

	// Stage 4: Sort types within each file (topological order).
	for fp, types := range fileMap {
		sorted, err := typegraph.SortTypesInFile(types)
		if err != nil {
			return nil, fmt.Errorf("sorting types in %s: %w", fp, err)
		}
		fileMap[fp] = sorted
	}

	// Stage 5: Validate inter-file dependency DAG and get emission order.
	fileOrder, err := typegraph.ValidateLoadDAG(fileMap)
	if err != nil {
		return nil, fmt.Errorf("validating load DAG: %w", err)
	}

	// Stage 6: Generate Starlark code.
	result, err := emitter.Emit(fileMap, fileOrder, opts.Package)
	if err != nil {
		return nil, fmt.Errorf("emitting starlark: %w", err)
	}

	// Stage 7: Write files to disk.
	fileCount, schemaCount, err := emitter.WriteFiles(result, opts.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("writing files: %w", err)
	}

	return &CRDResult{
		Files:       result,
		FileCount:   fileCount,
		SchemaCount: schemaCount,
		Warnings:    warnings,
		OutputDir:   opts.OutputDir,
	}, nil
}
