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

## In flight (v0.2)

Both target the same release because together they unblock an actual
end-to-end async-path run.

- **[A]** [zelosbackplane#11 — Pin substrate to NATS + implement NATS connector](https://github.com/ZelosAI/zelosbackplane/issues/11) — `Priority=P1`. Replaces TODO-returning skeleton with `nats.go` + JetStream against the `inference.requests.*` work queue.
- **[B]** [zelosclient#10 — Implement vLLM runtime adapter](https://github.com/ZelosAI/zelosclient/issues/10) — `Priority=P1`. Implements `Health()` and `Infer()` against the OpenAI-compatible vLLM API that `zelos.dgx` already provisions.

## Next (v0.3)

The broker rewrite + IDE extension; together they unlock the share-coordinator
primitive both flows depend on.

- **[C]** [zelosbroker#11 — Sync-channel + mount-coordinator primitive (WebDAV + HTTP-FUSE)](https://github.com/ZelosAI/zelosbroker/issues/11) — `Priority=P1`. The core rewrite. Replaces "asset puller + secure tunnel" with: WebSocket sync channel + pluggable share protocols + REST share-lifecycle API.
- **[D]** [zelosbroker#12 — Optional WireGuard wrapper for share traffic](https://github.com/ZelosAI/zelosbroker/issues/12) — `Priority=P2`. Optional encrypted-overlay wrap for IDE↔LLM-host traffic.
- **[E]** [zelos-vscode#1 — v0.1 IDE extension (config + OAuth + share lifecycle)](https://github.com/ZelosAI/zelos-vscode/issues/1) — `Priority=P1`. Settings panel, "Sign in to Zelos", "Open Zelos share", status-bar share state.

**Docs aligned with the v0.3 design change:**

- [zelosbroker#14 — docs: rewrite broker design (sync-channel + mount-coordinator)](https://github.com/ZelosAI/zelosbroker/issues/14) — `Priority=P1`.
- [zelosai#18 — docs: rewrite broker architecture pages + add ROADMAP.md](https://github.com/ZelosAI/zelosai/issues/18) — `Priority=P1`. (This file is the deliverable.)

## Following (v0.4)

- **[F]** [zelosbroker#13 — Add SMB share protocol (Samba sidecar)](https://github.com/ZelosAI/zelosbroker/issues/13) — `Priority=P2`. SMB rounds out the share-protocol palette for Windows-default and macOS-native customers. Split from C because mature pure-Go SMB-server libraries don't exist; this slice introduces a Samba sidecar in the broker Pod.

## Backlog (no release target)

Tracked as Backlog in the project; pull into a release as the need arises.

### Strategic decisions still TBD

- [zelosserver#10 — Decide zelosserver scope (UI / monitoring / doc store)](https://github.com/ZelosAI/zelosserver/issues/10).
- [zelosai#13 — Decide future Postgres role in the suite](https://github.com/ZelosAI/zelosai/issues/13).
- [zelosai#14 — Decide object-storage strategy (S3/MinIO/GCS) for LLM-generated assets](https://github.com/ZelosAI/zelosai/issues/14).
- [zelosai#15 — Decide search-backend strategy (OpenSearch / Meilisearch / none)](https://github.com/ZelosAI/zelosai/issues/15).
- [zelosai#16 — Decide cache-layer strategy for gateway and MCP (Redis vs in-memory vs none)](https://github.com/ZelosAI/zelosai/issues/16).
- [zelosai#17 — Decide operator job-scheduler strategy for periodic tasks](https://github.com/ZelosAI/zelosai/issues/17).

### Alternative implementations (will be needed when a customer demands them)

- [zelosbackplane#12 — Implement Redis Streams connector](https://github.com/ZelosAI/zelosbackplane/issues/12) — alt substrate.
- [zelosbackplane#13 — Implement Kafka connector](https://github.com/ZelosAI/zelosbackplane/issues/13) — alt substrate, high throughput / multi-DC.
- [zelosclient#11 — Implement Ollama runtime adapter](https://github.com/ZelosAI/zelosclient/issues/11) — alt runtime, edge / small-model.

## Open architectural questions

- **`zelosserver` scope.** Cleanest unblock would be a single concrete consumer (UI? monitoring hub?). See backlog #10.
- **Object-storage durability for assets.** The async path produces "usable assets" the IDE consumes; nothing in the suite persists them right now. Backlog #14.
- **Cache layer for the gateway.** Likely becomes inevitable once HPA-scaled gateway needs shared rate-limit / session state. Backlog #16.

## Process

All feature work follows the suite gitflow:

1. File the issue in the canonical repo (auto-adds to the [Zelos Platform Tracker](https://github.com/orgs/ZelosAI/projects/2) via `.github/workflows/add-to-project.yml`).
2. Set project fields: `Work type=Feature`, `Priority`, `Status=Todo`, `Release`. Set the matching repo milestone.
3. Branch off `develop`. PR back into `develop` with `Closes #N`. Promotion to `main` is a separate PR.
4. Back-merge `main → develop` when promotion adds commits develop doesn't have.

See [docs/architecture/05-gitflow.md](./docs/architecture/05-gitflow.md).
