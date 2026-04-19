// Package resolver converts libopenapi V2 model definitions into TypeNode structures.
//
// The resolver handles $ref resolution, circular reference detection, allOf composition
// merging, oneOf/anyOf union types, additionalProperties maps, and Kubernetes-specific
// OpenAPI extensions.
package resolver

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pb33f/libopenapi"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	v2high "github.com/pb33f/libopenapi/datamodel/high/v2"
	"github.com/wompipomp/starlark-gen/internal/types"
)

const refPrefix = "#/definitions/"

// resolverState holds the mutable state during resolution.
type resolverState struct {
	// defs holds all definitions from the swagger spec, keyed by definition key.
	defs map[string]*highbase.SchemaProxy

	// resolving tracks keys currently on the resolution stack (cycle detection).
	resolving map[string]bool

	// resolved caches already-resolved TypeNodes to prevent redundant work.
	resolved map[string]*types.TypeNode

	// warnings collects non-fatal issues found during resolution.
	warnings []string
}

// Resolve walks all definitions in the V2 model and converts them to TypeNode structures.
// It returns the resolved type nodes in definition order and a list of warning messages.
func Resolve(model *libopenapi.DocumentModel[v2high.Swagger]) ([]types.TypeNode, []string) {
	state := &resolverState{
		defs:      make(map[string]*highbase.SchemaProxy),
		resolving: make(map[string]bool),
		resolved:  make(map[string]*types.TypeNode),
	}

	// Build definition map and collect keys in spec order.
	var defKeys []string
	for key, proxy := range model.Model.Definitions.Definitions.FromOldest() {
		state.defs[key] = proxy
		defKeys = append(defKeys, key)
	}

	// Resolve each definition in spec order.
	var result []types.TypeNode
	for _, key := range defKeys {
		node := state.resolveDefinition(key)
		if node != nil {
			result = append(result, *node)
		}
	}

	return result, state.warnings
}

// resolveDefinition resolves a single definition key into a TypeNode.
// It handles circular reference detection, allOf merging, oneOf/anyOf, and
// additionalProperties.
func (s *resolverState) resolveDefinition(defKey string) *types.TypeNode {
	// Return cached result if already resolved.
	if cached, ok := s.resolved[defKey]; ok {
		return cached
	}

	proxy, ok := s.defs[defKey]
	if !ok {
		s.warnings = append(s.warnings, fmt.Sprintf("definition %q not found", defKey))
		return nil
	}

	// Detect circular reference: if we're already resolving this key, it's a cycle.
	if s.resolving[defKey] {
		// Create a minimal circular-ref placeholder.
		node := &types.TypeNode{
			Name:          shortName(defKey),
			DefinitionKey: defKey,
			IsCircularRef: true,
		}
		s.resolved[defKey] = node
		return node
	}

	// Mark as in-progress.
	s.resolving[defKey] = true

	schema := proxy.Schema()
	if schema == nil {
		s.warnings = append(s.warnings, fmt.Sprintf("cannot resolve schema for %q", defKey))
		delete(s.resolving, defKey)
		return nil
	}

	node := &types.TypeNode{
		Name:          shortName(defKey),
		DefinitionKey: defKey,
		Description:   schema.Description,
	}

	// Cache the node pointer early so circular refs can reference it.
	s.resolved[defKey] = node

	// Check for special types by canonical definition path FIRST.
	if special := IsSpecialType(defKey); special != types.SpecialNone {
		node.SpecialType = special
		// Special types like IntOrString and Quantity are leaf types --
		// no property recursion needed.
		sort.Strings(node.Dependencies)
		delete(s.resolving, defKey)
		return node
	}

	// Check for K8s extensions on the schema BEFORE processing properties.
	if special := CheckExtensions(schema); special != types.SpecialNone {
		node.SpecialType = special
		// PreserveUnknown and EmbeddedResource skip property recursion entirely.
		if special == types.SpecialPreserveUnknown || special == types.SpecialEmbeddedResource {
			sort.Strings(node.Dependencies)
			delete(s.resolving, defKey)
			return node
		}
	}

	// Resolve properties, handling allOf, oneOf/anyOf, additionalProperties.
	s.resolveSchema(node, schema, defKey)

	// Top-level resource types get apiVersion/kind defaulted from
	// x-kubernetes-group-version-kind so callers only need to set metadata/spec.
	if group, version, kind, ok := ExtractGVK(schema); ok {
		ApplyGVKDefaults(node, APIVersionString(group, version), kind)
	}

	// Sort dependencies for determinism.
	sort.Strings(node.Dependencies)

	// Remove from in-progress stack.
	delete(s.resolving, defKey)

	return node
}

