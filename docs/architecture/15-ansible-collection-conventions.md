# 15 — Ansible Collection Conventions

Canonical standard for **all `zelos.*` Ansible collections** — `zelos.common`, `zelos.proxmox`,
`zelos.kubernetes`, `zelos.foundry`, `zelos.bastion`, `zelos.dgx`. New collections are built to
this doc; existing ones converge on it (§20). Each collection ships a short
`CONTRIBUTING-ansible.md` that points back here.

The standard blends two proven collection styles:

- **Large-infrastructure profile** (modeled on the in-house AAL / padre collections): layered
  roles, verb dispatchers, a single inventory-facing dict per role, comprehensive sample
  inventories, thin orchestration playbooks.
- **Module-driven profile** (modeled on the jamf_pro / cerberus collections): custom modules with
  shared `module_utils` API clients, `*_info` read-only modules, paired integration-test targets.

A collection mixes both as needed — `zelos.common` carries the suite's custom plugins; the
infrastructure collections are role-first.

> Companion docs: [05-gitflow.md](05-gitflow.md) (branching/promotion),
> [06-naming-conventions.md](06-naming-conventions.md),
> [16-dns-and-hostname-standard.md](16-dns-and-hostname-standard.md),
> [18-organization-model.md](18-organization-model.md) (identity tuple + variable registry).

**The six non-negotiables**

1. Roles are standard-shaped and **importable by other projects** — nothing but inventory vars
   needed to `include_role` them (§1, §3).
2. Every role exposes **one inventory-facing dict** with clear defaults (§5).
3. Task lists are **verb-based** (`assert / deploy / destroy / info`) with `subtasks/` for
   processes (§3, §4).
4. Projects are **driven from inventory files** — static inventory, dynamic-inventory plugins,
   and vaulted secret dictionaries. Environment variables are sparing, documented, targeted
   overrides — never the driving mechanism (§8, §9).
5. Configuration is rendered from **Jinja2 templates**, never large inline text blocks (§10).
6. Tasks use **stable community collections**, not shell scripts — shell only where no module
   exists, wrapped and registered as an exception (§11).

## 1. Namespacing & FQCN

- Namespace is always **`zelos`**; the collection `name` is the domain (`common`, `proxmox`,
  `kubernetes`, `foundry`, `bastion`, `dgx`). Roles are referenced by **fully-qualified name**:
  `zelos.<name>.<layer>.<role>` (no layer for cross-cutting roles like `zelos.common.secrets`).
- One collection per **reusable concern**. Split by reuse boundary *and* churn surface (a fast-
  churning app layer should not live in a stable reusable layer).
- Collections depend on each other only through `requirements.yml` + FQCN, never via local paths.
- A role MUST be runnable by any consumer that installed the collection, with **nothing but
  inventory variables** — no reliance on the home repo's `ansible.cfg`, scripts, Makefile, or
  controller environment state.

## 2. Repository layout

```
zelos.<name>/
├── galaxy.yml            # ns/name/version/deps/build_ignore (§14)
├── meta/runtime.yml      # requires_ansible
├── ansible.cfg           # §13
├── requirements.yml      # collection deps (galaxy + git), pinned
├── .ansible-lint  .yamllint            # lint as config-as-code (§15)
├── README.md  LICENSE (Apache-2.0)  VERSION  CLAUDE.md  CONTRIBUTING-ansible.md
├── .github/              # suite gitflow + tracker workflows (§15)
├── roles/<layer>/<role>/ # §3 — layer subdirs group roles by responsibility
├── playbooks/            # broad verb orchestration + subtasks/ (§7)
├── inventory/            # per-environment dirs + committed sample (§8)
├── plugins/              # custom modules / module_utils / filters (§12) — when needed
├── tests/integration/targets/<module>/   # when plugins/ exists (§12)
├── src/zelos_<name>_cli/ + pyproject.toml  # the collection's CLI plugin package (§19) — when needed
└── docs/                 # §16
```

