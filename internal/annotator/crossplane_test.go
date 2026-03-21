package annotator

import (
	"strings"
	"testing"

	"github.com/wompipomp/starlark-gen/internal/types"
)

// makeTypeNode is a test helper that constructs a TypeNode with the given parameters.
func makeTypeNode(name, defKey, desc string, fields []types.FieldNode, deps []string) types.TypeNode {
	return types.TypeNode{
		Name:          name,
		DefinitionKey: defKey,
		Description:   desc,
		Fields:        fields,
		Dependencies:  deps,
		FilePath:      "test/v1.star",
	}
}

func findNode(nodes []types.TypeNode, name string) *types.TypeNode {
	for i := range nodes {
		if nodes[i].Name == name {
			return &nodes[i]
		}
	}
	return nil
}

func findField(node *types.TypeNode, name string) *types.FieldNode {
	for i := range node.Fields {
		if node.Fields[i].Name == name {
			return &node.Fields[i]
		}
	}
	return nil
}

// buildStandardProviderNodes constructs a minimal TypeNode graph for a standard
// Crossplane provider CRD with both forProvider and initProvider.
func buildStandardProviderNodes() []types.TypeNode {
	return []types.TypeNode{
		makeTypeNode("Bucket", "s3.aws.v1beta1.Bucket", "", []types.FieldNode{
			{Name: "spec", SchemaRef: "s3.aws.v1beta1.BucketSpec"},
			{Name: "status", SchemaRef: "s3.aws.v1beta1.BucketStatus"},
		}, []string{"s3.aws.v1beta1.BucketSpec", "s3.aws.v1beta1.BucketStatus"}),

		makeTypeNode("BucketSpec", "s3.aws.v1beta1.BucketSpec", "", []types.FieldNode{
			{Name: "deletionPolicy", TypeName: "string", Description: "DeletionPolicy specifies what happens."},
			{Name: "forProvider", SchemaRef: "s3.aws.v1beta1.BucketForProvider"},
			{Name: "initProvider", SchemaRef: "s3.aws.v1beta1.BucketInitProvider"},
			{Name: "managementPolicies", TypeName: "list"},
			{Name: "providerConfigRef", SchemaRef: "s3.aws.v1beta1.BucketProviderConfigRef", Description: "ProviderConfigReference specifies config."},
			{Name: "writeConnectionSecretToRef", SchemaRef: "s3.aws.v1beta1.BucketWriteConnectionSecretToRef"},
			{Name: "publishConnectionDetailsTo", SchemaRef: "s3.aws.v1beta1.BucketPublishConnectionDetailsTo"},
		}, []string{
			"s3.aws.v1beta1.BucketForProvider",
			"s3.aws.v1beta1.BucketInitProvider",
			"s3.aws.v1beta1.BucketProviderConfigRef",
			"s3.aws.v1beta1.BucketWriteConnectionSecretToRef",
			"s3.aws.v1beta1.BucketPublishConnectionDetailsTo",
		}),

		makeTypeNode("BucketForProvider", "s3.aws.v1beta1.BucketForProvider",
			"Parameters for the Bucket resource", []types.FieldNode{
				{Name: "region", TypeName: "string"},
				{Name: "tags", TypeName: "dict", IsMap: true},
			}, nil),

		makeTypeNode("BucketInitProvider", "s3.aws.v1beta1.BucketInitProvider",
			"InitProvider holds the same fields", []types.FieldNode{
				{Name: "tags", TypeName: "dict", IsMap: true},
			}, nil),

		makeTypeNode("BucketProviderConfigRef", "s3.aws.v1beta1.BucketProviderConfigRef", "", []types.FieldNode{
			{Name: "name", TypeName: "string", Required: true},
		}, nil),

		makeTypeNode("BucketWriteConnectionSecretToRef", "s3.aws.v1beta1.BucketWriteConnectionSecretToRef", "", []types.FieldNode{
			{Name: "name", TypeName: "string"},
			{Name: "namespace", TypeName: "string"},
		}, nil),

		makeTypeNode("BucketPublishConnectionDetailsTo", "s3.aws.v1beta1.BucketPublishConnectionDetailsTo", "", []types.FieldNode{
			{Name: "name", TypeName: "string"},
		}, nil),

		makeTypeNode("BucketStatus", "s3.aws.v1beta1.BucketStatus", "", []types.FieldNode{
			{Name: "atProvider", SchemaRef: "s3.aws.v1beta1.BucketAtProvider"},
			{Name: "conditions", TypeName: "list"},
		}, []string{"s3.aws.v1beta1.BucketAtProvider"}),

		makeTypeNode("BucketAtProvider", "s3.aws.v1beta1.BucketAtProvider", "", []types.FieldNode{
			{Name: "arn", TypeName: "string"},
			{Name: "bucketDomainName", TypeName: "string"},
		}, nil),
	}
}

