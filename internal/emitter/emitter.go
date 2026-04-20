// Package emitter generates Starlark schema code from resolved TypeNodes.
//
// The Emit function processes a FileMap (from the Organizer) and produces
// EmitResult, a map of file paths to generated .star file content. The
// EmitFile function handles a single file's code generation.
package emitter

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/wompipomp/starlark-gen/internal/organizer"
	"github.com/wompipomp/starlark-gen/internal/types"
)

// EmitResult maps file paths (e.g., "apps/v1.star") to their generated content.
type EmitResult map[string][]byte

// Emit generates Starlark code for all files in the given fileMap.
// Files are processed in the provided fileOrder (from ValidateLoadDAG).
// It builds an allNodes lookup from the fileMap for cross-file reference resolution.
func Emit(fileMap map[string][]*types.TypeNode, fileOrder []string, pkg string) (EmitResult, error) {
	// Build allNodes lookup from all types across all files.
	allNodes := make(map[string]*types.TypeNode)
	for _, nodes := range fileMap {
		for _, n := range nodes {
			allNodes[n.DefinitionKey] = n
		}
	}

	result := make(EmitResult)
	for _, fp := range fileOrder {
		nodes, ok := fileMap[fp]
		if !ok {
			continue
		}
		content, err := EmitFile(fp, nodes, allNodes, pkg)
		if err != nil {
			return nil, fmt.Errorf("emitting %s: %w", fp, err)
		}
		result[fp] = content
	}

	return result, nil
}

// EmitFile generates the Starlark content for a single output file.
//
// It produces:
//  1. load() statements for cross-file dependencies (sorted alphabetically by path, symbols grouped)
//  2. schema() definitions for each type (in the order provided, assumed pre-sorted)
//
// allNodes maps DefinitionKey to TypeNode for all types across all files,
// enabling cross-file reference resolution.
func EmitFile(filePath string, nodes []*types.TypeNode, allNodes map[string]*types.TypeNode, pkg string) ([]byte, error) {
	var buf bytes.Buffer

	// Build a set of DefinitionKeys for types in this file (for intra-file detection).
	fileTypes := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		fileTypes[n.DefinitionKey] = true
	}

	// Compute load() statements: group cross-file dependencies by source file.
	// map[sourceFilePath]map[symbolName]bool
	imports := make(map[string]map[string]bool)
	for _, n := range nodes {
		for _, depKey := range n.Dependencies {
			// Skip intra-file dependencies.
			if fileTypes[depKey] {
				continue
			}
			dep, ok := allNodes[depKey]
			if !ok {
				continue
			}
			if dep.FilePath == "" || dep.FilePath == filePath {
				continue
			}
			if imports[dep.FilePath] == nil {
				imports[dep.FilePath] = make(map[string]bool)
			}
			imports[dep.FilePath][dep.Name] = true
		}

		// Also check Items field for list types with cross-file schema items.
		for _, f := range n.Fields {
			if f.Items != "" && !fileTypes[f.Items] {
				dep, ok := allNodes[f.Items]
				if !ok {
					continue
				}
				if dep.FilePath == "" || dep.FilePath == filePath {
					continue
				}
				if imports[dep.FilePath] == nil {
					imports[dep.FilePath] = make(map[string]bool)
				}
				imports[dep.FilePath][dep.Name] = true
			}
		}
	}

	// Sort source files alphabetically for deterministic output.
	sortedFiles := make([]string, 0, len(imports))
	for fp := range imports {
		sortedFiles = append(sortedFiles, fp)
	}
	sort.Strings(sortedFiles)

	// Emit load() statements.
	for _, srcFile := range sortedFiles {
		symbols := imports[srcFile]
		sortedSymbols := make([]string, 0, len(symbols))
		for sym := range symbols {
			sortedSymbols = append(sortedSymbols, sym)
		}
		sort.Strings(sortedSymbols)

		loadPath := organizer.LoadPath(pkg, srcFile)
		fmt.Fprintf(&buf, "load(%q", loadPath)
		for _, sym := range sortedSymbols {
			fmt.Fprintf(&buf, ", %q", sym)
		}
		buf.WriteString(")\n")
	}

	// Blank line after load block (if any loads were emitted).
	if len(imports) > 0 {
		buf.WriteString("\n")
	}

	// Emit schema() definitions for each type.
	for i, n := range nodes {
		if i > 0 {
			buf.WriteString("\n")
		}
		emitSchema(&buf, n, fileTypes, allNodes)
	}

	return buf.Bytes(), nil
}