**Layers** are domain-specific subdirectories under `roles/` that group roles by responsibility
and deployment order. Examples: `zelos.proxmox` → `host` / `images`; `zelos.kubernetes` →
`provision` / `cluster` / `foundation` / `storage`; `zelos.foundry` → `cicd` / `registry` /
`results` / `routing` / `sim`; `zelos.bastion` → `vm` / `gateway` / `workstation`;
`zelos.common` → `google` (+ the unlayered `secrets`, `environment_facts`).

## 3. Role anatomy (every role)

```
roles/<layer>/<role>/
├── defaults/main.yml     # the role's variable surface — one inventory dict → computed vars (§5)
├── vars/main.yml         # non-overridable constants (chart repo URLs, pinned versions) — only when needed
├── handlers/main.yml
├── tasks/
│   ├── main.yml          # thin VERB DISPATCHER (§4)
│   ├── assert.yml        # read-only preflight: validate every consumed input (always present)
│   ├── deploy.yml  destroy.yml  info.yml   # standard verbs (+ build.yml / install.yml as needed)
│   └── subtasks/         # reusable subroutines included by the verb files
├── templates/{helm,k8s}/ # *.j2 grouped by technology (§10)
├── files/                # static assets (when needed)
├── tests/{inventory,test.yml}
└── README.md             # purpose · variables table · secrets consumed · env overrides · example · tags
```

Rules:

- `tasks/` contains **only** `main.yml`, the verb files, and `subtasks/` — no stray task files.
  Multi-step processes and per-item loop bodies live in `tasks/subtasks/<process>.yml` so the
  verb files stay linear and readable (model:
  `zelos.kubernetes.provision.proxmox` → `subtasks/{clone-lxc,wait-lxc,destroy-lxc}.yml`).