// buildForProviderOnlyNodes constructs a minimal TypeNode graph for a Crossplane
// provider CRD with forProvider but no initProvider.
func buildForProviderOnlyNodes() []types.TypeNode {
	return []types.TypeNode{
		makeTypeNode("Release", "helm.v1beta1.Release", "", []types.FieldNode{
			{Name: "spec", SchemaRef: "helm.v1beta1.ReleaseSpec"},
			{Name: "status", SchemaRef: "helm.v1beta1.ReleaseStatus"},
		}, []string{"helm.v1beta1.ReleaseSpec", "helm.v1beta1.ReleaseStatus"}),

		makeTypeNode("ReleaseSpec", "helm.v1beta1.ReleaseSpec", "", []types.FieldNode{
			{Name: "deletionPolicy", TypeName: "string"},
			{Name: "forProvider", SchemaRef: "helm.v1beta1.ReleaseForProvider"},
			{Name: "providerConfigRef", SchemaRef: "helm.v1beta1.ReleaseProviderConfigRef"},
		}, []string{
			"helm.v1beta1.ReleaseForProvider",
			"helm.v1beta1.ReleaseProviderConfigRef",
		}),

		makeTypeNode("ReleaseForProvider", "helm.v1beta1.ReleaseForProvider",
			"Parameters for the Helm Release", []types.FieldNode{
				{Name: "namespace", TypeName: "string"},
			}, nil),

		makeTypeNode("ReleaseProviderConfigRef", "helm.v1beta1.ReleaseProviderConfigRef", "", []types.FieldNode{
			{Name: "name", TypeName: "string", Required: true},
		}, nil),

		makeTypeNode("ReleaseStatus", "helm.v1beta1.ReleaseStatus", "", []types.FieldNode{
			{Name: "atProvider", SchemaRef: "helm.v1beta1.ReleaseAtProvider"},
		}, []string{"helm.v1beta1.ReleaseAtProvider"}),

		makeTypeNode("ReleaseAtProvider", "helm.v1beta1.ReleaseAtProvider", "", []types.FieldNode{
			{Name: "state", TypeName: "string"},
		}, nil),
	}
}

// buildNonStandardNodes constructs a minimal TypeNode graph for a CRD with
// no forProvider/initProvider structure.
func buildNonStandardNodes() []types.TypeNode {
	return []types.TypeNode{
		makeTypeNode("Thing", "custom.v1.Thing", "", []types.FieldNode{
			{Name: "spec", SchemaRef: "custom.v1.ThingSpec"},
			{Name: "status", SchemaRef: "custom.v1.ThingStatus"},
		}, []string{"custom.v1.ThingSpec", "custom.v1.ThingStatus"}),

		makeTypeNode("ThingSpec", "custom.v1.ThingSpec", "", []types.FieldNode{
			{Name: "someField", TypeName: "string"},
			{Name: "config", SchemaRef: "custom.v1.ThingConfig"},
		}, []string{"custom.v1.ThingConfig"}),

		makeTypeNode("ThingConfig", "custom.v1.ThingConfig", "", []types.FieldNode{
			{Name: "key", TypeName: "string"},
			{Name: "value", TypeName: "string"},
		}, nil),

		makeTypeNode("ThingStatus", "custom.v1.ThingStatus", "", []types.FieldNode{
			{Name: "observedState", SchemaRef: "custom.v1.ThingObservedState"},
		}, []string{"custom.v1.ThingObservedState"}),

		makeTypeNode("ThingObservedState", "custom.v1.ThingObservedState", "", []types.FieldNode{
			{Name: "ready", TypeName: "bool"},
		}, nil),
	}
}

