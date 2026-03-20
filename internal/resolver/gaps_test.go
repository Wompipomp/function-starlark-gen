package resolver

import (
	"testing"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/wompipomp/starlark-gen/internal/types"
	"go.yaml.in/yaml/v4"
)

// --- extractRefKey gaps ---

func TestExtractRefKeyNonMatchingPrefix(t *testing.T) {
	// Refs that don't start with #/definitions/ should return empty string.
	cases := []string{
		"",
		"#/components/schemas/Foo",
		"some-random-string",
		"definitions/Foo",
	}
	for _, ref := range cases {
		result := extractRefKey(ref)
		if result != "" {
			t.Errorf("extractRefKey(%q) = %q, want empty string", ref, result)
		}
	}
}

func TestExtractRefKeyValidPrefix(t *testing.T) {
	result := extractRefKey("#/definitions/io.k8s.api.apps.v1.Deployment")
	if result != "io.k8s.api.apps.v1.Deployment" {
		t.Errorf("extractRefKey = %q, want %q", result, "io.k8s.api.apps.v1.Deployment")
	}
}

// --- shortName gaps ---

func TestShortNameSingleSegment(t *testing.T) {
	result := shortName("Deployment")
	if result != "Deployment" {
		t.Errorf("shortName(\"Deployment\") = %q, want %q", result, "Deployment")
	}
}

func TestShortNameEmptyString(t *testing.T) {
	result := shortName("")
	if result != "" {
		t.Errorf("shortName(\"\") = %q, want empty string", result)
	}
}

// --- appendUnique gaps ---

func TestAppendUniqueExistingValue(t *testing.T) {
	slice := []string{"a", "b", "c"}
	result := appendUnique(slice, "b")
	if len(result) != 3 {
		t.Errorf("appendUnique with existing value: len = %d, want 3", len(result))
	}
}

func TestAppendUniqueNewValue(t *testing.T) {
	slice := []string{"a", "b"}
	result := appendUnique(slice, "c")
	if len(result) != 3 {
		t.Errorf("appendUnique with new value: len = %d, want 3", len(result))
	}
	if result[2] != "c" {
		t.Errorf("appendUnique new value = %q, want %q", result[2], "c")
	}
}

func TestAppendUniqueEmptySlice(t *testing.T) {
	var slice []string
	result := appendUnique(slice, "a")
	if len(result) != 1 || result[0] != "a" {
		t.Errorf("appendUnique on nil slice = %v, want [a]", result)
	}
}

// --- mapOpenAPIType gaps ---

func TestMapOpenAPITypeEmptyTypeSlice(t *testing.T) {
	schema := &highbase.Schema{}
	result := mapOpenAPIType(schema)
	if result != "" {
		t.Errorf("mapOpenAPIType(empty Type) = %q, want empty string", result)
	}
}

func TestMapOpenAPITypeUnknownType(t *testing.T) {
	schema := &highbase.Schema{Type: []string{"unknown"}}
	result := mapOpenAPIType(schema)
	if result != "" {
		t.Errorf("mapOpenAPIType(unknown) = %q, want empty string", result)
	}
}

func TestMapOpenAPITypeNumber(t *testing.T) {
	schema := &highbase.Schema{Type: []string{"number"}}
	result := mapOpenAPIType(schema)
	if result != "float" {
		t.Errorf("mapOpenAPIType(number) = %q, want %q", result, "float")
	}
}

func TestMapOpenAPITypeString(t *testing.T) {
	schema := &highbase.Schema{Type: []string{"string"}}
	result := mapOpenAPIType(schema)
	if result != "string" {
		t.Errorf("mapOpenAPIType(string) = %q, want %q", result, "string")
	}
}

// --- CheckExtensions gaps ---

func TestCheckExtensionsNilExtensions(t *testing.T) {
	schema := &highbase.Schema{Extensions: nil}
	result := CheckExtensions(schema)
	if result != types.SpecialNone {
		t.Errorf("CheckExtensions(nil extensions) = %v, want SpecialNone", result)
	}
}

func TestCheckExtensionsEmptyExtensions(t *testing.T) {
	extensions := orderedmap.New[string, *yaml.Node]()
	schema := &highbase.Schema{Extensions: extensions}
	result := CheckExtensions(schema)
	if result != types.SpecialNone {
		t.Errorf("CheckExtensions(empty extensions) = %v, want SpecialNone", result)
	}
}

func TestCheckExtensionsFalseValue(t *testing.T) {
	// BUG: extensions.go:19 checks `ext.Tag == "!!bool"` which matches both true and false.
	// Setting x-kubernetes-int-or-string: false should return SpecialNone, but currently
	// returns SpecialIntOrString because the Tag check doesn't verify the actual value.
	// This test documents the current (buggy) behavior.
	extensions := orderedmap.New[string, *yaml.Node]()
	extensions.Set("x-kubernetes-int-or-string", &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!bool",
		Value: "false",
	})
	schema := &highbase.Schema{Extensions: extensions}
	result := CheckExtensions(schema)
	// Current behavior: returns SpecialIntOrString (bug — should be SpecialNone).
	if result != types.SpecialIntOrString {
		t.Errorf("CheckExtensions(false value) = %v, want SpecialIntOrString (current behavior)", result)
	}
}

func TestCheckExtensionsPriority(t *testing.T) {
	// When multiple extensions are set, int-or-string takes priority.
	extensions := orderedmap.New[string, *yaml.Node]()
	trueNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"}
	extensions.Set("x-kubernetes-int-or-string", trueNode)
	extensions.Set("x-kubernetes-preserve-unknown-fields", trueNode)

	schema := &highbase.Schema{Extensions: extensions}
	result := CheckExtensions(schema)
	if result != types.SpecialIntOrString {
		t.Errorf("CheckExtensions(int-or-string + preserve) = %v, want SpecialIntOrString (higher priority)", result)
	}
}

// --- SpecialTypeToFieldNode gaps ---

func TestSpecialTypeToFieldNodeNone(t *testing.T) {
	field := SpecialTypeToFieldNode(types.SpecialNone, "myField")
	if field.Name != "myField" {
		t.Errorf("Name = %q, want %q", field.Name, "myField")
	}
	if field.TypeName != "" {
		t.Errorf("TypeName for SpecialNone = %q, want empty", field.TypeName)
	}
	if field.Description != "" {
		t.Errorf("Description for SpecialNone = %q, want empty", field.Description)
	}
}

// --- collectVariantDescriptions gaps ---

func TestCollectVariantDescriptionsEmpty(t *testing.T) {
	schema := &highbase.Schema{}
	result := collectVariantDescriptions(schema)
	if result != "" {
		t.Errorf("collectVariantDescriptions(empty) = %q, want empty", result)
	}
}
