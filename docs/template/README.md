# Suite-wide templates

This directory holds the canonical templates that every Zelos component
repository derives from. When you bootstrap a new repo (or retrofit an
existing one), copy from here.

## Files

| Template | Drop into | Notes |
|---|---|---|
| `CLAUDE.md.tmpl` | `<repo>/CLAUDE.md` | Replace `<REPO-NAME>` and the placeholder sections. The **Git / Workflow** section is byte-identical across the suite — don't edit it. |
| `README.md.tmpl` | `<repo>/README.md` | Replace `<repo-name>` and the one-paragraph description. |
| `pull_request_template.md` | `<repo>/.github/pull_request_template.md` | Identical across repos. |
| `lint.yml.python.tmpl` | `<repo>/.github/workflows/lint.yml` | Python repos. Runs ruff + pytest on PR and on pushes to `main` / `develop`. |
| `lint.yml.go.tmpl` | `<repo>/.github/workflows/lint.yml` | Go repos. Runs go vet + gofmt + go test + golangci-lint. |
| `release.yml.tmpl` | `<repo>/.github/workflows/release.yml` | Builds and pushes `ghcr.io/zelosai/<repo>` on `main` and on `v*` tag push. |
| `Dockerfile.python.tmpl` | `<repo>/Dockerfile` | Python service base. Replace `<PACKAGE_NAME>`. |
| `Dockerfile.go.tmpl` | `<repo>/Dockerfile` | Go service base. Replace `<REPO-NAME>`. |
| `Makefile.tmpl` | `<repo>/Makefile` | Standard targets: `build run test lint fmt image push clean`. |
| `pyproject.python.tmpl` | `<repo>/pyproject.toml` | Replace `<REPO-NAME>`. |
| `gitignore.python.tmpl` | `<repo>/.gitignore` | Python flavor. |
| `gitignore.go.tmpl` | `<repo>/.gitignore` | Go flavor. |
| `editorconfig.tmpl` | `<repo>/.editorconfig` | Identical across the suite. |
| `CODEOWNERS.tmpl` | `<repo>/.github/CODEOWNERS` | Identical across the suite. |
| `env.example.tmpl` | `<repo>/.env.example` | Starting point — add component-specific vars. |

## Substitution tokens

Replace these literally when applying a template to a repo:

| Token | Replace with | Example |
|---|---|---|
| `<REPO-NAME>` | The repo name | `zelosgateway` |
| `<PACKAGE_NAME>` | Python package (underscore form) | `zelosgateway` |
| `<repo-name>` | Lowercase repo name in prose | `zelosgateway` |

## Update policy

These templates are the source of truth. If a component repo's CLAUDE.md or
workflow needs to change in a way that should apply to the whole suite,
**update the template here first**, then propagate to every consuming repo
via a coordinated PR sweep.
