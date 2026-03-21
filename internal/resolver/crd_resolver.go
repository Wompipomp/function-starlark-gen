// Package resolver converts OpenAPI schemas into TypeNode structures.
//
// The CRD resolver walks CRD openAPIV3Schema trees and produces TypeNodes
// directly without libopenapi. CRD schemas are inline (no $ref), so the
// resolver is a straightforward recursive tree walker.
package resolver

import (
	"fmt"
	"sort"
	"strings"

	"github.com/wompipomp/starlark-gen/internal/loader"
	"github.com/wompipomp/starlark-gen/internal/types"
)

// CRDTypeInfo holds the group, version, and kind for constructing definition
// keys and file paths from CRD metadata.
type CRDTypeInfo struct {
	Group   string
	Version string
	Kind    string
}

// DefinitionKey returns the definition key for a named type under this CRD
// context. Format: {group}.{version}.{typeName}
func (c CRDTypeInfo) DefinitionKey(typeName string) string {
	return c.Group + "." + c.Version + "." + typeName
}

// FilePath returns the output file path for this CRD context.
// Format: {group}/{version}.star
func (c CRDTypeInfo) FilePath() string {
	return c.Group + "/" + c.Version + ".star"
}

// crdResolverState holds mutable state during CRD schema walking.
type crdResolverState struct {
	nodes    []types.TypeNode
	warnings []string
	info     CRDTypeInfo
}

// ResolveCRDs walks all CRD documents and returns TypeNodes and warnings.
// For each CRD, it iterates served versions and recursively walks the
// openAPIV3Schema tree to produce TypeNodes with kind-prefixed sub-type names.
func ResolveCRDs(crds []loader.CRDDocument) ([]types.TypeNode, []string) {
	var allNodes []types.TypeNode
	var allWarnings []string

	for _, crd := range crds {
		for _, version := range crd.Spec.Versions {
			if !version.Served {
				continue
			}
			if version.Schema == nil || version.Schema.OpenAPIV3Schema == nil {
				allWarnings = append(allWarnings, fmt.Sprintf(
					"CRD %s version %s has no openAPIV3Schema", crd.Spec.Names.Kind, version.Name))
				continue
			}

			info := CRDTypeInfo{
				Group:   crd.Spec.Group,
				Version: version.Name,
				Kind:    crd.Spec.Names.Kind,
			}

			state := &crdResolverState{info: info}
			state.walkObject(version.Schema.OpenAPIV3Schema, info.Kind, info.Kind)

			allNodes = append(allNodes, state.nodes...)
			allWarnings = append(allWarnings, state.warnings...)
		}
	}

	return allNodes, allWarnings
}

// walkObject recursively walks a schema object and produces a TypeNode.
// typeName is the name for this type (e.g., "Widget" or "WidgetSpec").
// parentKind is the original CRD kind prefix used for sub-type naming.
func (s *crdResolverState) walkObject(schema *loader.JSONSchemaProps, typeName string, parentKind string) {
	if schema == nil {
		return
	}

	node := types.TypeNode{
		Name:          typeName,
		DefinitionKey: s.info.DefinitionKey(typeName),
		Description:   schema.Description,
		FilePath:      s.info.FilePath(),
	}

	// Handle allOf: merge properties from all entries.
	if len(schema.AllOf) > 0 {
		merged := mergeAllOf(schema)
		schema = merged
	}

	// Build required set for this schema level.
	reqSet := make(map[string]bool)
	for _, r := range schema.Required {
		reqSet[r] = true
	}

	// Sort property keys for deterministic output.
	propNames := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		propNames = append(propNames, name)
	}
	sort.Strings(propNames)

	for _, propName := range propNames {
		prop := schema.Properties[propName]
		if prop == nil {
			continue
		}

		field := s.resolveField(prop, propName, parentKind, &node)
		if reqSet[propName] {
			field.Required = true
		}
		node.Fields = append(node.Fields, field)
	}

	// Sort dependencies for determinism.
	sort.Strings(node.Dependencies)

	s.nodes = append(s.nodes, node)
}

