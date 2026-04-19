package resolver

import (
	"sort"
	"testing"
	"time"

	"github.com/wompipomp/starlark-gen/internal/loader"
	"github.com/wompipomp/starlark-gen/internal/types"
)

// Test 1: Resolve produces a TypeNode for each definition in swagger-mini.json (count matches)
func TestResolveDefinitionCount(t *testing.T) {
	model, err := loader.LoadSwagger("../../testdata/swagger-mini.json")
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}

	typeNodes, warnings := Resolve(model)

	// swagger-mini.json has 30 definitions
	if len(typeNodes) != 30 {
		t.Errorf("expected 30 TypeNodes, got %d", len(typeNodes))
		for _, tn := range typeNodes {
			t.Logf("  TypeNode: %s (%s)", tn.Name, tn.DefinitionKey)
		}
	}

	// Warnings are informational, just log them
	for _, w := range warnings {
		t.Logf("warning: %s", w)
	}
}

// Test 2: A type referencing another type via $ref has the referenced definition key in its
// Dependencies and the field's SchemaRef is set correctly
func TestResolveRefDependencies(t *testing.T) {
	model, err := loader.LoadSwagger("../../testdata/swagger-mini.json")
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}

	typeNodes, _ := Resolve(model)

	// Find Deployment -- it references ObjectMeta, DeploymentSpec, DeploymentStatus via $ref
	var deployment *types.TypeNode
	for i := range typeNodes {
		if typeNodes[i].DefinitionKey == "io.k8s.api.apps.v1.Deployment" {
			deployment = &typeNodes[i]
			break
		}
	}
	if deployment == nil {
		t.Fatal("Deployment TypeNode not found")
	}

	// Check Dependencies contains the referenced types
	depSet := make(map[string]bool)
	for _, dep := range deployment.Dependencies {
		depSet[dep] = true
	}

	expectedDeps := []string{
		"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
		"io.k8s.api.apps.v1.DeploymentSpec",
		"io.k8s.api.apps.v1.DeploymentStatus",
	}
	for _, exp := range expectedDeps {
		if !depSet[exp] {
			t.Errorf("Deployment missing dependency %q", exp)
		}
	}

	// Check that the spec field has SchemaRef set correctly
	var specField *types.FieldNode
	for i := range deployment.Fields {
		if deployment.Fields[i].Name == "spec" {
			specField = &deployment.Fields[i]
			break
		}
	}
	if specField == nil {
		t.Fatal("Deployment missing 'spec' field")
	}
	if specField.SchemaRef != "io.k8s.api.apps.v1.DeploymentSpec" {
		t.Errorf("spec field SchemaRef = %q, want %q", specField.SchemaRef, "io.k8s.api.apps.v1.DeploymentSpec")
	}
}

// Test 3: A self-referencing type (circular) gets IsCircularRef=true and the back-edge
// field has TypeName="dict" (not infinite recursion)
func TestResolveCircularRef(t *testing.T) {
	model, err := loader.LoadSwagger("../../testdata/swagger-mini.json")
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}

	typeNodes, _ := Resolve(model)

	// JSONSchemaProps references itself via additionalProperties in "properties" field
	var jsonSchemaProps *types.TypeNode
	for i := range typeNodes {
		if typeNodes[i].DefinitionKey == "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.JSONSchemaProps" {
			jsonSchemaProps = &typeNodes[i]
			break
		}
	}
	if jsonSchemaProps == nil {
		t.Fatal("JSONSchemaProps TypeNode not found")
	}

	if !jsonSchemaProps.IsCircularRef {
		t.Error("JSONSchemaProps should have IsCircularRef=true")
	}

	// The "properties" field should be dict type (breaking the circular ref)
	var propsField *types.FieldNode
	for i := range jsonSchemaProps.Fields {
		if jsonSchemaProps.Fields[i].Name == "properties" {
			propsField = &jsonSchemaProps.Fields[i]
			break
		}
	}
	if propsField == nil {
		t.Fatal("JSONSchemaProps missing 'properties' field")
	}
	if propsField.TypeName != "dict" {
		t.Errorf("properties field TypeName = %q, want %q", propsField.TypeName, "dict")
	}
}