func TestAnnotateCrossplane_StandardProvider(t *testing.T) {
	nodes := buildStandardProviderNodes()
	result, warnings := AnnotateCrossplane(nodes)

	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for standard provider, got %d: %v", len(warnings), warnings)
	}

	// Status subtree should be removed.
	if findNode(result, "BucketStatus") != nil {
		t.Error("BucketStatus should be removed")
	}
	if findNode(result, "BucketAtProvider") != nil {
		t.Error("BucketAtProvider should be removed")
	}

	// Root should not have status field.
	bucket := findNode(result, "Bucket")
	if bucket == nil {
		t.Fatal("Bucket not found in result")
	}
	if findField(bucket, "status") != nil {
		t.Error("Bucket should not have status field after annotation")
	}

	// forProvider TypeNode description should be augmented.
	fp := findNode(result, "BucketForProvider")
	if fp == nil {
		t.Fatal("BucketForProvider not found")
	}
	if !strings.HasPrefix(fp.Description, "Reconcilable configuration. Fields here are continuously reconciled.") {
		t.Errorf("BucketForProvider.Description = %q, want prefix 'Reconcilable configuration...'", fp.Description)
	}
	if !strings.Contains(fp.Description, "Parameters for the Bucket resource") {
		t.Errorf("BucketForProvider.Description should contain original desc, got %q", fp.Description)
	}

	// initProvider TypeNode description should be augmented.
	ip := findNode(result, "BucketInitProvider")
	if ip == nil {
		t.Fatal("BucketInitProvider not found")
	}
	if !strings.HasPrefix(ip.Description, "Write-once initialization. Fields here are set only at creation.") {
		t.Errorf("BucketInitProvider.Description = %q, want prefix 'Write-once initialization...'", ip.Description)
	}

	// forProvider/initProvider fields on spec should also be annotated.
	spec := findNode(result, "BucketSpec")
	if spec == nil {
		t.Fatal("BucketSpec not found")
	}
	fpField := findField(spec, "forProvider")
	if fpField == nil {
		t.Fatal("BucketSpec.forProvider field not found")
	}
	if !strings.Contains(fpField.Description, "Reconcilable configuration") {
		t.Errorf("forProvider field Description = %q, want Reconcilable prefix", fpField.Description)
	}
	ipField := findField(spec, "initProvider")
	if ipField == nil {
		t.Fatal("BucketSpec.initProvider field not found")
	}
	if !strings.Contains(ipField.Description, "Write-once initialization") {
		t.Errorf("initProvider field Description = %q, want Write-once prefix", ipField.Description)
	}
}

func TestAnnotateCrossplane_ForProviderOnly(t *testing.T) {
	nodes := buildForProviderOnlyNodes()
	result, warnings := AnnotateCrossplane(nodes)

	// No warnings -- initProvider is optional.
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for forProvider-only, got %d: %v", len(warnings), warnings)
	}

	// Status should be removed.
	if findNode(result, "ReleaseStatus") != nil {
		t.Error("ReleaseStatus should be removed")
	}
	if findNode(result, "ReleaseAtProvider") != nil {
		t.Error("ReleaseAtProvider should be removed")
	}

	// forProvider should be annotated.
	fp := findNode(result, "ReleaseForProvider")
	if fp == nil {
		t.Fatal("ReleaseForProvider not found")
	}
	if !strings.HasPrefix(fp.Description, "Reconcilable configuration. Fields here are continuously reconciled.") {
		t.Errorf("ReleaseForProvider.Description = %q, want Reconcilable prefix", fp.Description)
	}

	// No initProvider to annotate -- just verify no crash.
	spec := findNode(result, "ReleaseSpec")
	if spec == nil {
		t.Fatal("ReleaseSpec not found")
	}
	if findField(spec, "initProvider") != nil {
		t.Error("ReleaseSpec should not have initProvider field")
	}
}

