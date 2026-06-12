# Zelos suite roadmap

Suite-wide forward-looking view. For per-repo CLAUDE.md context, see each
component repo. For the canonical architecture, see
[docs/architecture/](./docs/architecture/). Live triage lives in the
[Zelos Platform Tracker](https://github.com/orgs/ZelosAI/projects/2).

## Vision

The Zelos suite sits between developer IDEs and their paid subscription LLMs.
It **reduces tokens billed to those subscriptions** and **offloads work to
local / datacenter-hosted models** wherever that's cheaper than letting the
frontier model do the heavy lifting. Tool catalogs are compressed, IDE assets
are managed centrally, sub-agent runs happen with isolated context on
Zelos-hosted models, and the workspace is exposed to those models via an
ephemeral remote mount that leaves nothing on the LLM host's disk.

See [docs/architecture/00-overview.md](./docs/architecture/00-overview.md) for
the long-form rationale and the suite map.

## Path to Early Access (v0.5)

EA tag is **v0.5 = packaging-only** (no new features). The data path lands
in v0.2 / v0.3 / v0.4; v0.5 is runbooks, install bundles, and demos
validated against real hardware. Three EA deployment strategies — `full`,
`split`, `solo` — each ships with its own kustomize overlay, runbook,
and demo. See
[docs/architecture/14-deployment-strategies.md](./docs/architecture/14-deployment-strategies.md)
(landing in v0.4) for topologies. Total scope: ~58 OPEN issues across 9
repos as of 2026-05-22.

## In flight (v0.2) — async-path foundation

Tagged dev containers exist; `IDE → backplane → off-cluster client → vLLM → response`
works end-to-end against an embedded NATS test server.

- [zelosbackplane#11 — Pin substrate to NATS + implement NATS connector](https://github.com/ZelosAI/zelosbackplane/issues/11) — Feature · P1 · v0.2
- [zelosclient#10 — Implement vLLM runtime adapter](https://github.com/ZelosAI/zelosclient/issues/10) — Feature · P1 · v0.2
- [zelosai#30 — Chore: Wire release.yml (multi-arch GHCR build)](https://github.com/ZelosAI/zelosai/issues/30) — Chore · P1 · v0.2
- [zelosai#31 — Chore: Sync 04-components docs to Go implementation](https://github.com/ZelosAI/zelosai/issues/31) — Chore · P1 · v0.2
- [zelos-vscode#12 — Chore: Wire release.yml (build .vsix on tag push)](https://github.com/ZelosAI/zelos-vscode/issues/12) — Chore · P1 · v0.2
- [zelosmcp#20 — Chore: Bump pyproject.toml 0.1.0 → 0.2.0](https://github.com/ZelosAI/zelosmcp/issues/20) — Chore · P1 · v0.2
- Plus per-repo version-sync chores (`zelosserver#19`, `zelosgateway#19`, `zelosbackplane#22`, `zelosbroker#26`, `zelosclient#21`).

## Next (v0.3) — share coordinator + IDE + gateway + mcp tools

Synchronous and asynchronous paths both have real plumbing end-to-end.
Gateway terminates auth and dispatches. Broker brokers shares. The IDE
extension can sign in, open a share, and drive a full inference round-trip.

**Architecture prerequisite (P0, must land first):**

- [zelosai#32 — Feature: Write docs/architecture/12-auth.md](https://github.com/ZelosAI/zelosai/issues/32) — Docs · P0 · v0.3

**Broker share-coordinator primitive:**

- [zelosbroker#11 — Feature: Sync-channel + mount-coordinator primitive (WebDAV + HTTP-FUSE)](https://github.com/ZelosAI/zelosbroker/issues/11) — Feature · P1 · v0.3
- [zelosbroker#12 — Feature: Optional WireGuard wrapper for share traffic](https://github.com/ZelosAI/zelosbroker/issues/12) — Feature · P2 · v0.3

**Gateway (3 slices: auth, dispatch, correlation):**

- [zelosgateway — Feature: OIDC auth termination + downstream identity propagation](https://github.com/ZelosAI/zelosgateway/issues) — Feature · P1 · v0.3
- [zelosgateway — Feature: Sync/async routing + dispatch to mcp/backplane](https://github.com/ZelosAI/zelosgateway/issues) — Feature · P1 · v0.3
- [zelosgateway — Feature: Envelope round-trip + GET /v1/inference/{id}](https://github.com/ZelosAI/zelosgateway/issues) — Feature · P1 · v0.3

**Backplane runtime hardening:**

- [zelosbackplane — Feature: Envelope schema validation + topic bootstrap + /readyz substrate gating](https://github.com/ZelosAI/zelosbackplane/issues) — Feature · P1 · v0.3

**zelosmcp feature plumbing (5 new gap issues, EA-readiness audit 2026-05-22):**

- [zelosmcp#23 — Feature: Broker HTTP + WebSocket client (share lifecycle + sync-channel frame schema)](https://github.com/ZelosAI/zelosmcp/issues/23) — Feature · P1 · v0.3 (gates #24 + #25)
- [zelosmcp#24 — Feature: Sync-subagent MCP tools (expose subagents as tools; stream turns over broker WS)](https://github.com/ZelosAI/zelosmcp/issues/24) — Feature · P1 · v0.3
- [zelosmcp#25 — Feature: Backplane NATS publisher for async-task MCP tools](https://github.com/ZelosAI/zelosmcp/issues/25) — Feature · P1 · v0.3
- [zelosmcp#26 — Feature: Bearer-token issuance to broker + backplane on tool invoke](https://github.com/ZelosAI/zelosmcp/issues/26) — Feature · P1 · v0.3

**zelos-vscode end-to-end IDE surface:**

- [zelos-vscode#1 — Feature: v0.1 IDE extension (config + OAuth + share lifecycle)](https://github.com/ZelosAI/zelos-vscode/issues/1) — Feature · P1 · v0.3
- [zelos-vscode#13 — Feature: "Run inference prompt" command (POST /v1/inference + GET poll loop)](https://github.com/ZelosAI/zelos-vscode/issues/13) — Feature · P1 · v0.3
- [zelos-vscode#14 — Feature: Response display surface (Output channel)](https://github.com/ZelosAI/zelos-vscode/issues/14) — Feature · P1 · v0.3

**Docs aligned with the v0.3 design change:**

- [zelosbroker#14 — docs: rewrite broker design (sync-channel + mount-coordinator)](https://github.com/ZelosAI/zelosbroker/issues/14) — Docs · P1 · v0.3
- [zelosai#18 — docs: rewrite broker architecture pages + add ROADMAP.md](https://github.com/ZelosAI/zelosai/issues/18) — Docs · P1 · v0.3

## Following (v0.4) — hardening + provisioning + zelosserver scope + 3 overlays

Provisioning end-to-end. Operator reports real status. Security envelope
around shares. zelosserver scope locked. All three deployment overlays
(`full`, `split`, `solo`) merged. Integration test on kind. Hardware
procurement kicked off.

**Provisioning:**

- [zelos.dgx — Feature: Implement zelosclient delivery role](https://github.com/ZelosAI/zelos.dgx/issues) — Feature · P1 · v0.4
- [zelos.dgx — Feature: zdgx smoke subcommand](https://github.com/ZelosAI/zelos.dgx/issues) — Feature · P2 · v0.4

**zelosserver scope (re-targeted from Backlog to v0.4):**

- [zelosai#33 — Feature: Write docs/architecture/13-zelosserver-scope.md](https://github.com/ZelosAI/zelosai/issues/33) — Docs · P1 · v0.4
- [zelosserver#10 — Feature: zelosserver scope decision + MVP](https://github.com/ZelosAI/zelosserver/issues/10) — Feature · P1 · v0.4

**Operator + security hardening:**

- [zelosai#34 — Feature: Operator status conditions + readiness propagation](https://github.com/ZelosAI/zelosai/issues/34) — Feature · P2 · v0.4
- [zelosai#35 — Feature: Integration test on kind (operator + minimum deployment)](https://github.com/ZelosAI/zelosai/issues/35) — Feature · P1 · v0.4
- [zelosbroker — Feature: Share-token security hardening + signed TTL + allowedLLMHosts enforcement](https://github.com/ZelosAI/zelosbroker/issues) — Feature · P1 · v0.4

**Deployment strategies — architecture + three overlays:**

- [zelosai#36 — Feature: Write docs/architecture/14-deployment-strategies.md](https://github.com/ZelosAI/zelosai/issues/36) — Docs · P0 · v0.4
- [zelosai#37 — Feature: deploy/solo/ overlay](https://github.com/ZelosAI/zelosai/issues/37) — Feature · P1 · v0.4
- [zelosai#38 — Feature: deploy/split/ overlay](https://github.com/ZelosAI/zelosai/issues/38) — Feature · P1 · v0.4
- [zelosai#39 — Feature: deploy/full/ overlay](https://github.com/ZelosAI/zelosai/issues/39) — Feature · P1 · v0.4

**Install + observability + hardware:**

- [zelosai#40 — Feature: OTel collector recommended config + 2-3 Grafana dashboards](https://github.com/ZelosAI/zelosai/issues/40) — Feature · P2 · v0.4
- [zelosai#41 — Chore: Secure access to v0.5 validation hardware](https://github.com/ZelosAI/zelosai/issues/41) — Chore · P0 · v0.4

**CI test workflow rollout (EA-readiness gap C):**

- [zelosai#55 — Chore: Add docs/template/ci.yml.tmpl (Go + Python flavors)](https://github.com/ZelosAI/zelosai/issues/55) — Chore · P2 · v0.4
- Per-repo adoption chores: [zelosserver#23](https://github.com/ZelosAI/zelosserver/issues/23), [zelosmcp#28](https://github.com/ZelosAI/zelosmcp/issues/28), [zelosgateway#25](https://github.com/ZelosAI/zelosgateway/issues/25), [zelosbroker#30](https://github.com/ZelosAI/zelosbroker/issues/30), [zelosbackplane#26](https://github.com/ZelosAI/zelosbackplane/issues/26), [zelosclient#24](https://github.com/ZelosAI/zelosclient/issues/24), [zelos.dgx#32](https://github.com/ZelosAI/zelos.dgx/issues/32), [zelos-vscode#17](https://github.com/ZelosAI/zelos-vscode/issues/17) — all Chore · P2 · v0.4

**Ops housekeeping:**

- [zelosai#56 — Chore: Document ADD_TO_PROJECT_PAT lifecycle + rotation](https://github.com/ZelosAI/zelosai/issues/56) — Chore · P2 · v0.4

**zelosmcp subagent context (EA-nice-to-have):**

- [zelosmcp#27 — Feature: Subagent artifact loader (skills + hooks bundle)](https://github.com/ZelosAI/zelosmcp/issues/27) — Feature · P2 · v0.4

## v0.5 — EARLY ACCESS TAG (packaging only — no new features)

All three strategies smoke-validated against real hardware. Zero new
functionality; every issue is documentation, packaging, or validation. If
a feature sneaks in OR any one strategy fails its smoke run, EA slips.

**Per-strategy runbooks + bundles + demos (3 × 3 = 9 issues):**

- [zelosai#42 — Feature: full strategy — smoke runbook](https://github.com/ZelosAI/zelosai/issues/42) — Docs · P1 · v0.5
- [zelosai#43 — Feature: full strategy — EA install bundle (deploy/ea/full/)](https://github.com/ZelosAI/zelosai/issues/43) — Feature · P1 · v0.5
- [zelosai#44 — Feature: full strategy — demo scenario](https://github.com/ZelosAI/zelosai/issues/44) — Docs · P1 · v0.5
- [zelosai#45 — Feature: split strategy — smoke runbook (WireGuard recommended VPN)](https://github.com/ZelosAI/zelosai/issues/45) — Docs · P1 · v0.5
- [zelosai#46 — Feature: split strategy — EA install bundle (deploy/ea/split/, WireGuard default)](https://github.com/ZelosAI/zelosai/issues/46) — Feature · P1 · v0.5
- [zelosai#47 — Feature: split strategy — demo scenario](https://github.com/ZelosAI/zelosai/issues/47) — Docs · P1 · v0.5
- [zelosai#48 — Feature: solo strategy — smoke runbook](https://github.com/ZelosAI/zelosai/issues/48) — Docs · P1 · v0.5
- [zelosai#49 — Feature: solo strategy — EA install bundle (deploy/ea/solo/, WireGuard default)](https://github.com/ZelosAI/zelosai/issues/49) — Feature · P1 · v0.5
- [zelosai#50 — Feature: solo strategy — demo scenario](https://github.com/ZelosAI/zelosai/issues/50) — Docs · P1 · v0.5

**Cross-strategy docs:**

- [zelosai#51 — Feature: Getting-started doc + strategy-chooser flowchart](https://github.com/ZelosAI/zelosai/issues/51) — Docs · P1 · v0.5
- [zelosai#52 — Feature: Auth provider documentation + Dex recipe + Secret runbook](https://github.com/ZelosAI/zelosai/issues/52) — Docs · P1 · v0.5

**IDE extension per-strategy guidance + Marketplace publish:**

- [zelos-vscode#15 — Feature: Per-strategy connection-config guidance (full Ingress / split WireGuard / solo NodePort)](https://github.com/ZelosAI/zelos-vscode/issues/15) — Feature · P1 · v0.5
- [zelos-vscode#16 — Feature: .vsix Marketplace publish pipeline](https://github.com/ZelosAI/zelos-vscode/issues/16) — Feature · P2 · v0.5

## Infra collections — Ansible standards + refactor campaign (milestone v0.5, infra repos)

Distinct from the suite EA v0.5 packaging scope above: a cross-repo campaign over the
**infrastructure collections** (zelos.common · zelos.proxmox · zelos.kubernetes ·
zelos.foundry · zelos.bastion · zelos.dgx), tracked on the
[Zelos Foundry Tracker](https://github.com/orgs/ZelosAI/projects/5). Canon:
[15-ansible-collection-conventions.md](./docs/architecture/15-ansible-collection-conventions.md)
+ [18-organization-model.md](./docs/architecture/18-organization-model.md).

**In flight (Phase 0 — standards + propagation):**

- [zelosai#96 — docs: Ansible collection conventions v2](https://github.com/ZelosAI/zelosai/issues/96) — Docs · P1
- [zelosai#97 — docs: organization model + variable registry (18) + DNS tenancy update (16)](https://github.com/ZelosAI/zelosai/issues/97) — Docs · P1
- Per-repo propagation: [proxmox#41](https://github.com/ZelosAI/zelos.proxmox/issues/41) · [kubernetes#74](https://github.com/ZelosAI/zelos.kubernetes/issues/74) · [foundry#268](https://github.com/ZelosAI/zelos.foundry/issues/268) · [bastion#81](https://github.com/ZelosAI/zelos.bastion/issues/81) · [dgx#45](https://github.com/ZelosAI/zelos.dgx/issues/45) — Chore · P1

**Next (Phases 1–6, strict cross-repo ordering — see issue bodies):**

- Phase 1 foundations: [zelosai#98 scaffold zelos.common](https://github.com/ZelosAI/zelosai/issues/98) · common [#1](https://github.com/ZelosAI/zelos.common/issues/1)/[#2](https://github.com/ZelosAI/zelos.common/issues/2)/[#3](https://github.com/ZelosAI/zelos.common/issues/3) (google modules/roles, secrets role) · [foundry#269 legacy env path removal](https://github.com/ZelosAI/zelos.foundry/issues/269) · shell→module [kubernetes#75](https://github.com/ZelosAI/zelos.kubernetes/issues/75)/[proxmox#42](https://github.com/ZelosAI/zelos.proxmox/issues/42)
- Phase 2 adoption + deconfliction (order 15→16/17→18→19): common [#4 environment_facts](https://github.com/ZelosAI/zelos.common/issues/4) · [kubernetes#76](https://github.com/ZelosAI/zelos.kubernetes/issues/76)/[#77](https://github.com/ZelosAI/zelos.kubernetes/issues/77)/[#78](https://github.com/ZelosAI/zelos.kubernetes/issues/78) · [proxmox#43](https://github.com/ZelosAI/zelos.proxmox/issues/43) · [foundry#270](https://github.com/ZelosAI/zelos.foundry/issues/270)/[#271](https://github.com/ZelosAI/zelos.foundry/issues/271)
- Phase 3 GTS/Cloud DNS consumption: [kubernetes#79 cert_manager](https://github.com/ZelosAI/zelos.kubernetes/issues/79) · [kubernetes#80 external_dns](https://github.com/ZelosAI/zelos.kubernetes/issues/80)
- Phase 4 bastion roles conversion (greenfield validation; oobm.alpha FROZEN): [bastion#82](https://github.com/ZelosAI/zelos.bastion/issues/82) · [bastion#83](https://github.com/ZelosAI/zelos.bastion/issues/83)
- Phase 5 composition flip: [kubernetes#81](https://github.com/ZelosAI/zelos.kubernetes/issues/81) · [foundry#272](https://github.com/ZelosAI/zelos.foundry/issues/272)
- Phase 6 cutover + cleanup: zelosctl entrypoints [foundry#273](https://github.com/ZelosAI/zelos.foundry/issues/273)/[bastion#84](https://github.com/ZelosAI/zelos.bastion/issues/84) · per-repo cleanup [proxmox#44](https://github.com/ZelosAI/zelos.proxmox/issues/44)/[kubernetes#82](https://github.com/ZelosAI/zelos.kubernetes/issues/82)/[foundry#274](https://github.com/ZelosAI/zelos.foundry/issues/274)/[bastion#85](https://github.com/ZelosAI/zelos.bastion/issues/85) · umbrella [zelosai#99](https://github.com/ZelosAI/zelosai/issues/99)

## v0.6 — pre-1.0 hardening (post-EA, before v1.0)

User explicitly wanted SMB to land before v1.0. v0.6 absorbs SMB plus any
EA feedback themes.

- [zelosbroker#13 — Feature: Add SMB share protocol (Samba sidecar)](https://github.com/ZelosAI/zelosbroker/issues/13) — Feature · P2 · v0.6 (re-targeted from v0.4 on 2026-05-22)

## v1.0 — production grade (deferred)

Existing decisions stay backlog or move to v1.0:

- [zelosai#13 — Decide future Postgres role](https://github.com/ZelosAI/zelosai/issues/13) — Chore · P3 · v1.0
- [zelosai#14 — Decide object-storage strategy](https://github.com/ZelosAI/zelosai/issues/14) — Chore · P3 · v1.0
- [zelosai#15 — Decide search-backend strategy](https://github.com/ZelosAI/zelosai/issues/15) — Chore · P3 · v1.0
- [zelosai#16 — Decide cache-layer strategy](https://github.com/ZelosAI/zelosai/issues/16) — Chore · P3 · v1.0
- [zelosai#17 — Decide operator job-scheduler strategy](https://github.com/ZelosAI/zelosai/issues/17) — Chore · P3 · v1.0

New v1.0 themes to file once v0.5 ships: HA story, upgrade choreography,
multi-tenant isolation, Helm umbrella chart, shared schema libraries,
cert-manager/TLS ingress, in-cluster NVIDIA device-plugin recipe.

## Backlog (no release target)

Alternative implementations — pull into a release when a customer needs them.

- [zelosbackplane#12 — Implement Redis Streams connector](https://github.com/ZelosAI/zelosbackplane/issues/12) — alt substrate
- [zelosbackplane#13 — Implement Kafka connector](https://github.com/ZelosAI/zelosbackplane/issues/13) — alt substrate, high throughput
- [zelosclient#11 — Implement Ollama runtime adapter](https://github.com/ZelosAI/zelosclient/issues/11) — alt runtime, edge / small-model

## Open architectural questions

- **Object-storage durability for assets.** The async path produces "usable
  assets" the IDE consumes; nothing persists them today. zelosai#14.
- **Cache layer for the gateway.** Inevitable once HPA-scaled gateway needs
  shared rate-limit / session state. zelosai#16.
- **Multi-tenancy.** EA is single-tenant by decision (`12-auth.md`).
  Tenant claim → mcp/backplane scoping is a v1.0 item.

## Process

All feature work follows the suite gitflow:

1. File the issue in the canonical repo (auto-adds to the [Zelos Platform Tracker](https://github.com/orgs/ZelosAI/projects/2) via `.github/workflows/add-to-project.yml`).
2. Set project fields: `Work type=Feature`, `Priority`, `Status=Todo`, `Release`. Set the matching repo milestone.
3. Branch off `develop`. PR back into `develop` with `Closes #N`. Promotion to `main` is a separate PR.
4. Back-merge `main → develop` when promotion adds commits develop doesn't have.

See [docs/architecture/05-gitflow.md](./docs/architecture/05-gitflow.md) and the `## Planning and execution loop` section of every repo's `CLAUDE.md`.

## See also: per-repo ROADMAPs

Each component repo carries its own forward-looking view (issues filed
against that repo, bucketed into `In flight` / `Next` / `Backlog` /
`Recently shipped`). This file is the cross-component aggregator.

- [zelosbackplane/ROADMAP.md](https://github.com/ZelosAI/zelosbackplane/blob/main/ROADMAP.md)
- [zelosbroker/ROADMAP.md](https://github.com/ZelosAI/zelosbroker/blob/main/ROADMAP.md)
- [zelosclient/ROADMAP.md](https://github.com/ZelosAI/zelosclient/blob/main/ROADMAP.md)
- [zelosgateway/ROADMAP.md](https://github.com/ZelosAI/zelosgateway/blob/main/ROADMAP.md)
- [zelosmcp/ROADMAP.md](https://github.com/ZelosAI/zelosmcp/blob/main/ROADMAP.md)
- [zelosserver/ROADMAP.md](https://github.com/ZelosAI/zelosserver/blob/main/ROADMAP.md)
- [zelos.dgx/ROADMAP.md](https://github.com/ZelosAI/zelos.dgx/blob/main/ROADMAP.md)
- [zelos-vscode/ROADMAP.md](https://github.com/ZelosAI/zelos-vscode/blob/main/ROADMAP.md)