// emitSchema writes a single schema() definition to the buffer.
// Enum constants are emitted immediately above the schema() call.
func emitSchema(buf *bytes.Buffer, n *types.TypeNode, fileTypes map[string]bool, allNodes map[string]*types.TypeNode) {
	// Emit enum constants above the schema definition.
	emitEnumConstants(buf, n)

	fmt.Fprintf(buf, "%s = schema(\n", n.Name)
	fmt.Fprintf(buf, "    %q,\n", n.Name)

	if n.Description != "" {
		fmt.Fprintf(buf, "    doc=%q,\n", n.Description)
	}

	for _, f := range n.Fields {
		emitField(buf, f, fileTypes, allNodes)
	}

	buf.WriteString(")\n")
}

// emitField writes a single field() call to the buffer.
func emitField(buf *bytes.Buffer, f types.FieldNode, fileTypes map[string]bool, allNodes map[string]*types.TypeNode) {
	var parts []string

	// Determine type= parameter.
	if f.SchemaRef != "" {
		// Schema reference: use bare type name (whether intra-file or cross-file,
		// cross-file types are imported via load() at the top).
		typeName := organizer.TypeNameFromKey(f.SchemaRef)
		parts = append(parts, fmt.Sprintf("type=%s", typeName))
	} else {
		// Primitive, dict, list, or gradual typing: quoted string.
		parts = append(parts, fmt.Sprintf("type=%q", f.TypeName))
	}

	// Required flag.
	if f.Required {
		parts = append(parts, "required=True")
	}

	// Items for list types with schema items.
	if f.Items != "" {
		itemTypeName := organizer.TypeNameFromKey(f.Items)
		parts = append(parts, fmt.Sprintf("items=%s", itemTypeName))
	}

	// Enum values. Skipped for list types: function-starlark's `enum` applies
	// to the whole value, not elements, so emitting it on a list would reject
	// every non-empty list. Item-level enums are enforced by the API server;
	// the values still surface in the docstring and as named constants.
	if len(f.EnumValues) > 0 && f.TypeName != "list" {
		quoted := make([]string, len(f.EnumValues))
		for i, v := range f.EnumValues {
			quoted[i] = fmt.Sprintf("%q", v)
		}
		parts = append(parts, fmt.Sprintf("enum=[%s]", strings.Join(quoted, ", ")))
	}

	// Default value.
	if f.Default != nil {
		if formatted, ok := formatStarlarkDefault(f.Default); ok {
			parts = append(parts, fmt.Sprintf("default=%s", formatted))
		}
	}

	// Build doc string.
	doc := buildFieldDoc(f, allNodes)
	parts = append(parts, fmt.Sprintf("doc=%q", doc))

	fmt.Fprintf(buf, "    %s=field(%s),\n", sanitizeFieldName(f.Name), strings.Join(parts, ", "))
}

// sanitizeFieldName replaces characters that are not valid in Starlark
// identifiers (hyphens, dots, etc.) with underscores so that the emitted
// keyword argument is syntactically valid.
func sanitizeFieldName(name string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			return r
		}
		return '_'
	}, name)
}

// buildFieldDoc constructs the doc string for a field following user decisions:
// - Format: "<TypeLabel> - <Description>"
// - Append " (required)" if required
// - Append ". One of: val1, val2, val3" if enum values present
func buildFieldDoc(f types.FieldNode, allNodes map[string]*types.TypeNode) string {
	var typeLabel string

	switch {
	case f.SchemaRef != "":
		typeLabel = organizer.TypeNameFromKey(f.SchemaRef)
	case f.TypeName != "":
		typeLabel = f.TypeName
	default:
		// Gradual typing (empty string type): description is self-sufficient.
		typeLabel = ""
	}

	var doc string
	if typeLabel != "" && f.Description != "" {
		doc = typeLabel + " - " + f.Description
	} else if typeLabel != "" {
		doc = typeLabel
	} else {
		doc = f.Description
	}

	if f.Required {
		doc += " (required)"
	}

	if len(f.EnumValues) > 0 {
		doc += ". One of: " + strings.Join(f.EnumValues, ", ")
	}

	// Append default value indicator.
	if f.Default != nil {
		if docDefault, ok := formatDocDefault(f.Default); ok {
			doc += " (default: " + docDefault + ")"
		}
	}

	return doc
}