// resolveField converts a single JSON schema property into a FieldNode.
// It handles sub-types, extensions, enums, defaults, arrays, and maps.
func (s *crdResolverState) resolveField(
	prop *loader.JSONSchemaProps,
	propName string,
	parentKind string,
	parentNode *types.TypeNode,
) types.FieldNode {
	field := types.FieldNode{
		Name:        propName,
		Description: prop.Description,
	}

	// Check x-kubernetes-preserve-unknown-fields first -- emit dict, stop recursion.
	if prop.XPreserveUnknownFields != nil && *prop.XPreserveUnknownFields {
		field.TypeName = "dict"
		return field
	}

	// Check x-kubernetes-int-or-string.
	if prop.XIntOrString != nil && *prop.XIntOrString {
		field.TypeName = ""
		if field.Description == "" {
			field.Description = "int or string"
		}
		return field
	}

	// Check x-kubernetes-embedded-resource.
	if prop.XEmbeddedResource != nil && *prop.XEmbeddedResource {
		field.TypeName = "dict"
		return field
	}

	// Handle additionalProperties (map type).
	if prop.AdditionalProperties != nil {
		field.TypeName = "dict"
		field.IsMap = true
		return field
	}

	// Handle object with properties -- create sub-type.
	if prop.Type == "object" && len(prop.Properties) > 0 {
		subTypeName := parentKind + pascalCase(propName)
		subDefKey := s.info.DefinitionKey(subTypeName)

		field.SchemaRef = subDefKey
		parentNode.Dependencies = appendUnique(parentNode.Dependencies, subDefKey)

		// Recurse to build the sub-type.
		s.walkObject(prop, subTypeName, parentKind)
		return field
	}

	// Handle array type.
	if prop.Type == "array" {
		field.TypeName = "list"
		if prop.Items != nil {
			if prop.Items.Type == "object" && len(prop.Items.Properties) > 0 {
				// Array of objects: create sub-type for items.
				subTypeName := parentKind + pascalCase(propName)
				subDefKey := s.info.DefinitionKey(subTypeName)
				field.Items = subDefKey
				parentNode.Dependencies = appendUnique(parentNode.Dependencies, subDefKey)
				s.walkObject(prop.Items, subTypeName, parentKind)
			}
			// Array of primitives: TypeName stays "list", no Items ref.
		}

		// Propagate enum values from items (if primitive array with enum).
		if prop.Items != nil && len(prop.Items.Enum) > 0 {
			for _, e := range prop.Items.Enum {
				field.EnumValues = append(field.EnumValues, fmt.Sprintf("%v", e))
			}
		}
		return field
	}

	// Map OpenAPI primitive types to Starlark types.
	field.TypeName = mapCRDType(prop.Type)

	// Propagate enum values.
	if len(prop.Enum) > 0 {
		for _, e := range prop.Enum {
			field.EnumValues = append(field.EnumValues, fmt.Sprintf("%v", e))
		}
	}

	// Propagate default value (primitives only).
	if prop.Default != nil {
		switch prop.Default.(type) {
		case string, bool, int, float64:
			field.Default = prop.Default
		default:
			s.warnings = append(s.warnings, fmt.Sprintf(
				"skipping complex default for field %q (type %T)", propName, prop.Default))
		}
	}

	return field
}

// mapCRDType maps OpenAPI type strings to Starlark type strings.
func mapCRDType(openAPIType string) string {
	switch openAPIType {
	case "string":
		return "string"
	case "integer":
		return "int"
	case "number":
		return "float"
	case "boolean":
		return "bool"
	case "object":
		return "dict"
	case "array":
		return "list"
	default:
		return ""
	}
}

// pascalCase converts a field name to PascalCase for sub-type naming.
// "spec" -> "Spec", "apiVersion" -> "ApiVersion"
func pascalCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// mergeAllOf merges properties and required fields from all allOf entries
// into a single schema. Properties from later entries override earlier ones.
func mergeAllOf(schema *loader.JSONSchemaProps) *loader.JSONSchemaProps {
	merged := &loader.JSONSchemaProps{
		Type:        schema.Type,
		Description: schema.Description,
		Properties:  make(map[string]*loader.JSONSchemaProps),
	}

	// Also include top-level properties if any.
	for k, v := range schema.Properties {
		merged.Properties[k] = v
	}
	merged.Required = append(merged.Required, schema.Required...)

	for _, entry := range schema.AllOf {
		if entry == nil {
			continue
		}
		for k, v := range entry.Properties {
			merged.Properties[k] = v
		}
		merged.Required = append(merged.Required, entry.Required...)
	}

	return merged
}
