## What

<one-paragraph summary of the change>

## Why

<link to issue, ticket, or doc; if architectural, link to the relevant page in
[zelosai/docs/architecture](https://github.com/ZelosAI/zelosai/tree/main/docs/architecture)>

## Test plan

- [ ] `make lint` passes
- [ ] `make test` passes (unit tests — `unit-tests` workflow gate)
- [ ] `make test-integration` passes locally if the change touches runtime behaviour (otherwise the `integration-tests` workflow run against the dev container post-merge covers it)
- [ ] If container changed: `make image` succeeds
- [ ] If runtime behaviour changed: <describe manual verification>

## Gitflow check

- [ ] Targeting `develop` (NOT `main`). See [05-gitflow.md](https://github.com/ZelosAI/zelosai/blob/main/docs/architecture/05-gitflow.md).
- [ ] Branch name matches `^(feature|fix|chore|docs)/[0-9]+-[a-z0-9-]+$` (enforced by `branch-lint`). The numeric prefix is the GitHub issue this PR resolves.
- [ ] PR body contains `Closes #<N>` (or `Fixes #N` / `Resolves #N`) referencing that same issue number.
- [ ] No semver tag in this PR (tags are cut from `main` only).
