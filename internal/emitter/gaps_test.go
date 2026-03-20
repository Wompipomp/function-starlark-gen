package emitter

import (
	"testing"

	"github.com/wompipomp/starlark-gen/internal/types"
)

// --- buildFieldDoc gaps ---

func TestBuildFieldDocNoTypeNoDesc(t *testing.T) {
	f := types.FieldNode{Name: "x"}
	allNodes := map[string]*types.TypeNode{}
	doc := buildFieldDoc(f, allNodes)
	if doc != "" {
		t.Errorf("buildFieldDoc(no type, no desc) = %q, want empty", doc)
	}
}

func TestBuildFieldDocTypeOnly(t *testing.T) {
	f := types.FieldNode{Name: "x", TypeName: "string"}
	allNodes := map[string]*types.TypeNode{}
	doc := buildFieldDoc(f, allNodes)
	if doc != "string" {
		t.Errorf("buildFieldDoc(type only) = %q, want %q", doc, "string")
	}
}

func TestBuildFieldDocDescOnly(t *testing.T) {
	// Gradual typing (TypeName=""), no SchemaRef, with description.
	f := types.FieldNode{Name: "x", Description: "A flexible field"}
	allNodes := map[string]*types.TypeNode{}
	doc := buildFieldDoc(f, allNodes)
	if doc != "A flexible field" {
		t.Errorf("buildFieldDoc(desc only) = %q, want %q", doc, "A flexible field")
	}
}

func TestBuildFieldDocWithRequired(t *testing.T) {
	f := types.FieldNode{Name: "x", TypeName: "string", Description: "Name", Required: true}
	allNodes := map[string]*types.TypeNode{}
	doc := buildFieldDoc(f, allNodes)
	expected := "string - Name (required)"
	if doc != expected {
		t.Errorf("buildFieldDoc(required) = %q, want %q", doc, expected)
	}
}

func TestBuildFieldDocWithEnum(t *testing.T) {
	f := types.FieldNode{Name: "x", TypeName: "string", Description: "Policy", EnumValues: []string{"Always", "Never"}}
	allNodes := map[string]*types.TypeNode{}
	doc := buildFieldDoc(f, allNodes)
	expected := "string - Policy. One of: Always, Never"
	if doc != expected {
		t.Errorf("buildFieldDoc(enum) = %q, want %q", doc, expected)
	}
}

func TestBuildFieldDocWithSchemaRef(t *testing.T) {
	f := types.FieldNode{
		Name:        "spec",
		SchemaRef:   "io.k8s.api.apps.v1.DeploymentSpec",
		Description: "The desired state",
	}
	allNodes := map[string]*types.TypeNode{
		"io.k8s.api.apps.v1.DeploymentSpec": {Name: "DeploymentSpec"},
	}
	doc := buildFieldDoc(f, allNodes)
	expected := "DeploymentSpec - The desired state"
	if doc != expected {
		t.Errorf("buildFieldDoc(schemaRef) = %q, want %q", doc, expected)
	}
}

func TestBuildFieldDocRequiredAndEnum(t *testing.T) {
	f := types.FieldNode{
		Name:        "policy",
		TypeName:    "string",
		Description: "Restart policy",
		Required:    true,
		EnumValues:  []string{"Always", "OnFailure", "Never"},
	}
	allNodes := map[string]*types.TypeNode{}
	doc := buildFieldDoc(f, allNodes)
	expected := "string - Restart policy (required). One of: Always, OnFailure, Never"
	if doc != expected {
		t.Errorf("buildFieldDoc(required+enum) = %q, want %q", doc, expected)
	}
}

// --- Emit gaps ---

func TestEmitEmptyFileOrder(t *testing.T) {
	fileMap := map[string][]*types.TypeNode{
		"apps/v1.star": {{Name: "Deployment", DefinitionKey: "io.k8s.api.apps.v1.Deployment"}},
	}
	result, err := Emit(fileMap, []string{}, "test:v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 files in result with empty fileOrder, got %d", len(result))
	}
}

func TestEmitFileOrderSkipsMissingFile(t *testing.T) {
	fileMap := map[string][]*types.TypeNode{
		"apps/v1.star": {{Name: "Deployment", DefinitionKey: "io.k8s.api.apps.v1.Deployment", FilePath: "apps/v1.star"}},
	}
	// fileOrder references a file not in fileMap — should skip silently.
	result, err := Emit(fileMap, []string{"nonexistent/v1.star", "apps/v1.star"}, "test:v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 file in result, got %d", len(result))
	}
}

// --- WriteFiles gaps ---

func TestWriteFilesEmptyResult(t *testing.T) {
	result := make(EmitResult)
	fileCount, schemaCount, err := WriteFiles(result, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fileCount != 0 {
		t.Errorf("fileCount = %d, want 0", fileCount)
	}
	if schemaCount != 0 {
		t.Errorf("schemaCount = %d, want 0", schemaCount)
	}
}

// --- SummaryLine gaps ---

func TestSummaryLineFormat(t *testing.T) {
	line := SummaryLine(5, 42, "./output")
	expected := "Generated 5 files (42 schemas) in ./output"
	if line != expected {
		t.Errorf("SummaryLine = %q, want %q", line, expected)
	}
}
