package pipeline

import (
	"bytes"
	"strings"
	"testing"
)

const testCRDBasicPath = "../../testdata/crd-basic.yaml"
const testCRDPreservePath = "../../testdata/crd-preserve.yaml"
const testCRDMultiVersionPath = "../../testdata/crd-multi-version.yaml"
const testCRDMultiDocPath = "../../testdata/crd-multi-doc.yaml"

func TestRunCRD_Basic(t *testing.T) {
	opts := CRDOptions{
		Paths:     []string{testCRDBasicPath},
		Package:   "test:v1",
		OutputDir: t.TempDir(),
	}

	result, err := RunCRD(opts)
	if err != nil {
		t.Fatalf("RunCRD returned error: %v", err)
	}

	if len(result.Files) == 0 {
		t.Fatal("expected non-empty EmitResult, got empty")
	}

	// crd-basic.yaml is group=example.com, version=v1 -> example.com/v1.star
	if _, ok := result.Files["example.com/v1.star"]; !ok {
		t.Errorf("expected file 'example.com/v1.star' in result, got keys: %v", fileKeys(result.Files))
	}
}

func TestRunCRD_SchemaContent(t *testing.T) {
	opts := CRDOptions{
		Paths:     []string{testCRDBasicPath},
		Package:   "test:v1",
		OutputDir: t.TempDir(),
	}

	result, err := RunCRD(opts)
	if err != nil {
		t.Fatalf("RunCRD returned error: %v", err)
	}

	content, ok := result.Files["example.com/v1.star"]
	if !ok {
		t.Fatal("expected example.com/v1.star in result")
	}

	s := string(content)
	if !strings.Contains(s, "Widget = schema(") {
		t.Error("expected generated content to contain 'Widget = schema('")
	}
	if !strings.Contains(s, "WidgetSpec = schema(") {
		t.Error("expected generated content to contain 'WidgetSpec = schema('")
	}
}

func TestRunCRD_EnumConstants(t *testing.T) {
	opts := CRDOptions{
		Paths:     []string{testCRDBasicPath},
		Package:   "test:v1",
		OutputDir: t.TempDir(),
	}

	result, err := RunCRD(opts)
	if err != nil {
		t.Fatalf("RunCRD returned error: %v", err)
	}

	content, ok := result.Files["example.com/v1.star"]
	if !ok {
		t.Fatal("expected example.com/v1.star in result")
	}

	s := string(content)
	// Enum values from crd-basic.yaml: size enum has small, medium, large
	if !strings.Contains(s, "WIDGET_SPEC_SIZE_SMALL") {
		t.Error("expected SCREAMING_SNAKE_CASE enum constant WIDGET_SPEC_SIZE_SMALL")
	}
}

func TestRunCRD_Defaults(t *testing.T) {
	opts := CRDOptions{
		Paths:     []string{testCRDBasicPath},
		Package:   "test:v1",
		OutputDir: t.TempDir(),
	}

	result, err := RunCRD(opts)
	if err != nil {
		t.Fatalf("RunCRD returned error: %v", err)
	}

	content, ok := result.Files["example.com/v1.star"]
	if !ok {
		t.Fatal("expected example.com/v1.star in result")
	}

	s := string(content)
	// crd-basic.yaml has default: medium for size, default: 3 for replicas, default: true for enabled
	if !strings.Contains(s, "default=") {
		t.Error("expected generated content to contain 'default=' for primitive defaults")
	}
}

func TestRunCRD_MultiFile(t *testing.T) {
	opts := CRDOptions{
		Paths:     []string{testCRDBasicPath, testCRDPreservePath},
		Package:   "test:v1",
		OutputDir: t.TempDir(),
	}

	result, err := RunCRD(opts)
	if err != nil {
		t.Fatalf("RunCRD returned error: %v", err)
	}

	// Both CRDs are in example.com group, v1 version.
	// They should merge into example.com/v1.star.
	content, ok := result.Files["example.com/v1.star"]
	if !ok {
		t.Fatal("expected example.com/v1.star in result")
	}

	s := string(content)
	// Should have Widget from crd-basic.yaml and FlexType from crd-preserve.yaml.
	if !strings.Contains(s, "Widget = schema(") {
		t.Error("expected Widget from crd-basic.yaml")
	}
	if !strings.Contains(s, "FlexType = schema(") {
		t.Error("expected FlexType from crd-preserve.yaml")
	}
}

func TestRunCRD_MultiVersion(t *testing.T) {
	opts := CRDOptions{
		Paths:     []string{testCRDMultiVersionPath},
		Package:   "test:v1",
		OutputDir: t.TempDir(),
	}

	result, err := RunCRD(opts)
	if err != nil {
		t.Fatalf("RunCRD returned error: %v", err)
	}

	// crd-multi-version.yaml has v1 and v1alpha1 versions
	if _, ok := result.Files["example.com/v1.star"]; !ok {
		t.Error("expected example.com/v1.star for v1 version")
	}
	if _, ok := result.Files["example.com/v1alpha1.star"]; !ok {
		t.Error("expected example.com/v1alpha1.star for v1alpha1 version")
	}
}

func TestRunCRD_MultiDoc(t *testing.T) {
	opts := CRDOptions{
		Paths:     []string{testCRDMultiDocPath},
		Package:   "test:v1",
		OutputDir: t.TempDir(),
	}

	result, err := RunCRD(opts)
	if err != nil {
		t.Fatalf("RunCRD returned error: %v", err)
	}

	content, ok := result.Files["example.com/v1.star"]
	if !ok {
		t.Fatal("expected example.com/v1.star in result")
	}

	s := string(content)
	// Multi-doc has Alpha and Beta CRDs (ConfigMap is skipped).
	if !strings.Contains(s, "Alpha = schema(") {
		t.Error("expected Alpha from multi-doc")
	}
	if !strings.Contains(s, "Beta = schema(") {
		t.Error("expected Beta from multi-doc")
	}
}

func TestRunCRD_Determinism(t *testing.T) {
	opts := CRDOptions{
		Paths:     []string{testCRDBasicPath},
		Package:   "test:v1",
		OutputDir: t.TempDir(),
	}

	result1, err := RunCRD(opts)
	if err != nil {
		t.Fatalf("first RunCRD returned error: %v", err)
	}

	opts.OutputDir = t.TempDir()
	result2, err := RunCRD(opts)
	if err != nil {
		t.Fatalf("second RunCRD returned error: %v", err)
	}

	if len(result1.Files) != len(result2.Files) {
		t.Fatalf("file count mismatch: run1=%d, run2=%d", len(result1.Files), len(result2.Files))
	}

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

func TestRunCRD_LoadError(t *testing.T) {
	opts := CRDOptions{
		Paths:     []string{"/nonexistent/crd.yaml"},
		Package:   "test:v1",
		OutputDir: t.TempDir(),
	}

	_, err := RunCRD(opts)
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}

	if !strings.Contains(err.Error(), "loading CRDs") {
		t.Errorf("expected error to mention 'loading CRDs', got: %v", err)
	}
}

func TestRunCRD_FileCount(t *testing.T) {
	opts := CRDOptions{
		Paths:     []string{testCRDMultiVersionPath},
		Package:   "test:v1",
		OutputDir: t.TempDir(),
	}

	result, err := RunCRD(opts)
	if err != nil {
		t.Fatalf("RunCRD returned error: %v", err)
	}

	// crd-multi-version.yaml has 2 versions: v1 and v1alpha1 -> 2 files.
	if result.FileCount != 2 {
		t.Errorf("expected FileCount=2 (2 versions), got %d", result.FileCount)
	}
}
