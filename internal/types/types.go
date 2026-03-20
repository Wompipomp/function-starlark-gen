// Package types defines the central data structures shared by all pipeline stages.
//
// TypeNode and FieldNode form the data contract between the Resolver (which builds them
// from OpenAPI schemas) and the Emitter (which generates Starlark code from them).
// The Organizer assigns FilePath values, and the TypeGraph provides ordering.
package types

// SpecialType identifies Kubernetes-specific type semantics that require
// non-standard code generation (e.g., emitting field(type="") instead of a
// concrete type).
type SpecialType int

const (
	// SpecialNone indicates a regular type with no special handling.
	SpecialNone SpecialType = iota

	// SpecialIntOrString indicates a type that accepts both int and string values.
	// Detected by definition key "io.k8s.apimachinery.pkg.util.intstr.IntOrString"
	// or the x-kubernetes-int-or-string extension.
	SpecialIntOrString

	// SpecialQuantity indicates a Kubernetes resource quantity (e.g., "100m", "1Gi").
	// Detected by definition key "io.k8s.apimachinery.pkg.api.resource.Quantity".
	SpecialQuantity

	// SpecialPreserveUnknown indicates a type where unknown fields are preserved
	// as-is. Detected by the x-kubernetes-preserve-unknown-fields extension.
	// These emit field(type="dict") and stop property recursion.
	SpecialPreserveUnknown

	// SpecialEmbeddedResource indicates an embedded Kubernetes resource object.
	// Detected by the x-kubernetes-embedded-resource extension.
	// These emit field(type="dict") since the embedded resource is opaque.
	SpecialEmbeddedResource
)

// TypeNode represents a fully resolved OpenAPI schema ready for Starlark code generation.
// Each TypeNode maps to one schema() call in the generated output.
type TypeNode struct {
	// Name is the short type name used in generated Starlark code (e.g., "Deployment").
	Name string

	// DefinitionKey is the full OpenAPI definition path
	// (e.g., "io.k8s.api.apps.v1.Deployment").
	DefinitionKey string

	// Description is the OpenAPI description text, used verbatim in schema(doc=...).
	Description string

	// Fields is the ordered list of fields for this type. Order matches the OpenAPI
	// property order to ensure deterministic output.
	Fields []FieldNode

	// Dependencies lists the DefinitionKeys of types that this type references
	// via $ref. Used for topological sorting and cross-file load() generation.
	Dependencies []string

	// FilePath is the assigned output file path relative to the output directory
	// (e.g., "apps/v1.star"). Set by the Organizer stage.
	FilePath string

	// IsCircularRef is true when this type participates in a circular reference
	// chain. The self-referencing fields are emitted as field(type="dict") to
	// break the cycle.
	IsCircularRef bool

	// SpecialType indicates whether this type requires non-standard code generation
	// due to Kubernetes-specific semantics.
	SpecialType SpecialType
}

// FieldNode represents a single field within a TypeNode. Each FieldNode maps
// to one field() call in the generated Starlark schema.
type FieldNode struct {
	// Name is the Starlark-safe field name (e.g., "replicas", "api_version").
	Name string

	// TypeName is the Starlark type string: "string", "int", "float", "bool",
	// "list", "dict", or "" (empty for gradual typing / any). Empty when the
	// field references another schema via SchemaRef.
	TypeName string

	// SchemaRef is the DefinitionKey of the referenced schema type (e.g.,
	// "io.k8s.api.apps.v1.DeploymentSpec"). Empty for primitive-typed fields.
	SchemaRef string

	// Required indicates whether this field is marked as required in the OpenAPI spec.
	Required bool

	// Description is the OpenAPI description for this field, used in field(doc=...).
	Description string

	// Items is the DefinitionKey for list item schemas. Non-empty only when
	// TypeName is "list" and the list items are a referenced schema type.
	Items string

	// IsMap is true when the field uses additionalProperties in the OpenAPI spec,
	// indicating a map type. These emit field(type="dict").
	IsMap bool

	// EnumValues holds the allowed enum values from the OpenAPI spec, if any.
	// Used to generate field(enum=[...]) and document allowed values.
	EnumValues []string
}
