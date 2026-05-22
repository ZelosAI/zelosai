# CLAUDE.md

> **Note for Claude sessions:** this file follows the Zelos suite-wide template
> from [docs/template/CLAUDE.md.tmpl](./docs/template/CLAUDE.md.tmpl). The
> canonical gitflow rules every Zelos repo follows live in
> [docs/architecture/05-gitflow.md](./docs/architecture/05-gitflow.md).

## Repository

- **Repo:** `ZelosAI/zelosai`
- **Image:** `ghcr.io/zelosai/zelosai` — the operator manager binary.
- **Purpose:** Architecture hub + **Kubernetes operator** for the Zelos suite.
  Owns the umbrella `ZelosPlatform` CRD + per-component CRDs that manage
  every other component's Deployments / Services / ConfigMaps / Secrets /
  PVCs / HPAs. Also hosts the suite-wide templates and architecture docs
  that every component repo derives from.
- **State:** v0.2.0 — operator + CRDs + minimum-scaled deployment.
  Reconcilers are skeleton-grade (correct object rendering, stubbed
  status/upgrade logic). See [CHANGELOG.md](./CHANGELOG.md).

## Active Branch

- Work on: `claude/bootstrap-suite` (or whichever session branch the harness sets).

## Layout

```
zelosai/
├── README.md                       # suite-wide entry point
├── CLAUDE.md                       # this file
├── LICENSE  CHANGELOG.md  VERSION
├── PROJECT                         # kubebuilder project file
├── go.mod / go.sum  Makefile  Dockerfile
├── cmd/main.go                     # manager bootstrap
├── api/v1alpha1/                   # CRD Go types (one file per Kind)
│   ├── common_types.go             # WorkloadSpec + shared specs
│   ├── zelosplatform_types.go      # umbrella
│   ├── zelos{gateway,backplane,mcp,broker,server,client}_types.go
│   └── zz_generated.deepcopy.go    # controller-gen output
├── internal/controller/
│   ├── zelos*_controller.go        # one reconciler per Kind
│   └── render/                     # shared object builders
├── config/                         # kubebuilder layout
│   ├── crd/bases/                  # generated CRD YAMLs
│   ├── rbac/   manager/   samples/   default/
├── deploy/
│   ├── operator/                   # kubectl apply -k → installs operator
│   └── minimum/                    # kubectl apply -k → installs minimum suite
├── docs/
│   ├── architecture/
│   │   ├── 00-06 (existing suite docs)
│   │   ├── 07-container-contract.md  # env / paths / probes / ports
│   │   ├── 08-crds.md              # CRD field reference
│   │   ├── 09-dependencies.md      # external deps + provisioning
│   │   ├── 10-image-registry.md    # GHCR + pull-secret recipe
│   │   └── 11-telemetry.md         # OpenTelemetry pipeline
│   ├── runbooks/minimum-deployment.md
│   └── template/                   # suite-wide templates (unchanged)
└── .github/workflows/
    ├── lint.yml                    # markdown-lint + link-check
    └── docs.yml                    # mermaid block validation
```

## How to run it / How to build it

```bash
make tidy           # go mod tidy
make build          # build manager binary into bin/manager
make manifests      # regenerate CRD YAML via controller-gen
make generate       # regenerate zz_generated.deepcopy.go
make test           # go test ./...
make image          # docker build
make deploy         # kubectl apply -k deploy/operator/
```

Full end-to-end install (operator + minimum-scaled platform): see
[docs/runbooks/minimum-deployment.md](./docs/runbooks/minimum-deployment.md).

## What has been verified / What has NOT been verified

- **Verified:** `go build ./...` clean; `controller-gen` regenerates
  CRDs + deepcopy; `kustomize build deploy/operator/` and
  `kustomize build deploy/minimum/` produce valid YAML.
- **Not verified end-to-end in a live cluster:** kind smoke-test from
  [docs/runbooks/minimum-deployment.md](./docs/runbooks/minimum-deployment.md)
  is documented but not executed in this pass (no docker/kind in the dev
  environment used to author this code). Smoke test before tagging
  `v0.2.0` on `main`.
- **Skeleton reconcilers:** child controllers correctly render Deployments
  / Services / ConfigMaps / PVCs / HPAs / NATS StatefulSet, but advanced
  ordering, upgrade choreography, and conditions are stubbed. Treat the
  operator as v0.2.0 scaffold-grade.

