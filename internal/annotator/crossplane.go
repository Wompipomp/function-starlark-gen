// Package annotator provides post-resolution transforms for TypeNode slices.
//
// The Crossplane annotator detects Crossplane managed resource structure,
// removes status subtrees, and augments descriptions with lifecycle semantics.
package annotator

import (
	"fmt"
	"strings"

	"github.com/wompipomp/starlark-gen/internal/types"
)

// crossplaneFieldAnnotations maps known Crossplane spec-level field names
// to their doc annotation text.
var crossplaneFieldAnnotations = map[string]string{
	"providerConfigRef":          "Reference to the ProviderConfig for auth",
	"writeConnectionSecretToRef": "Where to write connection details secret",
	"publishConnectionDetailsTo": "Where to publish connection details",
	"deletionPolicy":             "Delete or Orphan the external resource on CR deletion",
	"managementPolicies":         "Actions Crossplane is allowed to take on the resource",
}

// forProviderAnnotation is the doc prefix for forProvider schema and field descriptions.
const forProviderAnnotation = "Reconcilable configuration. Fields here are continuously reconciled."

// initProviderAnnotation is the doc prefix for initProvider schema and field descriptions.
const initProviderAnnotation = "Write-once initialization. Fields here are set only at creation."

// AnnotateCrossplane processes resolved CRD TypeNodes to apply Crossplane-specific
// transformations:
//  1. Detect Crossplane managed resource structure (forProvider/initProvider in spec)
//  2. Remove status subtree (all status-rooted TypeNodes and status field from root)
//  3. Augment descriptions on forProvider/initProvider schemas and fields
//  4. Augment descriptions on known Crossplane-standard fields
//
// CRDs without forProvider/initProvider are returned unchanged with a warning.
func AnnotateCrossplane(nodes []types.TypeNode) ([]types.TypeNode, []string) {
	var warnings []string

	// Build index: definitionKey -> index in nodes slice.
	nodeIndex := make(map[string]int, len(nodes))
	for i := range nodes {
		nodeIndex[nodes[i].DefinitionKey] = i
	}

	// Track which definition keys to remove (status subtrees).
	removeKeys := make(map[string]bool)

	// Find root TypeNodes. A root TypeNode has a "spec" field with a SchemaRef.
	for i := range nodes {
		node := &nodes[i]
		specField := fieldByName(node.Fields, "spec")
		if specField == nil || specField.SchemaRef == "" {
			continue
		}

		// This is a root TypeNode. Look up the spec TypeNode.
		specIdx, ok := nodeIndex[specField.SchemaRef]
		if !ok {
			continue
		}
		specNode := &nodes[specIdx]

		// Check for forProvider/initProvider on the spec TypeNode.
		hasForProvider, hasInitProvider := isCrossplaneManagedResource(specNode)

		if !hasForProvider && !hasInitProvider {
			// Non-standard CRD: warn and skip annotation.
			warnings = append(warnings,
				fmt.Sprintf("warn: %s: no forProvider/initProvider structure found, generating as plain CRD", node.Name))
			continue
		}

		// This is a Crossplane managed resource. Apply annotations.

		// 1. Remove status subtree.
		statusField := fieldByName(node.Fields, "status")
		if statusField != nil {
			// Collect all status-reachable definition keys.
			if statusField.SchemaRef != "" {
				collectStatusSubtree(statusField.SchemaRef, nodes, nodeIndex, removeKeys)
			}

			// Remove status field from root.
			node.Fields = removeFieldByName(node.Fields, "status")

			// Remove status dependency from root.
			node.Dependencies = removeDep(node.Dependencies, statusField.SchemaRef)
		}

		// 2. Annotate forProvider.
		if hasForProvider {
			fpField := fieldByName(specNode.Fields, "forProvider")
			if fpField != nil {
				// Augment the forProvider field description on the spec TypeNode.
				fpField.Description = augmentDescription(forProviderAnnotation, fpField.Description)

				// Augment the forProvider TypeNode description.
				if fpField.SchemaRef != "" {
					if fpIdx, ok := nodeIndex[fpField.SchemaRef]; ok {
						nodes[fpIdx].Description = augmentDescription(
							forProviderAnnotation, nodes[fpIdx].Description)
					}
				}
			}
		}

		// 3. Annotate initProvider.
		if hasInitProvider {
			ipField := fieldByName(specNode.Fields, "initProvider")
			if ipField != nil {
				// Augment the initProvider field description on the spec TypeNode.
				ipField.Description = augmentDescription(initProviderAnnotation, ipField.Description)

				// Augment the initProvider TypeNode description.
				if ipField.SchemaRef != "" {
					if ipIdx, ok := nodeIndex[ipField.SchemaRef]; ok {
						nodes[ipIdx].Description = augmentDescription(
							initProviderAnnotation, nodes[ipIdx].Description)
					}
				}
			}
		}

		// 4. Annotate known Crossplane-standard fields on the spec TypeNode.
		for j := range specNode.Fields {
			f := &specNode.Fields[j]
			if annotation, ok := crossplaneFieldAnnotations[f.Name]; ok {
				f.Description = augmentDescription(annotation, f.Description)
			}
		}
	}

	// Filter out removed status TypeNodes.
	if len(removeKeys) > 0 {
		filtered := make([]types.TypeNode, 0, len(nodes)-len(removeKeys))
		for i := range nodes {
			if !removeKeys[nodes[i].DefinitionKey] {
				filtered = append(filtered, nodes[i])
			}
		}
		return filtered, warnings
	}

	return nodes, warnings
}

