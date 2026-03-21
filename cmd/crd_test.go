package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testCRDBasicPath = "../testdata/crd-basic.yaml"
const testCRDPreservePath = "../testdata/crd-preserve.yaml"

func TestCRDCmd_Basic(t *testing.T) {
	outDir := t.TempDir()
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"crd", testCRDBasicPath, "--package", "test:v1", "--output", outDir})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
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

func TestCRDCmd_MultipleFiles(t *testing.T) {
	outDir := t.TempDir()
	stdout := &bytes.Buffer{}
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"crd", testCRDBasicPath, testCRDPreservePath, "--package", "test:v1", "--output", outDir})
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(&bytes.Buffer{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	// Both CRDs should be processed -- output should mention schemas.
	output := stdout.String()
	if !strings.Contains(output, "Generated") {
		t.Errorf("expected summary output, got: %q", output)
	}

	// Verify output file exists with content from both CRDs.
	starFile := filepath.Join(outDir, "example.com", "v1.star")
	content, err := os.ReadFile(starFile)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "Widget = schema(") {
		t.Error("expected Widget from crd-basic.yaml in output")
	}
	if !strings.Contains(s, "FlexType = schema(") {
		t.Error("expected FlexType from crd-preserve.yaml in output")
	}
}

func TestCRDCmd_NoArgs(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"crd", "--package", "test:v1"})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when no file args provided")
	}
}

func TestCRDCmd_PackageRequired(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"crd", testCRDBasicPath})
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

func TestCRDCmd_DefaultOutput(t *testing.T) {
	rootCmd := newRootCmd()

	// Verify the default value of the --output flag.
	crdCmd := findSubcommand(rootCmd, "crd")
	if crdCmd == nil {
		t.Fatal("expected crd subcommand on root command")
	}
	flag := crdCmd.Flags().Lookup("output")
	if flag == nil {
		t.Fatal("expected --output flag on crd command")
	}
	if flag.DefValue != "./out" {
		t.Errorf("expected --output default to be './out', got %q", flag.DefValue)
	}
}

func TestCRDCmd_VerboseOutput(t *testing.T) {
	outDir := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"crd", testCRDBasicPath, "--package", "test:v1", "--output", outDir, "--verbose"})
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

func TestCRDCmd_SummaryOutput(t *testing.T) {
	outDir := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"crd", testCRDBasicPath, "--package", "test:v1", "--output", outDir})
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	// Default output should be a single summary line.
	lines := strings.Split(output, "\n")
	if len(lines) != 1 {
		t.Errorf("expected exactly 1 line of output, got %d: %q", len(lines), output)
	}
	if !strings.HasPrefix(output, "Generated") {
		t.Errorf("expected summary line starting with 'Generated', got: %q", output)
	}
	if !strings.Contains(output, "files") || !strings.Contains(output, "schemas") {
		t.Errorf("expected summary line to mention files and schemas, got: %q", output)
	}
}

func TestCRDCmd_Warnings(t *testing.T) {
	outDir := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"crd", testCRDBasicPath, "--package", "test:v1", "--output", outDir})
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	// stdout should have the summary, stderr may have warnings.
	if !strings.Contains(stdout.String(), "Generated") {
		t.Error("expected summary on stdout")
	}
}

func TestCRDCmd_SubcommandRegistered(t *testing.T) {
	rootCmd := newRootCmd()
	stdout := &bytes.Buffer{}
	rootCmd.SetArgs([]string{"crd", "--help"})
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(&bytes.Buffer{})

	// --help should succeed (subcommand exists).
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("crd --help failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "CRD") && !strings.Contains(output, "crd") {
		t.Errorf("expected help output to mention CRD/crd, got: %q", output)
	}
}
