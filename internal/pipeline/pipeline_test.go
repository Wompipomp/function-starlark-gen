package pipeline

import (
	"bytes"
	"strings"
	"testing"
)

const testSwaggerPath = "../../testdata/swagger-mini.json"

func TestRunK8s_ProducesNonEmptyResult(t *testing.T) {
	opts := K8sOptions{
		SwaggerPath: testSwaggerPath,
		Package:     "test:v1",
		OutputDir:   t.TempDir(),
	}

	result, err := RunK8s(opts)
	if err != nil {
		t.Fatalf("RunK8s returned error: %v", err)
	}

	if len(result.Files) == 0 {
		t.Fatal("expected non-empty EmitResult, got empty")
	}
}

func TestRunK8s_OutputContainsExpectedFilePaths(t *testing.T) {
	opts := K8sOptions{
		SwaggerPath: testSwaggerPath,
		Package:     "test:v1",
		OutputDir:   t.TempDir(),
	}

	result, err := RunK8s(opts)
	if err != nil {
		t.Fatalf("RunK8s returned error: %v", err)
	}

	expectedPaths := []string{"apps/v1.star", "core/v1.star", "meta/v1.star"}
	for _, p := range expectedPaths {
		if _, ok := result.Files[p]; !ok {
			t.Errorf("expected file path %q in result, got keys: %v", p, fileKeys(result.Files))
		}
	}
}

func TestRunK8s_GeneratedContentContainsSchemaAndFieldCalls(t *testing.T) {
	opts := K8sOptions{
		SwaggerPath: testSwaggerPath,
		Package:     "test:v1",
		OutputDir:   t.TempDir(),
	}

	result, err := RunK8s(opts)
	if err != nil {
		t.Fatalf("RunK8s returned error: %v", err)
	}

	hasSchema := false
	hasField := false
	for _, content := range result.Files {
		s := string(content)
		if strings.Contains(s, "schema(") {
			hasSchema = true
		}
		if strings.Contains(s, "field(") {
			hasField = true
		}
	}

	if !hasSchema {
		t.Error("expected generated content to contain 'schema(' calls")
	}
	if !hasField {
		t.Error("expected generated content to contain 'field(' calls")
	}
}

func TestRunK8s_GeneratedContentContainsLoadStatements(t *testing.T) {
	opts := K8sOptions{
		SwaggerPath: testSwaggerPath,
		Package:     "test:v1",
		OutputDir:   t.TempDir(),
	}

	result, err := RunK8s(opts)
	if err != nil {
		t.Fatalf("RunK8s returned error: %v", err)
	}

	hasLoad := false
	for _, content := range result.Files {
		if strings.Contains(string(content), "load(") {
			hasLoad = true
			break
		}
	}

	if !hasLoad {
		t.Error("expected generated content to contain load() statements for cross-file references")
	}
}

func TestRunK8s_DeterministicOutput(t *testing.T) {
	opts := K8sOptions{
		SwaggerPath: testSwaggerPath,
		Package:     "test:v1",
		OutputDir:   t.TempDir(),
	}

	result1, err := RunK8s(opts)
	if err != nil {
		t.Fatalf("first RunK8s returned error: %v", err)
	}

	opts.OutputDir = t.TempDir()
	result2, err := RunK8s(opts)
	if err != nil {
		t.Fatalf("second RunK8s returned error: %v", err)
	}

	// Same number of files.
	if len(result1.Files) != len(result2.Files) {
		t.Fatalf("file count mismatch: run1=%d, run2=%d", len(result1.Files), len(result2.Files))
	}

	// Same content for each file.
	for fp, content1 := range result1.Files {
		content2, ok := result2.Files[fp]
		if !ok {
			t.Errorf("file %q in run1 but not run2", fp)
			continue
		}
		if !bytes.Equal(content1, content2) {
			t.Errorf("file %q differs between runs", fp)
		}
	}
}

func TestRunK8s_InvalidFileReturnsError(t *testing.T) {
	opts := K8sOptions{
		SwaggerPath: "/nonexistent/swagger.json",
		Package:     "test:v1",
		OutputDir:   t.TempDir(),
	}

	_, err := RunK8s(opts)
	if err == nil {
		t.Fatal("expected error for invalid file path, got nil")
	}

	if !strings.Contains(err.Error(), "loading swagger") {
		t.Errorf("expected error to mention 'loading swagger', got: %v", err)
	}
}

func TestRunK8s_FullDepthAllDefinitionsHaveSchemas(t *testing.T) {
	opts := K8sOptions{
		SwaggerPath: testSwaggerPath,
		Package:     "test:v1",
		OutputDir:   t.TempDir(),
	}

	result, err := RunK8s(opts)
	if err != nil {
		t.Fatalf("RunK8s returned error: %v", err)
	}

	// Count total schemas across all files.
	totalSchemas := 0
	for _, content := range result.Files {
		totalSchemas += strings.Count(string(content), " = schema(")
	}

	// swagger-mini.json has 30 definitions, but 2 are special types
	// (IntOrString, Quantity) that don't generate schema definitions.
	// So we expect at least 28 schemas.
	if totalSchemas < 28 {
		t.Errorf("expected at least 28 schemas (full-depth), got %d", totalSchemas)
	}

	// Verify schema count matches result.SchemaCount.
	if result.SchemaCount != totalSchemas {
		t.Errorf("SchemaCount=%d does not match counted schemas=%d", result.SchemaCount, totalSchemas)
	}
}

func TestRunK8s_WarningsCollected(t *testing.T) {
	opts := K8sOptions{
		SwaggerPath: testSwaggerPath,
		Package:     "test:v1",
		OutputDir:   t.TempDir(),
	}

	result, err := RunK8s(opts)
	if err != nil {
		t.Fatalf("RunK8s returned error: %v", err)
	}

	// Warnings should be a non-nil slice (may be empty for clean inputs).
	if result.Warnings == nil {
		t.Error("expected Warnings to be non-nil (empty slice, not nil)")
	}
}

// fileKeys returns the keys of an EmitResult for error messages.
func fileKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
