package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

const testSwaggerPath = "../testdata/swagger-mini.json"

func TestK8sCmd_CreatesStarFiles(t *testing.T) {
	outDir := t.TempDir()
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"k8s", testSwaggerPath, "--package", "test:v1", "--output", outDir})
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

func TestK8sCmd_OutputDefaultsToOut(t *testing.T) {
	// Create a temp working directory to avoid polluting the project.
	tmpDir := t.TempDir()

	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"k8s", testSwaggerPath, "--package", "test:v1", "--output", filepath.Join(tmpDir, "out")})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	// Verify the default value of the --output flag.
	k8sCmd := findSubcommand(rootCmd, "k8s")
	if k8sCmd == nil {
		t.Fatal("expected k8s subcommand on root command")
	}
	flag := k8sCmd.Flags().Lookup("output")
	if flag == nil {
		t.Fatal("expected --output flag on k8s command")
	}
	if flag.DefValue != "./out" {
		t.Errorf("expected --output default to be './out', got %q", flag.DefValue)
	}
}

func TestK8sCmd_PackageRequired(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"k8s", testSwaggerPath})
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

func TestK8sCmd_MissingSwaggerArgReturnsError(t *testing.T) {
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"k8s", "--package", "test:v1"})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when swagger.json arg is missing")
	}
}

func TestK8sCmd_VerboseOutput(t *testing.T) {
	outDir := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"k8s", testSwaggerPath, "--package", "test:v1", "--output", outDir, "--verbose"})
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

func TestK8sCmd_DefaultOutputSummaryLine(t *testing.T) {
	outDir := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"k8s", testSwaggerPath, "--package", "test:v1", "--output", outDir})
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

func TestK8sCmd_WarningsOnStderrSummaryOnStdout(t *testing.T) {
	outDir := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"k8s", testSwaggerPath, "--package", "test:v1", "--output", outDir})
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	// stdout should have the summary, stderr may or may not have warnings
	// (depending on testdata), but the key check is that stdout has the summary.
	if !strings.Contains(stdout.String(), "Generated") {
		t.Error("expected summary on stdout")
	}
}

func TestK8sCmd_DeterminismCITest(t *testing.T) {
	outDir1 := t.TempDir()
	outDir2 := t.TempDir()

	// Run 1.
	rootCmd1 := newRootCmd()
	rootCmd1.SetArgs([]string{"k8s", testSwaggerPath, "--package", "test:v1", "--output", outDir1})
	rootCmd1.SetOut(&bytes.Buffer{})
	rootCmd1.SetErr(&bytes.Buffer{})
	if err := rootCmd1.Execute(); err != nil {
		t.Fatalf("run 1 failed: %v", err)
	}

	// Run 2.
	rootCmd2 := newRootCmd()
	rootCmd2.SetArgs([]string{"k8s", testSwaggerPath, "--package", "test:v1", "--output", outDir2})
	rootCmd2.SetOut(&bytes.Buffer{})
	rootCmd2.SetErr(&bytes.Buffer{})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("run 2 failed: %v", err)
	}

	// Compare all files between the two output directories.
	err := filepath.Walk(outDir1, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(outDir1, path)
		path2 := filepath.Join(outDir2, relPath)

		content1, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content2, err := os.ReadFile(path2)
		if err != nil {
			t.Errorf("file %s exists in run1 but not run2", relPath)
			return nil
		}

		if !bytes.Equal(content1, content2) {
			t.Errorf("file %s differs between runs", relPath)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking output dir: %v", err)
	}
}

// findSubcommand finds a named subcommand on a cobra command.
func findSubcommand(parent *cobra.Command, name string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}
