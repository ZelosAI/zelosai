# Suite-wide templates

This directory holds the canonical templates that every Zelos component
repository derives from. When you bootstrap a new repo (or retrofit an
existing one), copy from here.

## Files

| Template | Drop into | Notes |
|---|---|---|
| `README.md.tmpl` | `<repo>/README.md` | Replace `<repo-name>` and the one-paragraph description. |
| `pull_request_template.md` | `<repo>/.github/pull_request_template.md` | Identical across repos. |
| `lint.yml.go.tmpl` | `<repo>/.github/workflows/lint.yml` | Go repos. Runs go vet + gofmt + go test + golangci-lint. |
| `release.yml.go.tmpl` | `<repo>/.github/workflows/release.yml` | Go repos. Reads version from top-level `VERSION` file; multi-arch image push to `ghcr.io/zelosai/<repo>`. |
| `Dockerfile.go.tmpl` | `<repo>/Dockerfile` | Go service base. Replace `<REPO-NAME>`. |
| `Makefile.go.tmpl` | `<repo>/Makefile` | Go targets: `build run test lint fmt tidy image push clean`. |
| `Makefile.python.tmpl` | `<repo>/Makefile` | Python targets: `build run test lint fmt image push clean`. |
| `pyproject.python.tmpl` | `<repo>/pyproject.toml` | Python repos. Replace `<REPO-NAME>`. |
| `gitignore.go.tmpl` | `<repo>/.gitignore` | Go flavor. |
| `editorconfig.tmpl` | `<repo>/.editorconfig` | Identical across the suite. |
| `CODEOWNERS.tmpl` | `<repo>/.github/CODEOWNERS` | Identical across the suite. |
| `env.example.tmpl` | `<repo>/.env.example` | Starting point — add component-specific vars. |

Templates not yet materialised (planned): `CLAUDE.md.tmpl`, `lint.yml.python.tmpl`,
`release.yml.python.tmpl`, `Dockerfile.python.tmpl`, `gitignore.python.tmpl`.
Until they land, copy from an existing same-language repo of the same flavour.

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
