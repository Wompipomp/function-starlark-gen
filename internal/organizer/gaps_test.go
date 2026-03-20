package organizer

import (
	"testing"

	"github.com/wompipomp/starlark-gen/internal/types"
)

// --- DefinitionKeyToFilePath gaps ---

func TestDefinitionKeyToFilePathSingleSegment(t *testing.T) {
	_, _, err := DefinitionKeyToFilePath("X")
	if err == nil {
		t.Error("expected error for single-segment key, got nil")
	}
}

func TestDefinitionKeyToFilePathTwoSegments(t *testing.T) {
	// Two segments: fewer than 3, should error.
	_, _, err := DefinitionKeyToFilePath("a.b")
	if err == nil {
		t.Error("expected error for two-segment key, got nil")
	}
}

func TestDefinitionKeyToFilePathFallbackThreeSegments(t *testing.T) {
	// Three segments: fallback rule uses last three → group/version.star.
	fp, isSpecial, err := DefinitionKeyToFilePath("mygroup.v1.MyType")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isSpecial {
		t.Error("three-segment key should not be special")
	}
	if fp != "mygroup/v1.star" {
		t.Errorf("filePath = %q, want %q", fp, "mygroup/v1.star")
	}
}

func TestDefinitionKeyToFilePathRuntimePrefix(t *testing.T) {
	fp, isSpecial, err := DefinitionKeyToFilePath("io.k8s.apimachinery.pkg.runtime.RawExtension")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isSpecial {
		t.Error("runtime type should not be special")
	}
	if fp != "runtime/v1.star" {
		t.Errorf("filePath = %q, want %q", fp, "runtime/v1.star")
	}
}

// --- Organize gaps ---

func TestOrganizeEmptyInput(t *testing.T) {
	fm, warnings, err := Organize(nil, "test:v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fm) != 0 {
		t.Errorf("expected empty FileMap, got %d entries", len(fm))
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %d", len(warnings))
	}
}

func TestOrganizeSkipsSpecialTypes(t *testing.T) {
	nodes := []types.TypeNode{
		{
			Name:          "IntOrString",
			DefinitionKey: "io.k8s.apimachinery.pkg.util.intstr.IntOrString",
		},
		{
			Name:          "Deployment",
			DefinitionKey: "io.k8s.api.apps.v1.Deployment",
		},
	}
	fm, _, err := Organize(nodes, "test:v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// IntOrString should be skipped, only Deployment in the map.
	total := 0
	for _, types := range fm {
		total += len(types)
	}
	if total != 1 {
		t.Errorf("expected 1 type in FileMap, got %d", total)
	}
}

func TestOrganizeSetsFilePath(t *testing.T) {
	nodes := []types.TypeNode{
		{
			Name:          "Deployment",
			DefinitionKey: "io.k8s.api.apps.v1.Deployment",
		},
	}
	fm, _, err := Organize(nodes, "test:v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	types, ok := fm["apps/v1.star"]
	if !ok {
		t.Fatal("expected apps/v1.star in FileMap")
	}
	if len(types) != 1 {
		t.Fatalf("expected 1 type in apps/v1.star, got %d", len(types))
	}
	if types[0].FilePath != "apps/v1.star" {
		t.Errorf("FilePath = %q, want %q", types[0].FilePath, "apps/v1.star")
	}
}

// --- TypeNameFromKey gaps ---

func TestTypeNameFromKeyEmptyString(t *testing.T) {
	result := TypeNameFromKey("")
	if result != "" {
		t.Errorf("TypeNameFromKey(\"\") = %q, want empty", result)
	}
}

func TestTypeNameFromKeySingleSegment(t *testing.T) {
	result := TypeNameFromKey("Deployment")
	if result != "Deployment" {
		t.Errorf("TypeNameFromKey(\"Deployment\") = %q, want %q", result, "Deployment")
	}
}

// --- LoadPath gaps ---

func TestLoadPathFormat(t *testing.T) {
	result := LoadPath("schemas-k8s:v1.31", "apps/v1.star")
	expected := "schemas-k8s:v1.31/apps/v1.star"
	if result != expected {
		t.Errorf("LoadPath = %q, want %q", result, expected)
	}
}
