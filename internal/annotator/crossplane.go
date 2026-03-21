// Package annotator provides post-resolution transforms for TypeNode slices.
//
// The Crossplane annotator detects Crossplane managed resource structure,
// removes status subtrees, and augments descriptions with lifecycle semantics.
package annotator

import (
	"github.com/wompipomp/starlark-gen/internal/types"
)

// AnnotateCrossplane processes resolved CRD TypeNodes to apply Crossplane-specific
// transformations. It returns the modified nodes and any warnings.
func AnnotateCrossplane(nodes []types.TypeNode) ([]types.TypeNode, []string) {
	return nodes, nil
}
