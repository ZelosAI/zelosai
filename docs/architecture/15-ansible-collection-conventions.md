# 15 — Ansible Collection Conventions

Canonical standard for **all `zelos.*` Ansible collections** — `zelos.dgx`, `zelos.proxmox`,
`zelos.kubernetes`, `zelos.foundry`. New collections are built to this doc; existing ones
converge on it. Modeled on the in-house "AAL" (`automation_abstraction_infra`) collection
style. Each collection ships a short `CONTRIBUTING-ansible.md` that points back here.

> Companion docs: [05-gitflow.md](05-gitflow.md) (branching/promotion), [06-naming-conventions.md](06-naming-conventions.md).

## 1. Namespacing & FQCN

- Namespace is always **`zelos`**; the collection `name` is the domain (`proxmox`, `kubernetes`,
  `foundry`, `dgx`). Roles are referenced by **fully-qualified name**: `zelos.<name>.<layer>.<role>`.
- One collection per **reusable concern**. Split by reuse boundary *and* churn surface (a fast-
  churning app layer should not live in a stable reusable layer).
- Collections depend on each other only through `requirements.yml` + FQCN, never via local paths.

## 2. Repository layout

```
zelos.<name>/
├── galaxy.yml            # ns/name/version/deps/build_ignore (§9)
├── meta/runtime.yml      # requires_ansible
├── ansible.cfg           # §8
├── requirements.yml      # collection deps (galaxy + git)
├── README.md  LICENSE (Apache-2.0)  VERSION  CLAUDE.md  CONTRIBUTING-ansible.md
├── .github/              # suite gitflow + tracker workflows (§10)
├── roles/<layer>/<role>/ # §3 — layer subdirs group roles by responsibility
├── playbooks/            # one playbook per verb/connection-context (§6)
├── inventory/            # dynamic inventory config + sample (§7)
├── plugins/              # custom modules/filters/connection plugins (when needed)
└── docs/                 # §11
```

**Layers** are domain-specific subdirectories under `roles/` that group roles by responsibility
and deployment order. Examples: `zelos.proxmox` → `host` / `images`; `zelos.kubernetes` →
`provision` / `cluster` / `foundation` / `storage`; `zelos.foundry` → `cicd` / `registry` /
`results` / `routing` / `sim`. (The AAL reference uses `foundation`/`storage`/`platform`/`services`.)

## 3. Role anatomy (every role)

```
roles/<layer>/<role>/
├── defaults/main.yml     # the role's variable surface — one inventory dict → computed vars (§5)
├── vars/main.yml         # non-overridable constants (chart repo URLs, pinned versions) — only when needed
├── handlers/main.yml
├── tasks/
│   ├── main.yml          # thin VERB DISPATCHER (§4)
│   ├── deploy.yml  destroy.yml  info.yml   # standard verbs (+ build.yml / install.yml as needed)
│   └── subtasks/         # reusable subroutines included by the verb files
├── templates/{helm,k8s}/ # *.j2 grouped by technology
├── files/                # static assets (when needed)
├── tests/{inventory,test.yml}
└── README.md             # purpose · variables table · example · tags
```

## 4. Verb dispatcher — `tasks/main.yml`

`main.yml` does no work; it dispatches to verb files, gated on the role's `enabled` flag, with
consistent tags. Standard verbs: **`deploy` / `destroy` / `info`** (add `build` / `install` where
a role needs them). `info` is an always-run, read-only status verb.

```yaml
---
# tasks file for <role>
- name: Deploy <role>
  ansible.builtin.include_tasks: deploy.yml
  when: <role>_config.enabled | default(true)
  tags: [<layer>, <role>, deploy]

- name: <role> status
  ansible.builtin.include_tasks: info.yml
  tags: [<layer>, <role>, info, always]
```

## 5. Variable convention (the load-bearing rule)

**Every role exposes exactly one inventory-facing dict — `<role>: {}` — and nothing else.**
`defaults/main.yml` maps that dict into computed internal vars using deep `default()` chains, so
operators set one well-named block and the role fills in every default. `vars/main.yml` holds only
non-overridable constants.

