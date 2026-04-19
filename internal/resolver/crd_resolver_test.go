package resolver

import (
	"strings"
	"testing"

	"github.com/wompipomp/starlark-gen/internal/loader"
	"github.com/wompipomp/starlark-gen/internal/types"
)

func crdTestdataPath(name string) string {
	return "../../testdata/" + name
}

func loadTestCRDs(t *testing.T, files ...string) []loader.CRDDocument {
	t.Helper()
	crds, err := loader.LoadCRDs(files)
	if err != nil {
		t.Fatalf("LoadCRDs failed: %v", err)
	}
	return crds
}

func findNode(nodes []types.TypeNode, name string) *types.TypeNode {
	for i := range nodes {
		if nodes[i].Name == name {
			return &nodes[i]
		}
	}
	return nil
}

func findField(node *types.TypeNode, name string) *types.FieldNode {
	for i := range node.Fields {
		if node.Fields[i].Name == name {
			return &node.Fields[i]
		}
	}
	return nil
}

func TestResolveCRDs_Basic(t *testing.T) {
	crds := loadTestCRDs(t, crdTestdataPath("crd-basic.yaml"))
	nodes, _ := ResolveCRDs(crds)

	// Should produce TypeNodes for Widget, WidgetSpec, WidgetConfig, WidgetStatus.
	expectedTypes := []string{"Widget", "WidgetSpec", "WidgetConfig", "WidgetStatus"}
	for _, name := range expectedTypes {
		if findNode(nodes, name) == nil {
			t.Errorf("expected TypeNode %q not found in resolved nodes", name)
		}
	}

	// Widget should have spec and status fields.
	widget := findNode(nodes, "Widget")
	if widget == nil {
		t.Fatal("Widget not found")
	}
	specField := findField(widget, "spec")
	if specField == nil {
		t.Error("Widget missing spec field")
	}
	statusField := findField(widget, "status")
	if statusField == nil {
		t.Error("Widget missing status field")
	}
}

func TestResolveCRDs_SubTypePrefixed(t *testing.T) {
	crds := loadTestCRDs(t, crdTestdataPath("crd-basic.yaml"))
	nodes, _ := ResolveCRDs(crds)

	// Sub-types must use parent kind prefix.
	if findNode(nodes, "Spec") != nil {
		t.Error("found bare 'Spec' -- sub-types must be prefixed with parent kind")
	}
	if findNode(nodes, "Config") != nil {
		t.Error("found bare 'Config' -- sub-types must be prefixed with parent kind")
	}
	if findNode(nodes, "Status") != nil {
		t.Error("found bare 'Status' -- sub-types must be prefixed with parent kind")
	}

	// Confirm kind-prefixed names exist.
	if findNode(nodes, "WidgetSpec") == nil {
		t.Error("expected WidgetSpec (kind-prefixed)")
	}
	if findNode(nodes, "WidgetConfig") == nil {
		t.Error("expected WidgetConfig (kind-prefixed)")
	}
	if findNode(nodes, "WidgetStatus") == nil {
		t.Error("expected WidgetStatus (kind-prefixed)")
	}
}

func TestResolveCRDs_Enums(t *testing.T) {
	crds := loadTestCRDs(t, crdTestdataPath("crd-basic.yaml"))
	nodes, _ := ResolveCRDs(crds)

	// WidgetSpec.size should have enum values.
	spec := findNode(nodes, "WidgetSpec")
	if spec == nil {
		t.Fatal("WidgetSpec not found")
	}
	sizeField := findField(spec, "size")
	if sizeField == nil {
		t.Fatal("WidgetSpec.size not found")
	}
	expectedEnums := []string{"small", "medium", "large"}
	if len(sizeField.EnumValues) != len(expectedEnums) {
		t.Fatalf("expected %d enum values, got %d: %v", len(expectedEnums), len(sizeField.EnumValues), sizeField.EnumValues)
	}
	for i, v := range expectedEnums {
		if sizeField.EnumValues[i] != v {
			t.Errorf("enum[%d] = %q, want %q", i, sizeField.EnumValues[i], v)
		}
	}

	// WidgetStatus.phase should have enum values.
	status := findNode(nodes, "WidgetStatus")
	if status == nil {
		t.Fatal("WidgetStatus not found")
	}
	phaseField := findField(status, "phase")
	if phaseField == nil {
		t.Fatal("WidgetStatus.phase not found")
	}
	expectedPhaseEnums := []string{"Pending", "Running", "Failed"}
	if len(phaseField.EnumValues) != len(expectedPhaseEnums) {
		t.Fatalf("expected %d phase enum values, got %d", len(expectedPhaseEnums), len(phaseField.EnumValues))
	}
	for i, v := range expectedPhaseEnums {
		if phaseField.EnumValues[i] != v {
			t.Errorf("phase enum[%d] = %q, want %q", i, phaseField.EnumValues[i], v)
		}
	}
}

