# 05 — Suite-wide gitflow

> **This is the canonical gitflow for every Zelos suite repository.** Every
> component repo's CLAUDE.md cites this page; the **Git / Workflow** section
> of the [CLAUDE.md template](../template/CLAUDE.md.tmpl) is byte-identical
> across the suite.

## Branches

- **`main`** — protected release line. Tags are cut from here. Never PR a feature
  branch directly into `main`.
- **`develop`** — integration branch. All feature work merges here first. If
  `develop` doesn't exist on the remote yet, create it from `main` before opening
  the first feature PR.
- **`claude/<slug>`** — session branches created by Claude when working on a
  task. Pattern: `claude/<short-description>` (e.g. `claude/bootstrap-zelosgateway`).
- **Other topic branches** — `feat/...`, `fix/...`, `chore/...` for non-Claude work.

## Flow

```mermaid
---
title: Zelos suite-wide gitflow
---
gitGraph
   commit id: "scaffold" tag: "v0.1.0"
   branch develop
   checkout develop
   commit id: "develop created"
   branch claude/feature-a
   checkout claude/feature-a
   commit id: "WIP a"
   commit id: "polish a"
   checkout develop
   merge claude/feature-a id: "PR → develop"
   branch claude/feature-b
   checkout claude/feature-b
   commit id: "feature b"
   checkout develop
   merge claude/feature-b id: "PR → develop"
   checkout main
   merge develop tag: "v0.2.0"
```

Read top-to-bottom: every feature lands on `develop` first via its own PR.
A separate **promotion PR** later batches one or more feature merges from
`develop` into `main`, and the `v<X.Y.Z>` tag is cut from `main` at the same
time. The `release.yml` workflow then builds + pushes the
`ghcr.io/zelosai/<repo>` image.

Rules:

- **One feature, one PR**, targeting `develop`.
- **Never** open a PR directly from a feature branch into `main`.
- Promotion `develop` → `main` is its **own** PR, cut deliberately when a set of
  features has integrated cleanly on `develop`.
- Tags (`vX.Y.Z`) are pushed to `main` only.

## Commit messages

Clear, descriptive, imperative. If Claude authored or co-authored the change,
add a Co-Authored-By trailer:

```
Co-Authored-By: Claude <noreply@anthropic.com>
```

(Or a more specific model identifier if the session's tooling provides one.)

Don't `--amend` past commits unless explicitly asked. If a pre-commit hook
fails, the commit didn't happen — fix the issue and create a **new** commit.

## Tagging

Semver: `v<MAJOR>.<MINOR>.<PATCH>`.

- `MAJOR` bump: backwards-incompatible API or contract change.
- `MINOR`: new functionality, backwards-compatible.
- `PATCH`: bug fixes only.

Tags are pushed to `main` after a promotion PR lands. Don't tag from `develop`
or feature branches.

For repos with shared-lib content (like `zelosai/schemas` once it exists),
the schemas have their own semver inside the repo even though the repo's
release tags follow the repo as a whole.

## Container builds

Every component repo that ships a container has `.github/workflows/release.yml`
(from [`zelosai/docs/template/release.yml.tmpl`](../template/release.yml.tmpl)).
The workflow triggers on:

- Push to `main` → publishes `ghcr.io/zelosai/<repo>:main` and `:sha-<short>`.
- Tag push `v*` → publishes `ghcr.io/zelosai/<repo>:vX.Y.Z` and `:latest`, plus `:sha-<short>`.

Image references in the suite (e.g. inside Helm values) should pin to a tag
(`:vX.Y.Z`) or a digest, **not** to `:latest`.

## PRs

- **Do not create a PR unless explicitly asked.** Push branches; let the human
  open the PR.
- Use the standard [PR template](../template/pull_request_template.md) — checks
  for `make lint`, `make test`, container build (if applicable), and the
  gitflow checks (targeting `develop`, branch shape, no tag in the PR).

## Why `develop` and not trunk-based?

The suite has multiple independently-releasable components and several
work streams in flight at once. A `develop` integration branch lets us:

- Validate feature combinations together before they hit `main`.
- Cut promotion PRs that batch a coherent release.
- Let multiple feature PRs land without each one tagging a release.

If a repo becomes mature enough that trunk-based makes sense, that's a
per-repo decision made later — the canonical gitflow stays as-is for the suite.

## Source of truth

The canonical phrasing of the gitflow rule that goes into every repo's
CLAUDE.md lives in [`zelosai/docs/template/CLAUDE.md.tmpl`](../template/CLAUDE.md.tmpl)
under the **Git / Workflow** section. If you change the gitflow, change that
template **and this page** in the same PR, then propagate to every consuming
repo.
