package organizer

import (
	"testing"
)

func TestCRDDefinitionKeyToFilePath_Basic(t *testing.T) {
	fp, err := CRDDefinitionKeyToFilePath("example.com.v1.Widget")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fp != "example.com/v1.star" {
		t.Errorf("got %q, want %q", fp, "example.com/v1.star")
	}
}

func TestCRDDefinitionKeyToFilePath_DottedGroup(t *testing.T) {
	fp, err := CRDDefinitionKeyToFilePath("cert-manager.io.v1.Certificate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fp != "cert-manager.io/v1.star" {
		t.Errorf("got %q, want %q", fp, "cert-manager.io/v1.star")
	}
}

func TestCRDDefinitionKeyToFilePath_AlphaVersion(t *testing.T) {
	fp, err := CRDDefinitionKeyToFilePath("example.com.v1alpha1.Foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fp != "example.com/v1alpha1.star" {
		t.Errorf("got %q, want %q", fp, "example.com/v1alpha1.star")
	}
}

func TestCRDDefinitionKeyToFilePath_DeepGroup(t *testing.T) {
	fp, err := CRDDefinitionKeyToFilePath("some.deep.group.io.v2beta1.Bar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fp != "some.deep.group.io/v2beta1.star" {
		t.Errorf("got %q, want %q", fp, "some.deep.group.io/v2beta1.star")
	}
}