```yaml
# roles/foundation/istio/defaults/main.yml
---
istio: {}                                              # the ONLY inventory-facing var
k8s_namespace: "{{ istio.namespace | default('istio-system') }}"
istio_config:
  enabled:      "{{ istio.enabled      | default(true) }}"
  helm_version: "{{ istio.helm_version | default('1.28.1') }}"
gateway_api:
  enabled: "{{ (istio.gateway_api | default({})).enabled | default(true) }}"
  version: "{{ (istio.gateway_api | default({})).version | default('v1.4.1') }}"
```

Rules:
- Never read bare top-level vars inside tasks/templates — read the computed `<role>_*` vars.
- Nested optionals use the `(x | default({})).y | default(...)` idiom so a partial dict never errors.
- Cross-role values pass explicitly (registered facts / `set_fact` / vars on `include_role`) — never
  by reaching into another role's internals.
- The full surface (every var + its default) is documented once in `inventory/sample-inventory.yml`.

## 6. Playbooks & orchestration

One playbook per verb (and per connection-context where they differ). A playbook is a localhost
orchestration play that includes roles in **dependency order**, each gated + tagged:

```yaml
# playbooks/deploy.yml
- name: Deploy <collection>
  hosts: localhost
  gather_facts: true
  tasks:
    - name: Deploy cert-manager
      ansible.builtin.include_role:
        name: zelos.kubernetes.foundation.cert_manager
        tasks_from: deploy.yml
      when: cert_manager is defined and cert_manager is mapping and (cert_manager.enabled | default(false))
      tags: [foundation, cert-manager]
    # … later roles depend on earlier ones; destroy.yml runs the reverse order
```

- Double-tag at the play level `[<layer>, <role>]`; the verb tag comes from the role dispatcher.
- Prerequisite/config-load tasks: `tags: [always]`. Bulk fast-paths: `tags: [never, <name>]`.
- Destroy = reverse dependency order.

## 7. Inventory & per-bed config

- Prefer a **dynamic inventory** where the platform supports it (e.g. `community.proxmox.proxmox`
  discovers VMs + LXC and groups by tag; LXC nodes get `proxmox_pct_remote`). Per-bed config file:
  `inventory/<bed>.<name>.yml`.
- Ship **`inventory/sample-inventory.yml`** documenting ALL variables for ALL roles with their
  default values — operators copy + customize and flip `enabled: true` per role.
- Per-bed env files are gitignored except the sample; secrets via Vault / env vars.

## 8. `ansible.cfg` standard

```ini
[defaults]
collections_path = ~/.ansible/collections:./.ansible/collections
roles_path = roles
host_key_checking = False
retry_files_enabled = False
stdout_callback = yaml
callbacks_enabled = timer, profile_roles, profile_tasks   # timing visibility on long runs
jinja2_native = True
forks = 20
interpreter_python = auto_silent

[ssh_connection]
pipelining = True
ssh_args = -o ControlMaster=auto -o ControlPersist=300s -o StrictHostKeyChecking=no
```

## 9. Collection metadata

- **`galaxy.yml`** — `namespace: zelos`, `name`, semver `version`, `authors`, a real `description`,
  `license: [Apache-2.0]`, `tags`, `dependencies` (other collections), `repository`/`issues`, and a
  `build_ignore` that excludes `.github`, `.gitignore`, `.ansible`, `inventory`, `secrets`, venvs,
  `CLAUDE.md`, `VERSION`.
- **`meta/runtime.yml`** — `requires_ansible: ">=2.16.0"`.
- **`requirements.yml`** — runtime collection deps (Galaxy + git sources, pinned).

## 10. Gitflow, tracking & CI

- Follow the suite gitflow ([05-gitflow.md](05-gitflow.md)): `main` protected; `<type>/<N>-<slug>`
  (`^(feature|fix|chore|docs)/[0-9]+-[a-z0-9.-]+$`) → `develop`; `develop`→`main` promotion separate;
  semver tags on `main` only; PR body has `Closes #N`.
