// Package loader handles parsing OpenAPI specifications into libopenapi document models.
//
// The Swagger 2.0 loader reads K8s swagger.json files and builds a high-level V2 model
// that preserves definition ordering and handles circular references gracefully.
package loader

import (
	"fmt"
	"log"
	"os"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	v2high "github.com/pb33f/libopenapi/datamodel/high/v2"
)

// LoadSwagger reads a Swagger 2.0 JSON file and returns a parsed V2 document model.
//
// Circular reference warnings from libopenapi are logged to stderr but are not treated
// as errors, since K8s swagger.json contains known circular reference chains
// (e.g., JSONSchemaProps).
//
// Returns an error if the file cannot be read, is not valid JSON/YAML, or is not a
// Swagger 2.0 specification.
func LoadSwagger(path string) (*libopenapi.DocumentModel[v2high.Swagger], error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading swagger file %q: %w", path, err)
	}

	doc, err := libopenapi.NewDocumentWithConfiguration(data, &datamodel.DocumentConfiguration{
		IgnorePolymorphicCircularReferences: true,
		IgnoreArrayCircularReferences:       true,
	})
	if err != nil {
		return nil, fmt.Errorf("creating document from %q: %w", path, err)
	}

	model, err := doc.BuildV2Model()
	if err != nil {
		// Log build warnings (circular ref warnings are expected from K8s swagger)
		// but only fail if the model itself is nil.
		log.Printf("warn: building V2 model from %q: %v", path, err)
	}
	if model == nil {
		return nil, fmt.Errorf("failed to build V2 model from %q", path)
	}

	return model, nil
}
