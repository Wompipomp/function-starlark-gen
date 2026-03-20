package emitter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFiles_CreatesDirectoryStructure(t *testing.T) {
	tmpDir := t.TempDir()

	result := EmitResult{
		"apps/v1.star": []byte("# apps v1\n"),
		"core/v1.star": []byte("# core v1\n"),
		"meta/v1.star": []byte("# meta v1\n"),
	}

	_, _, err := WriteFiles(result, tmpDir)
	if err != nil {
		t.Fatalf("WriteFiles error: %v", err)
	}

	// Check directories exist
	dirs := []string{"apps", "core", "meta"}
	for _, d := range dirs {
		info, err := os.Stat(filepath.Join(tmpDir, d))
		if err != nil {
			t.Errorf("expected directory %s to exist: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", d)
		}
	}
}

func TestWriteFiles_CreatesCorrectContent(t *testing.T) {
	tmpDir := t.TempDir()

	content := []byte("Deployment = schema(\n    \"Deployment\",\n)\n")
	result := EmitResult{
		"apps/v1.star": content,
	}

	_, _, err := WriteFiles(result, tmpDir)
	if err != nil {
		t.Fatalf("WriteFiles error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(tmpDir, "apps/v1.star"))
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}

	if string(got) != string(content) {
		t.Errorf("content mismatch:\n  expected: %q\n  got: %q", string(content), string(got))
	}
}

func TestWriteFiles_OverwritesExistingFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Write initial content
	os.MkdirAll(filepath.Join(tmpDir, "apps"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "apps/v1.star"), []byte("old content"), 0o644)

	// Overwrite with new content
	newContent := []byte("new content\n")
	result := EmitResult{
		"apps/v1.star": newContent,
	}

	_, _, err := WriteFiles(result, tmpDir)
	if err != nil {
		t.Fatalf("WriteFiles error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(tmpDir, "apps/v1.star"))
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}

	if string(got) != string(newContent) {
		t.Errorf("expected overwritten content %q, got %q", string(newContent), string(got))
	}
}

func TestWriteFiles_DoesNotDeleteNonGeneratedFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Pre-create a file not in EmitResult
	os.MkdirAll(filepath.Join(tmpDir, "custom"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "custom/my-schemas.star"), []byte("my custom schema"), 0o644)

	result := EmitResult{
		"apps/v1.star": []byte("# apps\n"),
	}

	_, _, err := WriteFiles(result, tmpDir)
	if err != nil {
		t.Fatalf("WriteFiles error: %v", err)
	}

	// Custom file should still exist
	got, err := os.ReadFile(filepath.Join(tmpDir, "custom/my-schemas.star"))
	if err != nil {
		t.Fatal("non-generated file was deleted")
	}
	if string(got) != "my custom schema" {
		t.Error("non-generated file content was modified")
	}
}

func TestWriteFiles_ReturnsCounts(t *testing.T) {
	tmpDir := t.TempDir()

	result := EmitResult{
		"apps/v1.star": []byte("Deployment = schema(\n    \"Deployment\",\n)\n\nReplicaSet = schema(\n    \"ReplicaSet\",\n)\n"),
		"core/v1.star": []byte("Pod = schema(\n    \"Pod\",\n)\n"),
		"meta/v1.star": []byte("ObjectMeta = schema(\n    \"ObjectMeta\",\n)\n\nLabelSelector = schema(\n    \"LabelSelector\",\n)\n\nStatus = schema(\n    \"Status\",\n)\n"),
	}

	fileCount, schemaCount, err := WriteFiles(result, tmpDir)
	if err != nil {
		t.Fatalf("WriteFiles error: %v", err)
	}

	if fileCount != 3 {
		t.Errorf("expected fileCount=3, got %d", fileCount)
	}

	// apps: 2 schemas, core: 1 schema, meta: 3 schemas = 6 total
	if schemaCount != 6 {
		t.Errorf("expected schemaCount=6, got %d", schemaCount)
	}
}

func TestWriteFiles_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a file and verify it's complete (not partial)
	content := []byte("Schema1 = schema(\n    \"Schema1\",\n)\n")
	result := EmitResult{
		"apps/v1.star": content,
	}

	_, _, err := WriteFiles(result, tmpDir)
	if err != nil {
		t.Fatalf("WriteFiles error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(tmpDir, "apps/v1.star"))
	if err != nil {
		t.Fatal(err)
	}

	// Content should be complete (not truncated)
	if len(got) != len(content) {
		t.Errorf("expected %d bytes, got %d", len(content), len(got))
	}
}

func TestWriteFiles_DeterministicOutput(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	result := EmitResult{
		"apps/v1.star": []byte("Deployment = schema(\n    \"Deployment\",\n)\n"),
		"core/v1.star": []byte("Pod = schema(\n    \"Pod\",\n)\n"),
	}

	// Write twice to different dirs
	_, _, err := WriteFiles(result, tmpDir1)
	if err != nil {
		t.Fatalf("first WriteFiles error: %v", err)
	}

	_, _, err = WriteFiles(result, tmpDir2)
	if err != nil {
		t.Fatalf("second WriteFiles error: %v", err)
	}

	// Read back and compare
	for fp := range result {
		content1, err := os.ReadFile(filepath.Join(tmpDir1, fp))
		if err != nil {
			t.Fatal(err)
		}
		content2, err := os.ReadFile(filepath.Join(tmpDir2, fp))
		if err != nil {
			t.Fatal(err)
		}
		if string(content1) != string(content2) {
			t.Errorf("non-deterministic output for %s:\n  first: %q\n  second: %q", fp, string(content1), string(content2))
		}
	}
}

func TestSummaryLine(t *testing.T) {
	line := SummaryLine(42, 1247, "./out")
	expected := "Generated 42 files (1247 schemas) in ./out"
	if line != expected {
		t.Errorf("expected: %q, got: %q", expected, line)
	}
}
