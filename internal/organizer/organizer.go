package organizer

import (
	"fmt"

	"github.com/wompipomp/starlark-gen/internal/types"
)

// FileMap maps output file paths (e.g., "apps/v1.star") to the ordered list of
// TypeNodes assigned to each file. Within each file, nodes are in insertion
// order (the order they arrived from the Resolver). Topological sorting within
// files is done later by the TypeGraph package.
type FileMap map[string][]*types.TypeNode

// Organize assigns a FilePath to each TypeNode based on its DefinitionKey and
// groups nodes into a FileMap. Special types (IntOrString, Quantity) are excluded
// from file assignment since they don't generate standalone schema definitions.
//
// The pkg parameter is the OCI package prefix (e.g., "schemas-k8s:v1.31") used
// for constructing load paths. It is stored for downstream use but does not
// affect file assignment.
//
// Returns the FileMap, a list of warning messages, and an error (nil on success).
func Organize(nodes []types.TypeNode, pkg string) (FileMap, []string, error) {
	fm := make(FileMap)
	var warnings []string

	for i := range nodes {
		node := &nodes[i]

		filePath, isSpecial, err := DefinitionKeyToFilePath(node.DefinitionKey)
		if err != nil {
			return nil, warnings, fmt.Errorf("organizing %s: %w", node.DefinitionKey, err)
		}

		// Skip special types -- they don't get their own file.
		if isSpecial {
			continue
		}

		node.FilePath = filePath
		fm[filePath] = append(fm[filePath], node)
	}

	return fm, warnings, nil
}
