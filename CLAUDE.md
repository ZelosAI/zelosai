# CLAUDE.md

> **Note for Claude sessions:** this file follows the Zelos suite-wide template
> from [docs/template/CLAUDE.md.tmpl](./docs/template/CLAUDE.md.tmpl). The
> canonical gitflow rules every Zelos repo follows live in
> [docs/architecture/05-gitflow.md](./docs/architecture/05-gitflow.md).

## Repository

- **Repo:** `ZelosAI/zelosai`
- **Image:** *(none — `zelosai` is an architecture / docs / templates repo, not a runtime component)*
- **Purpose:** Architecture hub for the Zelos suite. Holds the suite-wide
  CLAUDE.md / Dockerfile / Makefile / CI templates that every component repo
  derives from, plus the architecture documentation that ties the suite together.
  In a future pass this repo will also host a Python + kopf Kubernetes operator,
  per-component Helm charts, an umbrella chart, and shared schema libraries.
- **State:** v0.1.0 scaffold — bootstrapped during the first-pass repo sweep.
  No runtime code or charts yet; docs + templates only.

## Active Branch

- Work on: `claude/bootstrap-suite` (or whichever session branch the harness sets).

## Layout

```
zelosai/
├── README.md                       # suite-wide entry point
├── CLAUDE.md                       # this file
├── LICENSE                         # MIT
├── CHANGELOG.md
├── .gitignore  .editorconfig
├── docs/
│   ├── architecture/
│   │   ├── 00-overview.md          # suite overview + component map
│   │   ├── 01-async-path.md        # IDE → gateway → backplane → client → asset
│   │   ├── 02-sync-path.md         # IDE → broker (secure tunnel) → LLM host
│   │   ├── 03-provisioning.md      # zelos.dgx (and future zelos.* collections)
│   │   ├── 04-components/          # one page per component
│   │   ├── 05-gitflow.md           # canonical gitflow every repo cites
│   │   └── 06-naming-conventions.md
│   └── template/                   # suite-wide templates (CLAUDE, Dockerfile, CI, ...)
└── .github/
    ├── workflows/lint.yml          # markdown-lint + link-check on docs
    ├── pull_request_template.md
    └── CODEOWNERS
```

## How to run it / How to build it

This repo doesn't ship a runtime today. Validation is:

```bash
# Lint markdown + check internal links (CI does this on PR)
npx markdownlint-cli '**/*.md' --ignore node_modules
```

When the operator + charts land in a later pass, the standard Makefile targets
(`build / run / test / lint / image / push`) will apply.

## What has been verified / What has NOT been verified

- **Verified:** Architecture docs render and internal cross-links resolve.
  Templates in `docs/template/` are self-consistent (the **Git / Workflow**
  section of `CLAUDE.md.tmpl` matches the rules in `docs/architecture/05-gitflow.md`).
- **Not verified:** No operator, no charts, no shared libs yet — those are
  out of scope for this first pass.

## Configuration surface

None at runtime (yet). For docs-only CI:

- `.github/workflows/lint.yml` — markdown lint, no external secrets needed.

## Git / Workflow

- **Branching policy:** `main` is the protected release line. Feature branches
  (the `claude/*` session branches and any other topic branches) MUST be PR'd
  into `develop` first. Promotion from `develop` to `main` is a separate PR
  cut from `develop` once a set of features has been integrated and validated.
  Never open a PR from a feature branch directly against `main`. If `develop`
  does not yet exist on the remote, create it from `main` before opening the
  first feature PR.
- **Commits:** clear, descriptive messages. Co-author with Claude where applicable.
- **Tagging:** semver — `v0.1.0`, `v0.2.0`, … Tag from `main` only.
- **Container builds:** *(not applicable — no container)*
- **PRs:** do not create unless explicitly asked.

## Relation to the Zelos suite

`zelosai` is the suite's architecture hub. Every other repo's CLAUDE.md cites
[docs/architecture/05-gitflow.md](./docs/architecture/05-gitflow.md) for the
gitflow rules, derives its CLAUDE.md / CI workflows / Dockerfile / Makefile from
[docs/template/](./docs/template/), and is documented under
[docs/architecture/04-components/](./docs/architecture/04-components/).

## Notes / Blockers

- This repo is **docs + templates only** in v0.1.0. The Kubernetes operator,
  Helm charts, and shared libraries (`schemas/`) referenced in
  [README.md](./README.md) and the architecture docs are deferred to a
  follow-up planning pass.
- Migration of [zelos.dgx](https://github.com/kmechlin/ansible-dgx-collection)
  from `kmechlin/*` to `ZelosAI/*` is out of scope; the architecture docs
  link to its current home.