// isCrossplaneManagedResource checks if a spec TypeNode represents a
// Crossplane managed resource by looking for forProvider/initProvider fields.
func isCrossplaneManagedResource(specNode *types.TypeNode) (hasForProvider, hasInitProvider bool) {
	for _, f := range specNode.Fields {
		switch f.Name {
		case "forProvider":
			hasForProvider = true
		case "initProvider":
			hasInitProvider = true
		}
	}
	return
}

// collectStatusSubtree traces all TypeNodes reachable from the status definition
// key and marks them for removal.
func collectStatusSubtree(statusDefKey string, nodes []types.TypeNode, nodeIndex map[string]int, removeKeys map[string]bool) {
	// BFS from the status definition key.
	queue := []string{statusDefKey}
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		if removeKeys[key] {
			continue
		}
		removeKeys[key] = true

		idx, ok := nodeIndex[key]
		if !ok {
			continue
		}
		// Follow dependencies of the status TypeNode.
		for _, dep := range nodes[idx].Dependencies {
			if !removeKeys[dep] {
				queue = append(queue, dep)
			}
		}
	}
}

// augmentDescription prepends a Crossplane annotation to an existing description.
// If the original description is empty, returns just the annotation.
// If the annotation ends with a period, uses space separator; otherwise ". ".
func augmentDescription(annotation, original string) string {
	if original == "" {
		return annotation
	}
	if strings.HasSuffix(annotation, ".") {
		return annotation + " " + original
	}
	return annotation + ". " + original
}

// fieldByName returns a pointer to the field with the given name, or nil.
func fieldByName(fields []types.FieldNode, name string) *types.FieldNode {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}
	return nil
}

// removeFieldByName returns a copy of the fields slice with the named field removed.
func removeFieldByName(fields []types.FieldNode, name string) []types.FieldNode {
	result := make([]types.FieldNode, 0, len(fields))
	for _, f := range fields {
		if f.Name != name {
			result = append(result, f)
		}
	}
	return result
}

// removeDep returns a copy of the deps slice with the given key removed.
func removeDep(deps []string, key string) []string {
	result := make([]string, 0, len(deps))
	for _, d := range deps {
		if d != key {
			result = append(result, d)
		}
	}
	return result
}
