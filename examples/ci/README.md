# Example CI Workflows

These GitHub Actions workflows automate Starlark schema regeneration for Kubernetes and Crossplane provider updates.

## Workflows

### k8s-schema-update.yaml

Regenerates K8s Starlark schemas when a new Kubernetes version is released. Downloads the swagger.json spec from the Kubernetes GitHub repository, runs `starlark-gen k8s`, and opens a pull request with the updated schemas.

### provider-schema-update.yaml

Regenerates Crossplane provider Starlark schemas when a new provider version is released. Downloads CRDs from Upbound GitHub releases (or a custom URL), runs `starlark-gen provider`, and opens a pull request with the updated schemas.

## Usage

1. Copy the desired workflow file into your repository's `.github/workflows/` directory.
2. Adjust input defaults to match your project (package name, output directory, etc.).
3. Run the workflow from the Actions tab using **Run workflow** and provide the required version inputs.

## Prerequisites

- **Go 1.25+** installed in the runner (handled by `actions/setup-go`).
- **`GITHUB_TOKEN`** with `contents: write` and `pull-requests: write` permissions for PR creation.
- For the provider workflow: `GITHUB_TOKEN` needs `repo` scope if downloading from private repositories, since `gh release download` uses it for authentication.

## Notes

- Both workflows use `workflow_dispatch` (manual trigger). Add `schedule` or other triggers as needed for your use case.
- The `starlark_gen_version` input defaults to `latest` but can be pinned to a specific release (e.g., `v0.1.0`) for reproducibility.
- Pull requests are created by [peter-evans/create-pull-request](https://github.com/peter-evans/create-pull-request). If your repository requires a PAT for PR creation (e.g., forks or branch protection rules), replace `GITHUB_TOKEN` with a PAT secret.

For more information, see [starlark-gen](https://github.com/wompipomp/starlark-gen).
