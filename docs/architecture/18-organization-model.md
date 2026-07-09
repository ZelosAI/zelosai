# 18 — Organization model & variable registry

The suite-wide identity model — **`<tenancy>.<environment>.<product>.<domain>`** — and the
canonical variable registry that expresses it in Ansible. This doc is the contract;
[15-ansible-collection-conventions.md](15-ansible-collection-conventions.md) (§6) binds roles to
it and [16-dns-and-hostname-standard.md](16-dns-and-hostname-standard.md) carries the DNS/TLS
form. It replaces the foundry-specific term **"bed"**: a foundry bed is simply *a foundry
environment* (the word may survive colloquially; variables and docs use `environment`).

## The identity tuple

```
<tenancy>.<environment>.<product>.<domain>
   │           │            │         └─ registered DNS root (zelosai.cloud; customers use their own)
   │           │            └─ the Zelos product (foundry, zelosai, bastion, …) — ALWAYS present
   │           └─ one full deployment instance (alpha, develop, production, …)
   └─ a logical container INSIDE an environment (optional level)
```

| Level | What it is | Maps to |
|---|---|---|
| **Product** | a deployable Zelos offering | its own k8s cluster **or** registration with an existing shared cluster; a GitHub/Okta OAuth app family |
| **Environment** | one fully deployed instance of a product (alpha, develop, production) | a Rancher **Project** on its hosting cluster; one Dex instance (GitHub or Okta connector); one inventory directory; one secrets file; VMs/LXC/bare-metal compute can also be associated with it |
| **Tenancy** | a logical container inside an environment | a Kubernetes **namespace** (+ container/VM/LXC/storage quotas); OIDC groups grant access at tenancy, environment, and product scope; RBAC eventually reaches tenant granularity |

### Decisions (locked, v0.4.8)

- **Tenancy is an optional deeper DNS level.** Environment-level platform services stay at
  `<service>.<environment>.<product>.<domain>` (zero migration of existing names/certs).
  Tenant-exposed ingress gets `<service>.<tenancy>.<environment>.<product>.<domain>` with a
  **per-tenancy wildcard** cert (`*.{tenancy}.{env}.{product}.{root}`) minted on demand — public
  wildcards cover exactly one label. A **reserved-name registry** at the shared label prevents a
  tenancy from claiming a service name (`harbor`, `argocd`, `auth`, `rancher`, `results`,
  `hooks`, `gateway`, `registry`, `api`, `mgmt` are reserved).
- **The product label is always present** in environment FQDNs — uniform shape for parsing,
  certs, and external-dns filters. (Single-service appliances still drop only the *service*
  label per doc 16: `prod.bastion.zelosai.cloud`.)
- **Environment ↔ cluster is an explicit binding, 1:1 in v0.4.8.** Products can build a dedicated
  cluster or register with an existing one; multiple clusters can share one Proxmox host. The
  binding lives in the tuple (`environment.cluster:`), so moving an environment between clusters
  later is a data change — live mobility (storage migration, DNS retarget, OIDC re-registration)
  is a future campaign, not v0.4.8.
- **Auth:** every environment runs a Dex (or Rancher) IdP with a GitHub **or Okta** application —
  the connector is inventory-driven. OIDC scopes/groups carry access for tenancies, environments,
  and products.
- **Word forms:** `environment` alone is an **Ansible-reserved variable name** — the inventory
  dict is therefore `environment_identity:` and derived facts use the full-word forms below. *Tenancy* is the level; a *tenant* is an instance of it.

## The inventory shape

Set once per environment (in `inventory/<env>/<env>.config.yml`, format 2 — the identity tuple
lives in the `common:` collection dict):

```yaml
zelos_inventory_format: 2     # the format-2 marker — every orchestration playbook asserts it
common:
  # NB: `environment` alone is an Ansible-RESERVED name — the tuple is environment_identity.
  environment_identity:
    name: alpha                 # ^[a-z0-9-]+$
    product: foundry
    root_domain: zelosai.cloud  # unset ⇒ legacy single-host path-based routing
    public_port: 9443           # optional; unset/443 ⇒ no suffix
    cluster:                    # explicit binding (1:1 in v0.4.8)
      kind: k3s                 # k3s | k8s | external
      # …provider-specific binding keys
    tenancies: []               # optional; [{name, namespace, quotas, oidc_group}, …]
```