- **`assert.yml` is a standard verb**: read-only, asserts every input the role consumes —
  including the reserved suite facts (§6) — with actionable `fail_msg` naming the inventory
  file/key to fix, `quiet: true`, and `no_log` on secret comparisons. It runs first from the
  verb files (or the dispatcher's `[always]` preflight) and is callable standalone via
  `tasks_from: assert.yml`.
- `vars/main.yml` holds only non-overridable constants — never tunables — and starts with the
  comment `# Internal constants — do not override in inventory`.

## 4. Verb dispatcher — `tasks/main.yml`

`main.yml` does no work; it lists **every verb the role implements**, gated and tagged
consistently. Standard verbs: **`deploy` / `destroy` / `info`** (+ `build` / `install` where a
role needs them). `info` is read-only status. Destructive verbs carry `never` so they cannot run
un-asked. Playbooks may also drive verbs directly with `tasks_from:` — both entry points are
sanctioned; `tasks_from` is the primary mechanism in orchestration playbooks.

```yaml
---
# tasks file for <role>
- name: <role> preflight
  ansible.builtin.include_tasks: assert.yml
  tags: [<layer>, <role>, always]

- name: Deploy <role>
  ansible.builtin.include_tasks: deploy.yml
  when: <role>_config.enabled
  tags: [<layer>, <role>, deploy]

- name: <role> status
  ansible.builtin.include_tasks: info.yml
  tags: [<layer>, <role>, info]

- name: Destroy <role>
  ansible.builtin.include_tasks: destroy.yml
  tags: [never, <layer>, <role>, destroy]
```

- `destroy.yml` preserves data-bearing resources (PVCs, CRDs, sealed secrets) unless an explicit
  `<role>_destroy_data` flag is passed — destruction of state is always a second, deliberate step.

## 5. Variable convention (the load-bearing rule)

**Every role exposes exactly one inventory-facing dict — `<role>: {}` — and nothing else.**
`defaults/main.yml` maps that dict into computed internal vars using deep `default()` chains, so
operators set one well-named block and the role fills in every default.

```yaml
# roles/foundation/istio/defaults/main.yml
---
istio: {}                                              # the ONLY inventory-facing var
k8s_namespace: "{{ istio.namespace | default('istio-system') }}"
istio_config:
  enabled:      "{{ istio.enabled      | default(true) }}"
  helm_version: "{{ istio.helm_version | default('1.28.1') }}"
  node_selector: "{{ istio.node_selector | default(omit) }}"   # optional → omit, not ''
gateway_api:
  enabled: "{{ (istio.gateway_api | default({})).enabled | default(true) }}"
  version: "{{ (istio.gateway_api | default({})).version | default('v1.4.1') }}"
```

Rules:

- Never read bare top-level vars inside tasks/templates — read the computed `<role>_*` vars.
- Nested optionals use the `(x | default({})).y | default(...)` idiom so a partial dict never errors.
- **Optional values default to `omit`, not `''`** — `default(omit)` keeps rendered YAML/JSON
  clean and makes "unset" testable. Reserve `''` for values where empty-string is meaningful.
- **Activation is presence-driven**: the existence of the role's dict (or list/field) in the
  inventory — refined by an `enabled` flag inside it — determines whether the role applies:
  `when: <role> is defined and <role> is mapping and (<role>.enabled | default(true))`.
- **Resolution order** for any input: *env override* (sparing + documented, §9) → *role dict
  key* → *vaulted/harvested inventory var* → *default / omit*. Every chain MUST terminate in an
  inventory var or default — never require an environment variable.
- Any **derived name** (Kubernetes Secret names, secrets-file keys, container env-var names) is
  computed once in `defaults/main.yml` — never assembled inline in tasks.
- Cross-role values pass explicitly (registered facts / `set_fact` / vars on `include_role`) —
  never by reaching into another role's internals.
- The full surface (every var + its default) is documented once in the sample environment (§8)
  and in the role README's variables table.

## 6. Organization model & the suite variable contract

The suite-wide identity model is `<tenancy>.<environment>.<product>.<domain>` — defined in
[18-organization-model.md](18-organization-model.md) (semantics + the full variable registry)
and [16-dns-and-hostname-standard.md](16-dns-and-hostname-standard.md) (DNS/TLS form). What
roles need to know:

- The **identity tuple** is set once per environment in inventory:
  `environment_identity: {name, product, root_domain, cluster: {...}, public_port}` (+ optional
  `tenancies:`) — `environment` alone is an Ansible-reserved word, hence the `_identity` suffix;
  derived facts use full-word names: `environment_name`, `environment_domain`, `environment_local_domain`,
  `public_port_suffix`, `auth_fqdn`, the `oidc_*` family.
- Facts are **derived exactly once** by `zelos.common.environment_facts` (tags `[always]`,
  included at the top of orchestration playbooks). Roles never re-derive them — they consume the
  facts and assert their presence in `assert.yml`.
- **Reserved names**: no role dict may shadow `environment*`, `tenancy*`, `product`,
  `root_domain`, `public_port*`, `auth_fqdn`, `oidc_*`, `gcp` — these belong to the suite
  contract (registry in doc 18).
- **Harvested-secret prefix rule**: keys written to an environment's secrets file are prefixed
  with the **full producing role name** — `pve_ceph_rbd_keyring`, not `ceph_rbd_keyring`;
  `bastion_github_oauth_client_id`, not `github_oauth_client_id` (when bastion-scoped). Consumer
  roles reference these names in their default chains.
- Cross-role consistency is **asserted**, not assumed: `environment_facts` carries the
  suite-level assertions (single public-port source, `gcp.project` equality across consumers,
  `environment_name` shape `^[a-z0-9-]+$`).

## 7. Playbooks & orchestration

Root `playbooks/` contains **broad verb-based playbooks for coordinating large-scale
provisioning only** — `site.yml` plus per-phase verbs (`provision` / `cluster` / `platform` /
`deploy` / `destroy` / `info` / `host_prep` / `build_images`). Anything that is a **reusable
process** is not a root playbook: it lives in `playbooks/subtasks/<process>.yml` (task-file
form, imported with `include_tasks` / `import_playbook` as appropriate).

- The secrets lifecycle is the canonical example: there are **no** `save_secrets.yml` /
  `seal_secrets.yml` / `unseal_secrets.yml` root playbooks. One `playbooks/subtasks/secrets.yml`,
  parameterized by `secrets_action: save|seal|unseal`, calls `zelos.common.secrets` with
  `tasks_from` (§18).
- A playbook is a localhost orchestration play that includes roles in **dependency order**, each
  gated + tagged. **Every dynamic `include_role` carries `apply: {tags: [...]}`** alongside its
  own `tags:` — without it, `--tags <role>` runs only the `[always]` preflight and silently
  skips the verb.

```yaml
# playbooks/platform.yml (excerpt)
- name: Deploy platform
  hosts: localhost
  gather_facts: true
  tasks:
    - name: Derive environment identity facts
      ansible.builtin.include_role:
        name: zelos.common.environment_facts
        apply: { tags: [always] }
      tags: [always]

    - name: Deploy cert-manager
      ansible.builtin.include_role:
        name: zelos.kubernetes.foundation.cert_manager
        tasks_from: deploy.yml
        apply: { tags: [foundation, cert-manager] }
      when: cert_manager is defined and cert_manager is mapping and (cert_manager.enabled | default(false))
      tags: [foundation, cert-manager]
```

- Destroy = reverse dependency order, and destructive playbooks gate on an explicit
  confirmation: `-e confirm_destroy=<environment_name>`, asserted against the inventory's
  `environment_name` before any teardown task runs — plus the `[never, destroy]` tags (§4).
- Prerequisite/config-load tasks: `tags: [always]`. Bulk fast-paths: `tags: [never, <name>]`.
- Playbooks take **no** `-e @<file>` configuration. Everything comes from `-i inventory/<env>`.
  The only sanctioned `-e` uses are ad-hoc verb flags (`confirm_destroy`, `<role>_destroy_data`,
  `secrets_action`).

## 8. Inventory: per-environment directories

Each deployment environment is a directory — the **single entry point** for everything:

```
inventory/
├── <env>/                      # one directory per environment (gitignored except sample)
│   ├── <env>.config.yml        # all environment vars (identity tuple + role dicts)
│   ├── <env>.secrets.yml       # THE secrets file — every value an inline `!vault` block
│   ├── <env>.proxmox.yml       # dynamic inventory (community.proxmox) — discovery by tag
│   └── <env>.pve.yml           # static host entries (PVE host / connection setup)
└── sample/                     # committed; documents EVERY role's full surface with defaults
    └── sample.{config,secrets,proxmox,pve}.yml
```

- Invoke with `ansible-playbook -i inventory/<env> playbooks/site.yml` — Ansible merges every
  file in the directory; `all: vars:` accumulate.
- **Dynamic inventory is preferred** where the platform supports it: `community.proxmox.proxmox`
  discovers VMs + LXC and groups by tag (LXC nodes get `proxmox_pct_remote`). Static files cover
  what discovery can't.
- **One vaulted secrets file per environment** (`<env>.secrets.yml`): the single home for every
  secret the environment needs — operator-provided and harvested (§18). Values are
  inline-`!vault` encrypted (mergeable, diffable, safe to commit if an operator opts in).
- The committed **sample environment** documents every variable for every role with its default —
  operators copy the directory, customize, and flip role dicts on. Real environment dirs are
  gitignored.
- The vault password resolves via `ANSIBLE_VAULT_PASSWORD_FILE` / `scripts/vault-pass-client.sh`
  — the **bootstrap exception** to §9 (you cannot vault the vault key).

## 9. Configuration precedence — inventory-primary

All configuration and secrets reach Ansible through the environment's inventory directory. This
is what makes an environment **reproducible** (fully described by one directory), roles
**importable** (consumers aren't asked to export N variables), and runs **container-safe** (§19).

- `lookup('env', ...)` is allowed **sparingly**, as a deliberate highest-precedence *override*
  of a specific variable at runtime — first in the chain, falling through to the inventory:

  ```yaml
  # documented override: CI smoke runs flip the registry without editing inventory
  harbor_external_url: "{{ lookup('env', 'HARBOR_EXTERNAL_URL') | default(harbor.external_url | default(harbor_default_url), true) }}"
  ```

- Every env override MUST be listed in the role README's **"env overrides"** row. A chain that
  *requires* an env var to function is a defect.
- Per-environment env-var name shapes (`<KEY>_<ENV>` suffixing) are retired — the per-environment
  dimension is the inventory directory itself.
- Custom modules use **no `env_fallback`** in argument specs — auth/config arrive as module args
  fed from inventory vars (§12).
- Sanctioned standing exceptions: the vault password (§8) and `KUBECONFIG` passed through play
  `environment:` for standard tooling.

## 10. Templating discipline

Configuration is **rendered, not concatenated** — in roles *and* in playbooks:

- Any multi-line file content MUST be a `templates/*.j2` rendered via `ansible.builtin.template`
  (or `lookup('template', ...)` for in-memory YAML such as Helm values). Static content belongs
  in `files/`.
- Large variable sets load from `vars_files` / `include_vars` out of the role/playbook/collection
  directories — not as giant inline `vars:` blocks in a play.
- **Banned:** large `copy: content: |` blobs and configs assembled from `blockinfile`/`lineinfile`
  fragments. (The pre-v0.4.8 `zelos.bastion/playbooks/devvm.yml` — 10+ inline content blocks — is
  the canonical negative example; unreadable by a human.)
- Allowed: trivial 1–3-line `content:` (marker files, single-value confs); `lineinfile` for
  single-line edits of foreign files; `blockinfile` ONLY for a marker-managed stanza in a file
  owned by an external system (e.g. PVE-managed `/etc/network/interfaces`) — and the block body
  comes from `lookup('template', ...)`, not inline YAML.
- Templates carry a `# {{ ansible_managed }}` header and group by technology subdir
  (`templates/{helm,k8s}/...`).

## 11. Module discipline — collections over shell

Use the stable community collection for the job, pinned in `requirements.yml`:

| Domain | Collection |
|---|---|
| Kubernetes / Helm | `kubernetes.core` |
| Proxmox API | `community.proxmox` |
| Docker / Compose | `community.docker` |
| Google Cloud | `google.cloud` |
| OS / misc | `community.general`, `ansible.posix` |

- `ansible.builtin.shell` / `command` ONLY where no module exists — and then wrapped:
  `changed_when` (or `creates`/`removes`) + `failed_when` where rc isn't truth, plus a comment
  **naming the gap**: `# no community.proxmox module for qm importdisk — registered exception (§11)`.
- Read-only probes in `info.yml`/`assert.yml` may use `command` freely but MUST set
  `changed_when: false`.
- **Registered exceptions** (the suite allowlist; additions require a PR to this doc):
  `qm create` / `qm importdisk` from a qcow2 cloud image; `pct create` + raw `lxc.*` config
  appends (k3s-in-LXC requirements); `qm set --hostpci0` (GPU passthrough — API tokens can't);
  `pveceph` / `ceph` CLI (no maintained collection; writing one is out of scope); `pct exec`
  reachability probes (the connection mechanism itself); the k3s install script;
  `ansible-vault encrypt_string` subprocess (secrets role); `timeshift`/`borg` CLIs (zelos.dgx).
- A *recurring* shell exception in a domain with an API is the trigger to write a custom module
  (§12) instead.

## 12. Custom plugins: modules & module_utils

When the suite needs automation no community module provides (e.g. Google Trust Services EAB
minting via `publicca.googleapis.com`), it ships a custom module — `zelos.common` is the
preferred home for cross-collection plugins.

```
plugins/
├── module_utils/<api>/auth.py     # shared session/auth base (token handling, retries,
│                                  #   structured error mapping → actionable messages)
├── module_utils/<api>/<resource>.py   # one client class per resource (jamf_pro pattern)
└── modules/
    ├── <resource>.py              # write module (CRUD; `state:` where applicable)
    └── <resource>_info.py         # read-only module
```

- Every module ships full `DOCUMENTATION` / `EXAMPLES` / `RETURN` blocks.
- `argument_spec` built with a shared auth-spec helper from `module_utils`; secrets marked
  `no_log: True`; **no `env_fallback`** (§9).
- `supports_check_mode=True` everywhere — check mode reports the predicted change without
  calling the API.
- Error handling at the client layer (decorator or base-class hook) converts API failures into
  *what-to-fix* messages — `fail_json(msg=...)`, never raw tracebacks.
- Read-only modules are named `*_info`. Single-use/irreversible server-side actions (e.g. EAB
  keys) expose an explicit `force` and default to no-op when the result already exists.
- Tests: `tests/integration/targets/<module>/` with `module_defaults` wired to inventory vars;
  `ansible-test sanity` joins CI as soon as `plugins/` exists.
- Roles consuming in-collection modules use the FQCN `zelos.<name>.<module>` and keep the §5
  variable convention (module args fed from `<role>_config.*`).

## 13. `ansible.cfg` standard

```ini
[defaults]
collections_path = ~/.ansible/collections:./.ansible/collections
roles_path = roles
host_key_checking = False
retry_files_enabled = False
stdout_callback = ansible.builtin.default
callback_result_format = yaml      # community.general.yaml callback was removed upstream
callbacks_enabled = timer, profile_roles, profile_tasks   # timing visibility on long runs
forks = 20
interpreter_python = auto_silent

[ssh_connection]
pipelining = True
ssh_args = -o ControlMaster=auto -o ControlPersist=300s -o StrictHostKeyChecking=no
```

The file is never environment-specific: no `inventory =` pointing at a real environment, no
vault keys, nothing that breaks importability (§1).

## 14. Collection metadata

- **`galaxy.yml`** — `namespace: zelos`, `name`, semver `version`, `authors`, a real `description`,
  `license: [Apache-2.0]`, `tags`, `dependencies` (other collections), `repository`/`issues`, and a
  `build_ignore` that excludes `.github`, `.gitignore`, `.ansible`, real `inventory/<env>` dirs,
  `secrets`, venvs, `CLAUDE.md`, `VERSION`, and the Python packaging (`src/`, `pyproject.toml`,
  `dist/`).
- **`meta/runtime.yml`** — `requires_ansible: ">=2.16.0"`.
- **`requirements.yml`** — runtime collection deps (Galaxy + git sources, pinned). Appliance and
  shared-concern dependencies (`zelos.common`, `zelos.bastion`) are declared here too (§17).

## 15. Gitflow, tracking & CI

- Follow the suite gitflow ([05-gitflow.md](05-gitflow.md)): `main` protected; `<type>/<N>-<slug>`
  (`^(feature|fix|chore|docs)/[0-9]+-[a-z0-9.-]+$`) → `develop`; `develop`→`main` promotion separate;
  semver tags on `main` only; PR body has `Closes #N`.
- `.github/workflows/`: `add-to-project`, `branch-lint`, `tracker-in-progress`, `tracker-ready-for-qa`
  (target the relevant GitHub Project), plus `unit-tests` (yamllint + ansible-lint +
  `ansible-galaxy collection build`) gating PRs into develop, and `release` (build/publish the
  collection on develop/main + `v*` tags). Branch protection: strict required checks
  `[validate-branch-name, test]`.
- **Lint is config-as-code:** every repo commits the standardized `.ansible-lint` (production
  profile; violations fixed or carried as inline `# noqa <rule>` with a justification comment —
  never silent `skip_list` growth) and `.yamllint`. CI runs them with no inline flags — the
  config files are authoritative. `ansible-test sanity` joins when `plugins/` exists.
- The collection must `ansible-galaxy collection build` clean.

## 16. Documentation

```
docs/
├── index.md  getting-started.md  Architecture.md  Makefile-guide.md
├── layers/<layer>.md
├── roles/<layer>/<role>.md
├── playbooks/<verb>.md
└── inventory/{index.md, sample environment}
```

- **`Architecture.md`** carries a mermaid **layer diagram**, a **component dependency graph**, the
  **deployment-order** phases, and the technology stack.
- Per-role `README.md`: purpose, a variables table (the `<role>` dict keys + defaults), a
  **secrets consumed** row (keys expected in `<env>.secrets.yml`), an **env overrides** row
  (the documented §9 overrides, if any), an example inventory snippet, and the role's tags.
- `docs/inventory/` documents the per-environment directory model + the sample environment.

## 17. Cross-collection dependencies

Two sanctioned patterns — and a default:

- **Provider pattern (default — loose coupling).** Consume a collection's *outputs* (template
  names, harvested secrets, facts, config) rather than importing its roles, so providers stay
  swappable: `zelos.kubernetes.provision.proxmox` consumes `zelos.proxmox`'s template name + API
  creds; a future vmware/eks provider slots in without touching the consumer.
- **Appliance pattern (hard import — when right).** A leaf **appliance collection** with no
  provider alternative (canonical: `zelos.bastion`) exposes roles consumed via FQCN
  `include_role`, behind **exactly one scoped interface dict** (`bastion: {}`) that is the whole
  contract. The dependency is declared in the consumer's `requirements.yml` (NOT `galaxy.yml
  dependencies` — the collection must install without it; dynamic includes resolve only when
  enabled), and the consumer's environment flips `bastion.enabled: true` in inventory.
- **Shared concerns live in `zelos.common`** — secrets machinery, environment facts, Google
  Cloud modules/roles, the CLI framework — rather than being duplicated per collection.
- Decision rule: hard-import only leaf appliances; never hard-import between peer infrastructure
  collections; never reach into another collection's internals in either pattern.

## 18. Secret harvesting (the `zelos.common.secrets` role)

A collection that **mints credentials on a target host** (API tokens, keyrings, …) brings them
**back to the controller** and stores them inline-encrypted in the environment's secrets file —
the ansible-native replacement for shell `emit-secrets`/`.env` flows. The single implementation
lives in `zelos.common`; consumers call it via `playbooks/subtasks/secrets.yml` (§7).

- **Shape:** the cross-cutting **`zelos.common.secrets`** role with verb task files
  (`save.yml` / `seal.yml` / `unseal.yml`) driven by `tasks_from`; callers pass their
  `sources:` spec.
- **Output:** the environment's own `inventory/<env>/<env>.secrets.yml` (§8), **0600**, each
  value an inline `!vault |` block (`ansible-vault encrypt_string --stdin-name`) — live on the
  next run with zero loading machinery.
- **Key names follow the §6 prefix rule** (`pve_ceph_*`, `bastion_*`, …).
- **Source spec** — a `sources:` list of `{name, from, …}` derived (with literal fallbacks) from
  the same inventory dicts the producing roles read. `from`: `file` (raw), `file_b64`
  (binary-safe), `file_grep` (`export KEY='…'`), `shell` (stdout). `optional: true` skips absent
  sources.
- **Harvest path:** gather the value **on the host**, pipe it via **stdin** into
  `encrypt_string` **on the controller** (`no_log` throughout — plaintext never lands on disk; a
  `secrets_debug` toggle lifts it for troubleshooting on a throwaway environment).
- **Merge, never overwrite:** harvested keys **update in place**, operator-**provided** keys are
  **preserved**, new keys appended, header kept (`files/merge_secrets.py`). Operators can
  hand-add secrets and a later harvest won't clobber them.
- **Vault password:** `secrets.vault_password_file` → `$ANSIBLE_VAULT_PASSWORD_FILE` →
  `$ANSIBLE_VAULT_PASSWORD` (staged to a temp 0600 file). Clear `ANSIBLE_VAULT_PASSWORD_FILE`
  for the `encrypt_string` call so the resolved pass-file is the sole default vault-id (else
  `rc 5`). `scripts/vault-pass-client.sh` (canonical copy in `zelos.common`) makes a single env
  var work for both harvest and decrypt.
- **Consume:** nothing to do — it's inventory.

## 19. Container execution & CLI contract

The collections are designed to run **from a container** (operator images, Kubernetes Jobs). A
pulled container must be **self-contained**: collections + CLI preinstalled, a mounted
`/inventory/<env>` + vault key as the only runtime inputs — no repo checkout, no Makefile, no
custom shell scripts at runtime. Make targets and `scripts/` remain for **building** images and
developer convenience only.

The CLI is **plugin-based** (kubectl/git model):

- **`zelosctl`** (ships in `zelos.common`'s `zelos-common` Python package) is the always-present
  wrapper and the operator-image **entrypoint**. It owns the universal commands —
  `env new/list` (scaffold from the sample environment), `secrets seal|unseal|save`, `doctor`,
  `versions` — and presents installed collection plugins as namespaced subcommands in one
  standard taxonomy.
- **Each collection ships a thin CLI plugin package** (`src/zelos_<name>_cli/` → pip package
  `zelos-<name>-cli` → binary `zelos-<name>`), built on the shared framework library (common
  options `--env/--inventory/--vault-pass`, inventory+secrets resolution, ansible-runner
  invocation, output/exit conventions). Its subcommands mirror the collection's playbook verbs:
  `zelos-foundry deploy --env alpha` ≡ `zelosctl foundry deploy --env alpha`.
- Discovery: Python entry-point group `zelos.cli.plugins` (PATH `zelos-*` fallback). Installing a
  collection's plugin into a container both delivers its binary and extends `zelosctl`.
- **Lockstep versioning:** the plugin package is versioned, tested, and released with its
  collection (same repo, same tag).
- The future per-layer **operators** (kopf, v0.6+) reuse the same command implementations —
  install the plugins an operator should handle and it reconciles exactly those infra CRDs
  (CRD specs mirror the §5 role dicts 1:1).

## 20. Convergence checklist (living appendix)

Known non-conformances at the time of the v0.4.8 rewrite — each row is a tracked issue in the
owning repo (Zelos Foundry Tracker, milestone v0.4.8):

| Non-conformance | Owner | Issue |
|---|---|---|
| Playbooks-first layout; devvm.yml inline content blocks (§3, §10) | zelos.bastion | #82 |
| `bastion_bed` dual-mode prelude; unprefixed bastion secrets (§5, §6) | zelos.bastion | #83 |
| Duplicated secrets role + 3 root secrets playbooks (§7, §18) | zelos.proxmox / zelos.kubernetes | #43 / #76 |
| Duplicated identity fact blocks in playbooks (§6) | zelos.kubernetes / zelos.foundry | #76 / #270 |
| `ceph_*` harvested keys unprefixed (§6) | zelos.proxmox (+ rook_ceph consumer) | #43 / #77 |
| Legacy env-file build path (`envs/`, Makefile `$(ENV_FILE)`, emit/publish-secrets) (§8, §9) | zelos.foundry | #269 |
| Env-var chains needing audit → inventory-primary + `default(omit)` (§5, §9) | zelos.kubernetes / zelos.foundry | #78 / #271 |
| `pct`/`qm` status shell probes where modules exist (§11) | zelos.kubernetes / zelos.proxmox | #75 / #42 |
| Large `copy: content:` blobs in pve_* / lxc_template / guest_base roles (§10) | zelos.proxmox / zelos.kubernetes | with #44 / #82-pattern cleanups |
| No `.ansible-lint`/`.yamllint` configs (§15) | all repos | #41 / #74 / #268 / #81 / #45 |
| GCP/GTS integration not yet module/role-shaped (§12) | zelos.common | #1 / #2 |
| Operator images need repo checkout (no zelosctl entrypoint) (§19) | zelos.foundry / zelos.bastion | #273 / #84 |
