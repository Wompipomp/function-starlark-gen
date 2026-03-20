package resolver

import (
	"testing"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/wompipomp/starlark-gen/internal/loader"
	"github.com/wompipomp/starlark-gen/internal/types"
	"go.yaml.in/yaml/v4"
)

// buildSchemaWithExtension creates a minimal Schema with the given extension key set to true.
func buildSchemaWithExtension(key string) *highbase.Schema {
	extensions := orderedmap.New[string, *yaml.Node]()
	extensions.Set(key, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!bool",
		Value: "true",
	})

	return &highbase.Schema{
		Extensions: extensions,
	}
}

// Test 1: CheckExtensions returns SpecialIntOrString for x-kubernetes-int-or-string
func TestCheckExtensionsIntOrString(t *testing.T) {
	schema := buildSchemaWithExtension("x-kubernetes-int-or-string")
	result := CheckExtensions(schema)
	if result != types.SpecialIntOrString {
		t.Errorf("CheckExtensions for x-kubernetes-int-or-string = %v, want SpecialIntOrString (%v)", result, types.SpecialIntOrString)
	}
}

// Test 2: CheckExtensions returns SpecialPreserveUnknown for x-kubernetes-preserve-unknown-fields
func TestCheckExtensionsPreserveUnknown(t *testing.T) {
	schema := buildSchemaWithExtension("x-kubernetes-preserve-unknown-fields")
	result := CheckExtensions(schema)
	if result != types.SpecialPreserveUnknown {
		t.Errorf("CheckExtensions for x-kubernetes-preserve-unknown-fields = %v, want SpecialPreserveUnknown (%v)", result, types.SpecialPreserveUnknown)
	}
}

// Test 3: CheckExtensions returns SpecialEmbeddedResource for x-kubernetes-embedded-resource
func TestCheckExtensionsEmbeddedResource(t *testing.T) {
	schema := buildSchemaWithExtension("x-kubernetes-embedded-resource")
	result := CheckExtensions(schema)
	if result != types.SpecialEmbeddedResource {
		t.Errorf("CheckExtensions for x-kubernetes-embedded-resource = %v, want SpecialEmbeddedResource (%v)", result, types.SpecialEmbeddedResource)
	}
}

// Test 4: IsSpecialType returns SpecialIntOrString for canonical IntOrString path
func TestIsSpecialTypeIntOrString(t *testing.T) {
	result := IsSpecialType("io.k8s.apimachinery.pkg.util.intstr.IntOrString")
	if result != types.SpecialIntOrString {
		t.Errorf("IsSpecialType(IntOrString) = %v, want SpecialIntOrString", result)
	}
}

// Test 5: IsSpecialType returns SpecialQuantity for canonical Quantity path
func TestIsSpecialTypeQuantity(t *testing.T) {
	result := IsSpecialType("io.k8s.apimachinery.pkg.api.resource.Quantity")
	if result != types.SpecialQuantity {
		t.Errorf("IsSpecialType(Quantity) = %v, want SpecialQuantity", result)
	}
}

// Test 6: IsSpecialType returns SpecialNone for regular definition keys
func TestIsSpecialTypeRegular(t *testing.T) {
	result := IsSpecialType("io.k8s.api.apps.v1.Deployment")
	if result != types.SpecialNone {
		t.Errorf("IsSpecialType(Deployment) = %v, want SpecialNone", result)
	}
}