func TestResolveCRDs_Defaults(t *testing.T) {
	crds := loadTestCRDs(t, crdTestdataPath("crd-basic.yaml"))
	nodes, _ := ResolveCRDs(crds)

	spec := findNode(nodes, "WidgetSpec")
	if spec == nil {
		t.Fatal("WidgetSpec not found")
	}

	// size default = "medium"
	sizeField := findField(spec, "size")
	if sizeField == nil {
		t.Fatal("WidgetSpec.size not found")
	}
	if sizeField.Default != "medium" {
		t.Errorf("size default = %v, want %q", sizeField.Default, "medium")
	}

	// replicas default = 3
	replicasField := findField(spec, "replicas")
	if replicasField == nil {
		t.Fatal("WidgetSpec.replicas not found")
	}
	// YAML integers are decoded as int by yaml.v3.
	if replicasField.Default != 3 {
		t.Errorf("replicas default = %v (%T), want 3", replicasField.Default, replicasField.Default)
	}

	// enabled default = true
	enabledField := findField(spec, "enabled")
	if enabledField == nil {
		t.Fatal("WidgetSpec.enabled not found")
	}
	if enabledField.Default != true {
		t.Errorf("enabled default = %v, want true", enabledField.Default)
	}
}

func TestResolveCRDs_PreserveUnknown(t *testing.T) {
	crds := loadTestCRDs(t, crdTestdataPath("crd-preserve.yaml"))
	nodes, _ := ResolveCRDs(crds)

	// FlexType should exist.
	flex := findNode(nodes, "FlexType")
	if flex == nil {
		// May be under FlexTypeSpec since top level is object with spec.
		// Let's check spec node.
		flexSpec := findNode(nodes, "FlexTypeSpec")
		if flexSpec == nil {
			t.Fatal("neither FlexType nor FlexTypeSpec found")
		}
		metaField := findField(flexSpec, "metadata")
		if metaField == nil {
			t.Fatal("FlexTypeSpec.metadata not found")
		}
		if metaField.TypeName != "dict" {
			t.Errorf("metadata type = %q, want dict (preserve-unknown)", metaField.TypeName)
		}
		return
	}

	// Check spec sub-type for metadata field.
	flexSpec := findNode(nodes, "FlexTypeSpec")
	if flexSpec == nil {
		t.Fatal("FlexTypeSpec not found")
	}
	metaField := findField(flexSpec, "metadata")
	if metaField == nil {
		t.Fatal("FlexTypeSpec.metadata not found")
	}
	if metaField.TypeName != "dict" {
		t.Errorf("metadata type = %q, want dict (preserve-unknown)", metaField.TypeName)
	}

	// Should NOT generate sub-types for preserved fields.
	if findNode(nodes, "FlexTypeMetadata") != nil {
		t.Error("should not generate sub-type for x-kubernetes-preserve-unknown-fields")
	}
	if findNode(nodes, "FlexTypeSpecMetadata") != nil {
		t.Error("should not generate sub-type for x-kubernetes-preserve-unknown-fields")
	}
}

func TestResolveCRDs_Dependencies(t *testing.T) {
	crds := loadTestCRDs(t, crdTestdataPath("crd-basic.yaml"))
	nodes, _ := ResolveCRDs(crds)

	widget := findNode(nodes, "Widget")
	if widget == nil {
		t.Fatal("Widget not found")
	}

	// Widget should depend on WidgetSpec definition key.
	foundSpecDep := false
	for _, dep := range widget.Dependencies {
		if dep == "example.com.v1.WidgetSpec" {
			foundSpecDep = true
		}
	}
	if !foundSpecDep {
		t.Errorf("Widget.Dependencies does not contain WidgetSpec key; got %v", widget.Dependencies)
	}

	// WidgetSpec should depend on WidgetConfig.
	spec := findNode(nodes, "WidgetSpec")
	if spec == nil {
		t.Fatal("WidgetSpec not found")
	}
	foundConfigDep := false
	for _, dep := range spec.Dependencies {
		if dep == "example.com.v1.WidgetConfig" {
			foundConfigDep = true
		}
	}
	if !foundConfigDep {
		t.Errorf("WidgetSpec.Dependencies does not contain WidgetConfig key; got %v", spec.Dependencies)
	}
}

