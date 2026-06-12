# zelosai docs

- **[architecture/](./architecture/)** — the suite-wide architecture. Start at
  [`architecture/00-overview.md`](./architecture/00-overview.md).
- **[template/](./template/)** — the canonical templates every Zelos component
  repo derives from (CLAUDE.md, Dockerfile, Makefile, lint + release workflows,
  PR template, gitignore, editorconfig, CODEOWNERS). See
  [`template/README.md`](./template/README.md) for substitution rules.

- [Getting started](getting-started.md) — the ONE path: copy example env → fill → seal → one command.

## Quick map

| You want to … | Read |
|---|---|
| Understand the whole suite | [`architecture/00-overview.md`](./architecture/00-overview.md) |
| Follow the async request flow | [`architecture/01-async-path.md`](./architecture/01-async-path.md) |
| Follow the sync (broker) flow | [`architecture/02-sync-path.md`](./architecture/02-sync-path.md) |
| Provision a host | [`architecture/03-provisioning.md`](./architecture/03-provisioning.md) |
| Look up a single component | [`architecture/04-components/`](./architecture/04-components/) |
| Follow the gitflow / branching rules | [`architecture/05-gitflow.md`](./architecture/05-gitflow.md) |
| Look up a naming convention | [`architecture/06-naming-conventions.md`](./architecture/06-naming-conventions.md) |
| Pick a deployment topology (solo / split / full) | [`architecture/14-deployment-strategies.md`](./architecture/14-deployment-strategies.md) |
| Know what zelosserver does | [`architecture/13-zelosserver-scope.md`](./architecture/13-zelosserver-scope.md) |
| Bootstrap a new component repo | [`template/README.md`](./template/README.md) |
| Add PR-validation CI to a repo | [`template/ci.yml.tmpl`](./template/ci.yml.tmpl) |
