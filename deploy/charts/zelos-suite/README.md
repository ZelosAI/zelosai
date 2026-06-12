# zelos-suite Helm chart

Renders a single `ZelosPlatform` CR for the [zelosai operator](../../../README.md)
from per-component **image values**. This exists (rather than a plain kustomize
overlay like [`deploy/minimum/`](../../minimum/)) so **Argo CD Image Updater** can
pin component image digests through the CR: Image Updater's git write-back targets
Helm values / kustomize `images:` — and the operator's per-component `image` lives
in a CRD field that the kustomize `images` transformer can't reach, but a Helm
value can.

The operator composes the final image with [`render.ImageRef`](../../../internal/controller/render/component.go)
(`repo@sha256:…` for a digest, `repo:tag` otherwise), so when Image Updater
rewrites `<component>.image.tag` to a digest the Deployment rolls.

## Layout

- `values.yaml` — defaults: GHCR repos + `:develop`, minimum-scaled set
  (gateway ×2 + backplane + mcp; broker/server/client off).
- `templates/zelosplatform.yaml` — the CR; per-component blocks gated on `enabled`.
- Bed overlays live in [`deploy/beds/<bed>/values.yaml`](../../beds/) (registry +
  enabled set per bed).

## Prerequisites (in the target namespace, before applying)

1. Operator + CRDs: `kubectl apply -k deploy/operator/`.
2. Pull secret (`imagePullSecret`) — docker-registry Secret for the chosen registry.
3. `zelos-mcp-auth` Secret (Fernet key + `providers.json`) and `zelos-mcp-servers`
   ConfigMap — see [`deploy/minimum/`](../../minimum/) examples.

## Render / apply

```bash
helm template zelos-suite deploy/charts/zelos-suite \
  -f deploy/beds/alpha/values.yaml | kubectl apply -f -
```

Normally driven by the `zelos-suite` Argo CD Application (source =
`deploy/charts/zelos-suite`, values = `deploy/beds/<bed>/values.yaml`) with
Image Updater annotations for develop-digest auto-deploy.