// Test 7: A field referencing IntOrString by def key gets appropriate handling
func TestResolveIntOrStringByPath(t *testing.T) {
	model, err := loader.LoadSwagger("../../testdata/swagger-mini.json")
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}

	typeNodes, _ := Resolve(model)

	// ServicePort has a targetPort field that $ref's IntOrString
	var servicePort *types.TypeNode
	for i := range typeNodes {
		if typeNodes[i].DefinitionKey == "io.k8s.api.core.v1.ServicePort" {
			servicePort = &typeNodes[i]
			break
		}
	}
	if servicePort == nil {
		t.Fatal("ServicePort TypeNode not found")
	}

	var targetPortField *types.FieldNode
	for i := range servicePort.Fields {
		if servicePort.Fields[i].Name == "targetPort" {
			targetPortField = &servicePort.Fields[i]
			break
		}
	}
	if targetPortField == nil {
		t.Fatal("ServicePort missing 'targetPort' field")
	}

	// Should have TypeName="" (gradual typing for int-or-string)
	if targetPortField.TypeName != "" {
		t.Errorf("targetPort TypeName = %q, want empty string (gradual typing)", targetPortField.TypeName)
	}

	// Description should mention int or string
	if targetPortField.Description == "" {
		t.Error("targetPort Description should not be empty")
	}
}

// Test 8: A field referencing Quantity by def key gets appropriate handling
func TestResolveQuantityByPath(t *testing.T) {
	model, err := loader.LoadSwagger("../../testdata/swagger-mini.json")
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}

	typeNodes, _ := Resolve(model)

	// Find the Quantity type itself -- it should be special
	var quantityNode *types.TypeNode
	for i := range typeNodes {
		if typeNodes[i].DefinitionKey == "io.k8s.apimachinery.pkg.api.resource.Quantity" {
			quantityNode = &typeNodes[i]
			break
		}
	}
	if quantityNode == nil {
		t.Fatal("Quantity TypeNode not found")
	}
	if quantityNode.SpecialType != types.SpecialQuantity {
		t.Errorf("Quantity SpecialType = %v, want SpecialQuantity", quantityNode.SpecialType)
	}

	// ResourceRequirements has limits/requests with additionalProperties.$ref to Quantity
	var resourceReqs *types.TypeNode
	for i := range typeNodes {
		if typeNodes[i].DefinitionKey == "io.k8s.api.core.v1.ResourceRequirements" {
			resourceReqs = &typeNodes[i]
			break
		}
	}
	if resourceReqs == nil {
		t.Fatal("ResourceRequirements TypeNode not found")
	}

	var limitsField *types.FieldNode
	for i := range resourceReqs.Fields {
		if resourceReqs.Fields[i].Name == "limits" {
			limitsField = &resourceReqs.Fields[i]
			break
		}
	}
	if limitsField == nil {
		t.Fatal("ResourceRequirements missing 'limits' field")
	}

	// limits is additionalProperties with $ref to Quantity -> should be dict/map
	if limitsField.TypeName != "dict" {
		t.Errorf("limits TypeName = %q, want %q", limitsField.TypeName, "dict")
	}
	if !limitsField.IsMap {
		t.Error("limits should have IsMap=true")
	}
}

// Test 9: A type with x-kubernetes-preserve-unknown-fields produces TypeName="dict" and
// resolver does NOT recurse into its sub-properties
func TestResolvePreserveUnknownFields(t *testing.T) {
	model, err := loader.LoadSwagger("../../testdata/swagger-mini.json")
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}

	typeNodes, _ := Resolve(model)

	// TypeWithPreserveUnknown has x-kubernetes-preserve-unknown-fields
	var preserveType *types.TypeNode
	for i := range typeNodes {
		if typeNodes[i].DefinitionKey == "io.k8s.api.core.v1.TypeWithPreserveUnknown" {
			preserveType = &typeNodes[i]
			break
		}
	}
	if preserveType == nil {
		t.Fatal("TypeWithPreserveUnknown TypeNode not found")
	}

	if preserveType.SpecialType != types.SpecialPreserveUnknown {
		t.Errorf("TypeWithPreserveUnknown SpecialType = %v, want SpecialPreserveUnknown", preserveType.SpecialType)
	}

	// Should have NO fields since recursion was stopped
	if len(preserveType.Fields) != 0 {
		t.Errorf("TypeWithPreserveUnknown should have 0 fields (no recursion), got %d", len(preserveType.Fields))
		for _, f := range preserveType.Fields {
			t.Logf("  unexpected field: %s", f.Name)
		}
	}
}