## Configuration surface

Operator flags (set in `config/manager/manager.yaml`):

- `--leader-elect` — enable leader election (on in cluster, off for local `make run`).
- `--metrics-bind-address` — Prometheus metrics endpoint (default `:8080`).
- `--health-probe-bind-address` — operator health probes (default `:8081`).

CI:

- `.github/workflows/lint.yml` — markdown lint, no external secrets needed.
- `.github/workflows/docs.yml` — mermaid block validation.

## Git / Workflow

- **Branching policy:** `main` is the protected release line. Feature branches
  (the `claude/*` session branches and any other topic branches) MUST be PR'd
  into `develop` first. Promotion from `develop` to `main` is a separate PR
  cut from `develop` once a set of features has been integrated and validated.
  Never open a PR from a feature branch directly against `main`. If `develop`
  does not yet exist on the remote, create it from `main` before opening the
  first feature PR.
- **Commits:** clear, descriptive messages. Co-author with Claude where applicable.
- **Tagging:** semver — `v0.1.0`, `v0.2.0`, … Tag from `main` only.
- **Container builds:** *(not applicable — no container)*
- **PRs:** do not create unless explicitly asked.

## Issue tracking & releases

All features, bugs, and chores in the Zelos suite are tracked in the org-level
GitHub Project [**Zelos Platform Tracker**](https://github.com/orgs/ZelosAI/projects/2).
Every issue opened in any ZelosAI repo auto-adds to the project via
`.github/workflows/add-to-project.yml` (uses the `ADD_TO_PROJECT_PAT` org secret).

**File issues in the repo they belong to**, not in `zelosai`, unless the work
genuinely spans multiple repos.

**Project fields to set on each item:**

- **Work type** — `Feature` / `Bug` / `Chore`.
- **Priority** — `P0` (drop everything) / `P1` (this sprint) / `P2` (this
  release) / `P3` (someday).
- **Status** — `Todo` / `In Progress` / `Ready for QA` / `Done` / `Blocked`.
  Transitions: `Todo` → `In Progress` is set manually when you start work.
  `In Progress` → `Ready for QA` fires **automatically** when the feature →
  develop PR merges and the `release` workflow's dev container build
  succeeds (see `.github/workflows/tracker-ready-for-qa.yml`).
  `Ready for QA` → `Done` fires **automatically** via the project's
  "Item closed" workflow when the linked issue is auto-closed on the
  develop → main promotion (per `Closes #N` in the feature PR body).
  Use `Blocked` (side-state, any phase) when you can't make forward
  progress; note the blocker in the issue.
- **Release** — cross-repo target: `v0.1`, `v0.2`, `v0.3`, `v1.0`, or
  `Backlog`.
- **Milestone** — matching repo-level milestone (same names exist in every
  repo). Keep Milestone and Release in sync so repo-native views match the
  project.

**When to file vs just fix:** if it's a self-contained change you're about to
ship this session, the PR is the record — no issue needed. File an issue for
work that won't ship this session, anything cross-repo, anything the user
asks to track, or follow-ups you discover but won't do now.

**Linking PRs:** PRs that resolve an issue must include `Closes #N` (or
`Fixes #N`) in the description so GitHub auto-closes the issue on merge and
the project's "Item closed" workflow moves it to `Done`.

## Planning and execution loop

This repo follows a structured planning ↔ execution flow with Claude. Three
artifacts stay in lockstep: the [Zelos Platform Tracker](https://github.com/orgs/ZelosAI/projects/2)
(structured state), this repo's `ROADMAP.md` (human-readable view of THIS
component), and the suite-wide [`zelosai/ROADMAP.md`](https://github.com/ZelosAI/zelosai/blob/main/ROADMAP.md)
(cross-component view).

### When a plan is accepted (planning → backlog)

The moment `ExitPlanMode` returns user approval, Claude must convert the
accepted plan into trackable work BEFORE starting any implementation:

1. **Identify feature boundaries.** Each implementable slice from the plan
   becomes one issue in the canonical repo for that work. Cross-repo slices
   get one canonical issue plus follow-up references in companion repos.
2. **File one issue per slice.** Title `Feature: <slice headline>` (or
   `Bug:` / `Chore:` if more accurate). Body carries the slice's **Why**,
   **Files to change**, **Verification**, and any decisions made during
   planning. Don't summarize — paste the slice content so future sessions
   can execute from the issue alone without re-reading the plan file.
3. **Apply project fields.** `Work type`, `Priority` (P0–P3),
   `Status=Todo`, `Release` (v0.x or `Backlog`). Field + option IDs change
   when the project schema is edited; re-fetch them with
   `gh project field-list 2 --owner zelosai --format json` instead of
   hardcoding.
4. **Apply the repo milestone.** Match `Release`.
   `gh issue create … --milestone v0.x`.
5. **Update this repo's `ROADMAP.md`.** Every filed feature lands in a lane:
   `In flight` (Status=In Progress), `Next` (Status=Todo with a v0.x
   release), `Backlog` (Release=Backlog), or `Recently shipped` (Status=Done,
   closed in the last release). Link by issue URL with the title + priority
   + release tags.
6. **Update `zelosai/ROADMAP.md`** as well if the feature matters at the
   suite level — anything in a v0.x release lane (in-flight / next /
   following) always goes in the suite roadmap; pure component-local backlog
   items can stay component-only.
7. **Update suite-architecture memory** if the plan introduces a new
   component or reshapes how existing ones interact.

This applies to plans of any size. Trivial single-file fixes the user asked
to be done in-session still skip the issue step (per "When to file vs just
fix" above) — but anything that came through `ExitPlanMode` is, by
definition, planned work and gets tracked.

### When given an issue to execute (backlog → implementation)

If the user references an issue by number or URL, Claude:

1. **Fetch the issue.** `gh issue view <N> -R zelosai/<repo> --json
   title,body,labels,milestone,assignees,projectItems`. Read end-to-end
   before touching code.
2. **Move the project item to `Status=In Progress`** and **move the entry
   in `ROADMAP.md` from `Next` (or `Backlog`) to `In flight`**. Same for
   the suite roadmap if the item lives there. Both happen in a single
   commit on the feature branch, before any implementation commits.
3. **Branch off `develop`.** Name: `claude/<short-slug-from-title>`.
4. **Implement** per the issue body's "Files to change" and "Verification"
   sections. Surface deviations to the user before pushing.
5. **PR feature → develop** with `Closes #<N>` in the body. Merge with
   `gh pr merge <PR> --squash --delete-branch --admin`. After merge: the
   `release` workflow builds and pushes the dev container; the
   `tracker-ready-for-qa` workflow then auto-moves the project item to
   `Status=Ready for QA`. Manually move the `ROADMAP.md` entry from
   `In flight` to `Ready for QA`.
6. **Promote develop → main** via a separate PR (`gh pr merge <PR> --merge
   --admin` to preserve commits). Every repo in the org defaults to `main`,
   so this is the merge that fires GitHub's `Closes #N` auto-close.
7. **Back-merge `main → develop`** to absorb the promotion's merge commit.
8. **Move the ROADMAP entries.** `Ready for QA` → `Recently shipped` in
   this repo's `ROADMAP.md` (and in `zelosai/ROADMAP.md` if it's there too).
   This can be folded into the back-merge PR or a tiny follow-up commit.
9. **Confirm.** The project's "Item closed" workflow moves Status to `Done`
   automatically; verify with `gh issue view <N>` and the project view.

If an issue turns out to be too coarse to execute as a single PR, propose
splitting it (in plan mode) before starting any code.

## Relation to the Zelos suite

`zelosai` is the suite's architecture hub. Every other repo's CLAUDE.md cites
[docs/architecture/05-gitflow.md](./docs/architecture/05-gitflow.md) for the
gitflow rules, derives its CLAUDE.md / CI workflows / Dockerfile / Makefile from
[docs/template/](./docs/template/), and is documented under
[docs/architecture/04-components/](./docs/architecture/04-components/).

## Notes / Blockers

- **Operator is v0.2.0 scaffold-grade.** Helm umbrella chart and shared
  schema libraries (`zelosai-schemas`, `@zelosai/schemas`) are still
  deferred to a later pass.
- **Live-cluster verification deferred.** The minimum-scaled deployment
  has been validated via `kustomize build` + `go build` only. Run the
  kind smoke test in [docs/runbooks/minimum-deployment.md](./docs/runbooks/minimum-deployment.md)
  before tagging `v0.2.0` on `main`.
- Migration of [zelos.dgx](https://github.com/kmechlin/ansible-dgx-collection)
  from `kmechlin/*` to `ZelosAI/*` is out of scope; the architecture docs
  link to its current home.
