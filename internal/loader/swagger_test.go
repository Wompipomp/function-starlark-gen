package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func testdataPath(name string) string {
	return filepath.Join("..", "..", "testdata", name)
}

func TestLoadSwaggerSuccess(t *testing.T) {
	// Test 1: LoadSwagger returns non-nil model with no error.
	model, err := LoadSwagger(testdataPath("swagger-mini.json"))
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestLoadSwaggerDefinitions(t *testing.T) {
	// Test 2: LoadSwagger returns model whose Definitions contain all expected keys.
	model, err := LoadSwagger(testdataPath("swagger-mini.json"))
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}

	defs := model.Model.Definitions.Definitions
	if defs == nil {
		t.Fatal("expected non-nil Definitions")
	}

	expectedKeys := []string{
		"io.k8s.api.apps.v1.Deployment",
		"io.k8s.api.apps.v1.DeploymentSpec",
		"io.k8s.api.apps.v1.DeploymentStatus",
		"io.k8s.api.apps.v1.DeploymentCondition",
		"io.k8s.api.apps.v1.DeploymentStrategy",
		"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
		"io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector",
		"io.k8s.api.core.v1.PodTemplateSpec",
		"io.k8s.api.core.v1.PodSpec",
		"io.k8s.api.core.v1.Container",
		"io.k8s.apimachinery.pkg.api.resource.Quantity",
		"io.k8s.apimachinery.pkg.util.intstr.IntOrString",
		"io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.JSONSchemaProps",
		"io.k8s.api.core.v1.TypeWithIntOrString",
		"io.k8s.api.core.v1.TypeWithPreserveUnknown",
		"io.k8s.api.core.v1.TypeWithEmbeddedResource",
		"io.k8s.api.networking.v1.NetworkPolicy",
		"io.k8s.api.core.v1.AllOfCompositeType",
		"io.k8s.api.core.v1.OneOfUnionType",
	}

	for _, key := range expectedKeys {
		val := defs.GetOrZero(key)
		if val == nil {
			t.Errorf("missing expected definition: %s", key)
		}
	}

	// Verify total count is at least the expected minimum.
	count := defs.Len()
	if count < 15 {
		t.Errorf("expected at least 15 definitions, got %d", count)
	}
}

func TestLoadSwaggerNonexistent(t *testing.T) {
	// Test 3: LoadSwagger with nonexistent file returns descriptive error.
	_, err := LoadSwagger("nonexistent.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	// Error should mention the file reading issue.
	errMsg := err.Error()
	if len(errMsg) == 0 {
		t.Error("expected non-empty error message")
	}
}

func TestLoadSwaggerMalformed(t *testing.T) {
	// Test 4: LoadSwagger on a malformed JSON file returns a parsing error.
	// Create a temporary malformed file.
	tmpDir := t.TempDir()
	malformedPath := filepath.Join(tmpDir, "malformed.json")
	if err := os.WriteFile(malformedPath, []byte("{not valid json!!!}"), 0o644); err != nil {
		t.Fatalf("failed to write malformed file: %v", err)
	}

	_, err := LoadSwagger(malformedPath)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestLoadSwaggerDefinitionOrdering(t *testing.T) {
	// Test 5: Loaded model's Definitions preserve spec ordering.
	model, err := LoadSwagger(testdataPath("swagger-mini.json"))
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}

	defs := model.Model.Definitions.Definitions

	// The first definition in swagger-mini.json is io.k8s.api.apps.v1.Deployment.
	for key := range defs.KeysFromOldest() {
		if key != "io.k8s.api.apps.v1.Deployment" {
			t.Errorf("expected first definition to be 'io.k8s.api.apps.v1.Deployment', got %q", key)
		}
		break
	}
}
