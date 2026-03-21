// Package loader handles parsing OpenAPI specifications and CRD YAML files.
//
// The CRD loader reads Kubernetes CustomResourceDefinition YAML files (v1 and v1beta1)
// and returns lightweight Go structs for consumption by the CRD resolver.
package loader

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// CRDDocument represents a parsed CRD YAML document.
type CRDDocument struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       CRDSpec  `yaml:"spec"`
}

// Metadata holds the CRD metadata fields.
type Metadata struct {
	Name string `yaml:"name"`
}

// CRDSpec holds the CRD spec fields.
type CRDSpec struct {
	Group    string       `yaml:"group"`
	Names    CRDNames     `yaml:"names"`
	Scope    string       `yaml:"scope"`
	Versions []CRDVersion `yaml:"versions"`
	// v1beta1 legacy: single version name at spec level.
	Version string `yaml:"version,omitempty"`
	// v1beta1 legacy: single schema at spec level.
	Validation *CRDValidation `yaml:"validation,omitempty"`
}

// CRDNames holds the CRD names (kind, plural, singular).
type CRDNames struct {
	Kind     string `yaml:"kind"`
	Plural   string `yaml:"plural"`
	Singular string `yaml:"singular"`
}

// CRDVersion holds per-version CRD metadata and schema.
type CRDVersion struct {
	Name    string         `yaml:"name"`
	Served  bool           `yaml:"served"`
	Storage bool           `yaml:"storage"`
	Schema  *CRDValidation `yaml:"schema,omitempty"`
}

// CRDValidation wraps the openAPIV3Schema for a CRD version.
type CRDValidation struct {
	OpenAPIV3Schema *JSONSchemaProps `yaml:"openAPIV3Schema"`
}

// JSONSchemaProps mirrors the OpenAPI v3 schema subset used in CRDs.
type JSONSchemaProps struct {
	Type                 string                       `yaml:"type,omitempty"`
	Description          string                       `yaml:"description,omitempty"`
	Properties           map[string]*JSONSchemaProps   `yaml:"properties,omitempty"`
	Required             []string                     `yaml:"required,omitempty"`
	Items                *JSONSchemaProps              `yaml:"items,omitempty"`
	Enum                 []interface{}                 `yaml:"enum,omitempty"`
	Default              interface{}                   `yaml:"default,omitempty"`
	AdditionalProperties *JSONSchemaPropsOrBool        `yaml:"additionalProperties,omitempty"`
	AllOf                []*JSONSchemaProps             `yaml:"allOf,omitempty"`
	OneOf                []*JSONSchemaProps             `yaml:"oneOf,omitempty"`
	AnyOf                []*JSONSchemaProps             `yaml:"anyOf,omitempty"`
	Format               string                       `yaml:"format,omitempty"`
	// K8s extensions
	XPreserveUnknownFields *bool `yaml:"x-kubernetes-preserve-unknown-fields,omitempty"`
	XIntOrString           *bool `yaml:"x-kubernetes-int-or-string,omitempty"`
	XEmbeddedResource      *bool `yaml:"x-kubernetes-embedded-resource,omitempty"`
}

// JSONSchemaPropsOrBool handles additionalProperties being either a boolean or
// a schema object.
type JSONSchemaPropsOrBool struct {
	Allowed bool
	Schema  *JSONSchemaProps
}

// UnmarshalYAML implements custom YAML unmarshaling for JSONSchemaPropsOrBool
// to handle both boolean and schema object forms.
func (j *JSONSchemaPropsOrBool) UnmarshalYAML(value *yaml.Node) error {
	// Try boolean first.
	if value.Kind == yaml.ScalarNode {
		var b bool
		if err := value.Decode(&b); err == nil {
			j.Allowed = b
			return nil
		}
	}

	// Try schema object.
	var schema JSONSchemaProps
	if err := value.Decode(&schema); err != nil {
		return err
	}
	j.Schema = &schema
	j.Allowed = true
	return nil
}

// LoadCRDs reads one or more CRD YAML files and returns parsed CRDDocument
// structs. Multi-document YAML files are supported: each document is decoded
// separately. Non-CRD documents (kind != CustomResourceDefinition) are silently
// skipped.
//
// For v1beta1 CRDs (apiVersion containing "v1beta1"), the schema is read from
// spec.validation.openAPIV3Schema and a single CRDVersion entry is synthesized
// from spec.version.
func LoadCRDs(paths []string) ([]CRDDocument, error) {
	var crds []CRDDocument

	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("opening CRD file %q: %w", path, err)
		}

		dec := yaml.NewDecoder(f)
		for {
			var doc CRDDocument
			if err := dec.Decode(&doc); err != nil {
				if err == io.EOF {
					break
				}
				f.Close()
				return nil, fmt.Errorf("decoding CRD YAML in %q: %w", path, err)
			}

			// Skip non-CRD documents.
			if doc.Kind != "CustomResourceDefinition" {
				continue
			}

			// Handle v1beta1 format: synthesize version entry from spec-level fields.
			if strings.Contains(doc.APIVersion, "v1beta1") {
				doc = synthesizeV1Beta1(doc)
			}

			crds = append(crds, doc)
		}
		f.Close()
	}

	return crds, nil
}

// synthesizeV1Beta1 converts a v1beta1 CRD into the v1 structure by creating
// a single CRDVersion entry from the spec-level version name and validation schema.
func synthesizeV1Beta1(doc CRDDocument) CRDDocument {
	versionName := doc.Spec.Version
	if versionName == "" {
		versionName = "v1"
	}

	var validation *CRDValidation
	if doc.Spec.Validation != nil {
		validation = doc.Spec.Validation
	}

	doc.Spec.Versions = []CRDVersion{
		{
			Name:    versionName,
			Served:  true,
			Storage: true,
			Schema:  validation,
		},
	}
	return doc
}
