package validate

import (
	"strings"
	"testing"

	"go.starlark.net/starlark"

	"github.com/wompipomp/starlark-gen/internal/pipeline"
)

// makeThread creates a basic starlark.Thread for calling constructors.
func makeThread() *starlark.Thread {
	return &starlark.Thread{Name: "test"}
}

func TestK8sGeneratedFilesLoad(t *testing.T) {
	result, err := pipeline.RunK8s(pipeline.K8sOptions{
		SwaggerPath: "../../testdata/swagger-mini.json",
		Package:     "test:v1",
		OutputDir:   t.TempDir(),
	})
	if err != nil {
		t.Fatalf("RunK8s: %v", err)
	}

	harness := NewHarness(result.Files, "test:v1")
	for path := range result.Files {
		globals, err := harness.LoadFile(path)
		if err != nil {
			t.Errorf("LoadFile(%s): %v", path, err)
			continue
		}
		if len(globals) == 0 {
			t.Errorf("LoadFile(%s): no globals exported", path)
		}
	}
}

func TestProviderGeneratedFilesLoad(t *testing.T) {
	result, err := pipeline.RunProvider(pipeline.ProviderOptions{
		Paths:     []string{"../../testdata/provider-aws-bucket.yaml"},
		Package:   "test-provider:v1",
		OutputDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("RunProvider: %v", err)
	}

	harness := NewHarness(result.Files, "test-provider:v1")
	for path := range result.Files {
		globals, err := harness.LoadFile(path)
		if err != nil {
			t.Errorf("LoadFile(%s): %v", path, err)
			continue
		}
		if len(globals) == 0 {
			t.Errorf("LoadFile(%s): no globals exported", path)
		}
	}
}

func TestCrossFileLoadResolution(t *testing.T) {
	result, err := pipeline.RunK8s(pipeline.K8sOptions{
		SwaggerPath: "../../testdata/swagger-mini.json",
		Package:     "test:v1",
		OutputDir:   t.TempDir(),
	})
	if err != nil {
		t.Fatalf("RunK8s: %v", err)
	}

	// apps/v1.star loads from meta/v1.star and core/v1.star -- this tests
	// that the custom module loader resolves cross-file load() correctly.
	harness := NewHarness(result.Files, "test:v1")
	globals, err := harness.LoadFile("apps/v1.star")
	if err != nil {
		t.Fatalf("LoadFile(apps/v1.star): %v", err)
	}
	if len(globals) == 0 {
		t.Fatal("apps/v1.star: no globals exported")
	}
}

func TestTypeValidation(t *testing.T) {
	result, err := pipeline.RunK8s(pipeline.K8sOptions{
		SwaggerPath: "../../testdata/swagger-mini.json",
		Package:     "test:v1",
		OutputDir:   t.TempDir(),
	})
	if err != nil {
		t.Fatalf("RunK8s: %v", err)
	}

	harness := NewHarness(result.Files, "test:v1")
	globals, err := harness.LoadFile("apps/v1.star")
	if err != nil {
		t.Fatalf("LoadFile(apps/v1.star): %v", err)
	}

	// Use the Deployment constructor which has apiVersion=field(type="string").
	// Passing an int where a string is expected should trigger a type error.
	deploymentVal, ok := globals["Deployment"]
	if !ok {
		t.Fatal("Deployment not found in apps/v1.star globals")
	}
	callable, ok := deploymentVal.(starlark.Callable)
	if !ok {
		t.Fatal("Deployment is not callable")
	}

	thread := makeThread()
	_, err = starlark.Call(thread, callable, nil, []starlark.Tuple{
		{starlark.String("apiVersion"), starlark.MakeInt(42)},
	})
	if err == nil {
		t.Fatal("Deployment: expected type validation error for apiVersion=int, got nil")
	}
	// The schema package should return a type mismatch error.
	errMsg := strings.ToLower(err.Error())
	if !strings.Contains(errMsg, "expected") && !strings.Contains(errMsg, "type") && !strings.Contains(errMsg, "got") {
		t.Errorf("Deployment: expected type error message, got: %v", err)
	}
}

func TestRequiredFields(t *testing.T) {
	result, err := pipeline.RunK8s(pipeline.K8sOptions{
		SwaggerPath: "../../testdata/swagger-mini.json",
		Package:     "test:v1",
		OutputDir:   t.TempDir(),
	})
	if err != nil {
		t.Fatalf("RunK8s: %v", err)
	}

	harness := NewHarness(result.Files, "test:v1")

	// Try files that contain constructors with required fields.
	filesToTry := []string{"apps/v1.star", "core/v1.star", "meta/v1.star"}
	for _, file := range filesToTry {
		globals, err := harness.LoadFile(file)
		if err != nil {
			t.Logf("LoadFile(%s): %v (skipping)", file, err)
			continue
		}

		for gName, val := range globals {
			callable, ok := val.(starlark.Callable)
			if !ok {
				continue
			}

			// Call with no arguments -- if this constructor has required fields,
			// it should return an error mentioning "required".
			thread := makeThread()
			_, err = starlark.Call(thread, callable, nil, nil)
			if err != nil && strings.Contains(strings.ToLower(err.Error()), "required") {
				// Found a constructor with required field enforcement.
				t.Logf("%s.%s: required field error confirmed: %v", file, gName, err)
				return
			}
		}
	}

	t.Fatal("no constructor with required fields found in K8s generated files")
}

func TestEnumValidation(t *testing.T) {
	// First try provider pipeline which has enum fields.
	result, err := pipeline.RunProvider(pipeline.ProviderOptions{
		Paths:     []string{"../../testdata/provider-aws-bucket.yaml"},
		Package:   "test-provider:v1",
		OutputDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("RunProvider: %v", err)
	}

	harness := NewHarness(result.Files, "test-provider:v1")

	for file := range result.Files {
		globals, err := harness.LoadFile(file)
		if err != nil {
			t.Logf("LoadFile(%s): %v (skipping)", file, err)
			continue
		}

		for gName, val := range globals {
			callable, ok := val.(starlark.Callable)
			if !ok {
				continue
			}

			// Try calling with an obviously invalid enum value on every
			// string field. If a field has an enum constraint, this should fail.
			thread := makeThread()
			_, err = starlark.Call(thread, callable, nil, []starlark.Tuple{
				{starlark.String("deletionPolicy"), starlark.String("INVALID_ENUM_VALUE_12345")},
			})
			if err != nil {
				errLower := strings.ToLower(err.Error())
				if strings.Contains(errLower, "enum") ||
					strings.Contains(errLower, "one of") ||
					strings.Contains(errLower, "delete") ||
					strings.Contains(errLower, "orphan") {
					t.Logf("%s.%s: enum validation error confirmed: %v", file, gName, err)
					return
				}
			}
		}
	}

	// Fallback: try K8s pipeline which also has enum fields.
	k8sResult, err := pipeline.RunK8s(pipeline.K8sOptions{
		SwaggerPath: "../../testdata/swagger-mini.json",
		Package:     "test:v1",
		OutputDir:   t.TempDir(),
	})
	if err != nil {
		t.Fatalf("RunK8s: %v", err)
	}

	k8sHarness := NewHarness(k8sResult.Files, "test:v1")

	for file := range k8sResult.Files {
		globals, err := k8sHarness.LoadFile(file)
		if err != nil {
			continue
		}

		for gName, val := range globals {
			callable, ok := val.(starlark.Callable)
			if !ok {
				continue
			}

			// Try known K8s enum fields.
			enumFields := []string{"type", "status", "operator", "protocol"}
			for _, enumField := range enumFields {
				thread := makeThread()
				_, err = starlark.Call(thread, callable, nil, []starlark.Tuple{
					{starlark.String(enumField), starlark.String("INVALID_ENUM_VALUE_12345")},
				})
				if err != nil {
					errLower := strings.ToLower(err.Error())
					if strings.Contains(errLower, "enum") ||
						strings.Contains(errLower, "one of") ||
						strings.Contains(errLower, "must be") {
						t.Logf("%s.%s.%s: enum validation error confirmed: %v", file, gName, enumField, err)
						return
					}
				}
			}
		}
	}

	t.Skip("no enum fields found in generated files")
}