func TestAnnotateCrossplane_NonStandard(t *testing.T) {
	nodes := buildNonStandardNodes()
	result, warnings := AnnotateCrossplane(nodes)

	// Should emit a warning.
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning for non-standard CRD, got %d: %v", len(warnings), warnings)
	}
	expected := "warn: Thing: no forProvider/initProvider structure found, generating as plain CRD"
	if warnings[0] != expected {
		t.Errorf("warning = %q, want %q", warnings[0], expected)
	}

	// All nodes should pass through unchanged.
	if len(result) != len(nodes) {
		t.Errorf("expected %d nodes (unchanged), got %d", len(nodes), len(result))
	}

	// Status should still be present for non-standard CRDs.
	if findNode(result, "ThingStatus") == nil {
		t.Error("ThingStatus should still be present for non-standard CRD")
	}
	thing := findNode(result, "Thing")
	if thing == nil {
		t.Fatal("Thing not found")
	}
	if findField(thing, "status") == nil {
		t.Error("Thing should still have status field for non-standard CRD")
	}
}

func TestAnnotateCrossplane_StatusRemoval(t *testing.T) {
	nodes := buildStandardProviderNodes()
	result, _ := AnnotateCrossplane(nodes)

	// Verify status field removed from root.
	bucket := findNode(result, "Bucket")
	if bucket == nil {
		t.Fatal("Bucket not found")
	}
	if findField(bucket, "status") != nil {
		t.Error("status field should be removed from Bucket root")
	}

	// Verify status TypeNode removed.
	if findNode(result, "BucketStatus") != nil {
		t.Error("BucketStatus TypeNode should be removed")
	}

	// Verify status sub-TypeNodes removed.
	if findNode(result, "BucketAtProvider") != nil {
		t.Error("BucketAtProvider TypeNode should be removed")
	}

	// Verify status dependency removed from root.
	for _, dep := range bucket.Dependencies {
		if dep == "s3.aws.v1beta1.BucketStatus" {
			t.Error("BucketStatus should be removed from Bucket.Dependencies")
		}
	}

	// Verify spec and other nodes are preserved.
	if findNode(result, "BucketSpec") == nil {
		t.Error("BucketSpec should be preserved")
	}
	if findNode(result, "BucketForProvider") == nil {
		t.Error("BucketForProvider should be preserved")
	}
	if findNode(result, "BucketProviderConfigRef") == nil {
		t.Error("BucketProviderConfigRef should be preserved")
	}
}

func TestAnnotateCrossplane_StandardFieldAnnotations(t *testing.T) {
	nodes := buildStandardProviderNodes()
	result, _ := AnnotateCrossplane(nodes)

	spec := findNode(result, "BucketSpec")
	if spec == nil {
		t.Fatal("BucketSpec not found")
	}

	tests := []struct {
		fieldName  string
		wantPrefix string
	}{
		{"providerConfigRef", "Reference to the ProviderConfig for auth"},
		{"writeConnectionSecretToRef", "Where to write connection details secret"},
		{"publishConnectionDetailsTo", "Where to publish connection details"},
		{"deletionPolicy", "Delete or Orphan the external resource on CR deletion"},
		{"managementPolicies", "Actions Crossplane is allowed to take on the resource"},
	}

	for _, tc := range tests {
		t.Run(tc.fieldName, func(t *testing.T) {
			field := findField(spec, tc.fieldName)
			if field == nil {
				t.Fatalf("BucketSpec.%s not found", tc.fieldName)
			}
			if !strings.Contains(field.Description, tc.wantPrefix) {
				t.Errorf("BucketSpec.%s.Description = %q, want to contain %q",
					tc.fieldName, field.Description, tc.wantPrefix)
			}
		})
	}
}

