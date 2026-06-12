# 20 — The buildable-repo contract (`.zelos.yaml`)

> Status: **v0.4.9 target** (gates v0.5.0, the POC-onboarding milestone).
> Companions: [15-ansible-collection-conventions](15-ansible-collection-conventions.md)
> (the collection-side standards), [18-organization-model](18-organization-model.md)
> (the tenancy level this contract consumes), [19-operator-and-cli](19-operator-and-cli.md).

## Why

zelos.foundry builds, tests, and delivers component repos. Pre-v0.4.9 each repo
required a hand-written WorkflowTemplate inside foundry
(`roles/cicd/workflowtemplates/files/<repo>.yaml`) — fine for ten first-party
repos, wrong for the destination: **onboarding repos foundry does not own**
(company POC projects and beyond). The fix is a thin **repo-side contract**: a
repo declares *what to build, how to test, where to deliver, what it needs* in
one manifest, and foundry runs ONE generic pipeline over it. Foundry-side
onboarding collapses to a registration entry in the environment inventory plus
a **tenancy**.

A manifest, not a directory convention: external repos can adopt a single file;
they cannot adopt a prescribed tree.

## The manifest — `.zelos.yaml`, schema v1

Lives at the repo root. Versioned from day one; foundry validates and fails
actionably on unknown schema versions.

```yaml
schema: v1
name: someapp                       # the project identity (and default tenancy name)

build:
  images:                           # one entry per container artifact
    - name: someapp                 # -> <registry>/<registry_project>/someapp
      context: .
      dockerfile: Dockerfile
      platforms: [amd64]            # arm64 opt-in per image

test:
  unit: "make test"                 # run containerized + unprivileged; non-zero fails the build
  integration:                      # OPTIONAL: how the CUT joins the bed integration suite
    cut: helm                       # helm | manifests
    path: deploy/chart              # what the suite deploys as the component-under-test

deliver:
  registry_project: someapp         # Harbor project (per-tenancy robot scopes to it)
  chart: deploy/chart               # OPTIONAL: chart/manifest delivery path
  versioning: tag                   # tag | sha | develop-floating

secrets: [SOMEAPP_API_KEY]          # NAMES ONLY — values resolve from the environment
                                    # inventory at run time; a manifest never carries values
```

Rules:

- **Names only for secrets** — the environment's sealed inventory maps them
  (doc 18 registry discipline). A manifest carrying a value fails validation.
- Everything beyond `schema`/`name`/`build` is optional — absent ⇒ feature off
  (the §5 presence convention, applied to repos).
- The manifest is the repo's "one inventory dict" (§5): foundry reads it
  AFTER clone, inside the pipeline — the registration entry (below) never
  duplicates its contents.

## The two tiers

| tier | who | definition lives | notes |
|---|---|---|---|
| **generic** | default; the ONLY tier for external repos | `.zelos.yaml` in-repo + `zelos-build-generic` in foundry | clone → validate manifest → build matrix → unit test → publish → optional integration trigger |
| **bespoke** | first-party exceptions only | a named WorkflowTemplate in foundry (`roles/cicd/workflowtemplates/files/`) | the integration suite, allure render, anything the manifest can't express |

**External repos never supply executable pipeline definitions.** An in-repo
workflow fragment is code running in the bed; that privilege is first-party
only, and even then the preference is extending the generic schema over adding
fragments.

## Registration — onboarding as inventory

A repo joins an environment via the **`cicd_component_repos[]`** list in
`inventory/<env>/<env>.config.yml` (the canonical name — bare `repos` is too
generic, and `registry.*` is doc-18 reserved):

```yaml
cicd_component_repos:
  - name: someapp                    # opaque project name (never a zelos.* FQCN)
    url: https://github.com/SomeOrg/someapp
    ref: develop                     # the watched branch
    tier: generic                    # generic | bespoke:<template-name>
    tenancy: someapp                 # doc-18 tenancy facet (defaults to name)
    registry_project: someapp        # Harbor project (defaults to name)
```

The entry carries **identity + tenancy only** — build/test/deliver details live
exclusively in the repo's `.zelos.yaml` (read at build time; never duplicated
into inventory). This ONE list drives: the Argo Events EventSource/Sensor
bindings (push → build), webhook registration (`register_webhooks`), and the
tenancy provisioning below. Onboarding = manifest in the repo + one
registration entry + `make seal` for its secrets. Nothing else.

**Migration note (v0.4.9):** the pre-contract names
`argo.events.component_repos` and `workflowtemplates.component_repos` are live
in real inventories — the foundry roles consume `cicd_component_repos` with
alias-first deep-default fallbacks to them; the fallbacks drop in the campaign's
final cleanup slice.

## Tenancy + trust (the external-repo model)

One onboarded project = one **tenancy** (doc 18):

- a namespace (+ ResourceQuota + NetworkPolicy) for its builds/deploys
- an OIDC group for its humans (Dex/Rancher per the environment's IdP)
- a Harbor **project** with a per-project robot — no shared push credentials
- a per-repo **webhook secret**
- builds run **unprivileged** (rootless image builds; no host mounts; no
  cluster credentials beyond the tenancy's own)

First-party repos ride the same mechanics with relaxed posture where the
environment chooses. The trust boundary is the tier system plus the tenancy
isolation — not review of external pipeline code, because there is none.

## Validation gates (per migrated repo)

push → webhook → Sensor → `zelos-build-generic` → Harbor artifact in the
tenancy's project (+ integration trigger where wired), green on the alpha
environment, BEFORE the repo's bespoke template is deleted.

## Boundaries

- GitHub Actions remains each repo's fast local unit gate; foundry owns
  build + integration + delivery. The manifest describes the foundry side only.
- This doc defines the contract; the generic WorkflowTemplate, registration
  plumbing, and tenancy wiring are zelos.foundry implementation (v0.4.9 issues).