// Test 4: An allOf type merges properties from ALL allOf entries
func TestResolveAllOf(t *testing.T) {
	model, err := loader.LoadSwagger("../../testdata/swagger-mini.json")
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}

	typeNodes, _ := Resolve(model)

	// AllOfCompositeType uses allOf with a $ref to ObjectMeta + inline properties
	var composite *types.TypeNode
	for i := range typeNodes {
		if typeNodes[i].DefinitionKey == "io.k8s.api.core.v1.AllOfCompositeType" {
			composite = &typeNodes[i]
			break
		}
	}
	if composite == nil {
		t.Fatal("AllOfCompositeType TypeNode not found")
	}

	// Should have fields from ObjectMeta (name, namespace, labels, annotations, ownerReferences)
	// AND inline properties (customField, replicas)
	fieldNames := make(map[string]bool)
	for _, f := range composite.Fields {
		fieldNames[f.Name] = true
	}

	expectedFields := []string{"name", "namespace", "labels", "annotations", "ownerReferences", "customField", "replicas"}
	for _, name := range expectedFields {
		if !fieldNames[name] {
			t.Errorf("AllOfCompositeType missing field %q", name)
		}
	}
}

// Test 5: A oneOf/anyOf type produces a field with TypeName="" and Description listing the variant types
func TestResolveOneOf(t *testing.T) {
	model, err := loader.LoadSwagger("../../testdata/swagger-mini.json")
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}

	typeNodes, _ := Resolve(model)

	// OneOfUnionType has a "handler" field with oneOf
	var unionType *types.TypeNode
	for i := range typeNodes {
		if typeNodes[i].DefinitionKey == "io.k8s.api.core.v1.OneOfUnionType" {
			unionType = &typeNodes[i]
			break
		}
	}
	if unionType == nil {
		t.Fatal("OneOfUnionType TypeNode not found")
	}

	var handlerField *types.FieldNode
	for i := range unionType.Fields {
		if unionType.Fields[i].Name == "handler" {
			handlerField = &unionType.Fields[i]
			break
		}
	}
	if handlerField == nil {
		t.Fatal("OneOfUnionType missing 'handler' field")
	}

	// TypeName should be "" (gradual typing)
	if handlerField.TypeName != "" {
		t.Errorf("handler field TypeName = %q, want empty string", handlerField.TypeName)
	}
}

// Test 6: An additionalProperties type produces a field with TypeName="dict" and IsMap=true
func TestResolveAdditionalProperties(t *testing.T) {
	model, err := loader.LoadSwagger("../../testdata/swagger-mini.json")
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}

	typeNodes, _ := Resolve(model)

	// ObjectMeta has "labels" with additionalProperties
	var objectMeta *types.TypeNode
	for i := range typeNodes {
		if typeNodes[i].DefinitionKey == "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta" {
			objectMeta = &typeNodes[i]
			break
		}
	}
	if objectMeta == nil {
		t.Fatal("ObjectMeta TypeNode not found")
	}

	var labelsField *types.FieldNode
	for i := range objectMeta.Fields {
		if objectMeta.Fields[i].Name == "labels" {
			labelsField = &objectMeta.Fields[i]
			break
		}
	}
	if labelsField == nil {
		t.Fatal("ObjectMeta missing 'labels' field")
	}

	if labelsField.TypeName != "dict" {
		t.Errorf("labels field TypeName = %q, want %q", labelsField.TypeName, "dict")
	}
	if !labelsField.IsMap {
		t.Error("labels field should have IsMap=true")
	}
}

