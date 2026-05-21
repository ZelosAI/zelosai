## What

<one-paragraph summary of the change>

## Why

<link to issue, ticket, or doc; if architectural, link to the relevant page in
[docs/architecture](./docs/architecture)>

## Test plan

- [ ] Markdown renders correctly on GitHub.
- [ ] Internal links resolve (the lychee CI job is green or the failures are intentional).
- [ ] If a template changed: every consuming repo has a follow-up PR identified.

## Gitflow check

- [ ] Targeting `develop` (NOT `main`). See [05-gitflow.md](./docs/architecture/05-gitflow.md).
- [ ] Branch is `claude/<slug>` or topic branch off `develop`.
- [ ] No semver tag in this PR (tags are cut from `main` only).