func TestResolveCRDs_FilePath(t *testing.T) {
	crds := loadTestCRDs(t, crdTestdataPath("crd-basic.yaml"))
	nodes, _ := ResolveCRDs(crds)

	for _, node := range nodes {
		if node.FilePath != "example.com/v1.star" {
			t.Errorf("TypeNode %q has FilePath %q, want %q", node.Name, node.FilePath, "example.com/v1.star")
		}
	}
}

func TestResolveCRDs_MultiVersion(t *testing.T) {
	crds := loadTestCRDs(t, crdTestdataPath("crd-multi-version.yaml"))
	nodes, _ := ResolveCRDs(crds)

	// Should produce types for both v1 and v1alpha1.
	v1Paths := 0
	v1alpha1Paths := 0
	for _, node := range nodes {
		switch node.FilePath {
		case "example.com/v1.star":
			v1Paths++
		case "example.com/v1alpha1.star":
			v1alpha1Paths++
		}
	}
	if v1Paths == 0 {
		t.Error("no TypeNodes with FilePath example.com/v1.star")
	}
	if v1alpha1Paths == 0 {
		t.Error("no TypeNodes with FilePath example.com/v1alpha1.star")
	}
}

func TestResolveCRDs_RequiredFields(t *testing.T) {
	crds := loadTestCRDs(t, crdTestdataPath("crd-basic.yaml"))
	nodes, _ := ResolveCRDs(crds)

	// Root Widget should have spec as required.
	widget := findNode(nodes, "Widget")
	if widget == nil {
		t.Fatal("Widget not found")
	}
	specField := findField(widget, "spec")
	if specField == nil {
		t.Fatal("Widget.spec not found")
	}
	if !specField.Required {
		t.Error("Widget.spec should be required")
	}

	// WidgetSpec should have size as required.
	spec := findNode(nodes, "WidgetSpec")
	if spec == nil {
		t.Fatal("WidgetSpec not found")
	}
	sizeField := findField(spec, "size")
	if sizeField == nil {
		t.Fatal("WidgetSpec.size not found")
	}
	if !sizeField.Required {
		t.Error("WidgetSpec.size should be required")
	}

	// WidgetSpec.replicas should NOT be required.
	replicasField := findField(spec, "replicas")
	if replicasField == nil {
		t.Fatal("WidgetSpec.replicas not found")
	}
	if replicasField.Required {
		t.Error("WidgetSpec.replicas should not be required")
	}
}