// resolveSchema populates a TypeNode's fields from a schema, handling composition.
func (s *resolverState) resolveSchema(node *types.TypeNode, schema *highbase.Schema, defKey string) {
	// Handle allOf composition: merge properties from all allOf entries.
	if len(schema.AllOf) > 0 {
		s.resolveAllOf(node, schema.AllOf, defKey)
		return
	}

	// Process properties from the schema directly.
	s.resolveProperties(node, schema, defKey)
}

// resolveAllOf merges properties from all allOf entries into the node.
func (s *resolverState) resolveAllOf(node *types.TypeNode, allOf []*highbase.SchemaProxy, defKey string) {
	// Collect all properties from allOf entries in order. Last wins for conflicts.
	fieldMap := make(map[string]types.FieldNode)
	var fieldOrder []string
	var allRequired []string

	for _, entry := range allOf {
		// Check if this allOf entry is a $ref.
		if entry.IsReference() {
			refKey := extractRefKey(entry.GetReference())
			if refKey != "" {
				// Resolve the referenced type and merge its fields.
				refNode := s.resolveDefinition(refKey)
				if refNode != nil {
					for _, f := range refNode.Fields {
						if _, exists := fieldMap[f.Name]; !exists {
							fieldOrder = append(fieldOrder, f.Name)
						}
						fieldMap[f.Name] = f
					}
					// Add as dependency.
					node.Dependencies = appendUnique(node.Dependencies, refKey)
				}
			}
			continue
		}

		// Inline schema in allOf -- resolve its properties.
		entrySchema := entry.Schema()
		if entrySchema == nil {
			continue
		}

		// Collect required fields from this entry.
		allRequired = append(allRequired, entrySchema.Required...)

		if entrySchema.Properties != nil {
			for propName, propProxy := range entrySchema.Properties.FromOldest() {
				field := s.resolveField(propName, propProxy, defKey)
				if _, exists := fieldMap[propName]; !exists {
					fieldOrder = append(fieldOrder, propName)
				}
				fieldMap[propName] = field
			}
		}
	}

	// Apply required flags.
	reqSet := make(map[string]bool)
	for _, r := range allRequired {
		reqSet[r] = true
	}

	// Build the fields list in order.
	for _, name := range fieldOrder {
		f := fieldMap[name]
		if reqSet[name] {
			f.Required = true
		}
		node.Fields = append(node.Fields, f)
	}
}

// resolveProperties resolves the properties of a schema into fields on the node.
func (s *resolverState) resolveProperties(node *types.TypeNode, schema *highbase.Schema, defKey string) {
	if schema.Properties == nil {
		return
	}

	// Build required set.
	reqSet := make(map[string]bool)
	for _, r := range schema.Required {
		reqSet[r] = true
	}

	for propName, propProxy := range schema.Properties.FromOldest() {
		field := s.resolveField(propName, propProxy, defKey)
		if reqSet[propName] {
			field.Required = true
		}
		node.Fields = append(node.Fields, field)
	}
}

