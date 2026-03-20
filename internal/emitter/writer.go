package emitter

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// WriteFiles writes the generated content to disk under the given output directory.
// Files are written in sorted order for deterministic behavior.
// It creates parent directories as needed and overwrites existing files.
// Non-generated files in the output directory are left untouched.
//
// Returns the number of files written, the total schema count across all files,
// and any error encountered during writing.
func WriteFiles(result EmitResult, outputDir string) (fileCount int, schemaCount int, err error) {
	// Sort file paths for deterministic write order.
	paths := make([]string, 0, len(result))
	for fp := range result {
		paths = append(paths, fp)
	}
	sort.Strings(paths)

	for _, fp := range paths {
		content := result[fp]
		fullPath := filepath.Join(outputDir, fp)

		// Create parent directories.
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return fileCount, schemaCount, fmt.Errorf("creating directory for %s: %w", fp, err)
		}

		// Write file content (overwrites existing).
		if err := os.WriteFile(fullPath, content, 0o644); err != nil {
			return fileCount, schemaCount, fmt.Errorf("writing %s: %w", fp, err)
		}

		// Count schemas by counting occurrences of " = schema(" in content.
		schemaCount += bytes.Count(content, []byte(" = schema("))
		fileCount++
	}

	return fileCount, schemaCount, nil
}

// SummaryLine returns the default summary line for successful generation.
// Format: "Generated N files (M schemas) in <outputDir>"
func SummaryLine(fileCount, schemaCount int, outputDir string) string {
	return fmt.Sprintf("Generated %d files (%d schemas) in %s", fileCount, schemaCount, outputDir)
}
