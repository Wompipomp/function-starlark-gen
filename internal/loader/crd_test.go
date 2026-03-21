package loader

import (
	"testing"
)

func TestLoadCRDs_Basic(t *testing.T) {
	crds, err := LoadCRDs([]string{testdataPath("crd-basic.yaml")})
	if err != nil {
		t.Fatalf("LoadCRDs failed: %v", err)
	}
	if len(crds) != 1 {
		t.Fatalf("expected 1 CRDDocument, got %d", len(crds))
	}

	doc := crds[0]
	if doc.Spec.Group != "example.com" {
		t.Errorf("expected group example.com, got %s", doc.Spec.Group)
	}
	if doc.Spec.Names.Kind != "Widget" {
		t.Errorf("expected kind Widget, got %s", doc.Spec.Names.Kind)
	}

	// Count served versions.
	servedCount := 0
	for _, v := range doc.Spec.Versions {
		if v.Served {
			servedCount++
		}
	}
	if servedCount != 1 {
		t.Errorf("expected 1 served version, got %d", servedCount)
	}

	// Check the v1 version has a non-nil schema.
	if len(doc.Spec.Versions) == 0 {
		t.Fatal("expected at least 1 version")
	}
	v := doc.Spec.Versions[0]
	if v.Name != "v1" {
		t.Errorf("expected version name v1, got %s", v.Name)
	}
	if v.Schema == nil || v.Schema.OpenAPIV3Schema == nil {
		t.Fatal("expected non-nil openAPIV3Schema")
	}
}

func TestLoadCRDs_MultiVersion(t *testing.T) {
	crds, err := LoadCRDs([]string{testdataPath("crd-multi-version.yaml")})
	if err != nil {
		t.Fatalf("LoadCRDs failed: %v", err)
	}
	if len(crds) != 1 {
		t.Fatalf("expected 1 CRDDocument, got %d", len(crds))
	}

	doc := crds[0]
	servedVersions := []string{}
	for _, v := range doc.Spec.Versions {
		if v.Served {
			servedVersions = append(servedVersions, v.Name)
		}
	}
	if len(servedVersions) != 2 {
		t.Fatalf("expected 2 served versions, got %d: %v", len(servedVersions), servedVersions)
	}
	// Check both v1 and v1alpha1 are present.
	found := map[string]bool{}
	for _, v := range servedVersions {
		found[v] = true
	}
	if !found["v1"] {
		t.Error("expected v1 to be served")
	}
	if !found["v1alpha1"] {
		t.Error("expected v1alpha1 to be served")
	}
}

func TestLoadCRDs_MultiDoc(t *testing.T) {
	crds, err := LoadCRDs([]string{testdataPath("crd-multi-doc.yaml")})
	if err != nil {
		t.Fatalf("LoadCRDs failed: %v", err)
	}
	if len(crds) != 2 {
		t.Fatalf("expected 2 CRDDocuments (non-CRD skipped), got %d", len(crds))
	}

	kinds := map[string]bool{}
	for _, doc := range crds {
		kinds[doc.Spec.Names.Kind] = true
	}
	if !kinds["Alpha"] {
		t.Error("expected Alpha CRD")
	}
	if !kinds["Beta"] {
		t.Error("expected Beta CRD")
	}
}

func TestLoadCRDs_Preserve(t *testing.T) {
	crds, err := LoadCRDs([]string{testdataPath("crd-preserve.yaml")})
	if err != nil {
		t.Fatalf("LoadCRDs failed: %v", err)
	}
	if len(crds) != 1 {
		t.Fatalf("expected 1 CRDDocument, got %d", len(crds))
	}

	schema := crds[0].Spec.Versions[0].Schema.OpenAPIV3Schema
	specProps := schema.Properties["spec"]
	if specProps == nil {
		t.Fatal("expected spec property")
	}
	metadataProp := specProps.Properties["metadata"]
	if metadataProp == nil {
		t.Fatal("expected metadata property in spec")
	}
	if metadataProp.XPreserveUnknownFields == nil || !*metadataProp.XPreserveUnknownFields {
		t.Error("expected x-kubernetes-preserve-unknown-fields=true on metadata property")
	}
}

func TestLoadCRDs_V1Beta1(t *testing.T) {
	crds, err := LoadCRDs([]string{testdataPath("crd-v1beta1.yaml")})
	if err != nil {
		t.Fatalf("LoadCRDs failed: %v", err)
	}
	if len(crds) != 1 {
		t.Fatalf("expected 1 CRDDocument, got %d", len(crds))
	}

	doc := crds[0]
	// v1beta1 detection should synthesize a version entry.
	if len(doc.Spec.Versions) != 1 {
		t.Fatalf("expected 1 synthesized version for v1beta1, got %d", len(doc.Spec.Versions))
	}
	v := doc.Spec.Versions[0]
	if v.Name != "v1" {
		t.Errorf("expected version name v1 from spec.version, got %s", v.Name)
	}
	if v.Schema == nil || v.Schema.OpenAPIV3Schema == nil {
		t.Fatal("expected non-nil schema from spec.validation.openAPIV3Schema")
	}
	if doc.Spec.Group != "legacy.example.com" {
		t.Errorf("expected group legacy.example.com, got %s", doc.Spec.Group)
	}
	if doc.Spec.Names.Kind != "OldStyle" {
		t.Errorf("expected kind OldStyle, got %s", doc.Spec.Names.Kind)
	}
}

func TestLoadCRDs_MultiFile(t *testing.T) {
	crds, err := LoadCRDs([]string{
		testdataPath("crd-basic.yaml"),
		testdataPath("crd-preserve.yaml"),
	})
	if err != nil {
		t.Fatalf("LoadCRDs failed: %v", err)
	}
	if len(crds) != 2 {
		t.Fatalf("expected 2 CRDDocuments from separate files, got %d", len(crds))
	}

	kinds := map[string]bool{}
	for _, doc := range crds {
		kinds[doc.Spec.Names.Kind] = true
	}
	if !kinds["Widget"] {
		t.Error("expected Widget CRD from crd-basic.yaml")
	}
	if !kinds["FlexType"] {
		t.Error("expected FlexType CRD from crd-preserve.yaml")
	}
}

func TestLoadCRDs_FileNotFound(t *testing.T) {
	_, err := LoadCRDs([]string{"nonexistent-crd.yaml"})
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestLoadCRDs_NonCRDSkipped(t *testing.T) {
	// crd-multi-doc.yaml contains a ConfigMap which should be silently skipped.
	crds, err := LoadCRDs([]string{testdataPath("crd-multi-doc.yaml")})
	if err != nil {
		t.Fatalf("LoadCRDs failed: %v", err)
	}
	// Should have exactly 2 CRDs (Alpha and Beta), not 3.
	if len(crds) != 2 {
		t.Errorf("expected 2 CRDs (non-CRD skipped), got %d", len(crds))
	}
	for _, doc := range crds {
		if doc.Kind != "CustomResourceDefinition" {
			t.Errorf("unexpected kind %q in results", doc.Kind)
		}
	}
}
