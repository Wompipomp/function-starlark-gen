package typegraph

import (
	"strings"
	"testing"

	"github.com/wompipomp/starlark-gen/internal/organizer"
	"github.com/wompipomp/starlark-gen/internal/types"
)

func TestSortTypesInFile_SimpleDependency(t *testing.T) {
	// A depends on B -> result should be [B, A]
	nodeB := &types.TypeNode{
		Name:          "B",
		DefinitionKey: "io.k8s.api.apps.v1.B",
		FilePath:      "apps/v1.star",
	}
	nodeA := &types.TypeNode{
		Name:          "A",
		DefinitionKey: "io.k8s.api.apps.v1.A",
		FilePath:      "apps/v1.star",
		Dependencies:  []string{"io.k8s.api.apps.v1.B"},
	}

	input := []*types.TypeNode{nodeA, nodeB}
	sorted, err := SortTypesInFile(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sorted) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(sorted))
	}
	if sorted[0].Name != "B" {
		t.Errorf("expected B first, got %s", sorted[0].Name)
	}
	if sorted[1].Name != "A" {
		t.Errorf("expected A second, got %s", sorted[1].Name)
	}
}

func TestSortTypesInFile_NoDependencies(t *testing.T) {
	// No dependencies -> lexicographic order
	nodeC := &types.TypeNode{
		Name:          "C",
		DefinitionKey: "io.k8s.api.apps.v1.C",
		FilePath:      "apps/v1.star",
	}
	nodeA := &types.TypeNode{
		Name:          "A",
		DefinitionKey: "io.k8s.api.apps.v1.A",
		FilePath:      "apps/v1.star",
	}
	nodeB := &types.TypeNode{
		Name:          "B",
		DefinitionKey: "io.k8s.api.apps.v1.B",
		FilePath:      "apps/v1.star",
	}

	input := []*types.TypeNode{nodeC, nodeA, nodeB}
	sorted, err := SortTypesInFile(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"A", "B", "C"}
	for i, n := range sorted {
		if n.Name != expected[i] {
			t.Errorf("position %d: expected %s, got %s", i, expected[i], n.Name)
		}
	}
}

func TestSortTypesInFile_DiamondDependency(t *testing.T) {
	// A->B, A->C, B->D, C->D => D first, then B and C in lex order, then A
	nodeD := &types.TypeNode{
		Name:          "D",
		DefinitionKey: "io.k8s.api.apps.v1.D",
		FilePath:      "apps/v1.star",
	}
	nodeB := &types.TypeNode{
		Name:          "B",
		DefinitionKey: "io.k8s.api.apps.v1.B",
		FilePath:      "apps/v1.star",
		Dependencies:  []string{"io.k8s.api.apps.v1.D"},
	}
	nodeC := &types.TypeNode{
		Name:          "C",
		DefinitionKey: "io.k8s.api.apps.v1.C",
		FilePath:      "apps/v1.star",
		Dependencies:  []string{"io.k8s.api.apps.v1.D"},
	}
	nodeA := &types.TypeNode{
		Name:          "A",
		DefinitionKey: "io.k8s.api.apps.v1.A",
		FilePath:      "apps/v1.star",
		Dependencies:  []string{"io.k8s.api.apps.v1.B", "io.k8s.api.apps.v1.C"},
	}

	input := []*types.TypeNode{nodeA, nodeB, nodeC, nodeD}
	sorted, err := SortTypesInFile(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sorted) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(sorted))
	}
	if sorted[0].Name != "D" {
		t.Errorf("expected D first, got %s", sorted[0].Name)
	}
	if sorted[1].Name != "B" {
		t.Errorf("expected B second, got %s", sorted[1].Name)
	}
	if sorted[2].Name != "C" {
		t.Errorf("expected C third, got %s", sorted[2].Name)
	}
	if sorted[3].Name != "A" {
		t.Errorf("expected A fourth, got %s", sorted[3].Name)
	}
}

