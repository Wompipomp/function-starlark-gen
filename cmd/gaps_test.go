package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestK8sCmd_NonexistentSwaggerFile(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"k8s", "nonexistent.json", "--package", "test:v1"})
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent swagger file")
	}
}

func TestK8sCmd_VerboseListsFilePaths(t *testing.T) {
	outDir := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"k8s", testSwaggerPath, "--package", "test:v1", "--output", outDir, "-v"})
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := stdout.String()
	// Should include specific known file paths.
	if !strings.Contains(output, "apps/v1.star") {
		t.Errorf("verbose output should contain apps/v1.star, got: %q", output)
	}
	if !strings.Contains(output, "meta/v1.star") {
		t.Errorf("verbose output should contain meta/v1.star, got: %q", output)
	}
}

func TestK8sCmd_SummaryContainsCounts(t *testing.T) {
	outDir := t.TempDir()
	stdout := &bytes.Buffer{}

	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"k8s", testSwaggerPath, "--package", "test:v1", "--output", outDir})
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(&bytes.Buffer{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "files") {
		t.Errorf("summary should mention files, got: %q", output)
	}
	if !strings.Contains(output, "schemas") {
		t.Errorf("summary should mention schemas, got: %q", output)
	}
}

func TestK8sCmd_ShortFlagP(t *testing.T) {
	outDir := t.TempDir()
	stdout := &bytes.Buffer{}

	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"k8s", testSwaggerPath, "-p", "test:v1", "-o", outDir})
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(&bytes.Buffer{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("short flags failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "Generated") {
		t.Error("expected success with short flags")
	}
}

func TestK8sCmd_PackagePropagatedToLoadPaths(t *testing.T) {
	outDir := t.TempDir()
	stdout := &bytes.Buffer{}

	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"k8s", testSwaggerPath, "--package", "my-schemas:v2.0", "--output", outDir})
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(&bytes.Buffer{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	// Check that generated files use the provided package prefix in load() paths.
	appsFile := filepath.Join(outDir, "apps", "v1.star")
	data, err := os.ReadFile(appsFile)
	if err != nil {
		t.Skipf("apps/v1.star not found: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "load(") && !strings.Contains(content, "my-schemas:v2.0") {
		t.Errorf("expected load paths to use package 'my-schemas:v2.0', got: %s", content)
	}
}