// Test 7: Primitive type mapping: string->string, integer->int, number->float, boolean->bool,
// array->list, object with no properties->dict
func TestResolvePrimitiveTypes(t *testing.T) {
	model, err := loader.LoadSwagger("../../testdata/swagger-mini.json")
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}

	typeNodes, _ := Resolve(model)

	// Find DeploymentSpec -- has integer field (replicas)
	var deploySpec *types.TypeNode
	for i := range typeNodes {
		if typeNodes[i].DefinitionKey == "io.k8s.api.apps.v1.DeploymentSpec" {
			deploySpec = &typeNodes[i]
			break
		}
	}
	if deploySpec == nil {
		t.Fatal("DeploymentSpec TypeNode not found")
	}

	// replicas is integer -> "int"
	var replicasField *types.FieldNode
	for i := range deploySpec.Fields {
		if deploySpec.Fields[i].Name == "replicas" {
			replicasField = &deploySpec.Fields[i]
			break
		}
	}
	if replicasField == nil {
		t.Fatal("DeploymentSpec missing 'replicas' field")
	}
	if replicasField.TypeName != "int" {
		t.Errorf("replicas TypeName = %q, want %q", replicasField.TypeName, "int")
	}

	// Find ConfigMapVolumeSource -- has boolean field (optional)
	var configMap *types.TypeNode
	for i := range typeNodes {
		if typeNodes[i].DefinitionKey == "io.k8s.api.core.v1.ConfigMapVolumeSource" {
			configMap = &typeNodes[i]
			break
		}
	}
	if configMap == nil {
		t.Fatal("ConfigMapVolumeSource TypeNode not found")
	}

	var optionalField *types.FieldNode
	for i := range configMap.Fields {
		if configMap.Fields[i].Name == "optional" {
			optionalField = &configMap.Fields[i]
			break
		}
	}
	if optionalField == nil {
		t.Fatal("ConfigMapVolumeSource missing 'optional' field")
	}
	if optionalField.TypeName != "bool" {
		t.Errorf("optional TypeName = %q, want %q", optionalField.TypeName, "bool")
	}

	// DeploymentStatus has array field (conditions) -> "list"
	var deployStatus *types.TypeNode
	for i := range typeNodes {
		if typeNodes[i].DefinitionKey == "io.k8s.api.apps.v1.DeploymentStatus" {
			deployStatus = &typeNodes[i]
			break
		}
	}
	if deployStatus == nil {
		t.Fatal("DeploymentStatus TypeNode not found")
	}

	var conditionsField *types.FieldNode
	for i := range deployStatus.Fields {
		if deployStatus.Fields[i].Name == "conditions" {
			conditionsField = &deployStatus.Fields[i]
			break
		}
	}
	if conditionsField == nil {
		t.Fatal("DeploymentStatus missing 'conditions' field")
	}
	if conditionsField.TypeName != "list" {
		t.Errorf("conditions TypeName = %q, want %q", conditionsField.TypeName, "list")
	}
	if conditionsField.Items != "io.k8s.api.apps.v1.DeploymentCondition" {
		t.Errorf("conditions Items = %q, want %q", conditionsField.Items, "io.k8s.api.apps.v1.DeploymentCondition")
	}

	// JSONSchemaPropsOrArray is object with no sub-properties -> "dict"
	var jspaNode *types.TypeNode
	for i := range typeNodes {
		if typeNodes[i].DefinitionKey == "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.JSONSchemaPropsOrArray" {
			jspaNode = &typeNodes[i]
			break
		}
	}
	if jspaNode == nil {
		t.Fatal("JSONSchemaPropsOrArray TypeNode not found")
	}
	// Should have no fields (empty object)
	if len(jspaNode.Fields) != 0 {
		t.Errorf("JSONSchemaPropsOrArray should have 0 fields, got %d", len(jspaNode.Fields))
	}
}

