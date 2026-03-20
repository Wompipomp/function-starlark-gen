package organizer

import (
	"testing"
)

func TestDefinitionKeyToFilePath_AppsV1Deployment(t *testing.T) {
	fp, isSpecial, err := DefinitionKeyToFilePath("io.k8s.api.apps.v1.Deployment")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isSpecial {
		t.Fatal("expected isSpecial=false for Deployment")
	}
	if fp != "apps/v1.star" {
		t.Errorf("expected apps/v1.star, got %s", fp)
	}
}

func TestDefinitionKeyToFilePath_CoreV1PodTemplateSpec(t *testing.T) {
	fp, isSpecial, err := DefinitionKeyToFilePath("io.k8s.api.core.v1.PodTemplateSpec")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isSpecial {
		t.Fatal("expected isSpecial=false for PodTemplateSpec")
	}
	if fp != "core/v1.star" {
		t.Errorf("expected core/v1.star, got %s", fp)
	}
}

func TestDefinitionKeyToFilePath_MetaV1ObjectMeta(t *testing.T) {
	fp, isSpecial, err := DefinitionKeyToFilePath("io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isSpecial {
		t.Fatal("expected isSpecial=false for ObjectMeta")
	}
	if fp != "meta/v1.star" {
		t.Errorf("expected meta/v1.star, got %s", fp)
	}
}

func TestDefinitionKeyToFilePath_Quantity(t *testing.T) {
	_, isSpecial, err := DefinitionKeyToFilePath("io.k8s.apimachinery.pkg.api.resource.Quantity")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isSpecial {
		t.Fatal("expected isSpecial=true for Quantity")
	}
}

func TestDefinitionKeyToFilePath_IntOrString(t *testing.T) {
	_, isSpecial, err := DefinitionKeyToFilePath("io.k8s.apimachinery.pkg.util.intstr.IntOrString")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isSpecial {
		t.Fatal("expected isSpecial=true for IntOrString")
	}
}

func TestDefinitionKeyToFilePath_RuntimeRawExtension(t *testing.T) {
	fp, isSpecial, err := DefinitionKeyToFilePath("io.k8s.apimachinery.pkg.runtime.RawExtension")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isSpecial {
		t.Fatal("expected isSpecial=false for RawExtension")
	}
	if fp != "runtime/v1.star" {
		t.Errorf("expected runtime/v1.star, got %s", fp)
	}
}

func TestDefinitionKeyToFilePath_KubeAggregatorAPIService(t *testing.T) {
	fp, isSpecial, err := DefinitionKeyToFilePath("io.k8s.kube-aggregator.pkg.apis.apiregistration.v1.APIService")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isSpecial {
		t.Fatal("expected isSpecial=false for APIService")
	}
	if fp != "apiregistration/v1.star" {
		t.Errorf("expected apiregistration/v1.star, got %s", fp)
	}
}

func TestDefinitionKeyToFilePath_UnknownPrefix(t *testing.T) {
	fp, isSpecial, err := DefinitionKeyToFilePath("com.example.custom.v2.MyType")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isSpecial {
		t.Fatal("expected isSpecial=false for unknown prefix")
	}
	// Fallback should produce something reasonable from the last segments.
	if fp == "" {
		t.Error("expected non-empty file path for unknown prefix")
	}
	// Expect "custom/v2.star" from fallback (last three segments: custom.v2.MyType -> group=custom, version=v2).
	if fp != "custom/v2.star" {
		t.Errorf("expected custom/v2.star, got %s", fp)
	}
}

func TestLoadPath(t *testing.T) {
	lp := LoadPath("schemas-k8s:v1.31", "apps/v1.star")
	if lp != "schemas-k8s:v1.31/apps/v1.star" {
		t.Errorf("expected schemas-k8s:v1.31/apps/v1.star, got %s", lp)
	}
}

func TestTypeNameFromKey(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"io.k8s.api.apps.v1.Deployment", "Deployment"},
		{"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta", "ObjectMeta"},
		{"SimpleType", "SimpleType"},
	}
	for _, tt := range tests {
		got := TypeNameFromKey(tt.key)
		if got != tt.want {
			t.Errorf("TypeNameFromKey(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}
