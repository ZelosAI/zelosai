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
| `ci.yml.tmpl` | `<repo>/.github/workflows/ci.yml` | PR validation. Two flavors in one file — keep one (see [ci.yml.tmpl](#ciymltmpl--per-repo-pr-validation) below). |
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

## `ci.yml.tmpl` — per-repo PR validation

`ci.yml.tmpl` is the canonical workflow that runs each repo's own unit tests +
lint on **pull requests into `develop` and `main`**. It is the source the
per-repo "Apply ci.yml" adoption chores consume. It complements, not replaces,
`lint.yml` (which runs on direct pushes to branches) and `release.yml` (image
build).

It ships **two flavors in a single file**, separated by a comment banner:

| Flavor | When to use | Setup + steps |
|---|---|---|
| **Go** | Go repos (`go.mod` present) | `actions/setup-go@v5` (`go-version-file: go.mod`) → `go vet ./...` → `go test -race -cover ./...` → `golangci-lint run`. |
| **Python** | Python repos (`pyproject.toml` present) | `actions/setup-python@v5` (`python-version-file: pyproject.toml`) → `pip install -e ".[dev]"` → `ruff check .` → `pytest -q --cov`. |

**How to apply:**

1. Copy `ci.yml.tmpl` to `<repo>/.github/workflows/ci.yml`.
2. **Delete the flavor you don't want** — keep exactly one of the two banner-
   delimited blocks. (Both blocks intentionally use `name: ci`; a valid
   workflow file must contain only one, so you *must* remove one block.)
3. Commit. No other edits — the template has **no substitution tokens** and
   renders verbatim.

**Expected per-repo customizations:** none beyond choosing the flavor. The Go
flavor assumes a `go.mod`; the Python flavor assumes a `pyproject.toml` with a
`[project.optional-dependencies].dev` extra that includes `ruff`, `pytest`, and
`pytest-cov` (see `pyproject.python.tmpl`). A repo with extra needs (e.g. a
service that needs Postgres for its tests) may add `services:` / steps, but the
core `test` job stays as shipped.

**Conventions baked in (and why):**

- **PR-only triggers** (`pull_request: branches: [develop, main]`) — this is the
  gate on PRs; push-to-branch lint is `lint.yml`'s job.
- **Concurrency group cancels superseded runs** — re-pushing to a PR cancels the
  in-flight run.
- **No matrix** — single OS (`ubuntu-latest`), single language version (from
  `go.mod` / `pyproject.toml`) for EA.
- **Coverage collected, not gated** — `-cover` / `--cov` produce reports but no
  minimum threshold is enforced for EA.

**Mixed Go+Python repos** (none exist today) would compose **both** `test` jobs
in one workflow rather than getting a third flavor. Note it here if it ever
happens; do not add a third flavor to the template.

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
