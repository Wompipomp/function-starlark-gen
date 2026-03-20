package organizer

import (
	"testing"

	"github.com/wompipomp/starlark-gen/internal/types"
)

func TestOrganize_AssignsFilePath(t *testing.T) {
	nodes := []types.TypeNode{
		{
			Name:          "Deployment",
			DefinitionKey: "io.k8s.api.apps.v1.Deployment",
		},
		{
			Name:          "Pod",
			DefinitionKey: "io.k8s.api.core.v1.Pod",
		},
	}

	fm, warnings, err := Organize(nodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) > 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	// Check that FilePath is set on returned nodes.
	appsNodes := fm["apps/v1.star"]
	if len(appsNodes) != 1 {
		t.Fatalf("expected 1 node in apps/v1.star, got %d", len(appsNodes))
	}
	if appsNodes[0].FilePath != "apps/v1.star" {
		t.Errorf("expected FilePath=apps/v1.star, got %s", appsNodes[0].FilePath)
	}
	if appsNodes[0].Name != "Deployment" {
		t.Errorf("expected Deployment, got %s", appsNodes[0].Name)
	}

	coreNodes := fm["core/v1.star"]
	if len(coreNodes) != 1 {
		t.Fatalf("expected 1 node in core/v1.star, got %d", len(coreNodes))
	}
	if coreNodes[0].Name != "Pod" {
		t.Errorf("expected Pod, got %s", coreNodes[0].Name)
	}
}

func TestOrganize_GroupsSameFile(t *testing.T) {
	nodes := []types.TypeNode{
		{
			Name:          "Deployment",
			DefinitionKey: "io.k8s.api.apps.v1.Deployment",
		},
		{
			Name:          "ReplicaSet",
			DefinitionKey: "io.k8s.api.apps.v1.ReplicaSet",
		},
		{
			Name:          "DaemonSet",
			DefinitionKey: "io.k8s.api.apps.v1.DaemonSet",
		},
	}

	fm, _, err := Organize(nodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	appsNodes := fm["apps/v1.star"]
	if len(appsNodes) != 3 {
		t.Fatalf("expected 3 nodes in apps/v1.star, got %d", len(appsNodes))
	}

	// Verify insertion order is maintained.
	names := []string{appsNodes[0].Name, appsNodes[1].Name, appsNodes[2].Name}
	expected := []string{"Deployment", "ReplicaSet", "DaemonSet"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("position %d: expected %s, got %s", i, expected[i], name)
		}
	}
}

func TestOrganize_SkipsSpecialTypes(t *testing.T) {
	nodes := []types.TypeNode{
		{
			Name:          "Deployment",
			DefinitionKey: "io.k8s.api.apps.v1.Deployment",
		},
		{
			Name:          "IntOrString",
			DefinitionKey: "io.k8s.apimachinery.pkg.util.intstr.IntOrString",
			SpecialType:   types.SpecialIntOrString,
		},
		{
			Name:          "Quantity",
			DefinitionKey: "io.k8s.apimachinery.pkg.api.resource.Quantity",
			SpecialType:   types.SpecialQuantity,
		},
	}

	fm, _, err := Organize(nodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Special types should not appear in any file.
	for filePath, fileNodes := range fm {
		for _, n := range fileNodes {
			if n.Name == "IntOrString" || n.Name == "Quantity" {
				t.Errorf("special type %s should not be in file %s", n.Name, filePath)
			}
		}
	}

	// Deployment should still be present.
	appsNodes := fm["apps/v1.star"]
	if len(appsNodes) != 1 {
		t.Fatalf("expected 1 node in apps/v1.star, got %d", len(appsNodes))
	}
}

func TestOrganize_OCILoadPath(t *testing.T) {
	lp := LoadPath("schemas-k8s:v1.31", "apps/v1.star")
	expected := "schemas-k8s:v1.31/apps/v1.star"
	if lp != expected {
		t.Errorf("expected %s, got %s", expected, lp)
	}
}