func TestAnnotateCrossplane_NoStandardFieldAnnotationOnNonStandard(t *testing.T) {
	// Build a non-standard CRD that happens to have providerConfigRef, deletionPolicy, etc.
	nodes := []types.TypeNode{
		makeTypeNode("Widget", "test.v1.Widget", "", []types.FieldNode{
			{Name: "spec", SchemaRef: "test.v1.WidgetSpec"},
		}, []string{"test.v1.WidgetSpec"}),

		makeTypeNode("WidgetSpec", "test.v1.WidgetSpec", "", []types.FieldNode{
			{Name: "providerConfigRef", SchemaRef: "test.v1.WidgetProviderConfigRef", Description: "Some config ref"},
			{Name: "deletionPolicy", TypeName: "string", Description: "Original deletion doc"},
			{Name: "someField", TypeName: "string"},
		}, []string{"test.v1.WidgetProviderConfigRef"}),

		makeTypeNode("WidgetProviderConfigRef", "test.v1.WidgetProviderConfigRef", "", []types.FieldNode{
			{Name: "name", TypeName: "string"},
		}, nil),
	}

	result, warnings := AnnotateCrossplane(nodes)

	// Should warn about non-standard.
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}

	// Standard Crossplane field annotations should NOT be applied.
	spec := findNode(result, "WidgetSpec")
	if spec == nil {
		t.Fatal("WidgetSpec not found")
	}

	providerRef := findField(spec, "providerConfigRef")
	if providerRef == nil {
		t.Fatal("providerConfigRef not found")
	}
	if strings.Contains(providerRef.Description, "Reference to the ProviderConfig for auth") {
		t.Error("providerConfigRef should NOT get Crossplane annotation on non-standard CRD")
	}

	deletionPolicy := findField(spec, "deletionPolicy")
	if deletionPolicy == nil {
		t.Fatal("deletionPolicy not found")
	}
	if strings.Contains(deletionPolicy.Description, "Delete or Orphan the external resource") {
		t.Error("deletionPolicy should NOT get Crossplane annotation on non-standard CRD")
	}
}

func TestAnnotateCrossplane_MixedCRDs(t *testing.T) {
	// Combine standard and non-standard nodes.
	standardNodes := buildStandardProviderNodes()
	nonStandardNodes := buildNonStandardNodes()

	combined := make([]types.TypeNode, 0, len(standardNodes)+len(nonStandardNodes))
	combined = append(combined, standardNodes...)
	combined = append(combined, nonStandardNodes...)

	result, warnings := AnnotateCrossplane(combined)

	// Should have exactly 1 warning for the non-standard CRD.
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning (from Thing), got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "Thing") {
		t.Errorf("warning should mention Thing, got %q", warnings[0])
	}

	// Standard CRD should get full annotation.
	if findNode(result, "BucketStatus") != nil {
		t.Error("BucketStatus should be removed (standard CRD)")
	}
	fp := findNode(result, "BucketForProvider")
	if fp == nil {
		t.Fatal("BucketForProvider not found")
	}
	if !strings.HasPrefix(fp.Description, "Reconcilable configuration") {
		t.Error("BucketForProvider should be annotated")
	}

	// Non-standard CRD should pass through unchanged.
	if findNode(result, "ThingStatus") == nil {
		t.Error("ThingStatus should still be present (non-standard CRD)")
	}
	thingSpec := findNode(result, "ThingSpec")
	if thingSpec == nil {
		t.Fatal("ThingSpec not found")
	}
	// No Crossplane annotations on ThingSpec fields.
	for _, f := range thingSpec.Fields {
		if strings.Contains(f.Description, "Reconcilable") || strings.Contains(f.Description, "Write-once") {
			t.Errorf("ThingSpec.%s should not have Crossplane annotations", f.Name)
		}
	}
}

