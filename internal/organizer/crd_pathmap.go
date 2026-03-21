package organizer

import (
	"fmt"
	"regexp"
	"strings"
)

// versionPattern matches Kubernetes API version segments like v1, v1alpha1,
// v1beta1, v2beta1, etc.
var versionPattern = regexp.MustCompile(`^v[0-9]`)

// CRDDefinitionKeyToFilePath maps a CRD definition key to an output file path.
// CRD definition keys follow the format: {group}.{version}.{TypeName}
// where the group can contain dots (e.g., "cert-manager.io").
//
// Strategy: scan parts from right to left to find the version segment matching
// the v[0-9]* pattern. Everything before it is the group, everything after is
// the type name.
//
// Examples:
//
//	"example.com.v1.Widget"            -> "example.com/v1.star"
//	"cert-manager.io.v1.Certificate"   -> "cert-manager.io/v1.star"
//	"some.deep.group.io.v2beta1.Bar"   -> "some.deep.group.io/v2beta1.star"
func CRDDefinitionKeyToFilePath(key string) (string, error) {
	parts := strings.Split(key, ".")

	// Scan from right to left to find the version segment.
	for i := len(parts) - 1; i >= 0; i-- {
		if versionPattern.MatchString(parts[i]) {
			if i == 0 {
				return "", fmt.Errorf("CRD definition key %q has no group before version segment", key)
			}
			group := strings.Join(parts[:i], ".")
			version := parts[i]
			return fmt.Sprintf("%s/%s.star", group, version), nil
		}
	}

	return "", fmt.Errorf("no version segment found in CRD definition key %q", key)
}
