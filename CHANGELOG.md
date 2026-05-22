# Changelog

All notable changes to this repository are documented here. The format is based
on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] — operator + CRDs

### Added

- **Go + kubebuilder operator** (`cmd/`, `api/v1alpha1/`, `internal/controller/`).
- **CRDs** in API group `zelos.zelosai.io/v1alpha1`:
  - Umbrella `ZelosPlatform` composing every component.
  - Per-component `ZelosGateway`, `ZelosBackplane`, `ZelosMCP`,
    `ZelosBroker`, `ZelosServer`, `ZelosClient`.
- **Operator-installed substrate**: NATS StatefulSet with JetStream when
  `ZelosBackplane.spec.substrate=nats` and no `externalURL`. Redis/Kafka
  stay external-only with docs links.
- **Operator-installed OTel collector** + suite-wide env ConfigMap that
  every workload mounts via `envFrom`. External endpoint supported via
  `spec.telemetry.externalEndpoint`.
- **Standard container contract** (env vars, secret file paths, PVC paths,
  probes, ports) documented in
  [docs/architecture/07-container-contract.md](./docs/architecture/07-container-contract.md).
- **Dependencies inventory** and **GHCR pull-secret recipe** in
  [docs/architecture/09-dependencies.md](./docs/architecture/09-dependencies.md)
  and [docs/architecture/10-image-registry.md](./docs/architecture/10-image-registry.md).
- **Minimum scaled deployment** Kustomize bundle in
  [deploy/minimum/](./deploy/minimum/).
- **`docs.yml` CI workflow** validating every ```` ```mermaid ```` block.
- `docs/template/Makefile.go.tmpl` — Go-flavoured Makefile template (build /
  run / test -race / lint with vet+gofmt+golangci-lint / fmt / tidy /
  image / push / clean).
- `docs/template/release.yml.go.tmpl` — Go-flavoured GHCR release workflow,
  reads version from a top-level `VERSION` file. Same tagging policy as the
  Python release flow.
- Renamed `docs/template/Makefile.tmpl` → `docs/template/Makefile.python.tmpl`
  to match the existing `pyproject.python.tmpl` / `*.go.tmpl` naming
  convention. `docs/template/README.md` updated to reflect the new file
  set.

### Changed

- README's "Future direction" section updated — operator and CRDs are now
  in this pass, not a future one. Documented operator stack is
  Go + kubebuilder (was Python + kopf).
- **Suite-wide language split documented.** The four async-path / sync-path
  daemons (`zelosgateway`, `zelosbackplane`, `zelosclient`, `zelosbroker`)
  were rewritten in Go in their respective repos. `zelosmcp` and `zelosserver`
  stay Python. The component table in [README.md](./README.md) and
  [docs/architecture/00-overview.md](./docs/architecture/00-overview.md) now
  carries a per-repo language column; the per-component pages under
  [docs/architecture/04-components/](./docs/architecture/04-components/) note
  the language and updated layout paths (`cmd/<repo>` + `internal/*`).
- **Backplane wire-contract path updated.** The envelope JSON-Schemas and
  topic catalog moved from `zelosbackplane/src/zelosbackplane/schemas/` to
  language-neutral `zelosbackplane/schemas/`. References in
  [docs/architecture/06-naming-conventions.md](./docs/architecture/06-naming-conventions.md)
  and the backplane component page updated.

## [0.1.0-docs-only] — first-pass scaffold (pre-operator)

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