func TestAnnotateCrossplane_AugmentExistingDescription(t *testing.T) {
	nodes := buildStandardProviderNodes()
	result, _ := AnnotateCrossplane(nodes)

	// BucketForProvider had Description "Parameters for the Bucket resource".
	fp := findNode(result, "BucketForProvider")
	if fp == nil {
		t.Fatal("BucketForProvider not found")
	}
	want := "Reconcilable configuration. Fields here are continuously reconciled. Parameters for the Bucket resource"
	if fp.Description != want {
		t.Errorf("BucketForProvider.Description = %q, want %q", fp.Description, want)
	}

	// BucketInitProvider had Description "InitProvider holds the same fields".
	ip := findNode(result, "BucketInitProvider")
	if ip == nil {
		t.Fatal("BucketInitProvider not found")
	}
	wantInit := "Write-once initialization. Fields here are set only at creation. InitProvider holds the same fields"
	if ip.Description != wantInit {
		t.Errorf("BucketInitProvider.Description = %q, want %q", ip.Description, wantInit)
	}

	// providerConfigRef field on BucketSpec had Description "ProviderConfigReference specifies config."
	spec := findNode(result, "BucketSpec")
	if spec == nil {
		t.Fatal("BucketSpec not found")
	}
	pcr := findField(spec, "providerConfigRef")
	if pcr == nil {
		t.Fatal("providerConfigRef not found")
	}
	wantPCR := "Reference to the ProviderConfig for auth. ProviderConfigReference specifies config."
	if pcr.Description != wantPCR {
		t.Errorf("providerConfigRef.Description = %q, want %q", pcr.Description, wantPCR)
	}

	// deletionPolicy field on BucketSpec had Description "DeletionPolicy specifies what happens."
	dp := findField(spec, "deletionPolicy")
	if dp == nil {
		t.Fatal("deletionPolicy not found")
	}
	wantDP := "Delete or Orphan the external resource on CR deletion. DeletionPolicy specifies what happens."
	if dp.Description != wantDP {
		t.Errorf("deletionPolicy.Description = %q, want %q", dp.Description, wantDP)
	}
}

func TestAugmentDescription(t *testing.T) {
	tests := []struct {
		name       string
		annotation string
		original   string
		want       string
	}{
		{
			name:       "period annotation empty original",
			annotation: "Ends with period.",
			original:   "",
			want:       "Ends with period.",
		},
		{
			name:       "no period annotation empty original",
			annotation: "No period",
			original:   "",
			want:       "No period",
		},
		{
			name:       "period annotation with original uses space",
			annotation: "Ends with period.",
			original:   "Original desc",
			want:       "Ends with period. Original desc",
		},
		{
			name:       "no period annotation with original uses dot space",
			annotation: "No period",
			original:   "Original desc",
			want:       "No period. Original desc",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := augmentDescription(tc.annotation, tc.original)
			if got != tc.want {
				t.Errorf("augmentDescription(%q, %q) = %q, want %q",
					tc.annotation, tc.original, got, tc.want)
			}
		})
	}
}

func TestAnnotateCrossplane_EmptyInput(t *testing.T) {
	// nil input
	result, warnings := AnnotateCrossplane(nil)
	if len(result) != 0 {
		t.Errorf("AnnotateCrossplane(nil): expected 0 nodes, got %d", len(result))
	}
	if len(warnings) != 0 {
		t.Errorf("AnnotateCrossplane(nil): expected 0 warnings, got %d: %v", len(warnings), warnings)
	}

	// empty slice input
	result, warnings = AnnotateCrossplane([]types.TypeNode{})
	if len(result) != 0 {
		t.Errorf("AnnotateCrossplane([]): expected 0 nodes, got %d", len(result))
	}
	if len(warnings) != 0 {
		t.Errorf("AnnotateCrossplane([]): expected 0 warnings, got %d: %v", len(warnings), warnings)
	}
}

