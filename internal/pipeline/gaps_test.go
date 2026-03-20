package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

// --- RunK8s error path gaps ---

func TestRunK8sNonexistentFile(t *testing.T) {
	opts := K8sOptions{
		SwaggerPath: "nonexistent.json",
		Package:     "test:v1",
		OutputDir:   t.TempDir(),
	}
	_, err := RunK8s(opts)
	if err == nil {
		t.Fatal("expected error for nonexistent swagger file")
	}
}

func TestRunK8sMalformedSwagger(t *testing.T) {
	// Create a malformed JSON file.
	tmpDir := t.TempDir()
	malformed := filepath.Join(tmpDir, "bad.json")
	if err := os.WriteFile(malformed, []byte("{not valid json!!!}"), 0o644); err != nil {
		t.Fatalf("failed to write malformed file: %v", err)
	}

	opts := K8sOptions{
		SwaggerPath: malformed,
		Package:     "test:v1",
		OutputDir:   t.TempDir(),
	}
	_, err := RunK8s(opts)
	if err == nil {
		t.Fatal("expected error for malformed swagger file")
	}
}

func TestRunK8sSuccessReturnsResult(t *testing.T) {
	opts := K8sOptions{
		SwaggerPath: "../../testdata/swagger-mini.json",
		Package:     "test:v1",
		OutputDir:   t.TempDir(),
	}
	result, err := RunK8s(opts)
	if err != nil {
		t.Fatalf("RunK8s failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.FileCount == 0 {
		t.Error("expected non-zero FileCount")
	}
	if result.SchemaCount == 0 {
		t.Error("expected non-zero SchemaCount")
	}
	if result.Warnings == nil {
		t.Error("expected non-nil Warnings slice")
	}
	if result.OutputDir != opts.OutputDir {
		t.Errorf("OutputDir = %q, want %q", result.OutputDir, opts.OutputDir)
	}
}

func TestRunK8sOutputDirContainsFiles(t *testing.T) {
	outDir := t.TempDir()
	opts := K8sOptions{
		SwaggerPath: "../../testdata/swagger-mini.json",
		Package:     "test:v1",
		OutputDir:   outDir,
	}
	result, err := RunK8s(opts)
	if err != nil {
		t.Fatalf("RunK8s failed: %v", err)
	}

	// Verify at least one .star file was written to disk.
	for fp := range result.Files {
		fullPath := filepath.Join(outDir, fp)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist on disk", fullPath)
		}
	}
}

func TestRunK8sWriteToReadOnlyDir(t *testing.T) {
	// Create a read-only directory to trigger write error.
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0o555); err != nil {
		t.Fatalf("failed to create read-only dir: %v", err)
	}
	t.Cleanup(func() {
		os.Chmod(readOnlyDir, 0o755)
	})

	opts := K8sOptions{
		SwaggerPath: "../../testdata/swagger-mini.json",
		Package:     "test:v1",
		OutputDir:   readOnlyDir,
	}
	_, err := RunK8s(opts)
	if err == nil {
		t.Fatal("expected error when writing to read-only directory")
	}
}