- `.github/workflows/`: `add-to-project`, `branch-lint`, `tracker-in-progress`, `tracker-ready-for-qa`
  (target the relevant GitHub Project), plus `unit-tests` (yamllint + ansible-lint + `ansible-galaxy
  collection build`) gating PRs into develop, and `release` (build/publish the collection on
  develop/main + `v*` tags). Branch protection: strict required checks `[validate-branch-name, test]`.
- **Lint:** `ansible-lint` (production profile, strict once roles are real), `yamllint`. The collection
  must `ansible-galaxy collection build` clean.

## 11. Documentation (Ansible + Zelos standards)

```
docs/
├── index.md  getting-started.md  Architecture.md  Makefile-guide.md
├── layers/<layer>.md
├── roles/<layer>/<role>.md
├── playbooks/<verb>.md
└── inventory/{index.md, sample-inventory.yml}
```

- **`Architecture.md`** carries a mermaid **layer diagram**, a **component dependency graph**, the
  **deployment-order** phases, and the technology stack.
- Per-role `README.md`: purpose, a variables table (the `<role>` dict keys + defaults), an example
  inventory snippet, and the role's tags.

## 12. Cross-collection dependencies

- Declare in `requirements.yml` (Galaxy name or `source:` git URL + `version:`), mirroring how
  `zelos.foundry` consumes `zelos.dgx`. Install into the image at build time (public) or via the
  entrypoint (private git).
- Reference other collections by **FQCN** only. Prefer **loose coupling** — consume a collection's
  *outputs* (templates, facts, config) rather than hard-importing its roles, so providers are
  swappable (e.g. `zelos.kubernetes.provision.proxmox` consumes `zelos.proxmox`'s template name +
  API creds rather than depending on the collection).

## 13. Secret harvesting (the `secrets` role)

A collection that **mints credentials on a target host** (API tokens, keyrings, …) must be able to
bring them **back to the controller** and store them inline-encrypted — the ansible-native
replacement for shell `emit-secrets`/`.env` flows.

- **Shape:** a top-level, cross-cutting **`secrets`** role (no provisioning layer; FQCN
  `zelos.<col>.secrets`) + a **`save_secrets`** playbook. Run after the provisioning playbooks.
- **Per-collection, per-bed output:** `secrets/<bed>.<col>.secrets.yaml`, **0600**, with **each
  value an inline `!vault |` block** (`ansible-vault encrypt_string --stdin-name`). Gitignored
  (except an `example.<col>.secrets.yaml`); inline-vault makes them safe to commit if an operator opts in.
- **Source spec** — a `sources:` list of `{name, from, …}` derived (with literal fallbacks) from the
  same inventory dicts the producing roles read. `from`: `file` (raw), `file_b64` (binary-safe),
  `file_grep` (`export KEY='…'`), `shell` (stdout). `optional: true` skips absent sources.
- **Harvest path:** gather the value **on the host**, pipe it via **stdin** into `encrypt_string`
  **on the controller** (`no_log` throughout — plaintext never lands on disk; a `secrets_debug`
  toggle lifts it for troubleshooting on a throwaway bed).
- **Merge, never overwrite:** harvested keys **update in place**, operator-**provided** keys are
  **preserved**, new keys appended, header kept (a small stdlib `files/merge_secrets.py`). Operators
  can hand-add secrets to the file and a later harvest won't clobber them.
- **Vault password:** `secrets.vault_password_file` → `$ANSIBLE_VAULT_PASSWORD_FILE` →
  `$ANSIBLE_VAULT_PASSWORD` (staged to a temp 0600 file). Clear `ANSIBLE_VAULT_PASSWORD_FILE` for the
  `encrypt_string` call so the resolved pass-file is the sole default vault-id (else `rc 5`,
  "vault-ids default,default … specify the vault-id"). Ship `scripts/vault-pass-client.sh` (an
  executable pass-file emitting `$ANSIBLE_VAULT_PASSWORD`) so a single env var works for **both**
  harvest and decrypt — ansible-core has no raw-env *decrypt* config, only the file form.
- **Consume** anywhere with `vars_files:` + a matching vault password.
