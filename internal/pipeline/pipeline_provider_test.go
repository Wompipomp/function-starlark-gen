package pipeline

import (
	"bytes"
	"strings"
	"testing"
)

const testProviderAWSPath = "../../testdata/provider-aws-bucket.yaml"
const testProviderHelmPath = "../../testdata/provider-helm-release.yaml"
const testProviderMinimalPath = "../../testdata/provider-minimal.yaml"

func TestRunProvider_StandardProvider(t *testing.T) {
	opts := ProviderOptions{
		Paths:     []string{testProviderAWSPath},
		Package:   "provider-aws:v1",
		OutputDir: t.TempDir(),
	}

	result, err := RunProvider(opts)
	if err != nil {
		t.Fatalf("RunProvider returned error: %v", err)
	}

	if len(result.Files) == 0 {
		t.Fatal("expected non-empty EmitResult, got empty")
	}

	if result.SchemaCount == 0 {
		t.Error("expected SchemaCount > 0")
	}

	// Check that output contains forProvider/initProvider lifecycle annotations.
	for fp, content := range result.Files {
		s := string(content)
		if !strings.Contains(s, "Reconcilable configuration") {
			t.Errorf("file %s: expected forProvider lifecycle annotation 'Reconcilable configuration'", fp)
		}
	}
}

func TestRunProvider_ForProviderOnly(t *testing.T) {
	opts := ProviderOptions{
		Paths:     []string{testProviderHelmPath},
		Package:   "provider-helm:v1",
		OutputDir: t.TempDir(),
	}

	result, err := RunProvider(opts)
	if err != nil {
		t.Fatalf("RunProvider returned error: %v", err)
	}

	if len(result.Files) == 0 {
		t.Fatal("expected non-empty EmitResult, got empty")
	}

	// Should have forProvider annotation.
	for fp, content := range result.Files {
		s := string(content)
		if !strings.Contains(s, "Reconcilable configuration") {
			t.Errorf("file %s: expected forProvider lifecycle annotation", fp)
		}
	}

	// No warnings about missing initProvider.
	for _, w := range result.Warnings {
		if strings.Contains(w, "no forProvider/initProvider") {
			t.Errorf("unexpected warning for forProvider-only CRD: %s", w)
		}
	}
}

func TestRunProvider_NonStandard(t *testing.T) {
	opts := ProviderOptions{
		Paths:     []string{testProviderMinimalPath},
		Package:   "custom:v1",
		OutputDir: t.TempDir(),
	}

	result, err := RunProvider(opts)
	if err != nil {
		t.Fatalf("RunProvider returned error: %v", err)
	}

	if len(result.Files) == 0 {
		t.Fatal("expected non-empty EmitResult for non-standard CRD")
	}

	// Should have a warning about non-standard structure.
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "no forProvider/initProvider structure found, generating as plain CRD") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about non-standard CRD, got warnings: %v", result.Warnings)
	}
}

func TestRunProvider_MultipleFiles(t *testing.T) {
	opts := ProviderOptions{
		Paths:     []string{testProviderAWSPath, testProviderHelmPath},
		Package:   "multi:v1",
		OutputDir: t.TempDir(),
	}

	result, err := RunProvider(opts)
	if err != nil {
		t.Fatalf("RunProvider returned error: %v", err)
	}

	// Should have files from both CRDs (different groups).
	if len(result.Files) < 2 {
		t.Errorf("expected at least 2 output files for 2 different CRD groups, got %d", len(result.Files))
	}
}

func TestRunProvider_InvalidPath(t *testing.T) {
	opts := ProviderOptions{
		Paths:     []string{"/nonexistent/provider.yaml"},
		Package:   "test:v1",
		OutputDir: t.TempDir(),
	}

	_, err := RunProvider(opts)
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestRunProvider_StatusExcluded(t *testing.T) {
	opts := ProviderOptions{
		Paths:     []string{testProviderAWSPath},
		Package:   "provider-aws:v1",
		OutputDir: t.TempDir(),
	}

	result, err := RunProvider(opts)
	if err != nil {
		t.Fatalf("RunProvider returned error: %v", err)
	}

	// Verify no status-related schemas in output.
	for fp, content := range result.Files {
		s := string(content)
		if strings.Contains(s, "AtProvider") {
			t.Errorf("file %s: expected no AtProvider schema in output (status should be excluded)", fp)
		}
		if bytes.Contains(content, []byte("Conditions = schema(")) {
			t.Errorf("file %s: expected no Conditions schema in output (status should be excluded)", fp)
		}
	}
}

func TestRunProvider_StandardFieldDocs(t *testing.T) {
	opts := ProviderOptions{
		Paths:     []string{testProviderAWSPath},
		Package:   "provider-aws:v1",
		OutputDir: t.TempDir(),
	}

	result, err := RunProvider(opts)
	if err != nil {
		t.Fatalf("RunProvider returned error: %v", err)
	}

	// Check that providerConfigRef has the standard annotation.
	for fp, content := range result.Files {
		s := string(content)
		if strings.Contains(s, "providerConfigRef") && !strings.Contains(s, "Reference to the ProviderConfig for auth") {
			t.Errorf("file %s: expected providerConfigRef to contain 'Reference to the ProviderConfig for auth'", fp)
		}
	}
}
