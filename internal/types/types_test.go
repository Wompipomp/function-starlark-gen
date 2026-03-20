package types

import (
	"encoding/json"
	"os"
	"testing"
)

func TestTypeNodeFields(t *testing.T) {
	// Test 1: TypeNode struct has all required fields.
	node := TypeNode{
		Name:          "Deployment",
		DefinitionKey: "io.k8s.api.apps.v1.Deployment",
		Description:   "Deployment enables declarative updates for Pods and ReplicaSets.",
		Fields: []FieldNode{
			{Name: "replicas", TypeName: "int"},
		},
		Dependencies:  []string{"io.k8s.api.apps.v1.DeploymentSpec"},
		FilePath:      "apps/v1.star",
		IsCircularRef: false,
		SpecialType:   SpecialNone,
	}

	if node.Name != "Deployment" {
		t.Errorf("expected Name 'Deployment', got %q", node.Name)
	}
	if node.DefinitionKey != "io.k8s.api.apps.v1.Deployment" {
		t.Errorf("expected DefinitionKey, got %q", node.DefinitionKey)
	}
	if node.Description == "" {
		t.Error("expected non-empty Description")
	}
	if len(node.Fields) != 1 {
		t.Errorf("expected 1 field, got %d", len(node.Fields))
	}
	if len(node.Dependencies) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(node.Dependencies))
	}
	if node.FilePath != "apps/v1.star" {
		t.Errorf("expected FilePath 'apps/v1.star', got %q", node.FilePath)
	}
	if node.IsCircularRef {
		t.Error("expected IsCircularRef to be false")
	}
	if node.SpecialType != SpecialNone {
		t.Errorf("expected SpecialNone, got %d", node.SpecialType)
	}
}

func TestFieldNodeFields(t *testing.T) {
	// Test 2: FieldNode struct has all required fields.
	field := FieldNode{
		Name:        "selector",
		TypeName:    "",
		SchemaRef:   "io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector",
		Required:    true,
		Description: "Label query over pods",
		Items:       "",
		IsMap:       false,
		EnumValues:  []string{"Always", "OnFailure", "Never"},
	}

	if field.Name != "selector" {
		t.Errorf("expected Name 'selector', got %q", field.Name)
	}
	if field.TypeName != "" {
		t.Errorf("expected empty TypeName for ref field, got %q", field.TypeName)
	}
	if field.SchemaRef == "" {
		t.Error("expected non-empty SchemaRef")
	}
	if !field.Required {
		t.Error("expected Required to be true")
	}
	if field.Description == "" {
		t.Error("expected non-empty Description")
	}
	if field.Items != "" {
		t.Errorf("expected empty Items, got %q", field.Items)
	}
	if field.IsMap {
		t.Error("expected IsMap to be false")
	}
	if len(field.EnumValues) != 3 {
		t.Errorf("expected 3 EnumValues, got %d", len(field.EnumValues))
	}
}

func TestSpecialTypeConstants(t *testing.T) {
	// Test 3: SpecialType constants exist with correct values.
	if SpecialNone != 0 {
		t.Errorf("expected SpecialNone == 0, got %d", SpecialNone)
	}
	if SpecialIntOrString != 1 {
		t.Errorf("expected SpecialIntOrString == 1, got %d", SpecialIntOrString)
	}
	if SpecialQuantity != 2 {
		t.Errorf("expected SpecialQuantity == 2, got %d", SpecialQuantity)
	}
	if SpecialPreserveUnknown != 3 {
		t.Errorf("expected SpecialPreserveUnknown == 3, got %d", SpecialPreserveUnknown)
	}
	if SpecialEmbeddedResource != 4 {
		t.Errorf("expected SpecialEmbeddedResource == 4, got %d", SpecialEmbeddedResource)
	}

	// Verify they are distinct.
	seen := map[SpecialType]bool{}
	for _, st := range []SpecialType{SpecialNone, SpecialIntOrString, SpecialQuantity, SpecialPreserveUnknown, SpecialEmbeddedResource} {
		if seen[st] {
			t.Errorf("duplicate SpecialType value: %d", st)
		}
		seen[st] = true
	}
}

func TestSwaggerMiniIsValidJSON(t *testing.T) {
	// Test 4: testdata/swagger-mini.json is valid JSON.
	data, err := os.ReadFile("../../testdata/swagger-mini.json")
	if err != nil {
		t.Fatalf("failed to read swagger-mini.json: %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("swagger-mini.json is not valid JSON: %v", err)
	}

	// Check required Swagger 2.0 fields.
	if _, ok := doc["swagger"]; !ok {
		t.Error("missing 'swagger' field")
	}
	if _, ok := doc["info"]; !ok {
		t.Error("missing 'info' field")
	}
	if _, ok := doc["paths"]; !ok {
		t.Error("missing 'paths' field")
	}
	if _, ok := doc["definitions"]; !ok {
		t.Error("missing 'definitions' field")
	}

	// Verify definitions count is at least 15 (covering all requirement categories).
	defs, ok := doc["definitions"].(map[string]interface{})
	if !ok {
		t.Fatal("'definitions' is not an object")
	}
	if len(defs) < 15 {
		t.Errorf("expected at least 15 definitions, got %d", len(defs))
	}
}
