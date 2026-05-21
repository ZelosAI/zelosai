# Changelog

All notable changes to this repository are documented here. The format is based
on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Suite-wide CLAUDE.md / Dockerfile / Makefile / CI workflow templates in
  [docs/template/](./docs/template/).
- Architecture documentation in [docs/architecture/](./docs/architecture/):
  - `00-overview.md` — suite overview and component map.
  - `01-async-path.md` — IDE → gateway → backplane → client → asset.
  - `02-sync-path.md` — IDE → broker (secure tunnel) → LLM host.
  - `03-provisioning.md` — `zelos.dgx` and the future `zelos.<hosttype>` collections.
  - `04-components/` — one page per component (mcp, gateway, backplane, client, broker, server, dgx).
  - `05-gitflow.md` — canonical gitflow every repo follows.
  - `06-naming-conventions.md` — repo / image / topic / tag / env-var naming.
- Repo-level scaffolding: `README.md`, `CLAUDE.md`, `.gitignore`, `.editorconfig`,
  `.github/{workflows/lint.yml, pull_request_template.md, CODEOWNERS}`.

## [0.1.0] — first-pass scaffold

Initial public scaffold. Docs and templates only; no runtime components yet.
