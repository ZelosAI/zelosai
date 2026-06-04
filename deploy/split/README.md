# `split` deployment overlay

The middle topology: a **laptop / workstation running k3s** as a light control
plane, with the **DGX a remote inference host**. The `zelosclient` runs as a
**host process on the DGX** (provisioned by the
[`zelos.dgx`](../../docs/architecture/03-provisioning.md) Ansible collection),
**not** in the cluster. The DGX-side client dials the laptop's backplane over
the network; the cluster never dials out.

See [docs/architecture/14-deployment-strategies.md](../../docs/architecture/14-deployment-strategies.md)
for the topology rationale and the two-trust-domain secret model this overlay
implements.

## What this overlay assumes

- **A Linux laptop / workstation with k3s** as the control plane (ships
  `local-path` storage, which this overlay uses).
- **A reachable remote DGX** running the `zelosclient` host process + a local
  vLLM/Ollama runtime, provisioned by `zelos.dgx`. The DGX is **not** a cluster
  node.
- **Network reachability** from the DGX to the laptop's NATS NodePort
  (`30422`). WireGuard is the recommended substrate (see below).

## What it changes vs `deploy/minimum/`

| Concern | `minimum` | `split` |
|---|---|---|
| Gateway replicas | 2 + HPA | **1, no HPA** |
| `zelosclient` | off | off (**runs on the DGX host**, not the cluster) |
| NATS exposure | headless ClusterIP | **+ NodePort `30422`** for the DGX client to dial in |
| Public ingress | none | none (gateway/broker over `localhost`) |
| StorageClass | cluster default | unset → k3s `local-path` |

There is intentionally **no in-cluster GPU**: the laptop has none, the operator
renders no GPU workload, and no NVIDIA device-plugin is installed (contrast
`solo`).

## Networking

The structural divide vs `solo`: the **backplane ↔ client edge leaves the
cluster**. NATS must be reachable from the DGX, and the client always *dials in*.
This overlay exposes only the NATS client port (4222) on NodePort `30422`. The
gateway and broker stay loopback-only on the laptop (reach them with
`kubectl -n zelos port-forward svc/zelosgateway-default 8000:8000`).

Choose one network mode for the DGX → laptop NATS link, **recommended first**:

1. **WireGuard (recommended).** The canonical Zelos VPN substrate across the
   suite (matches the optional WireGuard wrapper in `zelosbroker#12`). Stand up
   a WireGuard tunnel between the laptop and the DGX, then point the DGX client
   at the laptop's WireGuard address on `30422`. Bind/firewall the NodePort to
   the WireGuard interface so it is never world-reachable. **No Tailscale-only
   recipes** — WireGuard is the supported VPN substrate.
2. **LAN (fallback).** Laptop and DGX on the same trusted L2/L3 segment; the DGX
   reaches `nats://<laptop-lan-ip>:30422` directly. Acceptable for a single
   trusted lab network.
3. **Public IP with TLS (fallback, discouraged).** Only if the laptop has a
   routable address: terminate TLS on NATS (`backplane.tlsSecretRef`) and add a
   NATS auth token; never expose `30422` unauthenticated to the internet.

Because the link crosses a network, it needs its own **cross-host credential**
(NATS auth token / TLS), provisioned to the DGX by `zelos.dgx` — not just
network reachability (see 14-deployment-strategies.md → Secret surfaces).

### Sample `zelos.dgx` inventory snippet

Point the DGX-side `zelosclient` at the laptop's reachable NATS address. With
WireGuard, `<laptop-addr>` is the laptop's WireGuard IP:

```yaml
# inventory/hosts.yml for the zelos.dgx collection
all:
  hosts:
    dgx-01:
      ansible_host: 10.10.0.2          # DGX over WireGuard
  vars:
    # The laptop control plane's reachable NATS endpoint (WireGuard IP + NodePort).
    zelosclient_backplane_url: "nats://10.10.0.1:30422"
    zelosclient_runtime: vllm
    zelosclient_runtime_url: "http://localhost:8000"
    zelosclient_subscribe_topics:
      - "zelos.infer.>"
    # Cross-host credential for the backplane link (token or TLS material).
    zelosclient_backplane_auth_token: "{{ vault_zelos_nats_token }}"
```

## How to apply

```bash
kubectl apply -k deploy/operator/        # operator + CRDs (once)
# fill in the deploy/minimum/ example Secrets, then:
kubectl apply -k deploy/split/
kubectl -n zelos wait --for=condition=Ready pod --all --timeout=180s
```

Then bring up the DGX host process via the `zelos.dgx` playbook pointed at the
inventory above. End-to-end async path: workstation IDE → laptop gateway →
laptop backplane → DGX `zelosclient` → vLLM → reply.

## Validation

`kubectl kustomize deploy/split/` renders valid YAML offline. Live end-to-end
(laptop k3s + reachable DGX) is deferred.