func TestSortTypesInFile_IgnoresCrossFileDependencies(t *testing.T) {
	// A depends on something in a different file -> that dependency is ignored for intra-file sorting
	nodeA := &types.TypeNode{
		Name:          "A",
		DefinitionKey: "io.k8s.api.apps.v1.A",
		FilePath:      "apps/v1.star",
		Dependencies:  []string{"io.k8s.api.core.v1.Pod"}, // different file
	}
	nodeB := &types.TypeNode{
		Name:          "B",
		DefinitionKey: "io.k8s.api.apps.v1.B",
		FilePath:      "apps/v1.star",
	}

	input := []*types.TypeNode{nodeA, nodeB}
	sorted, err := SortTypesInFile(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Since A's dependency is cross-file (ignored), both have no intra-file deps.
	// Should be sorted lexicographically: A, B
	if sorted[0].Name != "A" {
		t.Errorf("expected A first (cross-file dep ignored), got %s", sorted[0].Name)
	}
	if sorted[1].Name != "B" {
		t.Errorf("expected B second, got %s", sorted[1].Name)
	}
}

func TestValidateLoadDAG_NoCycle(t *testing.T) {
	// apps/v1.star depends on meta/v1.star (no cycle)
	fm := organizer.FileMap{
		"meta/v1.star": {
			&types.TypeNode{
				Name:          "ObjectMeta",
				DefinitionKey: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
				FilePath:      "meta/v1.star",
			},
		},
		"apps/v1.star": {
			&types.TypeNode{
				Name:          "Deployment",
				DefinitionKey: "io.k8s.api.apps.v1.Deployment",
				FilePath:      "apps/v1.star",
				Dependencies:  []string{"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta"},
			},
		},
	}

	order, err := ValidateLoadDAG(fm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// meta/v1.star has no deps, should come first.
	if len(order) != 2 {
		t.Fatalf("expected 2 files, got %d", len(order))
	}
	if order[0] != "meta/v1.star" {
		t.Errorf("expected meta/v1.star first, got %s", order[0])
	}
	if order[1] != "apps/v1.star" {
		t.Errorf("expected apps/v1.star second, got %s", order[1])
	}
}

func TestValidateLoadDAG_CircularDependency(t *testing.T) {
	// A.star loads B.star, B.star loads A.star -> cycle
	fm := organizer.FileMap{
		"a.star": {
			&types.TypeNode{
				Name:          "TypeA",
				DefinitionKey: "io.k8s.api.a.v1.TypeA",
				FilePath:      "a.star",
				Dependencies:  []string{"io.k8s.api.b.v1.TypeB"},
			},
		},
		"b.star": {
			&types.TypeNode{
				Name:          "TypeB",
				DefinitionKey: "io.k8s.api.b.v1.TypeB",
				FilePath:      "b.star",
				Dependencies:  []string{"io.k8s.api.a.v1.TypeA"},
			},
		},
	}

	_, err := ValidateLoadDAG(fm)
	if err == nil {
		t.Fatal("expected error for circular dependency, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("expected error mentioning circular, got: %v", err)
	}
}

func TestSortTypesInFile_Deterministic(t *testing.T) {
	// Run the same sort twice and verify identical output
	makeInput := func() []*types.TypeNode {
		return []*types.TypeNode{
			{
				Name:          "Z",
				DefinitionKey: "io.k8s.api.apps.v1.Z",
				FilePath:      "apps/v1.star",
				Dependencies:  []string{"io.k8s.api.apps.v1.M"},
			},
			{
				Name:          "M",
				DefinitionKey: "io.k8s.api.apps.v1.M",
				FilePath:      "apps/v1.star",
			},
			{
				Name:          "A",
				DefinitionKey: "io.k8s.api.apps.v1.A",
				FilePath:      "apps/v1.star",
			},
		}
	}

	sorted1, err1 := SortTypesInFile(makeInput())
	if err1 != nil {
		t.Fatalf("unexpected error run 1: %v", err1)
	}

	sorted2, err2 := SortTypesInFile(makeInput())
	if err2 != nil {
		t.Fatalf("unexpected error run 2: %v", err2)
	}

	if len(sorted1) != len(sorted2) {
		t.Fatalf("different lengths: %d vs %d", len(sorted1), len(sorted2))
	}
	for i := range sorted1 {
		if sorted1[i].Name != sorted2[i].Name {
			t.Errorf("position %d: %s vs %s", i, sorted1[i].Name, sorted2[i].Name)
		}
	}
}

func TestValidateLoadDAG_ReturnsSortedFileOrder(t *testing.T) {
	// Three files: c depends on b, b depends on a -> order: a, b, c
	fm := organizer.FileMap{
		"c.star": {
			&types.TypeNode{
				Name:          "TypeC",
				DefinitionKey: "io.k8s.api.c.v1.TypeC",
				FilePath:      "c.star",
				Dependencies:  []string{"io.k8s.api.b.v1.TypeB"},
			},
		},
		"b.star": {
			&types.TypeNode{
				Name:          "TypeB",
				DefinitionKey: "io.k8s.api.b.v1.TypeB",
				FilePath:      "b.star",
				Dependencies:  []string{"io.k8s.api.a.v1.TypeA"},
			},
		},
		"a.star": {
			&types.TypeNode{
				Name:          "TypeA",
				DefinitionKey: "io.k8s.api.a.v1.TypeA",
				FilePath:      "a.star",
			},
		},
	}

	order, err := ValidateLoadDAG(fm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("expected 3 files, got %d", len(order))
	}
	if order[0] != "a.star" {
		t.Errorf("expected a.star first, got %s", order[0])
	}
	if order[1] != "b.star" {
		t.Errorf("expected b.star second, got %s", order[1])
	}
	if order[2] != "c.star" {
		t.Errorf("expected c.star third, got %s", order[2])
	}
}
