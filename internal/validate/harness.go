// Package validate provides a Starlark test harness for runtime validation
// of generated .star files against the function-starlark v1.7 builtins.
//
// The Harness embeds the Starlark interpreter with schema() and field()
// builtins pre-declared, and provides a custom module loader that resolves
// OCI short-form load() paths (e.g., "test:v1/meta/v1.star") to generated
// file content held in memory.
package validate

import (
	"fmt"
	"strings"

	"github.com/wompipomp/function-starlark/schema"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// cacheEntry stores the result of loading a module so that repeated
// Thread.Load calls return the same StringDict (required by the Starlark
// specification).
type cacheEntry struct {
	globals starlark.StringDict
	err     error
}

// Harness provides a test environment for executing generated .star files
// against the function-starlark runtime builtins.
type Harness struct {
	// predeclared holds the schema() and field() builtins.
	predeclared starlark.StringDict

	// files maps relative file paths (e.g., "apps/v1.star") to their content.
	files map[string][]byte

	// packagePrefix is the OCI package prefix (e.g., "test:v1").
	packagePrefix string

	// cache stores loaded module globals for repeated load() calls.
	cache map[string]*cacheEntry
}

// NewHarness creates a Harness from pipeline output.
//
// The files map should be the EmitResult from a pipeline run, keyed by
// relative file path. The packagePrefix should match the Package option
// passed to the pipeline (e.g., "test:v1").
func NewHarness(files map[string][]byte, packagePrefix string) *Harness {
	return &Harness{
		predeclared: starlark.StringDict{
			"schema": schema.SchemaBuiltin(),
			"field":  schema.FieldBuiltin(),
		},
		files:         files,
		packagePrefix: packagePrefix,
		cache:         make(map[string]*cacheEntry),
	}
}

// LoadFile executes a .star file and returns its globals.
//
// The file is looked up by relative path in the files map. A new
// starlark.Thread is created with the custom module loader attached
// so that cross-file load() statements resolve correctly.
func (h *Harness) LoadFile(relPath string) (starlark.StringDict, error) {
	content, ok := h.files[relPath]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", relPath)
	}

	thread := &starlark.Thread{
		Name: relPath,
		Load: h.load,
	}

	return starlark.ExecFileOptions(&syntax.FileOptions{}, thread, relPath, content, h.predeclared)
}

// load implements the Thread.Load callback.
//
// It maps OCI short-form paths like "test:v1/meta/v1.star" to "meta/v1.star"
// by stripping the package prefix. Results are cached so that repeated calls
// with the same module name return the identical StringDict (required by the
// Starlark specification).
func (h *Harness) load(_ *starlark.Thread, module string) (starlark.StringDict, error) {
	// Check cache first (required: repeated calls must return same result).
	if e, ok := h.cache[module]; ok {
		return e.globals, e.err
	}

	// Strip package prefix to get relative path.
	// "test:v1/meta/v1.star" -> "meta/v1.star"
	relPath := strings.TrimPrefix(module, h.packagePrefix+"/")

	globals, err := h.LoadFile(relPath)
	h.cache[module] = &cacheEntry{globals, err}
	return globals, err
}
