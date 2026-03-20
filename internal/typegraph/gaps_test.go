package typegraph

import (
	"testing"

	"github.com/wompipomp/starlark-gen/internal/organizer"
	"github.com/wompipomp/starlark-gen/internal/types"
)

// --- SortTypesInFile gaps ---

func TestSortTypesInFileEmpty(t *testing.T) {
	result, err := SortTypesInFile(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for nil input, got %v", result)
	}
}

func TestSortTypesInFileSingleNode(t *testing.T) {
	nodes := []*types.TypeNode{
		{Name: "Foo", DefinitionKey: "io.k8s.api.core.v1.Foo", FilePath: "core/v1.star"},
	}
	result, err := SortTypesInFile(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 node, got %d", len(result))
	}
	if result[0].Name != "Foo" {
		t.Errorf("result[0].Name = %q, want %q", result[0].Name, "Foo")
	}
}

func TestSortTypesInFileNoDeps(t *testing.T) {
	// Multiple nodes with no dependencies should be sorted lexicographically.
	nodes := []*types.TypeNode{
		{Name: "Zulu", DefinitionKey: "io.k8s.api.core.v1.Zulu", FilePath: "core/v1.star"},
		{Name: "Alpha", DefinitionKey: "io.k8s.api.core.v1.Alpha", FilePath: "core/v1.star"},
		{Name: "Mike", DefinitionKey: "io.k8s.api.core.v1.Mike", FilePath: "core/v1.star"},
	}
	result, err := SortTypesInFile(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(result))
	}
	// Lexicographic tiebreaking: Alpha, Mike, Zulu.
	expected := []string{"Alpha", "Mike", "Zulu"}
	for i, name := range expected {
		if result[i].Name != name {
			t.Errorf("result[%d].Name = %q, want %q", i, result[i].Name, name)
		}
	}
}

// --- ValidateLoadDAG gaps ---

func TestValidateLoadDAGEmptyFileMap(t *testing.T) {
	fm := make(organizer.FileMap)
	order, err := ValidateLoadDAG(fm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 0 {
		t.Errorf("expected empty order for empty FileMap, got %v", order)
	}
}

func TestValidateLoadDAGSingleFile(t *testing.T) {
	fm := organizer.FileMap{
		"core/v1.star": {
			{Name: "Pod", DefinitionKey: "io.k8s.api.core.v1.Pod", FilePath: "core/v1.star"},
		},
	}
	order, err := ValidateLoadDAG(fm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 1 {
		t.Fatalf("expected 1 file in order, got %d", len(order))
	}
	if order[0] != "core/v1.star" {
		t.Errorf("order[0] = %q, want %q", order[0], "core/v1.star")
	}
}

func TestValidateLoadDAGNoCrossFileDeps(t *testing.T) {
	// Two files with only intra-file dependencies — no cross-file edges.
	fm := organizer.FileMap{
		"apps/v1.star": {
			{
				Name:          "Deployment",
				DefinitionKey: "io.k8s.api.apps.v1.Deployment",
				FilePath:      "apps/v1.star",
				Dependencies:  []string{"io.k8s.api.apps.v1.DeploymentSpec"},
			},
			{
				Name:          "DeploymentSpec",
				DefinitionKey: "io.k8s.api.apps.v1.DeploymentSpec",
				FilePath:      "apps/v1.star",
			},
		},
		"core/v1.star": {
			{
				Name:          "Pod",
				DefinitionKey: "io.k8s.api.core.v1.Pod",
				FilePath:      "core/v1.star",
			},
		},
	}
	order, err := ValidateLoadDAG(fm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 2 {
		t.Fatalf("expected 2 files in order, got %d", len(order))
	}
	// Lexicographic: apps before core.
	if order[0] != "apps/v1.star" {
		t.Errorf("order[0] = %q, want %q", order[0], "apps/v1.star")
	}
}