func TestAnnotateCrossplane_DeepStatusSubtree(t *testing.T) {
	// Build a 3-level deep status subtree:
	// Root -> Spec (forProvider) + Status -> AtProvider -> Detail -> NestedDetail
	nodes := []types.TypeNode{
		makeTypeNode("Deep", "test.v1.Deep", "", []types.FieldNode{
			{Name: "spec", SchemaRef: "test.v1.DeepSpec"},
			{Name: "status", SchemaRef: "test.v1.DeepStatus"},
		}, []string{"test.v1.DeepSpec", "test.v1.DeepStatus"}),

		makeTypeNode("DeepSpec", "test.v1.DeepSpec", "", []types.FieldNode{
			{Name: "forProvider", SchemaRef: "test.v1.DeepForProvider"},
		}, []string{"test.v1.DeepForProvider"}),

		makeTypeNode("DeepForProvider", "test.v1.DeepForProvider", "", []types.FieldNode{
			{Name: "name", TypeName: "string"},
		}, nil),

		makeTypeNode("DeepStatus", "test.v1.DeepStatus", "", []types.FieldNode{
			{Name: "atProvider", SchemaRef: "test.v1.DeepAtProvider"},
		}, []string{"test.v1.DeepAtProvider"}),

		makeTypeNode("DeepAtProvider", "test.v1.DeepAtProvider", "", []types.FieldNode{
			{Name: "detail", SchemaRef: "test.v1.DeepDetail"},
		}, []string{"test.v1.DeepDetail"}),

		makeTypeNode("DeepDetail", "test.v1.DeepDetail", "", []types.FieldNode{
			{Name: "nested", SchemaRef: "test.v1.DeepNestedDetail"},
		}, []string{"test.v1.DeepNestedDetail"}),

		makeTypeNode("DeepNestedDetail", "test.v1.DeepNestedDetail", "", []types.FieldNode{
			{Name: "value", TypeName: "string"},
		}, nil),
	}

	result, warnings := AnnotateCrossplane(nodes)

	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d: %v", len(warnings), warnings)
	}

	// All 4 status-rooted TypeNodes should be removed.
	for _, name := range []string{"DeepStatus", "DeepAtProvider", "DeepDetail", "DeepNestedDetail"} {
		if findNode(result, name) != nil {
			t.Errorf("%s should be removed (status subtree)", name)
		}
	}

	// Non-status TypeNodes should be preserved.
	for _, name := range []string{"Deep", "DeepSpec", "DeepForProvider"} {
		if findNode(result, name) == nil {
			t.Errorf("%s should be preserved", name)
		}
	}

	// Root should not have status field.
	deep := findNode(result, "Deep")
	if deep == nil {
		t.Fatal("Deep not found")
	}
	if findField(deep, "status") != nil {
		t.Error("Deep should not have status field after annotation")
	}
}

func TestAnnotateCrossplane_StatusWithoutSchemaRef(t *testing.T) {
	// Root TypeNode with status field that has no SchemaRef.
	nodes := []types.TypeNode{
		makeTypeNode("Widget", "test.v1.Widget", "", []types.FieldNode{
			{Name: "spec", SchemaRef: "test.v1.WidgetSpec"},
			{Name: "status", TypeName: "dict"}, // no SchemaRef
		}, []string{"test.v1.WidgetSpec"}),

		makeTypeNode("WidgetSpec", "test.v1.WidgetSpec", "", []types.FieldNode{
			{Name: "forProvider", SchemaRef: "test.v1.WidgetForProvider"},
		}, []string{"test.v1.WidgetForProvider"}),

		makeTypeNode("WidgetForProvider", "test.v1.WidgetForProvider", "Config params", []types.FieldNode{
			{Name: "region", TypeName: "string"},
		}, nil),
	}

	result, warnings := AnnotateCrossplane(nodes)

	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d: %v", len(warnings), warnings)
	}

	// Status field should be removed from root even without SchemaRef.
	widget := findNode(result, "Widget")
	if widget == nil {
		t.Fatal("Widget not found")
	}
	if findField(widget, "status") != nil {
		t.Error("status field should be removed even without SchemaRef")
	}

	// All 3 TypeNodes should still exist (no TypeNode removal without SchemaRef).
	if len(result) != 3 {
		t.Errorf("expected 3 nodes (no TypeNode removal), got %d", len(result))
	}

	// forProvider should still be annotated.
	fp := findNode(result, "WidgetForProvider")
	if fp == nil {
		t.Fatal("WidgetForProvider not found")
	}
	if !strings.HasPrefix(fp.Description, "Reconcilable configuration") {
		t.Errorf("WidgetForProvider.Description = %q, want Reconcilable prefix", fp.Description)
	}
}
