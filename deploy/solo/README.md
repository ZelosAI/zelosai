# `solo` deployment overlay

The one-box install path: a single **NVIDIA DGX** running k3s with **every**
Zelos component — the operator-managed control plane *and* the `zelosclient`
inference worker — on that one machine. The developer's workstation only
connects in. This is the simplest topology and the only one where the
`zelosclient` runs as an in-cluster Pod and an NVIDIA device-plugin runs inside
Kubernetes.

See [docs/architecture/14-deployment-strategies.md](../../docs/architecture/14-deployment-strategies.md)
for the topology rationale, networking model, and the GPU-placement decision
this overlay implements.

## What this overlay assumes

- **A single NVIDIA DGX** (or equivalent GPU box) that is *both* the k3s node
  and the inference host.
- **k3s already installed** on the DGX (ships Traefik + `local-path` storage,
  both of which this overlay relies on).
- **NVIDIA host stack present**: the datacenter driver is loaded and
  `nvidia-container-toolkit` is installed with the container runtime set to the
  `nvidia` runtime class. The device-plugin DaemonSet this overlay installs
  surfaces the GPU to Kubernetes but does **not** install the driver/toolkit.
- **The device-plugin image is reachable** — `nvcr.io/nvidia/k8s-device-plugin`
  (pulled from NGC; no auth needed for this public image).
- **A vLLM (or Ollama) runtime** is reachable from the client Pod at
  `runtimeURL` (default `http://localhost:8000`). Run it as a sibling workload
  or on the host; the client only bridges the backplane to it.

## What it changes vs `deploy/minimum/`

| Concern | `minimum` | `solo` |
|---|---|---|
| Gateway replicas | 2 + HPA | **1, no HPA** |
| MCP replicas | 1 | 1 |
| Backplane PVC | 1Gi | 5Gi (room for JetStream on the one box) |
| `zelosclient` | off | **on, `inCluster: true`, runtime `vllm`** |
| NVIDIA device-plugin | none | **DaemonSet in `kube-system`** |
| Gateway ingress | none | **Traefik Ingress + NodePort 30080** |
| StorageClass | cluster default | unset → k3s `local-path` (correct for solo) |

## How to apply

```bash
# 1. Operator + CRDs (once).
kubectl apply -k deploy/operator/
kubectl -n zelos-system wait --for=condition=Available \
  deploy/zelosai-controller-manager --timeout=120s

# 2. Fill in the example Secrets that deploy/minimum/ carries (GHCR pull secret,
#    zelos-mcp-auth, zelos-mcp-servers) — see deploy/minimum/ and the
#    minimum-deployment runbook. Then apply this overlay:
kubectl apply -k deploy/solo/

# 3. Verify (target: all Pods Ready in < 3 min, ZelosPlatform/default Ready).
kubectl -n zelos wait --for=condition=Ready pod --all --timeout=180s
kubectl -n zelos get zelosplatform default
```

Reach the gateway from the workstation at `http://<dgx-ip>:30080` (NodePort) or
via the Traefik Ingress once you point a hostname at the DGX.

## GPU request — known v0.2.0 limitation

The NVIDIA device-plugin advertises `nvidia.com/gpu` on the node, and the
`client.inCluster: true` setting makes the operator render the `zelosclient`
DaemonSet. **However**, the v0.2.0 operator's client reconciler
([internal/controller/zelosclient_controller.go](../../internal/controller/zelosclient_controller.go))
renders only default CPU/memory requests — it does **not yet emit a
`limits: nvidia.com/gpu: 1`** request, and `PlatformClient` exposes no
`resources` knob to inject one through the CR. Until that lands, on a DGX the
client still co-schedules with the GPU (single GPU node, default scheduling),
but it does not *reserve* the device. Pinning the GPU request is tracked as
operator work; do not assume hard GPU isolation in `solo` on v0.2.0.

## Validation

`kubectl kustomize deploy/solo/` renders valid YAML offline (no cluster
needed). Live smoke per step 3 above is deferred to a DGX with k3s.
