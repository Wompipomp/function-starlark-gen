package validate

import (
	"strings"
	"testing"

	"go.starlark.net/starlark"
)

func TestNewHarness_Predeclared(t *testing.T) {
	h := NewHarness(nil, "pkg:v1")

	for _, name := range []string{"schema", "field"} {
		v, ok := h.predeclared[name]
		if !ok {
			t.Errorf("predeclared missing %q builtin", name)
			continue
		}
		if _, ok := v.(starlark.Callable); !ok {
			t.Errorf("predeclared[%q] is %T, want starlark.Callable", name, v)
		}
	}
}

func TestLoadFile_FileNotFound(t *testing.T) {
	h := NewHarness(map[string][]byte{}, "pkg:v1")

	_, err := h.LoadFile("nonexistent.star")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent.star") {
		t.Errorf("error should mention filename, got: %v", err)
	}
}

func TestLoadFile_ValidFile(t *testing.T) {
	files := map[string][]byte{
		"hello.star": []byte(`x = 1`),
	}
	h := NewHarness(files, "pkg:v1")

	globals, err := h.LoadFile("hello.star")
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	v, ok := globals["x"]
	if !ok {
		t.Fatal("expected global 'x' to be exported")
	}
	if v.String() != "1" {
		t.Errorf("x = %s, want 1", v.String())
	}
}

func TestLoad_PrefixStripping(t *testing.T) {
	files := map[string][]byte{
		"group/v1.star": []byte(`y = "hello"`),
	}
	h := NewHarness(files, "pkg:v1")

	// Simulate a load() call with full OCI path.
	thread := &starlark.Thread{Name: "test"}
	globals, err := h.load(thread, "pkg:v1/group/v1.star")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	v, ok := globals["y"]
	if !ok {
		t.Fatal("expected global 'y' after prefix stripping")
	}
	if v.String() != `"hello"` {
		t.Errorf("y = %s, want \"hello\"", v.String())
	}
}

func TestLoad_CacheBehavior(t *testing.T) {
	files := map[string][]byte{
		"cached.star": []byte(`z = 42`),
	}
	h := NewHarness(files, "pkg:v1")

	thread := &starlark.Thread{Name: "test"}

	first, err := h.load(thread, "pkg:v1/cached.star")
	if err != nil {
		t.Fatalf("first load: %v", err)
	}

	second, err := h.load(thread, "pkg:v1/cached.star")
	if err != nil {
		t.Fatalf("second load: %v", err)
	}

	// Per Starlark spec, repeated load() must return the same StringDict.
	if len(first) != len(second) {
		t.Fatalf("cache returned different lengths: %d vs %d", len(first), len(second))
	}
	for k, v1 := range first {
		v2, ok := second[k]
		if !ok {
			t.Errorf("cached result missing key %q", k)
			continue
		}
		if v1 != v2 {
			t.Errorf("cached result for %q: got different starlark.Value instances", k)
		}
	}
}
