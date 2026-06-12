# Getting started — build a Zelos environment

There is ONE path (v0.4.8). Everything an environment needs lives in a single
per-environment inventory directory; secrets are inline-vaulted in one file;
one command builds the bed.

## 0. Prerequisites (once per operator)

- Docker (the operator image is self-contained — no local ansible needed), or
  a local venv with `ansible-core ≥ 2.17` + `pip install zelos-common` and the
  collection CLI plugins for the repos you drive.
- The vault identity file `~/.zelos-vault-identities.json` carrying
  `<env>_bed` (each environment's vault id), readable by
  `scripts/vault-pass-client.sh`.
- SSH key authorized on the target PVE host (`make push-key BED=<env>`).

## 1. Create the environment inventory

```bash
cd zelos.foundry           # the composition repo (installs proxmox+kubernetes+bastion)
cp -r inventory/example inventory/<env>
cd inventory/<env> && for f in example.*; do mv "$f" "${f/example/<env>}"; done
```

Fill `<env>.config.yml`:

- `environment_identity:` — `{name, product, root_domain, public_port}`
  (doc 18; every hostname, cert, and OIDC issuer derives from this tuple).
- The provider + cluster dicts (`proxmox`, `network`, `vms`, per-role dicts) —
  every key is annotated in the example.
- Optional appliance: the unified `bastion:` dict in `<env>.bastion.yml`
  (`bastion.enabled: true` — see zelos.bastion/inventory/example/).

Seed `<env>.secrets.yml` with the CHANGEME values (GitHub OAuth apps, PVE
token, …). Names follow the doc-18 registry (`bastion_*` for the bastion's own).

## 2. Seal the secrets

```bash
make seal BED=<env>        # inline-vaults every value in place (id <env>_bed)
```

(Direct form: `zelosctl foundry secrets seal --env <env>` — or any collection's
`secrets` verb; they share zelos.common.secrets, merge-never-overwrite.)

## 3. Build the environment

```bash
make foundry-bed BED=<env>     # bare metal → host prep → images → cluster →
                               # platform → foundry CI (+ bastion when enabled)
```

Re-runs are idempotent; on a REBUILD where the PVE host persists, add
`SKIP_TAGS=host,images`.

## 4. Day-2 — the self-contained operator container (§19)

A pulled image needs ONLY the inventory + vault key mounted:

```bash
docker run --rm \
  -v $PWD/inventory/<env>:/work/inventory/<env> \
  -v ~/.zelos-vault-identities.json:/root/.zelos-vault-identities.json:ro \
  -v ~/.ssh/<key>:/root/.ssh/<key>:ro \
  -e ANSIBLE_VAULT_IDENTITY_LIST=<env>_bed@/work/scripts/vault-pass-client.sh \
  ghcr.io/zelosai/zelos.foundry:<tag> foundry info --env <env>
```

`zelosctl --help` lists every installed collection plugin
(`foundry` / `kubernetes` / `proxmox` / `bastion`) — subcommands mirror each
collection's playbook verbs (conventions doc 15 §19).

## Reference

- Conventions: [architecture/15](architecture/15-ansible-collection-conventions.md)
- Identity model: [architecture/18](architecture/18-organization-model.md)
- DNS/hostname standard: [architecture/16](architecture/16-dns-and-hostname-standard.md)
- CLI/operator design: [architecture/19](architecture/19-operator-and-cli.md)
