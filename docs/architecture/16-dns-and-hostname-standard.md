# 16 — DNS & hostname standard

The suite-wide convention for how every Zelos service is addressed on the network, the TLS
certificates that cover those names, and the unified OIDC contract that rides on them. This is
the **source of truth**; the component repos (`zelos.kubernetes`, `zelos.foundry`,
`zelos.bastion`) implement it and link back here. The identity levels themselves (tenancy /
environment / product) are defined in [18-organization-model.md](18-organization-model.md).

## The standard

```
<service>.<environment>.<product>.<domain>                  # environment-level services
<service>.<tenancy>.<environment>.<product>.<domain>        # tenant-exposed services (optional level)
```

| Label | Meaning | Examples |
|---|---|---|
| `<service>` | the component / UI being addressed | `harbor`, `argo`, `argocd`, `results`, `hooks`, `auth`, `gateway`, `rancher` |
| `<tenancy>` | *(optional)* a logical container inside the environment — a k8s namespace with quotas/RBAC | `zelosai`, `teamx` |
| `<environment>` | the deployment instance (formerly a foundry **"bed"**) | `alpha`, `develop`, `production`, `staging` |
| `<product>` | the Zelos product the deployment belongs to — **always present** | `foundry`, `zelosai`, `bastion` |
| `<domain>` | the registered DNS root | `zelosai.cloud` (ZelosAI's own infra); customers use their own |

**Worked example (alpha foundry environment):** `harbor.alpha.foundry.zelosai.cloud`.

The labels read left-to-right from most-specific to least: *which service*, (optionally *in which
tenancy*,) on *which environment*, of *which product*, under *which org/customer domain*. One
glance localises any hostname in the suite — useful when troubleshooting several
clusters/products at once.

### The tenancy level (optional, additive)

Environment-level platform services stay at `<service>.<environment>.<product>.<domain>` —
introducing tenancies migrates **nothing**. A tenancy that exposes ingress gets the deeper form
`<service>.<tenancy>.<environment>.<product>.<domain>`, covered by a **per-tenancy wildcard**
certificate (`*.{tenancy}.{environment}.{product}.{root}`) minted on demand — public CA
wildcards cover exactly one label, so each exposing tenancy carries its own.

Because tenancies and services share the label right of `<environment>`'s services, a
**reserved-name registry** applies: `harbor`, `argocd`, `argo`, `auth`, `rancher`, `results`,
`hooks`, `gateway`, `registry`, `api`, `mgmt` are service names a tenancy cannot claim
(authoritative list in [18-organization-model.md](18-organization-model.md)).

### Single-service appliances drop `<service>`

A deployment that is **not** on a cluster/bed and exposes a **single** service collapses the
first two labels — it has no service-vs-instance ambiguity to resolve:

```
<env>.<product>.<domain>
```

`zelos.bastion` is the canonical case: a standalone break-glass VM →
**`prod.bastion.zelosai.cloud`** (or `staging.bastion.zelosai.cloud`). Here `<env>` plays the
role of the instance label and the product's single UI (Guacamole) answers at the apex.

### Products

| Product | What it is | Typical environment label | Example host |
|---|---|---|---|
| `foundry` | the CI / test-bed platform (an environment = a "bed") | `alpha`, `beta` | `argocd.alpha.foundry.zelosai.cloud` |
| `zelosai` | the Zelos AI platform (operator + gateway/mcp/backplane) | `develop`, `production` | `gateway.production.zelosai.<domain>` |
| `bastion` | the standalone OOB recovery gateway (single-service) | `prod` | `prod.bastion.zelosai.cloud` |

New products slot in by adding a `<product>` label; nothing else about the standard changes. The
`<product>` label is **always present** — the uniform shape keeps parsing, wildcard issuance, and
external-dns zone filters special-case-free (even when the product name matches the root's first
label: `gateway.develop.zelosai.zelosai.cloud`).

## Two parallel domains (same hierarchy)

Every instance serves the **same names** on two domains so the identical hostname structure
works from the public internet and from the LAN:

| Domain | Form | Resolves to | Cert |
|---|---|---|---|
| **public** | `<env>.<product>.<domain>` → `*.<env>.<product>.<domain>` | WAN A-record (DDNS / router) via public DNS | `zelos-public-tls` |
| **LAN** | `<env>.<product>.local` → `*.<env>.<product>.local` | environment VIP via in-cluster dnsmasq | `zelos-local-tls` |

The `.local` domain mirrors the public hierarchy exactly (`harbor.alpha.foundry.local` ↔
`harbor.alpha.foundry.zelosai.cloud`) so an operator troubleshooting on the LAN uses the same
muscle memory, and names never collide across clusters/products.

### Split-horizon

The public names also resolve **on the LAN** (the environment's dnsmasq answers `*.<env>.<product>.<domain>`
with the gateway VIP), so an operator can use the real, publicly-trusted names locally and still
get a valid certificate. Per-host overrides point management-only services (e.g. Rancher) at the
management VIP instead of the public one.

## TLS — dual wildcard, always-on internal CA, port-agnostic

cert-manager issues **two** wildcard certificates per environment:

- **`zelos-public-tls`** = `*.<env>.<product>.<domain>` (+ apex) — issued by **ACME**
  (Let's Encrypt or Google Trust Services via DNS-01 on Google Cloud DNS) **or** the internal
  self-signed CA, selected by `cert_manager.mode`.
- **`zelos-local-tls`** = `*.<env>.<product>.local` (+ apex) — **always** the internal
  self-signed CA, because no public CA will sign a `.local` name.

The internal CA chain (`zelos-selfsigned-bootstrap` → `zelos-root-ca` → `zelos-self-signed-ca`)
and trust-manager stay enabled in **all** modes — they sign the `.local` wildcard regardless of
how the public one is issued. In self-signed mode a single gateway block serves both wildcards.

**Certs are port-agnostic.** TLS is matched by SNI / Host header, never by port — so the same
certificate is valid on the WAN, on a non-standard public port, on the private management port,
and on the `.local` names. There is nothing port-specific in a Zelos certificate.

### Optional non-standard public port

`environment:` config may set an optional **public port** when the standard `:443` is taken
by another tenant on the same WAN IP. The alpha environment runs publicly on **`:9443`** (its
`https://<svc>.alpha.foundry.zelosai.cloud:9443`), the router forwards WAN `:9443` → the gateway
VIP, and — because certs are port-agnostic — the wildcard is valid there unchanged. All derived
URLs (OIDC issuer/callback, OAuth redirect, registry `externalURL`, webhook target) carry the
port suffix; `:443` / unset ⇒ no suffix.

## Unified OIDC contract (interchangeable IdP)

Auth rides on the environment **apex** host so the two identity providers — **Dex** and **Rancher** —
are drop-in interchangeable (mutually exclusive, chosen per environment). Pick **Rancher** when you
want its full management UI; pick **Dex** as the light option on small hosts. The contract is
identical for both, so swapping the IdP needs **zero** changes in GitHub or in any OIDC client:

| Endpoint | Value | Notes |
|---|---|---|
| **OIDC issuer** | `https://<environment-domain><:port>/oidc` | Rancher-native; **Dex reconfigured from `/dex` → `/oidc`** |
| **GitHub OAuth callback** | `https://<environment-domain><:port>/verify-auth` | Rancher-native; Dex's connector redirectURI set to it + an apex route rewrites `/verify-auth` → Dex's `/oidc/callback` |
| **SSO gate** | `auth.<environment-domain>` | oauth2-proxy; cookie scoped to `.<environment-domain>` for cross-subdomain SSO |
| **Apps** | `<svc>.<environment-domain>` | each gated app behind the SSO gate (except registry + webhook hosts) |

**One GitHub OAuth App** per environment, with one Homepage + one callback (`…/verify-auth`),
serves whichever IdP is deployed. `zelos.bastion` follows the same `/oidc` + `/verify-auth`
contract for its local Dex even though it has no Rancher — so OAuth Apps are registered
identically everywhere in the suite.

### The `oidc_*` parity layer

Dex and Rancher differ in scope handling and group claims; the platform resolves an `oidc_*`
fact set once from the chosen provider and threads it into every client, hiding the difference:

| Fact | Dex | Rancher |
|---|---|---|
| `oidc_issuer_url` | `<environment-domain>/oidc` | `<environment-domain>/oidc` |
| `oidc_scopes` | `openid profile email groups` | `openid profile offline_access` (Rancher **rejects** `groups`/`email` scopes) |
| `oidc_groups_claim` | `groups` | `groups` |
| `oidc_admin_group` | `<org>:<team-slug>` | Rancher principal format (overridable) |
| `insecure_enable_groups` | n/a | `true` |

See [12-auth.md](./12-auth.md) for the auth model and `zelos.foundry`
[`docs/architecture/23-idp-strategy.md`](https://github.com/ZelosAI/zelos.foundry/blob/develop/docs/architecture/23-idp-strategy.md)
for the interchangeable-IdP rationale.

## How it's activated

- **`zelos.kubernetes` / `zelos.foundry` environments** set the `environment:` identity tuple —
  `name`, `product`, `root_domain`, optional `public_port`, `cluster:` binding (see
  [18-organization-model.md](18-organization-model.md); the pre-v0.5 `cluster:` dict with
  `bed` is read as a deprecated alias for one phase). When `environment.root_domain` is set,
  `zelos.common.environment_facts` derives every hostname, the dual certs, the SSO gate, and the
  unified OIDC issuer/callback (`[always]` facts: `environment_domain`,
  `environment_local_domain`, `public_port_suffix`, `oidc_*`). **Unset ⇒ legacy single-host
  path-based routing**, byte-identical to pre-standard behaviour.
- **`zelos.bastion`** sets `BASTION_HOSTNAME=<env>.bastion.zelosai.cloud` (cloud-init), and its
  Caddy + local Dex serve `/oidc` + `/verify-auth`.

Full setup, including the Google Cloud variables (managed zone, `dns.admin` SA, GTS EAB):
`zelos.kubernetes`
[`docs/google-trust-services-cert-manager.md`](https://github.com/ZelosAI/zelos.kubernetes/blob/develop/docs/google-trust-services-cert-manager.md)
and [`docs/cluster-tls-and-dns.md`](https://github.com/ZelosAI/zelos.kubernetes/blob/develop/docs/cluster-tls-and-dns.md).

## Worked examples

```
# foundry — alpha environment, product foundry, root zelosai.cloud, public port 9443
harbor.alpha.foundry.zelosai.cloud:9443      # registry UI + OCI endpoint (ungated)
argocd.alpha.foundry.zelosai.cloud:9443      # Argo CD (SSO-gated)
auth.alpha.foundry.zelosai.cloud:9443        # oauth2-proxy SSO gate
alpha.foundry.zelosai.cloud:9443/oidc        # OIDC issuer (Dex or Rancher)
alpha.foundry.zelosai.cloud:9443/verify-auth # GitHub OAuth callback
harbor.alpha.foundry.local                   # same service, LAN name, internal-CA cert

# bastion — single-service appliance (drops <service>)
prod.bastion.zelosai.cloud                   # Guacamole UI
prod.bastion.zelosai.cloud/oidc              # local Dex issuer
prod.bastion.zelosai.cloud/verify-auth       # GitHub OAuth callback

# zelosai platform — gateway is the single exposed control-plane service
gateway.prod.zelosai.<domain>                # the ZelosPlatform gateway Ingress host
```

## Why this standard

- **Localisable** — four labels place any hostname (service / instance / product / org) at a
  glance; no per-deployment cheat sheet.
- **Collision-free across instances** — many environments/products coexist without name clashes,
  including a parallel `.local` mirror for LAN troubleshooting.
- **One auth contract** — `/oidc` + `/verify-auth` on the apex means one GitHub OAuth App and
  interchangeable IdPs; clients never learn which IdP is deployed.
- **Real certs, anywhere** — dual wildcards + split-horizon + port-agnostic TLS give valid
  certificates on the WAN, on a non-standard port, on the management port, and on `.local`.
- **Backward compatible** — gated on the `cluster:` dict; existing single-host path-based
  deployments are unaffected.
