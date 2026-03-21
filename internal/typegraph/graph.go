// Package typegraph provides topological sorting of TypeNodes within files
// and validation of the inter-file load dependency graph.
//
// SortTypesInFile sorts types within a single output file so that dependencies
// appear before their consumers, preventing Starlark NameError failures.
//
// ValidateLoadDAG validates that the file-level dependency graph is acyclic,
// ensuring no circular load() statements between generated .star files.
package typegraph

import (
	"fmt"
	"strings"

	"github.com/dominikbraun/graph"
	"github.com/wompipomp/starlark-gen/internal/organizer"
	"github.com/wompipomp/starlark-gen/internal/types"
)

// SortTypesInFile performs a topological sort on the given types, considering
// only intra-file dependencies (dependencies where both types are in the same
// file). Cross-file dependencies are ignored for ordering purposes.
//
// Uses dominikbraun/graph StableTopologicalSort with lexicographic tiebreaking
// to ensure deterministic output.
//
// Returns the sorted slice of TypeNode pointers, or an error if a cycle is
// detected among intra-file dependencies.
func SortTypesInFile(nodes []*types.TypeNode) ([]*types.TypeNode, error) {
	if len(nodes) == 0 {
		return nodes, nil
	}

	// Build a lookup from DefinitionKey to TypeNode for types in this file.
	byDefKey := make(map[string]*types.TypeNode, len(nodes))
	byName := make(map[string]*types.TypeNode, len(nodes))
	for _, n := range nodes {
		byDefKey[n.DefinitionKey] = n
		byName[n.Name] = n
	}

	// The file path for this group (all nodes should share the same FilePath).
	filePath := nodes[0].FilePath

	// Build the directed graph with cycle prevention. Edges go from
	// dependency -> consumer so that topological sort places dependencies first.
	g := graph.New(graph.StringHash, graph.Directed(), graph.PreventCycles())

	// Add all type names as vertices.
	for _, n := range nodes {
		_ = g.AddVertex(n.Name)
	}

	// Add edges for intra-file dependencies. When an edge would create a
	// cycle, break it by converting the referencing field to field(type="dict").
	for _, n := range nodes {
		for _, depKey := range n.Dependencies {
			dep, exists := byDefKey[depKey]
			if !exists {
				// Cross-file dependency or unresolved -- skip.
				continue
			}
			// Only add edge if the dependency is in the same file.
			if dep.FilePath != filePath {
				continue
			}
			// Skip self-edges: circular ref types reference themselves,
			// but the resolver already broke the cycle with field(type="dict").
			if dep.Name == n.Name {
				continue
			}
			// Edge from dependency to consumer: dep must come before n.
			err := g.AddEdge(dep.Name, n.Name)
			if err == graph.ErrEdgeCreatesCycle {
				breakCycleDependency(n, depKey)
			}
		}
	}

	// Perform stable topological sort with lexicographic tiebreaking.
	order, err := graph.StableTopologicalSort(g, func(a, b string) bool {
		return a < b
	})
	if err != nil {
		return nil, fmt.Errorf("circular dependency within file %s: %w", filePath, err)
	}

	// Reorder nodes according to topological order.
	result := make([]*types.TypeNode, 0, len(order))
	for _, name := range order {
		result = append(result, byName[name])
	}

	return result, nil
}

// breakCycleDependency breaks a circular dependency by converting fields that
// reference depKey to field(type="dict") and removing the dependency edge.
// This mirrors how the K8s swagger resolver handles circular $ref chains.
func breakCycleDependency(node *types.TypeNode, depKey string) {
	node.IsCircularRef = true
	for i := range node.Fields {
		if node.Fields[i].SchemaRef == depKey {
			shortName := depKey[strings.LastIndex(depKey, ".")+1:]
			node.Fields[i].SchemaRef = ""
			node.Fields[i].TypeName = "dict"
			if node.Fields[i].Description != "" {
				node.Fields[i].Description = "dict - " + node.Fields[i].Description + " (circular reference to " + shortName + ")"
			} else {
				node.Fields[i].Description = "dict - Recursive reference to " + shortName
			}
		}
		if node.Fields[i].Items == depKey {
			node.Fields[i].Items = ""
		}
	}
	// Remove the dependency from the list.
	filtered := node.Dependencies[:0]
	for _, d := range node.Dependencies {
		if d != depKey {
			filtered = append(filtered, d)
		}
	}
	node.Dependencies = filtered
}

// ValidateLoadDAG validates that the inter-file dependency graph is a DAG
// (directed acyclic graph). It builds a file-level graph where an edge from
// file A to file B means file B loads (imports) types from file A.
//
// Returns the topologically sorted file order (files with no dependencies first)
// suitable for emission order, or an error if circular file dependencies exist.
func ValidateLoadDAG(fileMap organizer.FileMap) ([]string, error) {
	// Build a lookup from DefinitionKey to FilePath for all types.
	defKeyToFile := make(map[string]string)
	for fp, nodes := range fileMap {
		for _, n := range nodes {
			defKeyToFile[n.DefinitionKey] = fp
		}
	}

	// Build the file-level directed graph.
	g := graph.New(graph.StringHash, graph.Directed(), graph.PreventCycles())

	// Add all file paths as vertices.
	for fp := range fileMap {
		_ = g.AddVertex(fp)
	}

	// Add edges: for each type in a file, if it depends on a type in a
	// DIFFERENT file, add an edge from that file to this file.
	for fp, nodes := range fileMap {
		for _, n := range nodes {
			for _, depKey := range n.Dependencies {
				depFile, exists := defKeyToFile[depKey]
				if !exists {
					// Dependency not in any file (e.g., special type) -- skip.
					continue
				}
				if depFile == fp {
					// Same file -- not an inter-file dependency.
					continue
				}
				// Edge from depFile to fp: depFile must be loaded before fp.
				err := g.AddEdge(depFile, fp)
				if err != nil {
					// If PreventCycles detects a cycle, wrap with a clear message.
					if err == graph.ErrEdgeCreatesCycle {
						return nil, fmt.Errorf("circular load() dependency: %s and %s form a cycle", depFile, fp)
					}
					// Duplicate edge is not an error -- just skip.
				}
			}
		}
	}

	// Perform stable topological sort for deterministic emission order.
	order, err := graph.StableTopologicalSort(g, func(a, b string) bool {
		return a < b
	})
	if err != nil {
		return nil, fmt.Errorf("circular load() dependency detected: %w", err)
	}

	return order, nil
}
