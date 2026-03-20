package resolver

import (
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/wompipomp/starlark-gen/internal/types"
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