## Derived facts — `zelos.common.environment_facts`

Derived **exactly once** per run (tags `[always]`, included at the top of orchestration
playbooks). Roles consume the facts and assert presence in `assert.yml`; they never re-derive.

| Fact | Derivation | Example |
|---|---|---|
| `environment_name` | `environment.name` | `alpha` |
| `environment_product` | `environment.product` | `foundry` |
| `environment_domain` | `<name>.<product>.<root_domain>` | `alpha.foundry.zelosai.cloud` |
| `environment_local_domain` | `<name>.<product>.local` | `alpha.foundry.local` |
| `public_port_suffix` | `:<public_port>` or `''` | `:9443` |
| `auth_fqdn` | `auth.<environment_domain>` | `auth.alpha.foundry.zelosai.cloud` |
| `oidc_issuer_url`, `oidc_scopes`, `oidc_groups_claim`, `oidc_admin_group` | per the doc-16 parity layer | — |
| `bed_name`, `bed_domain`, `bed_local_domain` | **deprecated aliases** of the above, emitted for one phase | removed in v0.4.8 Phase 6 |

Suite-level assertions carried by `environment_facts`:

- `environment_name` matches `^[a-z0-9-]+$`.
- **Single public-port source**: `environment.public_port` is the only public port in rendered
  output; `istio.listener_port` / `istio.mgmt_gateway.listener_port` are internal Service
  targetPorts only; equality is asserted if a role sets both.
- `gcp.project` equality across consumers (cert_manager, external_dns, clouddns) when more than
  one sets it explicitly.

## Variable registry — renames (old → new)

Alias-first staging: consumers accept both for one phase; producers rename second; fallbacks
removed in Phase 6. Each rename lands via its v0.4.8 issue.

| Old | New | Owner |
|---|---|---|
| `cluster: {root_domain, product, bed, public_port}` | `environment_identity: {name, product, root_domain, public_port, cluster:{}}` | all |
| `bed_name` / `bed_domain` / `bed_local_domain` | `environment_name` / `environment_domain` / `environment_local_domain` | zelos.common.environment_facts |
| duplicated `set_fact` identity blocks (platform.yml / deploy.yml) | `include_role: zelos.common.environment_facts` | zelos.kubernetes, zelos.foundry |
| `(istio.gateway).public_port` fallback chains | `environment.public_port` fact | zelos.kubernetes |
| `ceph_cluster_id` | `pve_ceph_cluster_id` | zelos.proxmox secrets → zelos.kubernetes rook_ceph |
| `ceph_rbd_keyring` / `ceph_cephfs_keyring` / `ceph_csi_*_keyring` / `ceph_healthchecker_keyring` / `ceph_mon_data` / `ceph_monitoring_endpoint` | same names with `pve_ceph_` prefix | zelos.proxmox secrets → consumers |
| `CEPH_CLUSTER_ID_<BED>` / `HARBOR_ADMIN_PASSWORD_<BED>` / `MINIO_ROOT_*_<BED>` env shapes | retired — vaulted per-env keys (`harbor_admin_password`, `minio_root_password`, …); the env dimension is the inventory dir | zelos.kubernetes, zelos.foundry |
| `cert_manager.acme.dns01.gcp_project` + `external_dns.gcp_project` (independent) | both default from shared `gcp: {project}` (+ equality assert) | zelos.kubernetes, zelos.common |
| bare bastion secrets (`github_oauth_client_id`, `postgres_password`, …) in standalone mode | `bastion_`-prefixed in **both** modes | zelos.bastion |
| `bastion_bed:` scoped dict (bed mode) + legacy top-level dicts (standalone) | one `bastion:` interface dict | zelos.bastion |
| **format 2 (HARD CUTOVER — no alias phase; `zelosctl inventory migrate` converts):** | | |
| every flat top-level role dict | nested under its owning collection root: `common:` / `proxmox:` / `kubernetes:` / `foundry:` / `bastion:` (ownership = the collection whose role consumes it; the authoritative map is `zelos_common.inventory_migrate.CONFIG_MAP`) | all |
| `proxmox: {api_host, api_user, node, ssh_user, ssh_key_path, verify_tls}` (the PVE conn dict) | `proxmox.api` (bare name collided with the collection root) | zelos.proxmox → all consumers |
| `kubernetes: {version, cluster_endpoint, *_cidr, disable, tls_sans}` (the cluster dict) | `kubernetes.cluster` (bare name collided with the collection root) | zelos.kubernetes |
| flat secret keys | grouped into `<collection>_secrets:` sections, key NAMES unchanged (distinct roots — same-named top-level keys across inventory sources clobber) | all |
| (absent) | `zelos_inventory_format: 2` marker, asserted by every orchestration playbook | zelos.common.environment_facts |

