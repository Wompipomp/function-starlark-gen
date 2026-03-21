# starlark-gen

Generate typed Starlark schema libraries from Kubernetes and Crossplane OpenAPI specs. Provides compile-time validation for resource definitions in [function-starlark](https://github.com/wompipomp/function-starlark) compositions.

## Install

```bash
go install github.com/wompipomp/starlark-gen@latest
```

Or download a binary from [Releases](https://github.com/Wompipomp/function-starlark-gen/releases).

## Usage

### Kubernetes

Generate schemas from a K8s swagger.json:

```bash
curl -fsSL -o swagger.json \
  "https://raw.githubusercontent.com/kubernetes/kubernetes/v1.31.0/api/openapi-spec/swagger.json"

starlark-gen k8s swagger.json \
  --package schemas-k8s:v1.31 \
  --output ./schemas/k8s
```

### CRDs

Generate schemas from CustomResourceDefinition YAML files:

```bash
starlark-gen crd cert-manager-crds.yaml \
  --package schemas-cert-manager:v1 \
  --output ./schemas/cert-manager
```

Supports multiple files, multi-document YAML, and both v1 and v1beta1 CRD formats.

### Crossplane Providers

Generate schemas from Crossplane provider CRDs with lifecycle annotations and status exclusion:

```bash
starlark-gen provider provider-aws-crds/*.yaml \
  --package schemas-provider-aws:v1.14 \
  --output ./schemas/provider-aws
```

Provider schemas annotate `forProvider` fields as "Reconcilable configuration" and `initProvider` fields as "Write-once initialization". The `status` subtree is excluded entirely.

## Output

Generated `.star` files use function-starlark's `schema()` and `field()` builtins:

```starlark
load("schemas-k8s:v1.31/meta/v1.star", "LabelSelector", "ObjectMeta")
load("schemas-k8s:v1.31/core/v1.star", "PodTemplateSpec")

DeploymentSpec = schema(
    "DeploymentSpec",
    doc="DeploymentSpec defines the desired state of a Deployment.",
    replicas=field(type="int", doc="int - Number of desired pods"),
    selector=field(type=LabelSelector, required=True, doc="LabelSelector - (required)"),
    template=field(type=PodTemplateSpec, required=True, doc="PodTemplateSpec - (required)"),
)

Deployment = schema(
    "Deployment",
    doc="Deployment enables declarative updates for Pods and ReplicaSets.",
    metadata=field(type=ObjectMeta),
    spec=field(type=DeploymentSpec),
)
```

Use them in function-starlark compositions:

```starlark
load("schemas-k8s:v1.31/apps/v1.star", "Deployment")

def compose(req):
    deployment = Deployment(
        metadata=ObjectMeta(name="my-app"),
        spec=DeploymentSpec(
            replicas=3,
            selector=LabelSelector(matchLabels={"app": "my-app"}),
        ),
    )
    return [Resource(body=deployment, apiVersion="apps/v1", kind="Deployment")]
```

Typos, wrong types, and missing required fields are caught at construction time.

## Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--package`, `-p` | OCI package prefix for `load()` paths (required) | — |
| `--output`, `-o` | Output directory | `./out` |
| `--verbose`, `-v` | Show per-file listing instead of summary | `false` |

## Pre-built Schemas

Pre-generated schemas for Kubernetes and Crossplane providers are available at [function-starlark-schemas](https://github.com/Wompipomp/function-starlark-schemas).