// emitEnumConstants writes SCREAMING_SNAKE_CASE named constants for all enum
// fields in the given TypeNode. Constants are emitted immediately above the
// schema() definition so they are grouped with their related code.
func emitEnumConstants(buf *bytes.Buffer, n *types.TypeNode) {
	for _, f := range n.Fields {
		if len(f.EnumValues) == 0 {
			continue
		}
		for _, val := range f.EnumValues {
			name := enumConstantName(n.Name, f.Name, val)
			fmt.Fprintf(buf, "%s = %q\n", name, val)
		}
		buf.WriteString("\n")
	}
}

// enumConstantName builds a SCREAMING_SNAKE_CASE constant name from the type
// name, field name, and enum value. The full path prevents collisions across
// types. Example: enumConstantName("Certificate", "privateKeyAlgorithm", "RSA")
// returns "CERTIFICATE_PRIVATE_KEY_ALGORITHM_RSA".
func enumConstantName(typeName, fieldName, value string) string {
	return toScreamingSnake(typeName) + "_" +
		toScreamingSnake(fieldName) + "_" +
		toScreamingSnake(value)
}

// toScreamingSnake converts a string to SCREAMING_SNAKE_CASE.
//
// Rules:
//   - Insert '_' before uppercase letters preceded by lowercase: PrivateKey -> PRIVATE_KEY
//   - Handle transitions from uppercase run to uppercase+lowercase: HTTPSProxy -> HTTPS_PROXY
//   - Replace non-alphanumeric characters with '_'
//   - Collapse consecutive underscores
//   - Strip leading/trailing underscores
//   - Convert everything to uppercase
func toScreamingSnake(s string) string {
	if s == "" {
		return ""
	}

	var buf bytes.Buffer
	runes := []rune(s)

	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			buf.WriteRune('_')
			continue
		}

		if unicode.IsUpper(r) && i > 0 {
			prev := runes[i-1]
			// Insert '_' before uppercase preceded by lowercase: aB -> a_B
			if unicode.IsLower(prev) {
				buf.WriteRune('_')
			} else if unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
				// Insert '_' at transitions like HTTPSProxy -> HTTPS_Proxy boundary
				buf.WriteRune('_')
			}
		}

		buf.WriteRune(unicode.ToUpper(r))
	}

	// Collapse consecutive underscores and strip leading/trailing.
	result := buf.String()
	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}
	result = strings.Trim(result, "_")

	return result
}

// formatStarlarkDefault converts a Go value to its Starlark literal representation
// for use in field(default=...). Returns the formatted string and whether the
// value is a supported primitive type.
//
// Supported types: string (quoted), bool (True/False), int (%d), float64 (%g).
// Unsupported types (map, slice, nil): returns "", false.
func formatStarlarkDefault(val interface{}) (string, bool) {
	switch v := val.(type) {
	case string:
		return fmt.Sprintf("%q", v), true
	case bool:
		if v {
			return "True", true
		}
		return "False", true
	case int:
		return fmt.Sprintf("%d", v), true
	case float64:
		// If value equals its integer conversion, format as int.
		if v == float64(int(v)) {
			return fmt.Sprintf("%d", int(v)), true
		}
		return fmt.Sprintf("%g", v), true
	default:
		return "", false
	}
}

// formatDocDefault converts a Go value to its display representation for use in
// doc strings: (default: value). Unlike formatStarlarkDefault, string values are
// NOT quoted in the doc string. Bool values use Starlark True/False.
func formatDocDefault(val interface{}) (string, bool) {
	switch v := val.(type) {
	case string:
		return v, true
	case bool:
		if v {
			return "True", true
		}
		return "False", true
	case int:
		return fmt.Sprintf("%d", v), true
	case float64:
		if v == float64(int(v)) {
			return fmt.Sprintf("%d", int(v)), true
		}
		return fmt.Sprintf("%g", v), true
	default:
		return "", false
	}
}