## Per-environment secrets key inventory

The single `inventory/<env>/<env>.secrets.yml` carries (inline-`!vault`, §18 of doc 15), each
key in its owning collection's `<collection>_secrets:` section (names unchanged — the section
is pure grouping; the authoritative map is `zelos_common.inventory_migrate.SECRETS_MAP` +
prefix rules):

- **`common_secrets`** (environment-level identity/creds): `github_oauth_client_id/_secret`,
  `gcp_dns_sa_json`, `gts_eab_key_id`, `gts_eab_hmac`, `gts_eab_minted_at`, Okta client
  credentials where used.
- **`proxmox_secrets`** (infra-minted by the host roles): `proxmox_api_token_id/_secret`, the
  `pve_ceph_*` family, `zelosadmin_password`, `zelosadmin_ssh_private_key`, `idrac_username`,
  `idrac_password`, `idrac_ssh_password`.
- **`kubernetes_secrets`** (cluster + foundation harvest): `kubeconfig_content`,
  `dex_*_client_secret`, `oauth2_proxy_cookie_secret`, `rancher_bootstrap_password`.
- **`foundry_secrets`** (product-layer): `harbor_admin_password`, `argocd_admin_password`,
  `minio_root_password`, `ghcr_push_token`, `gh_token`, `zelos_suite_git_token`,
  `zelos_mcp_fernet_key`.
- **`bastion_secrets`** (its own instance): `bastion_github_oauth_client_id/_secret` (+ `_alt`),
  `bastion_postgres_password`, `bastion_gts_eab_key_id/_hmac`, `bastion_gcp_dns_sa_json`
  (falls back to the shared `common_secrets.gcp_dns_sa_json`),
  `bastion_sync_proxmox_token_id/_secret` (falls back to
  `proxmox_secrets.proxmox_api_token_id/_secret`), `bastion_ssh_private_key`,
  `bastion_kubernetes_client_cert/_key/_ca_cert`, `bastion_cas_ca_chain`,
  `bastion_alt_tls_cert/_key`, `bastion_github_user_sync_token`,
  `bastion_workstation_user_secret`, `bastion_console_oauth2_client_secret/_cookie_secret`.

## Reserved names

No role's interface dict may shadow the format-2 top-level namespace: the collection roots
(`common`, `proxmox`, `kubernetes`, `foundry`, `bastion`), the secrets sections
(`*_secrets`), `zelos_inventory_format`, and the derived-fact space: `environment*`,
`tenancy*`, `tenancies`, `product`, `root_domain`, `public_port*`, `auth_fqdn`, `oidc_*`,
`bed_*` (retired alias space). Inside `common:`, `gcp` keeps its reserved meaning (the shared
GCP project dict).

## Worked examples

```
# foundry product, alpha environment, platform service (no tenancy level)
harbor.alpha.foundry.zelosai.cloud:9443

# zelosai tenancy inside the alpha foundry environment (tenant-exposed ingress)
app.zelosai.alpha.foundry.zelosai.cloud:9443     # covered by *.zelosai.alpha.foundry.zelosai.cloud

# zelosai product, develop / production environments (product label always present)
gateway.develop.zelosai.zelosai.cloud
gateway.production.zelosai.zelosai.cloud

# bastion appliance (single-service: drops only the <service> label, per doc 16)
prod.bastion.zelosai.cloud
```

### Registered (v0.4.9): the CI registration list

`cicd_component_repos[]` — the per-environment list of repos foundry builds
(doc 20): items `{name, url, ref, tier, tenancy, registry_project}`. `name` is
an opaque project label (never a `zelos.*` FQCN); `tenancy` defaults to `name`.
Deprecated aliases (alias-first, removed at the v0.4.9 cleanup):
`argo.events.component_repos`, `workflowtemplates.component_repos`.