// Test: TypeWithIntOrString gets SpecialIntOrString via extension
func TestResolveIntOrStringExtension(t *testing.T) {
	model, err := loader.LoadSwagger("../../testdata/swagger-mini.json")
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}

	typeNodes, _ := Resolve(model)

	var intOrStrType *types.TypeNode
	for i := range typeNodes {
		if typeNodes[i].DefinitionKey == "io.k8s.api.core.v1.TypeWithIntOrString" {
			intOrStrType = &typeNodes[i]
			break
		}
	}
	if intOrStrType == nil {
		t.Fatal("TypeWithIntOrString TypeNode not found")
	}

	if intOrStrType.SpecialType != types.SpecialIntOrString {
		t.Errorf("TypeWithIntOrString SpecialType = %v, want SpecialIntOrString", intOrStrType.SpecialType)
	}
}

// Test: TypeWithEmbeddedResource gets SpecialEmbeddedResource via extension
func TestResolveEmbeddedResourceExtension(t *testing.T) {
	model, err := loader.LoadSwagger("../../testdata/swagger-mini.json")
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}

	typeNodes, _ := Resolve(model)

	var embeddedType *types.TypeNode
	for i := range typeNodes {
		if typeNodes[i].DefinitionKey == "io.k8s.api.core.v1.TypeWithEmbeddedResource" {
			embeddedType = &typeNodes[i]
			break
		}
	}
	if embeddedType == nil {
		t.Fatal("TypeWithEmbeddedResource TypeNode not found")
	}

	if embeddedType.SpecialType != types.SpecialEmbeddedResource {
		t.Errorf("TypeWithEmbeddedResource SpecialType = %v, want SpecialEmbeddedResource", embeddedType.SpecialType)
	}

	// Should have NO fields since recursion was stopped
	if len(embeddedType.Fields) != 0 {
		t.Errorf("TypeWithEmbeddedResource should have 0 fields (no recursion), got %d", len(embeddedType.Fields))
	}
}

// Test: IntOrString type detected by canonical definition path
func TestResolveIntOrStringDefinitionPath(t *testing.T) {
	model, err := loader.LoadSwagger("../../testdata/swagger-mini.json")
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}

	typeNodes, _ := Resolve(model)

	var intOrStr *types.TypeNode
	for i := range typeNodes {
		if typeNodes[i].DefinitionKey == "io.k8s.apimachinery.pkg.util.intstr.IntOrString" {
			intOrStr = &typeNodes[i]
			break
		}
	}
	if intOrStr == nil {
		t.Fatal("IntOrString TypeNode not found")
	}

	if intOrStr.SpecialType != types.SpecialIntOrString {
		t.Errorf("IntOrString SpecialType = %v, want SpecialIntOrString", intOrStr.SpecialType)
	}
}

// Test: SpecialTypeToFieldNode produces correct field nodes for each special type
func TestSpecialTypeToFieldNode(t *testing.T) {
	tests := []struct {
		special     types.SpecialType
		name        string
		wantType    string
		wantDescHas string
	}{
		{types.SpecialIntOrString, "myField", "", "int or string"},
		{types.SpecialQuantity, "myField", "", "quantity"},
		{types.SpecialPreserveUnknown, "myField", "dict", "preserve-unknown-fields"},
		{types.SpecialEmbeddedResource, "myField", "dict", "embedded-resource"},
	}

	for _, tc := range tests {
		t.Run(tc.name+"-"+tc.wantDescHas, func(t *testing.T) {
			field := SpecialTypeToFieldNode(tc.special, tc.name)
			if field.TypeName != tc.wantType {
				t.Errorf("TypeName = %q, want %q", field.TypeName, tc.wantType)
			}
			if field.Name != tc.name {
				t.Errorf("Name = %q, want %q", field.Name, tc.name)
			}
		})
	}
}