// Test 8: Required fields array propagates to FieldNode.Required=true for matching field names
func TestResolveRequiredFields(t *testing.T) {
	model, err := loader.LoadSwagger("../../testdata/swagger-mini.json")
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}

	typeNodes, _ := Resolve(model)

	// DeploymentSpec requires "selector" and "template"
	var deploySpec *types.TypeNode
	for i := range typeNodes {
		if typeNodes[i].DefinitionKey == "io.k8s.api.apps.v1.DeploymentSpec" {
			deploySpec = &typeNodes[i]
			break
		}
	}
	if deploySpec == nil {
		t.Fatal("DeploymentSpec TypeNode not found")
	}

	requiredFields := map[string]bool{
		"selector": true,
		"template": true,
	}
	optionalFields := map[string]bool{
		"replicas":        true,
		"strategy":        true,
		"minReadySeconds": true,
	}

	for _, f := range deploySpec.Fields {
		if requiredFields[f.Name] && !f.Required {
			t.Errorf("field %q should be required", f.Name)
		}
		if optionalFields[f.Name] && f.Required {
			t.Errorf("field %q should NOT be required", f.Name)
		}
	}
}

// Test 9: Resolver does NOT stack overflow on the circular reference type (completes in <5 seconds)
func TestResolveNoStackOverflow(t *testing.T) {
	model, err := loader.LoadSwagger("../../testdata/swagger-mini.json")
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}

	done := make(chan struct{})
	go func() {
		Resolve(model)
		close(done)
	}()

	select {
	case <-done:
		// Success -- completed without stack overflow
	case <-time.After(5 * time.Second):
		t.Fatal("Resolve did not complete within 5 seconds -- possible stack overflow or infinite loop")
	}
}

// Test: Dependencies are sorted for determinism
func TestResolveDependenciesSorted(t *testing.T) {
	model, err := loader.LoadSwagger("../../testdata/swagger-mini.json")
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}

	typeNodes, _ := Resolve(model)

	for _, tn := range typeNodes {
		if len(tn.Dependencies) > 1 {
			sorted := make([]string, len(tn.Dependencies))
			copy(sorted, tn.Dependencies)
			sort.Strings(sorted)
			for i := range sorted {
				if sorted[i] != tn.Dependencies[i] {
					t.Errorf("TypeNode %q Dependencies not sorted: %v", tn.DefinitionKey, tn.Dependencies)
					break
				}
			}
		}
	}
}

// TestResolveDeploymentGVKDefaults: top-level resource types read
// x-kubernetes-group-version-kind and default apiVersion/kind so callers
// only need to set metadata/spec.
func TestResolveDeploymentGVKDefaults(t *testing.T) {
	model, err := loader.LoadSwagger("../../testdata/swagger-mini.json")
	if err != nil {
		t.Fatalf("LoadSwagger failed: %v", err)
	}

	typeNodes, _ := Resolve(model)

	var dep *types.TypeNode
	for i := range typeNodes {
		if typeNodes[i].DefinitionKey == "io.k8s.api.apps.v1.Deployment" {
			dep = &typeNodes[i]
			break
		}
	}
	if dep == nil {
		t.Fatal("Deployment TypeNode not found")
	}

	fieldDefault := func(name string) interface{} {
		for _, f := range dep.Fields {
			if f.Name == name {
				return f.Default
			}
		}
		return nil
	}

	if got := fieldDefault("apiVersion"); got != "apps/v1" {
		t.Errorf("Deployment.apiVersion Default = %v, want %q", got, "apps/v1")
	}
	if got := fieldDefault("kind"); got != "Deployment" {
		t.Errorf("Deployment.kind Default = %v, want %q", got, "Deployment")
	}

	// Sub-types must NOT pick up defaults.
	for _, n := range typeNodes {
		if n.DefinitionKey == "io.k8s.api.apps.v1.Deployment" {
			continue
		}
		for _, f := range n.Fields {
			if (f.Name == "apiVersion" || f.Name == "kind") && f.Default != nil {
				t.Errorf("sub-type %s has unexpected default on %s: %v",
					n.DefinitionKey, f.Name, f.Default)
			}
		}
	}
}
