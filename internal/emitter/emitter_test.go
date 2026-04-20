package emitter

import (
	"bytes"
	"sort"
	"strings"
	"testing"

	"github.com/wompipomp/starlark-gen/internal/types"
)

// helper builds a map from DefinitionKey to TypeNode for test use.
func buildAllNodes(nodes ...*types.TypeNode) map[string]*types.TypeNode {
	m := make(map[string]*types.TypeNode)
	for _, n := range nodes {
		m[n.DefinitionKey] = n
	}
	return m
}

// helper builds the file-type set from a list of TypeNodes.
func buildFileTypes(nodes []*types.TypeNode) map[string]bool {
	m := make(map[string]bool)
	for _, n := range nodes {
		m[n.DefinitionKey] = true
	}
	return m
}

func TestEmitFile_SingleTypeWithPrimitiveFields(t *testing.T) {
	node := &types.TypeNode{
		Name:          "DeploymentSpec",
		DefinitionKey: "io.k8s.api.apps.v1.DeploymentSpec",
		Description:   "DeploymentSpec is the specification of the desired behavior of the Deployment.",
		FilePath:      "apps/v1.star",
		Fields: []types.FieldNode{
			{Name: "replicas", TypeName: "int", Description: "Number of desired pods"},
			{Name: "paused", TypeName: "bool", Description: "Indicates that the deployment is paused"},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("apps/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// Should have schema() call
	if !strings.Contains(content, `DeploymentSpec = schema(`) {
		t.Error("expected DeploymentSpec = schema( call")
	}
	// Should have the name as first positional arg
	if !strings.Contains(content, `    "DeploymentSpec",`) {
		t.Error("expected name as first positional arg")
	}
	// Should have doc= with description
	if !strings.Contains(content, `    doc="DeploymentSpec is the specification of the desired behavior of the Deployment.",`) {
		t.Error("expected doc= with description")
	}
	// Should have field() calls with type= quoted for primitives
	if !strings.Contains(content, `replicas=field(type="int"`) {
		t.Errorf("expected replicas field, got:\n%s", content)
	}
	if !strings.Contains(content, `paused=field(type="bool"`) {
		t.Errorf("expected paused field, got:\n%s", content)
	}
	// Should have closing paren
	if !strings.Contains(content, ")\n") {
		t.Error("expected closing paren for schema()")
	}
}

func TestEmitFile_CrossFileDependencyProducesLoad(t *testing.T) {
	meta := &types.TypeNode{
		Name:          "ObjectMeta",
		DefinitionKey: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
		FilePath:      "meta/v1.star",
	}
	deploy := &types.TypeNode{
		Name:          "Deployment",
		DefinitionKey: "io.k8s.api.apps.v1.Deployment",
		Description:   "Deployment enables declarative updates for Pods and ReplicaSets.",
		FilePath:      "apps/v1.star",
		Dependencies:  []string{"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta"},
		Fields: []types.FieldNode{
			{Name: "metadata", SchemaRef: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta", Description: "Standard object metadata"},
		},
	}

	nodes := []*types.TypeNode{deploy}
	allNodes := buildAllNodes(meta, deploy)

	out, err := EmitFile("apps/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// Should have a load() statement for ObjectMeta from meta/v1.star
	if !strings.Contains(content, `load("schemas-k8s:v1.31/meta/v1.star", "ObjectMeta")`) {
		t.Errorf("expected load() for ObjectMeta, got:\n%s", content)
	}
}

func TestEmitFile_LoadStatementsUseOCIShortForm(t *testing.T) {
	meta := &types.TypeNode{
		Name:          "ObjectMeta",
		DefinitionKey: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
		FilePath:      "meta/v1.star",
	}
	deploy := &types.TypeNode{
		Name:          "Deployment",
		DefinitionKey: "io.k8s.api.apps.v1.Deployment",
		FilePath:      "apps/v1.star",
		Dependencies:  []string{"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta"},
		Fields: []types.FieldNode{
			{Name: "metadata", SchemaRef: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta", Description: "Standard object metadata"},
		},
	}

	nodes := []*types.TypeNode{deploy}
	allNodes := buildAllNodes(meta, deploy)

	out, err := EmitFile("apps/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// OCI short-form: load("schemas-k8s:v1.31/meta/v1.star", "ObjectMeta")
	expected := `load("schemas-k8s:v1.31/meta/v1.star", "ObjectMeta")`
	if !strings.Contains(content, expected) {
		t.Errorf("expected OCI short-form load path:\n  expected: %s\n  got:\n%s", expected, content)
	}
}

func TestEmitFile_MultipleSymbolsGroupedInOneLoad(t *testing.T) {
	metaNode1 := &types.TypeNode{
		Name:          "ObjectMeta",
		DefinitionKey: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
		FilePath:      "meta/v1.star",
	}
	metaNode2 := &types.TypeNode{
		Name:          "LabelSelector",
		DefinitionKey: "io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector",
		FilePath:      "meta/v1.star",
	}
	deploy := &types.TypeNode{
		Name:          "DeploymentSpec",
		DefinitionKey: "io.k8s.api.apps.v1.DeploymentSpec",
		FilePath:      "apps/v1.star",
		Dependencies: []string{
			"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
			"io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector",
		},
		Fields: []types.FieldNode{
			{Name: "metadata", SchemaRef: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta", Description: "Standard object metadata"},
			{Name: "selector", SchemaRef: "io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector", Required: true, Description: "Label query over pods"},
		},
	}

	nodes := []*types.TypeNode{deploy}
	allNodes := buildAllNodes(metaNode1, metaNode2, deploy)

	out, err := EmitFile("apps/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// Both symbols should be in one load() statement, sorted alphabetically
	expected := `load("schemas-k8s:v1.31/meta/v1.star", "LabelSelector", "ObjectMeta")`
	if !strings.Contains(content, expected) {
		t.Errorf("expected grouped load:\n  expected: %s\n  got:\n%s", expected, content)
	}
}

func TestEmitFile_LoadStatementsSortedAlphabetically(t *testing.T) {
	meta := &types.TypeNode{
		Name:          "ObjectMeta",
		DefinitionKey: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
		FilePath:      "meta/v1.star",
	}
	podSpec := &types.TypeNode{
		Name:          "PodTemplateSpec",
		DefinitionKey: "io.k8s.api.core.v1.PodTemplateSpec",
		FilePath:      "core/v1.star",
	}
	deploy := &types.TypeNode{
		Name:          "DeploymentSpec",
		DefinitionKey: "io.k8s.api.apps.v1.DeploymentSpec",
		FilePath:      "apps/v1.star",
		Dependencies: []string{
			"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
			"io.k8s.api.core.v1.PodTemplateSpec",
		},
		Fields: []types.FieldNode{
			{Name: "metadata", SchemaRef: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta", Description: "Standard object metadata"},
			{Name: "template", SchemaRef: "io.k8s.api.core.v1.PodTemplateSpec", Required: true, Description: "Template describes the pods"},
		},
	}

	nodes := []*types.TypeNode{deploy}
	allNodes := buildAllNodes(meta, podSpec, deploy)

	out, err := EmitFile("apps/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// core/v1.star should come before meta/v1.star alphabetically
	coreIdx := strings.Index(content, `load("schemas-k8s:v1.31/core/v1.star"`)
	metaIdx := strings.Index(content, `load("schemas-k8s:v1.31/meta/v1.star"`)

	if coreIdx < 0 || metaIdx < 0 {
		t.Fatalf("expected both load() statements, got:\n%s", content)
	}
	if coreIdx >= metaIdx {
		t.Error("expected core/v1.star load() before meta/v1.star load() (alphabetical)")
	}
}

func TestEmitFile_FieldDocIncludesTypePrefix(t *testing.T) {
	node := &types.TypeNode{
		Name:          "DeploymentSpec",
		DefinitionKey: "io.k8s.api.apps.v1.DeploymentSpec",
		FilePath:      "apps/v1.star",
		Fields: []types.FieldNode{
			{Name: "replicas", TypeName: "int", Description: "Number of desired pods"},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("apps/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// doc should have type prefix: "int - Number of desired pods"
	if !strings.Contains(content, `doc="int - Number of desired pods"`) {
		t.Errorf("expected type prefix in doc, got:\n%s", content)
	}
}

func TestEmitFile_RequiredFieldIncludesMarkerInDoc(t *testing.T) {
	meta := &types.TypeNode{
		Name:          "LabelSelector",
		DefinitionKey: "io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector",
		FilePath:      "meta/v1.star",
	}
	node := &types.TypeNode{
		Name:          "DeploymentSpec",
		DefinitionKey: "io.k8s.api.apps.v1.DeploymentSpec",
		FilePath:      "apps/v1.star",
		Dependencies:  []string{"io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector"},
		Fields: []types.FieldNode{
			{Name: "selector", SchemaRef: "io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector", Required: true, Description: "Label query over pods"},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(meta, node)

	out, err := EmitFile("apps/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// doc should include (required) marker
	if !strings.Contains(content, `doc="LabelSelector - Label query over pods (required)"`) {
		t.Errorf("expected required marker in doc, got:\n%s", content)
	}
}

func TestEmitFile_ListFieldOmitsEnumKwarg(t *testing.T) {
	node := &types.TypeNode{
		Name:          "EntrySpec",
		DefinitionKey: "com.eon.atlantis.leanix.v1alpha1.EntrySpec",
		FilePath:      "leanix.atlantis.eon.com/v1alpha1.star",
		Fields: []types.FieldNode{
			{
				Name:        "managementPolicies",
				TypeName:    "list",
				Description: "Management policies",
				EnumValues:  []string{"Observe", "Create", "Update", "Delete", "LateInitialize", "*"},
			},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("leanix.atlantis.eon.com/v1alpha1.star", nodes, allNodes, "schemas-provider-leanix:v1alpha1")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	if strings.Contains(content, "enum=[") {
		t.Errorf("list field must not emit enum kwarg (enum applies to whole value in function-starlark), got:\n%s", content)
	}

	// Docstring should still advertise the allowed item values.
	if !strings.Contains(content, "One of: Observe, Create, Update, Delete, LateInitialize, *") {
		t.Errorf("expected item enum values in docstring, got:\n%s", content)
	}

	// Field should still be emitted as a list.
	if !strings.Contains(content, `managementPolicies=field(type="list"`) {
		t.Errorf("expected managementPolicies field with type=\"list\", got:\n%s", content)
	}
}

func TestEmitFile_EnumValuesListedInDoc(t *testing.T) {
	node := &types.TypeNode{
		Name:          "PodSpec",
		DefinitionKey: "io.k8s.api.core.v1.PodSpec",
		FilePath:      "core/v1.star",
		Fields: []types.FieldNode{
			{Name: "restart_policy", TypeName: "string", Description: "Restart policy", EnumValues: []string{"Always", "OnFailure", "Never"}},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("core/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	if !strings.Contains(content, `One of: Always, OnFailure, Never`) {
		t.Errorf("expected enum values in doc, got:\n%s", content)
	}
}

func TestEmitFile_SchemaDocUsesOpenAPIDescriptionVerbatim(t *testing.T) {
	desc := "Deployment enables declarative updates for Pods and ReplicaSets."
	node := &types.TypeNode{
		Name:          "Deployment",
		DefinitionKey: "io.k8s.api.apps.v1.Deployment",
		Description:   desc,
		FilePath:      "apps/v1.star",
		Fields: []types.FieldNode{
			{Name: "replicas", TypeName: "int", Description: "Number of desired pods"},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("apps/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// doc= should contain the verbatim OpenAPI description
	if !strings.Contains(content, `doc="Deployment enables declarative updates for Pods and ReplicaSets.",`) {
		t.Errorf("expected verbatim description in doc=, got:\n%s", content)
	}
}

func TestEmitFile_IntraFileRefUsesBareNameNotLoad(t *testing.T) {
	spec := &types.TypeNode{
		Name:          "DeploymentSpec",
		DefinitionKey: "io.k8s.api.apps.v1.DeploymentSpec",
		FilePath:      "apps/v1.star",
		Fields: []types.FieldNode{
			{Name: "replicas", TypeName: "int", Description: "Number of desired pods"},
		},
	}
	deploy := &types.TypeNode{
		Name:          "Deployment",
		DefinitionKey: "io.k8s.api.apps.v1.Deployment",
		FilePath:      "apps/v1.star",
		Dependencies:  []string{"io.k8s.api.apps.v1.DeploymentSpec"},
		Fields: []types.FieldNode{
			{Name: "spec", SchemaRef: "io.k8s.api.apps.v1.DeploymentSpec", Description: "Specification of the desired behavior"},
		},
	}

	nodes := []*types.TypeNode{spec, deploy}
	allNodes := buildAllNodes(spec, deploy)

	out, err := EmitFile("apps/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// Intra-file ref: bare type name, no quoted string
	if !strings.Contains(content, `type=DeploymentSpec`) {
		t.Errorf("expected bare type reference, got:\n%s", content)
	}
	// Should NOT have a load() for an intra-file type
	if strings.Contains(content, `load(`) {
		t.Errorf("should not have load() for intra-file reference, got:\n%s", content)
	}
}

func TestEmitFile_ListFieldWithSchemaItems(t *testing.T) {
	condition := &types.TypeNode{
		Name:          "DeploymentCondition",
		DefinitionKey: "io.k8s.api.apps.v1.DeploymentCondition",
		FilePath:      "apps/v1.star",
		Fields: []types.FieldNode{
			{Name: "type", TypeName: "string", Description: "Type of condition"},
		},
	}
	status := &types.TypeNode{
		Name:          "DeploymentStatus",
		DefinitionKey: "io.k8s.api.apps.v1.DeploymentStatus",
		FilePath:      "apps/v1.star",
		Dependencies:  []string{"io.k8s.api.apps.v1.DeploymentCondition"},
		Fields: []types.FieldNode{
			{Name: "conditions", TypeName: "list", Items: "io.k8s.api.apps.v1.DeploymentCondition", Description: "Represents the latest observations"},
		},
	}

	nodes := []*types.TypeNode{condition, status}
	allNodes := buildAllNodes(condition, status)

	out, err := EmitFile("apps/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// Should have type="list" and items=DeploymentCondition
	if !strings.Contains(content, `type="list"`) {
		t.Errorf("expected type=\"list\", got:\n%s", content)
	}
	if !strings.Contains(content, `items=DeploymentCondition`) {
		t.Errorf("expected items=DeploymentCondition, got:\n%s", content)
	}
}

func TestEmitFile_DictFieldForMapTypes(t *testing.T) {
	node := &types.TypeNode{
		Name:          "ObjectMeta",
		DefinitionKey: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
		FilePath:      "meta/v1.star",
		Fields: []types.FieldNode{
			{Name: "labels", TypeName: "dict", IsMap: true, Description: "Map of string keys and values"},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("meta/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	if !strings.Contains(content, `type="dict"`) {
		t.Errorf("expected type=\"dict\" for map field, got:\n%s", content)
	}
}

func TestEmitFile_GradualTypingEmitsEmptyStringType(t *testing.T) {
	node := &types.TypeNode{
		Name:          "IntOrString",
		DefinitionKey: "io.k8s.apimachinery.pkg.util.intstr.IntOrString",
		Description:   "IntOrString is a type that can hold an int or a string.",
		FilePath:      "intstr/v1.star",
		SpecialType:   types.SpecialIntOrString,
		Fields: []types.FieldNode{
			{Name: "value", TypeName: "", Description: "IntOrString - accepts int or string values"},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("intstr/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	if !strings.Contains(content, `type=""`) {
		t.Errorf("expected type=\"\" for gradual typing, got:\n%s", content)
	}
}

func TestEmitFile_CircularRefFieldEmitsDictWithDoc(t *testing.T) {
	node := &types.TypeNode{
		Name:          "JSONSchemaProps",
		DefinitionKey: "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.JSONSchemaProps",
		Description:   "JSONSchemaProps is a JSON-Schema following Spec Draft 4.",
		FilePath:      "apiextensions/v1.star",
		IsCircularRef: true,
		Dependencies:  []string{"io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.JSONSchemaProps"},
		Fields: []types.FieldNode{
			{Name: "properties", TypeName: "dict", Description: "Recursive structure - see JSONSchemaProps"},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("apiextensions/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	if !strings.Contains(content, `type="dict"`) {
		t.Errorf("expected type=\"dict\" for circular ref field, got:\n%s", content)
	}
}

func TestEmitFile_ConsistentLineEndings(t *testing.T) {
	node := &types.TypeNode{
		Name:          "Simple",
		DefinitionKey: "io.k8s.api.core.v1.Simple",
		Description:   "A simple type.",
		FilePath:      "core/v1.star",
		Fields: []types.FieldNode{
			{Name: "name", TypeName: "string", Description: "The name"},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("core/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	// Check for \r\n (should not be present)
	if bytes.Contains(out, []byte("\r\n")) {
		t.Error("output contains \\r\\n line endings, expected \\n only")
	}

	// Check for trailing whitespace on any line
	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		if line != strings.TrimRight(line, " \t") {
			t.Errorf("line %d has trailing whitespace: %q", i+1, line)
		}
	}
}

func TestEmitFile_DeterministicOutput(t *testing.T) {
	meta := &types.TypeNode{
		Name:          "ObjectMeta",
		DefinitionKey: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
		FilePath:      "meta/v1.star",
	}
	labelSel := &types.TypeNode{
		Name:          "LabelSelector",
		DefinitionKey: "io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector",
		FilePath:      "meta/v1.star",
	}
	deploy := &types.TypeNode{
		Name:          "DeploymentSpec",
		DefinitionKey: "io.k8s.api.apps.v1.DeploymentSpec",
		Description:   "DeploymentSpec is the specification of the desired behavior of the Deployment.",
		FilePath:      "apps/v1.star",
		Dependencies: []string{
			"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
			"io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector",
		},
		Fields: []types.FieldNode{
			{Name: "replicas", TypeName: "int", Description: "Number of desired pods"},
			{Name: "selector", SchemaRef: "io.k8s.apimachinery.pkg.apis.meta.v1.LabelSelector", Required: true, Description: "Label query over pods"},
			{Name: "template", SchemaRef: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta", Description: "Standard object metadata"},
		},
	}

	nodes := []*types.TypeNode{deploy}
	allNodes := buildAllNodes(meta, labelSel, deploy)

	out1, err := EmitFile("apps/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile (first call) error: %v", err)
	}

	out2, err := EmitFile("apps/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile (second call) error: %v", err)
	}

	if !bytes.Equal(out1, out2) {
		t.Errorf("EmitFile is not deterministic:\nfirst:\n%s\nsecond:\n%s", string(out1), string(out2))
	}
}

func TestEmit_FullPipeline(t *testing.T) {
	// Test the Emit function that processes an entire FileMap.
	meta := &types.TypeNode{
		Name:          "ObjectMeta",
		DefinitionKey: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
		Description:   "ObjectMeta is metadata that all persisted resources must have.",
		FilePath:      "meta/v1.star",
		Fields: []types.FieldNode{
			{Name: "name", TypeName: "string", Description: "Name must be unique within a namespace"},
			{Name: "labels", TypeName: "dict", IsMap: true, Description: "Map of string keys and values"},
		},
	}
	deploy := &types.TypeNode{
		Name:          "Deployment",
		DefinitionKey: "io.k8s.api.apps.v1.Deployment",
		Description:   "Deployment enables declarative updates.",
		FilePath:      "apps/v1.star",
		Dependencies:  []string{"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta"},
		Fields: []types.FieldNode{
			{Name: "metadata", SchemaRef: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta", Description: "Standard object metadata"},
			{Name: "replicas", TypeName: "int", Description: "Number of desired pods"},
		},
	}

	fileMap := map[string][]*types.TypeNode{
		"meta/v1.star": {meta},
		"apps/v1.star": {deploy},
	}
	fileOrder := []string{"meta/v1.star", "apps/v1.star"}

	result, err := Emit(fileMap, fileOrder, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("Emit error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 files, got %d", len(result))
	}

	// Check meta/v1.star has no load statements
	metaContent := string(result["meta/v1.star"])
	if strings.Contains(metaContent, "load(") {
		t.Errorf("meta/v1.star should have no load() statements, got:\n%s", metaContent)
	}

	// Check apps/v1.star has a load statement for ObjectMeta
	appsContent := string(result["apps/v1.star"])
	if !strings.Contains(appsContent, `load("schemas-k8s:v1.31/meta/v1.star", "ObjectMeta")`) {
		t.Errorf("apps/v1.star should load ObjectMeta, got:\n%s", appsContent)
	}
}

func TestEmitFile_FieldDocForSchemaRef(t *testing.T) {
	// Verify schema ref fields show type name in doc.
	meta := &types.TypeNode{
		Name:          "ObjectMeta",
		DefinitionKey: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
		FilePath:      "meta/v1.star",
	}
	node := &types.TypeNode{
		Name:          "Deployment",
		DefinitionKey: "io.k8s.api.apps.v1.Deployment",
		FilePath:      "apps/v1.star",
		Dependencies:  []string{"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta"},
		Fields: []types.FieldNode{
			{Name: "metadata", SchemaRef: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta", Description: "Standard object metadata"},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(meta, node)

	out, err := EmitFile("apps/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)
	// Schema ref doc includes type name
	if !strings.Contains(content, `doc="ObjectMeta - Standard object metadata"`) {
		t.Errorf("expected schema ref type in doc, got:\n%s", content)
	}
}

func TestEmitFile_ListFieldWithPrimitiveItems(t *testing.T) {
	// List of primitives: no items= parameter, just type="list"
	node := &types.TypeNode{
		Name:          "PodSpec",
		DefinitionKey: "io.k8s.api.core.v1.PodSpec",
		FilePath:      "core/v1.star",
		Fields: []types.FieldNode{
			{Name: "dns_servers", TypeName: "list", Description: "List of DNS servers"},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("core/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// Should have type="list" without items=
	if !strings.Contains(content, `type="list"`) {
		t.Errorf("expected type=\"list\", got:\n%s", content)
	}
	if strings.Contains(content, "items=") {
		t.Errorf("expected no items= for primitive list, got:\n%s", content)
	}
}

func TestEmitFile_EmptyDescription(t *testing.T) {
	// When description is empty, no doc= line in schema()
	node := &types.TypeNode{
		Name:          "Simple",
		DefinitionKey: "io.k8s.api.core.v1.Simple",
		FilePath:      "core/v1.star",
		Fields: []types.FieldNode{
			{Name: "name", TypeName: "string", Description: "The name"},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("core/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// No doc= line since description is empty
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// The schema doc= should not appear (distinct from field doc=)
		if strings.HasPrefix(trimmed, "doc=") {
			// This is fine if it's inside a field() call -- we need to check schema-level doc
			// Just make sure there is no schema-level doc= (second line after name)
		}
	}

	// Better check: schema() should go directly from name to fields
	if strings.Contains(content, "Simple = schema(\n    \"Simple\",\n    doc=") {
		t.Errorf("expected no doc= for empty description, got:\n%s", content)
	}
}

func TestEmitFile_EnumFieldFormat(t *testing.T) {
	node := &types.TypeNode{
		Name:          "PodSpec",
		DefinitionKey: "io.k8s.api.core.v1.PodSpec",
		FilePath:      "core/v1.star",
		Fields: []types.FieldNode{
			{Name: "restart_policy", TypeName: "string", Description: "Restart policy for all containers", EnumValues: []string{"Always", "OnFailure", "Never"}},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("core/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// Full expected doc format
	expectedDoc := `doc="string - Restart policy for all containers. One of: Always, OnFailure, Never"`
	if !strings.Contains(content, expectedDoc) {
		t.Errorf("expected enum doc format:\n  expected: %s\n  got:\n%s", expectedDoc, content)
	}
}

// Test load() deduplication from Dependencies list.
func TestEmitFile_LoadDeduplication(t *testing.T) {
	meta := &types.TypeNode{
		Name:          "ObjectMeta",
		DefinitionKey: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
		FilePath:      "meta/v1.star",
	}
	// Two types in the same file both reference ObjectMeta
	spec := &types.TypeNode{
		Name:          "DeploymentSpec",
		DefinitionKey: "io.k8s.api.apps.v1.DeploymentSpec",
		FilePath:      "apps/v1.star",
		Dependencies:  []string{"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta"},
		Fields: []types.FieldNode{
			{Name: "metadata", SchemaRef: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta", Description: "Standard object metadata"},
		},
	}
	deploy := &types.TypeNode{
		Name:          "Deployment",
		DefinitionKey: "io.k8s.api.apps.v1.Deployment",
		FilePath:      "apps/v1.star",
		Dependencies:  []string{"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta", "io.k8s.api.apps.v1.DeploymentSpec"},
		Fields: []types.FieldNode{
			{Name: "metadata", SchemaRef: "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta", Description: "Standard object metadata"},
			{Name: "spec", SchemaRef: "io.k8s.api.apps.v1.DeploymentSpec", Description: "Spec"},
		},
	}

	nodes := []*types.TypeNode{spec, deploy}
	allNodes := buildAllNodes(meta, spec, deploy)

	out, err := EmitFile("apps/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// ObjectMeta should appear in load() exactly once
	count := strings.Count(content, `"ObjectMeta"`)
	if count != 1 {
		t.Errorf("expected ObjectMeta to appear in load() once, got %d times:\n%s", count, content)
	}
}

// Test that file-level items dependencies also generate load() statements.
func TestEmitFile_ListItemsCrossFileLoad(t *testing.T) {
	condition := &types.TypeNode{
		Name:          "PodCondition",
		DefinitionKey: "io.k8s.api.core.v1.PodCondition",
		FilePath:      "core/v1.star",
		Fields: []types.FieldNode{
			{Name: "type", TypeName: "string", Description: "Type of condition"},
		},
	}

	status := &types.TypeNode{
		Name:          "DeploymentStatus",
		DefinitionKey: "io.k8s.api.apps.v1.DeploymentStatus",
		FilePath:      "apps/v1.star",
		Dependencies:  []string{"io.k8s.api.core.v1.PodCondition"},
		Fields: []types.FieldNode{
			{Name: "conditions", TypeName: "list", Items: "io.k8s.api.core.v1.PodCondition", Description: "List of conditions"},
		},
	}

	nodes := []*types.TypeNode{status}
	allNodes := buildAllNodes(condition, status)

	out, err := EmitFile("apps/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// Should have load() for PodCondition from core/v1.star
	if !strings.Contains(content, `load("schemas-k8s:v1.31/core/v1.star", "PodCondition")`) {
		t.Errorf("expected load() for cross-file list items, got:\n%s", content)
	}
	// Should have items=PodCondition
	if !strings.Contains(content, `items=PodCondition`) {
		t.Errorf("expected items=PodCondition, got:\n%s", content)
	}
}

// Ensure Emit builds allNodes from fileMap correctly
func TestEmit_BuildsAllNodes(t *testing.T) {
	spec := &types.TypeNode{
		Name:          "DeploymentSpec",
		DefinitionKey: "io.k8s.api.apps.v1.DeploymentSpec",
		FilePath:      "apps/v1.star",
		Fields: []types.FieldNode{
			{Name: "replicas", TypeName: "int", Description: "Number of desired pods"},
		},
	}
	deploy := &types.TypeNode{
		Name:          "Deployment",
		DefinitionKey: "io.k8s.api.apps.v1.Deployment",
		Description:   "A Deployment.",
		FilePath:      "apps/v1.star",
		Dependencies:  []string{"io.k8s.api.apps.v1.DeploymentSpec"},
		Fields: []types.FieldNode{
			{Name: "spec", SchemaRef: "io.k8s.api.apps.v1.DeploymentSpec", Description: "Desired behavior"},
		},
	}

	fileMap := map[string][]*types.TypeNode{
		"apps/v1.star": {spec, deploy},
	}
	fileOrder := []string{"apps/v1.star"}

	result, err := Emit(fileMap, fileOrder, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("Emit error: %v", err)
	}

	content := string(result["apps/v1.star"])

	// Intra-file ref should use bare name
	if !strings.Contains(content, `type=DeploymentSpec`) {
		t.Errorf("expected bare type ref for intra-file dependency, got:\n%s", content)
	}
	// Should NOT have load() for intra-file types
	if strings.Contains(content, "load(") {
		t.Errorf("should not load() for intra-file ref, got:\n%s", content)
	}
}

// Test that field names produce correct Starlark field kwarg names.
func TestEmitFile_FieldOrderPreserved(t *testing.T) {
	node := &types.TypeNode{
		Name:          "DeploymentSpec",
		DefinitionKey: "io.k8s.api.apps.v1.DeploymentSpec",
		FilePath:      "apps/v1.star",
		Fields: []types.FieldNode{
			{Name: "replicas", TypeName: "int", Description: "Number of desired pods"},
			{Name: "paused", TypeName: "bool", Description: "Paused"},
			{Name: "min_ready_seconds", TypeName: "int", Description: "Min ready seconds"},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("apps/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// Fields should appear in the given order
	replicasIdx := strings.Index(content, "replicas=field(")
	pausedIdx := strings.Index(content, "paused=field(")
	minReadyIdx := strings.Index(content, "min_ready_seconds=field(")

	if replicasIdx < 0 || pausedIdx < 0 || minReadyIdx < 0 {
		t.Fatalf("missing field declarations, got:\n%s", content)
	}

	if !(replicasIdx < pausedIdx && pausedIdx < minReadyIdx) {
		t.Error("fields not in expected order")
	}
}

// --- Enum constant and default value tests (Phase 2, Plan 02) ---

func TestEmitEnumConstants_StringValues(t *testing.T) {
	node := &types.TypeNode{
		Name:          "Widget",
		DefinitionKey: "example.Widget",
		FilePath:      "example/v1.star",
		Fields: []types.FieldNode{
			{Name: "algorithm", TypeName: "string", Description: "Algorithm", EnumValues: []string{"RSA", "ECDSA", "Ed25519"}},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("example/v1.star", nodes, allNodes, "test:v1")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// Should emit SCREAMING_SNAKE_CASE constants above schema
	if !strings.Contains(content, `WIDGET_ALGORITHM_RSA = "RSA"`) {
		t.Errorf("expected WIDGET_ALGORITHM_RSA constant, got:\n%s", content)
	}
	if !strings.Contains(content, `WIDGET_ALGORITHM_ECDSA = "ECDSA"`) {
		t.Errorf("expected WIDGET_ALGORITHM_ECDSA constant, got:\n%s", content)
	}
	if !strings.Contains(content, `WIDGET_ALGORITHM_ED25519 = "Ed25519"`) {
		t.Errorf("expected WIDGET_ALGORITHM_ED25519 constant, got:\n%s", content)
	}
}

func TestEmitEnumConstants_MultipleFields(t *testing.T) {
	node := &types.TypeNode{
		Name:          "Config",
		DefinitionKey: "example.Config",
		FilePath:      "example/v1.star",
		Fields: []types.FieldNode{
			{Name: "mode", TypeName: "string", Description: "Mode", EnumValues: []string{"fast", "slow"}},
			{Name: "level", TypeName: "string", Description: "Level", EnumValues: []string{"high", "low"}},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("example/v1.star", nodes, allNodes, "test:v1")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// Both fields should have constants
	if !strings.Contains(content, `CONFIG_MODE_FAST = "fast"`) {
		t.Errorf("expected CONFIG_MODE_FAST constant, got:\n%s", content)
	}
	if !strings.Contains(content, `CONFIG_MODE_SLOW = "slow"`) {
		t.Errorf("expected CONFIG_MODE_SLOW constant, got:\n%s", content)
	}
	if !strings.Contains(content, `CONFIG_LEVEL_HIGH = "high"`) {
		t.Errorf("expected CONFIG_LEVEL_HIGH constant, got:\n%s", content)
	}
	if !strings.Contains(content, `CONFIG_LEVEL_LOW = "low"`) {
		t.Errorf("expected CONFIG_LEVEL_LOW constant, got:\n%s", content)
	}
}

func TestEmitEnumConstants_SpecialChars(t *testing.T) {
	node := &types.TypeNode{
		Name:          "Type",
		DefinitionKey: "example.Type",
		FilePath:      "example/v1.star",
		Fields: []types.FieldNode{
			{Name: "field", TypeName: "string", Description: "Field", EnumValues: []string{"digital signature"}},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("example/v1.star", nodes, allNodes, "test:v1")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// Non-alphanumeric replaced with _, collapsed
	if !strings.Contains(content, `TYPE_FIELD_DIGITAL_SIGNATURE = "digital signature"`) {
		t.Errorf("expected TYPE_FIELD_DIGITAL_SIGNATURE constant, got:\n%s", content)
	}
}

func TestEmitEnumConstants_BoolValues(t *testing.T) {
	node := &types.TypeNode{
		Name:          "Config",
		DefinitionKey: "example.Config",
		FilePath:      "example/v1.star",
		Fields: []types.FieldNode{
			{Name: "enabled", TypeName: "string", Description: "Enabled", EnumValues: []string{"true", "false"}},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("example/v1.star", nodes, allNodes, "test:v1")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	if !strings.Contains(content, `CONFIG_ENABLED_TRUE = "true"`) {
		t.Errorf("expected CONFIG_ENABLED_TRUE constant, got:\n%s", content)
	}
	if !strings.Contains(content, `CONFIG_ENABLED_FALSE = "false"`) {
		t.Errorf("expected CONFIG_ENABLED_FALSE constant, got:\n%s", content)
	}
}

func TestEmitFieldEnum(t *testing.T) {
	node := &types.TypeNode{
		Name:          "Widget",
		DefinitionKey: "example.Widget",
		FilePath:      "example/v1.star",
		Fields: []types.FieldNode{
			{Name: "algorithm", TypeName: "string", Description: "Algorithm", EnumValues: []string{"RSA", "ECDSA", "Ed25519"}},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("example/v1.star", nodes, allNodes, "test:v1")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// field() call should include enum=[...]
	if !strings.Contains(content, `enum=["RSA", "ECDSA", "Ed25519"]`) {
		t.Errorf("expected enum= argument in field(), got:\n%s", content)
	}
}

func TestEmitFieldDefault_String(t *testing.T) {
	node := &types.TypeNode{
		Name:          "Config",
		DefinitionKey: "example.Config",
		FilePath:      "example/v1.star",
		Fields: []types.FieldNode{
			{Name: "priority", TypeName: "string", Description: "Priority", Default: "medium"},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("example/v1.star", nodes, allNodes, "test:v1")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	if !strings.Contains(content, `default="medium"`) {
		t.Errorf("expected default=\"medium\" in field(), got:\n%s", content)
	}
}

func TestEmitFieldDefault_Int(t *testing.T) {
	node := &types.TypeNode{
		Name:          "Config",
		DefinitionKey: "example.Config",
		FilePath:      "example/v1.star",
		Fields: []types.FieldNode{
			{Name: "replicas", TypeName: "int", Description: "Number of replicas", Default: 3},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("example/v1.star", nodes, allNodes, "test:v1")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	if !strings.Contains(content, `default=3`) {
		t.Errorf("expected default=3 in field(), got:\n%s", content)
	}
}

func TestEmitFieldDefault_Float(t *testing.T) {
	node := &types.TypeNode{
		Name:          "Config",
		DefinitionKey: "example.Config",
		FilePath:      "example/v1.star",
		Fields: []types.FieldNode{
			{Name: "ratio", TypeName: "float", Description: "Ratio", Default: 1.5},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("example/v1.star", nodes, allNodes, "test:v1")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	if !strings.Contains(content, `default=1.5`) {
		t.Errorf("expected default=1.5 in field(), got:\n%s", content)
	}
}

func TestEmitFieldDefault_Bool(t *testing.T) {
	node := &types.TypeNode{
		Name:          "Config",
		DefinitionKey: "example.Config",
		FilePath:      "example/v1.star",
		Fields: []types.FieldNode{
			{Name: "enabled", TypeName: "bool", Description: "Enabled", Default: true},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("example/v1.star", nodes, allNodes, "test:v1")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// Starlark uses True (capital T)
	if !strings.Contains(content, `default=True`) {
		t.Errorf("expected default=True in field(), got:\n%s", content)
	}
}

func TestEmitFieldDefault_BoolFalse(t *testing.T) {
	node := &types.TypeNode{
		Name:          "Config",
		DefinitionKey: "example.Config",
		FilePath:      "example/v1.star",
		Fields: []types.FieldNode{
			{Name: "disabled", TypeName: "bool", Description: "Disabled", Default: false},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("example/v1.star", nodes, allNodes, "test:v1")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// Starlark uses False (capital F)
	if !strings.Contains(content, `default=False`) {
		t.Errorf("expected default=False in field(), got:\n%s", content)
	}
}

func TestEmitFieldDefault_ComplexSkipped(t *testing.T) {
	node := &types.TypeNode{
		Name:          "Config",
		DefinitionKey: "example.Config",
		FilePath:      "example/v1.star",
		Fields: []types.FieldNode{
			{Name: "metadata", TypeName: "dict", Description: "Metadata", Default: map[string]interface{}{}},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("example/v1.star", nodes, allNodes, "test:v1")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// Complex defaults should NOT emit default= argument
	if strings.Contains(content, "default=") {
		t.Errorf("expected no default= for complex type, got:\n%s", content)
	}
}

func TestEmitFieldDoc_WithDefault(t *testing.T) {
	node := &types.TypeNode{
		Name:          "Config",
		DefinitionKey: "example.Config",
		FilePath:      "example/v1.star",
		Fields: []types.FieldNode{
			{Name: "priority", TypeName: "string", Description: "Priority level", Default: "medium"},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("example/v1.star", nodes, allNodes, "test:v1")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// Doc should include (default: medium) with unquoted string value
	if !strings.Contains(content, `(default: medium)`) {
		t.Errorf("expected (default: medium) in doc, got:\n%s", content)
	}
}

func TestEmitFieldDoc_WithBoolDefault(t *testing.T) {
	node := &types.TypeNode{
		Name:          "Config",
		DefinitionKey: "example.Config",
		FilePath:      "example/v1.star",
		Fields: []types.FieldNode{
			{Name: "enabled", TypeName: "bool", Description: "Enabled", Default: true},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("example/v1.star", nodes, allNodes, "test:v1")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// Doc should include (default: True) with Starlark capitalization
	if !strings.Contains(content, `(default: True)`) {
		t.Errorf("expected (default: True) in doc, got:\n%s", content)
	}
}

func TestEmitSchema_EnumConstantsAbove(t *testing.T) {
	node := &types.TypeNode{
		Name:          "PodSpec",
		DefinitionKey: "io.k8s.api.core.v1.PodSpec",
		FilePath:      "core/v1.star",
		Fields: []types.FieldNode{
			{Name: "restart_policy", TypeName: "string", Description: "Restart policy", EnumValues: []string{"Always", "OnFailure", "Never"}},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("core/v1.star", nodes, allNodes, "test:v1")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// Constants must appear ABOVE the schema() call
	constIdx := strings.Index(content, `POD_SPEC_RESTART_POLICY_ALWAYS`)
	schemaIdx := strings.Index(content, `PodSpec = schema(`)

	if constIdx < 0 {
		t.Fatalf("expected enum constant, got:\n%s", content)
	}
	if schemaIdx < 0 {
		t.Fatalf("expected schema() call, got:\n%s", content)
	}
	if constIdx >= schemaIdx {
		t.Errorf("expected constants above schema(), constIdx=%d >= schemaIdx=%d\n%s", constIdx, schemaIdx, content)
	}
}

func TestEmitSchema_NoEnums(t *testing.T) {
	node := &types.TypeNode{
		Name:          "Simple",
		DefinitionKey: "example.Simple",
		FilePath:      "example/v1.star",
		Fields: []types.FieldNode{
			{Name: "name", TypeName: "string", Description: "The name"},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("example/v1.star", nodes, allNodes, "test:v1")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// No enum constants -- should start directly with schema()
	if strings.Contains(content, " = \"") {
		// Check this isn't a constant assignment (schema assignments use "= schema(")
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, " = \"") && !strings.Contains(trimmed, "= schema(") {
				t.Errorf("unexpected constant-like assignment in output:\n%s", content)
			}
		}
	}
}

func TestExistingEmitterTests(t *testing.T) {
	// This test verifies no regressions by running a representative existing scenario.
	node := &types.TypeNode{
		Name:          "DeploymentSpec",
		DefinitionKey: "io.k8s.api.apps.v1.DeploymentSpec",
		Description:   "DeploymentSpec is the specification of the desired behavior of the Deployment.",
		FilePath:      "apps/v1.star",
		Fields: []types.FieldNode{
			{Name: "replicas", TypeName: "int", Description: "Number of desired pods"},
			{Name: "paused", TypeName: "bool", Description: "Indicates that the deployment is paused"},
		},
	}

	nodes := []*types.TypeNode{node}
	allNodes := buildAllNodes(node)

	out, err := EmitFile("apps/v1.star", nodes, allNodes, "schemas-k8s:v1.31")
	if err != nil {
		t.Fatalf("EmitFile error: %v", err)
	}

	content := string(out)

	// Basic structure checks from existing tests
	if !strings.Contains(content, `DeploymentSpec = schema(`) {
		t.Error("expected DeploymentSpec = schema( call")
	}
	if !strings.Contains(content, `    "DeploymentSpec",`) {
		t.Error("expected name as first positional arg")
	}
	if !strings.Contains(content, `replicas=field(type="int"`) {
		t.Errorf("expected replicas field, got:\n%s", content)
	}
	if !strings.Contains(content, `paused=field(type="bool"`) {
		t.Errorf("expected paused field, got:\n%s", content)
	}
	// No enum constants should appear
	if strings.Contains(content, "DEPLOYMENT_SPEC_") {
		t.Errorf("unexpected enum constants for type without enums:\n%s", content)
	}
}

func TestToScreamingSnake_Empty(t *testing.T) {
	got := toScreamingSnake("")
	if got != "" {
		t.Errorf("toScreamingSnake(\"\") = %q, want \"\"", got)
	}
}

func TestToScreamingSnake_AllCapsRun(t *testing.T) {
	got := toScreamingSnake("HTTPSProxy")
	if got != "HTTPS_PROXY" {
		t.Errorf("toScreamingSnake(\"HTTPSProxy\") = %q, want \"HTTPS_PROXY\"", got)
	}
}

func TestToScreamingSnake_WithNumbers(t *testing.T) {
	got := toScreamingSnake("v2beta1")
	if got != "V2BETA1" {
		t.Errorf("toScreamingSnake(\"v2beta1\") = %q, want \"V2BETA1\"", got)
	}
}

func TestToScreamingSnake_NonAlphanumeric(t *testing.T) {
	got := toScreamingSnake("some-field.name")
	if got != "SOME_FIELD_NAME" {
		t.Errorf("toScreamingSnake(\"some-field.name\") = %q, want \"SOME_FIELD_NAME\"", got)
	}
}

func TestFormatStarlarkDefault_FloatAsInt(t *testing.T) {
	got, ok := formatStarlarkDefault(float64(3.0))
	if !ok {
		t.Fatal("formatStarlarkDefault(3.0) returned ok=false")
	}
	if got != "3" {
		t.Errorf("formatStarlarkDefault(3.0) = %q, want \"3\"", got)
	}
}

func TestFormatDocDefault_FloatWithDecimals(t *testing.T) {
	got, ok := formatDocDefault(float64(3.14))
	if !ok {
		t.Fatal("formatDocDefault(3.14) returned ok=false")
	}
	if got != "3.14" {
		t.Errorf("formatDocDefault(3.14) = %q, want \"3.14\"", got)
	}
}

// Verify sorting behavior directly.
func TestSortedKeys(t *testing.T) {
	m := map[string][]string{
		"z/v1.star": {"Beta", "Alpha"},
		"a/v1.star": {"Gamma"},
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if keys[0] != "a/v1.star" || keys[1] != "z/v1.star" {
		t.Errorf("expected sorted keys, got: %v", keys)
	}
}