// resolveField resolves a single property SchemaProxy into a FieldNode.
func (s *resolverState) resolveField(name string, proxy *highbase.SchemaProxy, parentDefKey string) types.FieldNode {
	field := types.FieldNode{
		Name: name,
	}

	// Handle $ref fields.
	if proxy.IsReference() {
		refKey := extractRefKey(proxy.GetReference())
		if refKey != "" {
			// Check if the $ref target is a special type -- use SpecialTypeToFieldNode.
			if special := IsSpecialType(refKey); special != types.SpecialNone {
				specialField := SpecialTypeToFieldNode(special, name)
				// Preserve the original description if available.
				schema := proxy.Schema()
				if schema != nil && schema.Description != "" {
					specialField.Description = schema.Description
				}
				// Add dependency on the referenced type.
				if parentNode, ok := s.resolved[parentDefKey]; ok {
					parentNode.Dependencies = appendUnique(parentNode.Dependencies, refKey)
				}
				return specialField
			}

			field.SchemaRef = refKey

			// Check if it's a circular reference back to the parent.
			if s.resolving[refKey] {
				field.TypeName = "dict"
				field.Description = fmt.Sprintf("dict - Recursive reference to %s", shortName(refKey))
				// Mark the parent as having a circular ref.
				if parentNode, ok := s.resolved[parentDefKey]; ok {
					parentNode.IsCircularRef = true
				}
			}

			// Add dependency on the referenced type.
			if parentNode, ok := s.resolved[parentDefKey]; ok {
				parentNode.Dependencies = appendUnique(parentNode.Dependencies, refKey)
			}
		}

		// Also resolve the schema to get the description.
		schema := proxy.Schema()
		if schema != nil && field.Description == "" {
			field.Description = schema.Description
		}

		return field
	}

	// Resolve the schema.
	schema := proxy.Schema()
	if schema == nil {
		return field
	}

	field.Description = schema.Description

	// Handle oneOf/anyOf -- emit field(type="") with doc listing variants.
	if len(schema.OneOf) > 0 || len(schema.AnyOf) > 0 {
		field.TypeName = ""
		variants := collectVariantDescriptions(schema)
		if variants != "" {
			if field.Description != "" {
				field.Description = field.Description + " (variants: " + variants + ")"
			} else {
				field.Description = "One of: " + variants
			}
		}
		return field
	}

	// Handle additionalProperties.
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.IsA() {
		field.TypeName = "dict"
		field.IsMap = true

		// Check if additionalProperties references another type.
		addlProxy := schema.AdditionalProperties.A
		if addlProxy.IsReference() {
			refKey := extractRefKey(addlProxy.GetReference())
			if refKey != "" {
				// Detect circular reference through additionalProperties.
				if s.resolving[refKey] {
					if parentNode, ok := s.resolved[parentDefKey]; ok {
						parentNode.IsCircularRef = true
					}
					field.Description = fmt.Sprintf("dict - Recursive reference to %s", shortName(refKey))
				}
				if parentNode, ok := s.resolved[parentDefKey]; ok {
					parentNode.Dependencies = appendUnique(parentNode.Dependencies, refKey)
				}
			}
		}
		return field
	}

	// Map OpenAPI types to Starlark types.
	field.TypeName = mapOpenAPIType(schema)

	// Handle array items.
	if field.TypeName == "list" && schema.Items != nil && schema.Items.IsA() {
		itemProxy := schema.Items.A
		if itemProxy.IsReference() {
			refKey := extractRefKey(itemProxy.GetReference())
			if refKey != "" {
				field.Items = refKey
				if parentNode, ok := s.resolved[parentDefKey]; ok {
					parentNode.Dependencies = appendUnique(parentNode.Dependencies, refKey)
				}
			}
		}
		// If items is an inline schema, resolve its type name.
		itemSchema := itemProxy.Schema()
		if itemSchema != nil && field.Items == "" {
			// Inline item type -- get the type name for enum propagation.
			if len(itemSchema.Enum) > 0 {
				for _, e := range itemSchema.Enum {
					field.EnumValues = append(field.EnumValues, fmt.Sprintf("%v", e.Value))
				}
			}
		}
	}

	// Propagate enum values.
	if len(schema.Enum) > 0 {
		for _, e := range schema.Enum {
			field.EnumValues = append(field.EnumValues, fmt.Sprintf("%v", e.Value))
		}
	}

	return field
}

// mapOpenAPIType maps OpenAPI type strings to Starlark type strings.
func mapOpenAPIType(schema *highbase.Schema) string {
	if len(schema.Type) == 0 {
		return ""
	}

	switch schema.Type[0] {
	case "string":
		return "string"
	case "integer":
		return "int"
	case "number":
		return "float"
	case "boolean":
		return "bool"
	case "array":
		return "list"
	case "object":
		// Object with properties is a structured type, not dict.
		// But object with NO properties and no additionalProperties is dict.
		if schema.Properties == nil || schema.Properties.Len() == 0 {
			if schema.AdditionalProperties == nil {
				return "dict"
			}
		}
		return "dict"
	default:
		return ""
	}
}

// collectVariantDescriptions returns a string describing the variants for oneOf/anyOf.
func collectVariantDescriptions(schema *highbase.Schema) string {
	var variants []string
	sources := schema.OneOf
	if len(sources) == 0 {
		sources = schema.AnyOf
	}

	for _, variant := range sources {
		variantSchema := variant.Schema()
		if variantSchema == nil {
			continue
		}
		if variantSchema.Properties != nil {
			// Collect property names as variant description.
			var props []string
			for propName := range variantSchema.Properties.FromOldest() {
				props = append(props, propName)
			}
			variants = append(variants, strings.Join(props, "+"))
		} else if len(variantSchema.Type) > 0 {
			variants = append(variants, variantSchema.Type[0])
		}
	}

	return strings.Join(variants, ", ")
}

// extractRefKey extracts the definition key from a $ref string.
// E.g., "#/definitions/io.k8s.api.apps.v1.Deployment" -> "io.k8s.api.apps.v1.Deployment"
func extractRefKey(ref string) string {
	if strings.HasPrefix(ref, refPrefix) {
		return strings.TrimPrefix(ref, refPrefix)
	}
	return ""
}

// shortName extracts the short type name from a definition key.
// E.g., "io.k8s.api.apps.v1.Deployment" -> "Deployment"
func shortName(defKey string) string {
	parts := strings.Split(defKey, ".")
	if len(parts) == 0 {
		return defKey
	}
	return parts[len(parts)-1]
}

// appendUnique appends a value to a slice if it's not already present.
func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}