func TestResolveCRDs_ListType(t *testing.T) {
	// Create a CRD with a list of primitives inline to test.
	doc := loader.CRDDocument{
		APIVersion: "apiextensions.k8s.io/v1",
		Kind:       "CustomResourceDefinition",
		Spec: loader.CRDSpec{
			Group: "test.io",
			Names: loader.CRDNames{Kind: "ListTest"},
			Versions: []loader.CRDVersion{
				{
					Name:    "v1",
					Served:  true,
					Storage: true,
					Schema: &loader.CRDValidation{
						OpenAPIV3Schema: &loader.JSONSchemaProps{
							Type: "object",
							Properties: map[string]*loader.JSONSchemaProps{
								"tags": {
									Type: "array",
									Items: &loader.JSONSchemaProps{
										Type: "string",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	nodes, _ := ResolveCRDs([]loader.CRDDocument{doc})

	listTest := findNode(nodes, "ListTest")
	if listTest == nil {
		t.Fatal("ListTest not found")
	}
	tagsField := findField(listTest, "tags")
	if tagsField == nil {
		t.Fatal("ListTest.tags not found")
	}
	if tagsField.TypeName != "list" {
		t.Errorf("tags TypeName = %q, want list", tagsField.TypeName)
	}
	// Items should be empty for primitive list items (no schema ref needed).
	if tagsField.Items != "" {
		t.Errorf("tags Items = %q, want empty for primitive list items", tagsField.Items)
	}
}

func TestResolveCRDs_IntOrString(t *testing.T) {
	trueVal := true
	doc := loader.CRDDocument{
		APIVersion: "apiextensions.k8s.io/v1",
		Kind:       "CustomResourceDefinition",
		Spec: loader.CRDSpec{
			Group: "test.io",
			Names: loader.CRDNames{Kind: "IntOrStringTest"},
			Versions: []loader.CRDVersion{
				{
					Name:    "v1",
					Served:  true,
					Storage: true,
					Schema: &loader.CRDValidation{
						OpenAPIV3Schema: &loader.JSONSchemaProps{
							Type: "object",
							Properties: map[string]*loader.JSONSchemaProps{
								"port": {
									XIntOrString: &trueVal,
								},
							},
						},
					},
				},
			},
		},
	}

	nodes, _ := ResolveCRDs([]loader.CRDDocument{doc})

	node := findNode(nodes, "IntOrStringTest")
	if node == nil {
		t.Fatal("IntOrStringTest not found")
	}
	portField := findField(node, "port")
	if portField == nil {
		t.Fatal("IntOrStringTest.port not found")
	}
	if portField.TypeName != "" {
		t.Errorf("port TypeName = %q, want empty for int-or-string", portField.TypeName)
	}
	if portField.Description != "int or string" {
		t.Errorf("port Description = %q, want %q", portField.Description, "int or string")
	}
}

func TestResolveCRDs_EmbeddedResource(t *testing.T) {
	trueVal := true
	doc := loader.CRDDocument{
		APIVersion: "apiextensions.k8s.io/v1",
		Kind:       "CustomResourceDefinition",
		Spec: loader.CRDSpec{
			Group: "test.io",
			Names: loader.CRDNames{Kind: "EmbedTest"},
			Versions: []loader.CRDVersion{
				{
					Name:    "v1",
					Served:  true,
					Storage: true,
					Schema: &loader.CRDValidation{
						OpenAPIV3Schema: &loader.JSONSchemaProps{
							Type: "object",
							Properties: map[string]*loader.JSONSchemaProps{
								"resource": {
									XEmbeddedResource: &trueVal,
								},
							},
						},
					},
				},
			},
		},
	}

	nodes, _ := ResolveCRDs([]loader.CRDDocument{doc})

	node := findNode(nodes, "EmbedTest")
	if node == nil {
		t.Fatal("EmbedTest not found")
	}
	resField := findField(node, "resource")
	if resField == nil {
		t.Fatal("EmbedTest.resource not found")
	}
	if resField.TypeName != "dict" {
		t.Errorf("resource TypeName = %q, want dict", resField.TypeName)
	}
}

func TestResolveCRDs_AllOfMerge(t *testing.T) {
	doc := loader.CRDDocument{
		APIVersion: "apiextensions.k8s.io/v1",
		Kind:       "CustomResourceDefinition",
		Spec: loader.CRDSpec{
			Group: "test.io",
			Names: loader.CRDNames{Kind: "AllOfTest"},
			Versions: []loader.CRDVersion{
				{
					Name:    "v1",
					Served:  true,
					Storage: true,
					Schema: &loader.CRDValidation{
						OpenAPIV3Schema: &loader.JSONSchemaProps{
							Type: "object",
							AllOf: []*loader.JSONSchemaProps{
								{
									Properties: map[string]*loader.JSONSchemaProps{
										"name": {Type: "string"},
									},
									Required: []string{"name"},
								},
								{
									Properties: map[string]*loader.JSONSchemaProps{
										"age": {Type: "integer"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	nodes, _ := ResolveCRDs([]loader.CRDDocument{doc})

	node := findNode(nodes, "AllOfTest")
	if node == nil {
		t.Fatal("AllOfTest not found")
	}
	nameField := findField(node, "name")
	if nameField == nil {
		t.Fatal("AllOfTest.name not found (should be merged from allOf[0])")
	}
	if !nameField.Required {
		t.Error("name should be required (from allOf[0].required)")
	}
	ageField := findField(node, "age")
	if ageField == nil {
		t.Fatal("AllOfTest.age not found (should be merged from allOf[1])")
	}
}

func TestResolveCRDs_ArrayOfObjects(t *testing.T) {
	doc := loader.CRDDocument{
		APIVersion: "apiextensions.k8s.io/v1",
		Kind:       "CustomResourceDefinition",
		Spec: loader.CRDSpec{
			Group: "test.io",
			Names: loader.CRDNames{Kind: "ArrayObjTest"},
			Versions: []loader.CRDVersion{
				{
					Name:    "v1",
					Served:  true,
					Storage: true,
					Schema: &loader.CRDValidation{
						OpenAPIV3Schema: &loader.JSONSchemaProps{
							Type: "object",
							Properties: map[string]*loader.JSONSchemaProps{
								"items": {
									Type: "array",
									Items: &loader.JSONSchemaProps{
										Type: "object",
										Properties: map[string]*loader.JSONSchemaProps{
											"key":   {Type: "string"},
											"value": {Type: "string"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	nodes, _ := ResolveCRDs([]loader.CRDDocument{doc})

	parent := findNode(nodes, "ArrayObjTest")
	if parent == nil {
		t.Fatal("ArrayObjTest not found")
	}
	itemsField := findField(parent, "items")
	if itemsField == nil {
		t.Fatal("ArrayObjTest.items not found")
	}
	if itemsField.TypeName != "list" {
		t.Errorf("items TypeName = %q, want list", itemsField.TypeName)
	}
	if itemsField.Items == "" {
		t.Error("items Items should be non-empty (referencing sub-type)")
	}

	// Sub-type should be created.
	subType := findNode(nodes, "ArrayObjTestItems")
	if subType == nil {
		t.Fatal("ArrayObjTestItems sub-type not found")
	}
	if findField(subType, "key") == nil {
		t.Error("ArrayObjTestItems.key not found")
	}
	if findField(subType, "value") == nil {
		t.Error("ArrayObjTestItems.value not found")
	}
}

func TestResolveCRDs_SkippedVersion(t *testing.T) {
	doc := loader.CRDDocument{
		APIVersion: "apiextensions.k8s.io/v1",
		Kind:       "CustomResourceDefinition",
		Spec: loader.CRDSpec{
			Group: "test.io",
			Names: loader.CRDNames{Kind: "SkipTest"},
			Versions: []loader.CRDVersion{
				{
					Name:    "v1",
					Served:  false, // not served
					Storage: true,
					Schema: &loader.CRDValidation{
						OpenAPIV3Schema: &loader.JSONSchemaProps{
							Type: "object",
							Properties: map[string]*loader.JSONSchemaProps{
								"name": {Type: "string"},
							},
						},
					},
				},
			},
		},
	}

	nodes, _ := ResolveCRDs([]loader.CRDDocument{doc})

	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes for unserved version, got %d", len(nodes))
	}
}

func TestResolveCRDs_MissingSchemaWarning(t *testing.T) {
	doc := loader.CRDDocument{
		APIVersion: "apiextensions.k8s.io/v1",
		Kind:       "CustomResourceDefinition",
		Spec: loader.CRDSpec{
			Group: "test.io",
			Names: loader.CRDNames{Kind: "NoSchema"},
			Versions: []loader.CRDVersion{
				{
					Name:   "v1",
					Served: true,
					Schema: nil, // no schema
				},
			},
		},
	}

	nodes, warnings := ResolveCRDs([]loader.CRDDocument{doc})

	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes for missing schema, got %d", len(nodes))
	}
	if len(warnings) == 0 {
		t.Fatal("expected warning for missing schema")
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "no openAPIV3Schema") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning mentioning 'no openAPIV3Schema', got: %v", warnings)
	}
}

func TestResolveCRDs_ComplexDefaultWarning(t *testing.T) {
	doc := loader.CRDDocument{
		APIVersion: "apiextensions.k8s.io/v1",
		Kind:       "CustomResourceDefinition",
		Spec: loader.CRDSpec{
			Group: "test.io",
			Names: loader.CRDNames{Kind: "ComplexDefault"},
			Versions: []loader.CRDVersion{
				{
					Name:    "v1",
					Served:  true,
					Storage: true,
					Schema: &loader.CRDValidation{
						OpenAPIV3Schema: &loader.JSONSchemaProps{
							Type: "object",
							Properties: map[string]*loader.JSONSchemaProps{
								"config": {
									Type:    "string",
									Default: map[string]interface{}{"key": "val"},
								},
							},
						},
					},
				},
			},
		},
	}

	nodes, warnings := ResolveCRDs([]loader.CRDDocument{doc})

	// Should still produce a node.
	if len(nodes) == 0 {
		t.Fatal("expected at least 1 node")
	}
	// The complex default should be skipped with a warning.
	if len(warnings) == 0 {
		t.Fatal("expected warning for complex default")
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "complex default") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning mentioning 'complex default', got: %v", warnings)
	}

	// The field should not have Default set.
	node := findNode(nodes, "ComplexDefault")
	if node == nil {
		t.Fatal("ComplexDefault not found")
	}
	configField := findField(node, "config")
	if configField == nil {
		t.Fatal("ComplexDefault.config not found")
	}
	if configField.Default != nil {
		t.Error("config Default should be nil for complex defaults")
	}
}

func TestMapCRDType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"string", "string"},
		{"integer", "int"},
		{"number", "float"},
		{"boolean", "bool"},
		{"object", "dict"},
		{"array", "list"},
		{"unknown", ""},
		{"", ""},
	}

	for _, tc := range tests {
		got := mapCRDType(tc.input)
		if got != tc.want {
			t.Errorf("mapCRDType(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestResolveCRDs_MapType(t *testing.T) {
	doc := loader.CRDDocument{
		APIVersion: "apiextensions.k8s.io/v1",
		Kind:       "CustomResourceDefinition",
		Spec: loader.CRDSpec{
			Group: "test.io",
			Names: loader.CRDNames{Kind: "MapTest"},
			Versions: []loader.CRDVersion{
				{
					Name:    "v1",
					Served:  true,
					Storage: true,
					Schema: &loader.CRDValidation{
						OpenAPIV3Schema: &loader.JSONSchemaProps{
							Type: "object",
							Properties: map[string]*loader.JSONSchemaProps{
								"labels": {
									Type: "object",
									AdditionalProperties: &loader.JSONSchemaPropsOrBool{
										Allowed: true,
										Schema: &loader.JSONSchemaProps{
											Type: "string",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	nodes, _ := ResolveCRDs([]loader.CRDDocument{doc})

	mapTest := findNode(nodes, "MapTest")
	if mapTest == nil {
		t.Fatal("MapTest not found")
	}
	labelsField := findField(mapTest, "labels")
	if labelsField == nil {
		t.Fatal("MapTest.labels not found")
	}
	if !labelsField.IsMap {
		t.Error("labels should have IsMap=true")
	}
	if labelsField.TypeName != "dict" {
		t.Errorf("labels TypeName = %q, want dict", labelsField.TypeName)
	}
}

// TestResolveCRDsTopLevelGVKDefaults: top-level CRD kind types get
// apiVersion/kind fields prepended with defaults derived from group/version/kind.
// Sub-types (WidgetSpec, WidgetStatus, WidgetConfig) must not get them.
func TestResolveCRDsTopLevelGVKDefaults(t *testing.T) {
	crds := loadTestCRDs(t, crdTestdataPath("crd-basic.yaml"))
	nodes, _ := ResolveCRDs(crds)

	widget := findNode(nodes, "Widget")
	if widget == nil {
		t.Fatal("Widget not found")
	}

	if len(widget.Fields) < 2 {
		t.Fatalf("Widget has %d fields, expected at least apiVersion/kind/spec/status", len(widget.Fields))
	}
	if widget.Fields[0].Name != "apiVersion" {
		t.Errorf("Fields[0].Name = %q, want apiVersion", widget.Fields[0].Name)
	}
	if widget.Fields[0].Default != "example.com/v1" {
		t.Errorf("Fields[0].Default = %v, want %q", widget.Fields[0].Default, "example.com/v1")
	}
	if widget.Fields[0].TypeName != "string" {
		t.Errorf("Fields[0].TypeName = %q, want string", widget.Fields[0].TypeName)
	}
	if widget.Fields[1].Name != "kind" {
		t.Errorf("Fields[1].Name = %q, want kind", widget.Fields[1].Name)
	}
	if widget.Fields[1].Default != "Widget" {
		t.Errorf("Fields[1].Default = %v, want %q", widget.Fields[1].Default, "Widget")
	}

	// Sub-types must not get apiVersion/kind fields.
	for _, subName := range []string{"WidgetSpec", "WidgetStatus", "WidgetConfig"} {
		sub := findNode(nodes, subName)
		if sub == nil {
			t.Errorf("%s not found", subName)
			continue
		}
		for _, f := range sub.Fields {
			if f.Name == "apiVersion" || f.Name == "kind" {
				t.Errorf("sub-type %s unexpectedly has %s field", subName, f.Name)
			}
		}
	}
}
