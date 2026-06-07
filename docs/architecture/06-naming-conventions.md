# 06 — Naming conventions

## Repos

- All component repos: `ZelosAI/<lowercase-name>`, no hyphens, no underscores.
  Examples: `zelosai`, `zelosmcp`, `zelosgateway`, `zelosbackplane`, `zelosclient`, `zelosbroker`, `zelosserver`.
- Ansible collections: `zelos.<hosttype>` (Ansible Galaxy FQCN form). The
  GitHub repo name may differ (e.g. `kmechlin/ansible-dgx-collection` for
  `zelos.dgx`). Future collections should adopt `ZelosAI/zelos.<hosttype>`.

## Container images

- Registry: `ghcr.io/zelosai`.
- Image name = repo name: `ghcr.io/zelosai/<repo>`.
- Tags:
  - `vX.Y.Z` — semver release tag, pushed when a tag is cut from `main`.
  - `latest` — alias for the most recent `vX.Y.Z`.
  - `main` — head of `main`.
  - `sha-<short>` — every build, for traceability.

Pin images in deployment manifests by **tag** (`:vX.Y.Z`) or **digest**
(`@sha256:...`). Never pin to `:latest` outside of dev environments.

## Backplane topics

Topics live in `zelosbackplane/schemas/topics.yaml` (top-level,
language-neutral). Naming:

- Dotted hierarchy, lowercase: `<domain>.<action>.<qualifier>`.
- Examples:
  - `inference.requests.codegen`
  - `inference.requests.analysis`
  - `inference.responses.<corrId>` (one per correlation id)
  - `provisioning.events`
  - `metrics.<component>`

Reserve `system.*` for suite-internal lifecycle events (component up/down,
schema-version negotiation, etc.).

## Envelope fields

Every backplane message uses the canonical envelope from
`zelosbackplane/schemas/envelopes/v1/`:

| Field | Type | Meaning |
|---|---|---|
| `id` | string (UUID) | Globally unique message id. |
| `ts` | RFC 3339 timestamp | Emit time. |
| `source` | string | Component name that emitted (`zelosgateway`, etc.). |
| `kind` | string | Event/request kind. Matches the topic suffix. |
| `traceId` | string | Distributed trace id; propagated across hops. |
| `payload` | object | Kind-specific body. |

## CRDs (future — for `zelosai`'s operator)

API group: `zelos.ai`. Versions: `v1alpha1` → `v1beta1` → `v1`. Names:

- Cluster-scoped: `ZelosCluster`, `BareMetalHost`.
- Namespaced: `MCPGateway`, `Backplane`, `ModelWorker`, future `ZelosGateway`,
  `ZelosServer` if they grow non-trivial control logic.

Avoid generic names like `Service` or `Deployment` that collide with core
Kubernetes types.

## Environment variables

- Prefix component-specific vars with the **uppercase repo name** plus an
  underscore: `ZELOSMCP_PORT`, `ZELOSGATEWAY_AUTH_KEY`, `ZELOSBROKER_TUNNEL_CERT`, etc.
- Suite-wide vars use the prefix `ZELOSAI_*`: e.g. `ZELOSAI_SCHEMAS_VERSION`.
- Sensitive values live in `.env` (gitignored) or a Secret. Commit only `.env.example`.

## DNS & hostnames

Every Zelos service is addressed as:

```
<service>.<bed-or-cluster>.<product>.<domain>
```

e.g. `harbor.alpha.foundry.zelosai.cloud` — *which service*, on *which
instance* (a foundry **bed** or kube **cluster**/env), of *which product*
(`foundry` / `zelosai` / `bastion`), under *which* registered `<domain>`.

- **Single-service appliances** that aren't on a cluster (the `zelos.bastion`
  break-glass VM) drop `<service>` → `<env>.<product>.<domain>`
  (`prod.bastion.zelosai.cloud`).
- Each instance also serves a `.local` LAN mirror of the same hierarchy
  (`harbor.alpha.foundry.local`), covered by an internal-CA wildcard
  alongside the public one.
- Auth rides the bed **apex**: issuer `<bed-domain>/oidc`, GitHub callback
  `<bed-domain>/verify-auth`, SSO gate `auth.<bed-domain>`.

[16-dns-and-hostname-standard.md](./16-dns-and-hostname-standard.md) is the full
reference (dual wildcard certs, split-horizon, optional non-standard public
port, the interchangeable-IdP `oidc_*` parity layer).

## Branches

Detailed in [05-gitflow.md](./05-gitflow.md). Short version:

- `main` — protected.
- `develop` — integration.
- `claude/<slug>` — Claude session branches.
- `feat/<slug>`, `fix/<slug>`, `chore/<slug>` — non-Claude topic branches.

## Tags

- Repos: `vX.Y.Z` semver from `main` only.
- Schemas (inside `zelosai/schemas/` once it exists): independent semver in
  the schemas `VERSION` file.
- Ansible collections: semver in `galaxy.yml`.

## Why these conventions

To keep things scannable:

- Eyes can predict a repo URL from a component name.
- An image tag tells you whether it's a release, the HEAD of a branch, or a
  specific build.
- A topic name tells you what's on it without reading the schema.
- A CRD name doesn't collide with anything in Kubernetes core.

Deviating is fine when the convention actively gets in the way — but flag it
explicitly in the affected repo's CLAUDE.md "Notes / Blockers" section.
