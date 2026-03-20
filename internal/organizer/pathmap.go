// Package organizer assigns TypeNodes to output file paths based on Kubernetes
// API group and version structure, and validates the resulting file dependency graph.
package organizer

import (
	"fmt"
	"strings"
)

// DefinitionKeyToFilePath maps a Kubernetes OpenAPI definition key to an output
// file path. It returns the file path (e.g., "apps/v1.star"), whether the key
// identifies a special type that should be excluded from file assignment, and
// any error encountered.
//
// Priority-ordered prefix rules:
//  1. Known special types (IntOrString, Quantity) -> isSpecial=true
//  2. io.k8s.api.<group>.<version>.<Type> -> <group>/<version>.star
//  3. io.k8s.apimachinery.pkg.apis.meta.<version>.<Type> -> meta/<version>.star
//  4. io.k8s.apimachinery.pkg.apis.<group>.<version>.<Type> -> <group>/<version>.star
//  5. io.k8s.apimachinery.pkg.runtime.<Type> -> runtime/v1.star
//  6. io.k8s.kube-aggregator.pkg.apis.<group>.<version>.<Type> -> <group>/<version>.star
//  7. Fallback: use last three segments as <group>/<version>.star, log warning
func DefinitionKeyToFilePath(key string) (filePath string, isSpecial bool, err error) {
	// Rule 1: Check for well-known special types first.
	switch key {
	case "io.k8s.apimachinery.pkg.util.intstr.IntOrString":
		return "", true, nil
	case "io.k8s.apimachinery.pkg.api.resource.Quantity":
		return "", true, nil
	}

	parts := strings.Split(key, ".")

	// Rule 2: io.k8s.api.<group>.<version>.<Type>
	if strings.HasPrefix(key, "io.k8s.api.") && len(parts) >= 6 {
		group := parts[3]  // e.g., "apps", "core"
		version := parts[4] // e.g., "v1"
		return fmt.Sprintf("%s/%s.star", group, version), false, nil
	}

	// Rule 3: io.k8s.apimachinery.pkg.apis.meta.<version>.<Type>
	if strings.HasPrefix(key, "io.k8s.apimachinery.pkg.apis.meta.") && len(parts) >= 8 {
		version := parts[6] // e.g., "v1"
		return fmt.Sprintf("meta/%s.star", version), false, nil
	}

	// Rule 4: io.k8s.apimachinery.pkg.apis.<group>.<version>.<Type>
	if strings.HasPrefix(key, "io.k8s.apimachinery.pkg.apis.") && len(parts) >= 8 {
		group := parts[5]   // e.g., "apiregistration" -- wait, this is under apimachinery
		version := parts[6] // e.g., "v1"
		return fmt.Sprintf("%s/%s.star", group, version), false, nil
	}

	// Rule 5: io.k8s.apimachinery.pkg.runtime.<Type> -> runtime/v1.star
	if strings.HasPrefix(key, "io.k8s.apimachinery.pkg.runtime.") {
		return "runtime/v1.star", false, nil
	}

	// Rule 6: io.k8s.kube-aggregator.pkg.apis.<group>.<version>.<Type>
	// Note: "kube-aggregator" is a single segment because it contains a hyphen within
	// the definition key's dot-separated notation. The key looks like:
	// "io.k8s.kube-aggregator.pkg.apis.apiregistration.v1.APIService"
	// After splitting on ".", parts are: [io, k8s, kube-aggregator, pkg, apis, apiregistration, v1, APIService]
	if strings.HasPrefix(key, "io.k8s.kube-aggregator.pkg.apis.") && len(parts) >= 8 {
		group := parts[5]   // e.g., "apiregistration"
		version := parts[6] // e.g., "v1"
		return fmt.Sprintf("%s/%s.star", group, version), false, nil
	}

	// Rule 7: Fallback -- use last three segments: <group>.<version>.<Type>
	if len(parts) >= 3 {
		group := parts[len(parts)-3]
		version := parts[len(parts)-2]
		return fmt.Sprintf("%s/%s.star", group, version), false, nil
	}

	return "", false, fmt.Errorf("cannot determine file path for definition key %q", key)
}

// LoadPath constructs an OCI short-form load path by combining the package
// prefix with the file path. Example: LoadPath("schemas-k8s:v1.31", "apps/v1.star")
// returns "schemas-k8s:v1.31/apps/v1.star".
func LoadPath(pkg, filePath string) string {
	return fmt.Sprintf("%s/%s", pkg, filePath)
}

// TypeNameFromKey extracts the short type name (last segment) from a
// dot-separated definition key. Example: "io.k8s.api.apps.v1.Deployment" -> "Deployment".
func TypeNameFromKey(definitionKey string) string {
	parts := strings.Split(definitionKey, ".")
	if len(parts) == 0 {
		return definitionKey
	}
	return parts[len(parts)-1]
}
