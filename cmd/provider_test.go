package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testProviderAWSPath = "../testdata/provider-aws-bucket.yaml"
const testProviderHelmPath = "../testdata/provider-helm-release.yaml"
const testProviderMinimalPath = "../testdata/provider-minimal.yaml"

func TestProviderCmd_Help(t *testing.T) {
	stdout := &bytes.Buffer{}
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"provider", "--help"})
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(&bytes.Buffer{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("provider --help failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Crossplane provider CRD") {
		t.Errorf("expected help to mention 'Crossplane provider CRD', got: %q", output)
	}
}

func TestProviderCmd_NoArgs(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"provider", "--package", "test:v1"})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when no file args provided")
	}
}

func TestProviderCmd_MissingPackage(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"provider", testProviderAWSPath})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when --package is missing")
	}
	if !strings.Contains(err.Error(), "package") {
		t.Errorf("expected error to mention 'package', got: %v", err)
	}
}

func TestProviderCmd_BasicRun(t *testing.T) {
	outDir := t.TempDir()
	stdout := &bytes.Buffer{}
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"provider", testProviderAWSPath, "--package", "test:v1", "--output", outDir})
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(&bytes.Buffer{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Generated") {
		t.Errorf("expected summary output containing 'Generated', got: %q", output)
	}

	// Check that .star files were created.
	found := false
	err := filepath.Walk(outDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".star") {
			found = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking output dir: %v", err)
	}
	if !found {
		t.Error("expected .star files in output directory, found none")
	}
}

func TestProviderCmd_Verbose(t *testing.T) {
	outDir := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"provider", testProviderAWSPath, "--package", "test:v1", "--output", outDir, "--verbose"})
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := stdout.String()
	// Verbose output should contain per-file listing with schema counts.
	if !strings.Contains(output, ".star") {
		t.Errorf("expected verbose output to contain .star file paths, got: %q", output)
	}
	if !strings.Contains(output, "schemas)") {
		t.Errorf("expected verbose output to contain schema counts, got: %q", output)
	}
	// Should also contain the summary line at the end.
	if !strings.Contains(output, "Generated") {
		t.Errorf("expected verbose output to end with summary line, got: %q", output)
	}
}

func TestProviderCmd_Warnings(t *testing.T) {
	outDir := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"provider", testProviderMinimalPath, "--package", "test:v1", "--output", outDir})
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	errOutput := stderr.String()
	if !strings.Contains(errOutput, "no forProvider/initProvider") {
		t.Errorf("expected warning on stderr about non-standard CRD, got: %q", errOutput)
	}
}

func TestProviderCmd_MultipleFiles(t *testing.T) {
	outDir := t.TempDir()
	stdout := &bytes.Buffer{}
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"provider", testProviderAWSPath, testProviderHelmPath, "--package", "test:v1", "--output", outDir})
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(&bytes.Buffer{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Generated") {
		t.Errorf("expected summary output, got: %q", output)
	}

	// Both CRDs should produce output files (different groups).
	starCount := 0
	err := filepath.Walk(outDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".star") {
			starCount++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking output dir: %v", err)
	}
	if starCount < 2 {
		t.Errorf("expected at least 2 .star files for 2 different CRD groups, got %d", starCount)
	}
}

func TestProviderCmd_InvalidFile(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"provider", "/nonexistent/file.yaml", "--package", "test:v1"})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestProviderCmd_RegisteredInRoot(t *testing.T) {
	stdout := &bytes.Buffer{}
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"--help"})
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(&bytes.Buffer{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root --help failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "provider") {
		t.Errorf("expected root --help to list 'provider' subcommand, got: %q", output)
	}
}
