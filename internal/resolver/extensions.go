package resolver

import (
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/wompipomp/starlark-gen/internal/types"
	"go.yaml.in/yaml/v4"
)

// CheckExtensions checks a schema's extensions for Kubernetes-specific markers
// and returns the corresponding SpecialType. Extension priority:
// 1. x-kubernetes-int-or-string
// 2. x-kubernetes-preserve-unknown-fields
// 3. x-kubernetes-embedded-resource
func CheckExtensions(schema *highbase.Schema) types.SpecialType {
	if schema.Extensions == nil {
		return types.SpecialNone
	}

	if ext := schema.Extensions.GetOrZero("x-kubernetes-int-or-string"); ext != nil {
		if ext.Value == "true" || ext.Tag == "!!bool" {
			return types.SpecialIntOrString
		}
	}

	if ext := schema.Extensions.GetOrZero("x-kubernetes-preserve-unknown-fields"); ext != nil {
		if ext.Value == "true" || ext.Tag == "!!bool" {
			return types.SpecialPreserveUnknown
		}
	}

	if ext := schema.Extensions.GetOrZero("x-kubernetes-embedded-resource"); ext != nil {
		if ext.Value == "true" || ext.Tag == "!!bool" {
			return types.SpecialEmbeddedResource
		}
	}

	return types.SpecialNone
}

// IsSpecialType checks if a definition key corresponds to a well-known Kubernetes
// special type by its canonical definition path.
func IsSpecialType(definitionKey string) types.SpecialType {
	switch definitionKey {
	case "io.k8s.apimachinery.pkg.util.intstr.IntOrString":
		return types.SpecialIntOrString
	case "io.k8s.apimachinery.pkg.api.resource.Quantity":
		return types.SpecialQuantity
	default:
		return types.SpecialNone
	}
}

// ExtractGVK reads the x-kubernetes-group-version-kind extension and returns
// the single (group, version, kind). Returns ok=false when the extension is
// absent, malformed, or has multiple entries (DeleteOptions, WatchEvent) — the
// multi-entry case is skipped because those types aren't user-composed.
func ExtractGVK(schema *highbase.Schema) (group, version, kind string, ok bool) {
	if schema == nil || schema.Extensions == nil {
		return "", "", "", false
	}
	ext := schema.Extensions.GetOrZero("x-kubernetes-group-version-kind")
	if ext == nil || ext.Kind != yaml.SequenceNode || len(ext.Content) != 1 {
		return "", "", "", false
	}
	entry := ext.Content[0]
	if entry.Kind != yaml.MappingNode {
		return "", "", "", false
	}
	for i := 0; i+1 < len(entry.Content); i += 2 {
		switch entry.Content[i].Value {
		case "group":
			group = entry.Content[i+1].Value
		case "version":
			version = entry.Content[i+1].Value
		case "kind":
			kind = entry.Content[i+1].Value
		}
	}
	if version == "" || kind == "" {
		return "", "", "", false
	}
	return group, version, kind, true
}

// APIVersionString joins (group, version) into the apiVersion wire string.
// Core types have group="" and produce just the version ("v1").
func APIVersionString(group, version string) string {
	if group == "" {
		return version
	}
	return group + "/" + version
}

// ApplyGVKDefaults sets Default on the apiVersion/kind fields of a top-level
// resource node and places them in canonical order [apiVersion, kind, ...rest]
// at the front of node.Fields. Existing apiVersion/kind fields are preserved
// (description is kept) but their Default/TypeName/SchemaRef/Required are
// overwritten to reflect the fixed GVK.
func ApplyGVKDefaults(node *types.TypeNode, apiVersion, kind string) {
	var apiField, kindField types.FieldNode
	var apiSeen, kindSeen bool
	var rest []types.FieldNode
	for _, f := range node.Fields {
		switch f.Name {
		case "apiVersion":
			apiField = f
			apiSeen = true
		case "kind":
			kindField = f
			kindSeen = true
		default:
			rest = append(rest, f)
		}
	}

	if !apiSeen {
		apiField = types.FieldNode{
			Name:        "apiVersion",
			Description: "APIVersion of the resource (fixed).",
		}
	}
	apiField.TypeName = "string"
	apiField.SchemaRef = ""
	apiField.Required = false
	apiField.Default = apiVersion

	if !kindSeen {
		kindField = types.FieldNode{
			Name:        "kind",
			Description: "Kind of the resource (fixed).",
		}
	}
	kindField.TypeName = "string"
	kindField.SchemaRef = ""
	kindField.Required = false
	kindField.Default = kind

	node.Fields = append([]types.FieldNode{apiField, kindField}, rest...)
}

// SpecialTypeToFieldNode converts a special type into an appropriate FieldNode
// for use when a field references a special type.
func SpecialTypeToFieldNode(special types.SpecialType, fieldName string) types.FieldNode {
	field := types.FieldNode{Name: fieldName}

	switch special {
	case types.SpecialIntOrString:
		field.TypeName = ""
		field.Description = "int or string - Accepts both integer and string values"
	case types.SpecialQuantity:
		field.TypeName = ""
		field.Description = "string - Kubernetes resource quantity (e.g., '100m', '1Gi')"
	case types.SpecialPreserveUnknown:
		field.TypeName = "dict"
		field.Description = "dict - Arbitrary JSON (x-kubernetes-preserve-unknown-fields)"
	case types.SpecialEmbeddedResource:
		field.TypeName = "dict"
		field.Description = "dict - Embedded Kubernetes resource (x-kubernetes-embedded-resource)"
	}

	return field
}
