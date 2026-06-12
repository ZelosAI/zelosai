# `full` deployment overlay

The production-shape install for teams: a **multi-node Kubernetes** control
plane behind a real Ingress controller and a network-backed StorageClass, with
**one or more separate DGX hosts** running host-side `zelosclient` processes
(provisioned by [`zelos.dgx`](../../docs/architecture/03-provisioning.md), **not**
in the cluster).

This overlay is **HA-flavored, not HA** — full HA (multi-replica StatefulSets,
PDBs, multi-zone) is v1.0 work. The point of `full` for v0.4 is to prove the
manifests don't assume single-node behavior and that EA users can install onto a
real cluster **without forking kustomize**.

See [docs/architecture/14-deployment-strategies.md](../../docs/architecture/14-deployment-strategies.md)
for the topology rationale.

## Prerequisites

- **A 3+ node Kubernetes cluster** (control-plane + workers).
- **A CSI driver + network-backed StorageClass** (Ceph RBD, EBS, Cinder,
  Longhorn, …). The overlay pins `ceph-block` by default — swap it (see below).
- **An Ingress controller** — `nginx` by default (swap `ingressClassName`).
- **cert-manager** with a `ClusterIssuer` for the gateway TLS cert (or supply
  your own cert Secret and drop the annotation). Three ready-to-edit issuer
  examples ship here — self-signed, Let's Encrypt, Google Trust Services —
  see **[TLS & DNS](#tls--dns)** below.
- **A DNS record** resolving your gateway hostname to the Ingress — manage it
  by hand, or apply `external-dns.example.yaml` to create it automatically in
  Google Cloud DNS.
- **A CNI that enforces NetworkPolicy** (Calico / Cilium) for `networkpolicy.yaml`
  to take effect — flannel ignores it.
- **One or more DGX hosts** provisioned **separately** via `zelos.dgx`, each
  running a host-side `zelosclient` + local runtime, dialing the cluster's NATS.

## What it changes vs `deploy/minimum/`

| Concern | `minimum` | `full` |
|---|---|---|
| Gateway | 2 replicas + HPA (2–5) | 2 replicas + HPA (2–6) |
| Backplane PVC | 1Gi, default SC | **10Gi on `ceph-block`** (network SC) |
| MCP PVC | 1Gi, default SC | **5Gi on `ceph-block`** (network SC) |
| Gateway exposure | none | **Ingress (nginx, TLS via cert-manager)** |
| NATS exposure | headless ClusterIP | **+ LoadBalancer** for the DGX fleet |
| NetworkPolicy | none | **gateway-only ingress to mcp/backplane** |
| `zelosclient` | off | off (**host-side on the DGX fleet**) |

## Swapping the StorageClass

`ceph-block` is the default. To use a different CSI SC, edit
`storageClassName` in `zelosplatform-patch.yaml` (both `backplane.persistence`
and `mcp.persistence`). The operator threads it into the NATS StatefulSet's
volumeClaimTemplate (see [render/nats.go](../../internal/controller/render/nats.go))
and the mcp PVC. Do **not** leave it unset in `full` — an unset SC falls back to
the cluster default, which may be node-local and would nail the NATS PVC to one
node.

## Swapping the IngressClass

Edit `gateway-ingress.yaml`: change `ingressClassName` (e.g. `traefik`,
`alb`) and the cert annotation to match your controller / issuer. The backend
Service (`zelosgateway-default:8000`) is stable across controllers.

## TLS & DNS

The gateway Ingress terminates TLS with a cert-manager-issued Secret. Pick **one**
`ClusterIssuer` — three example manifests ship here (each is an EXAMPLE, applied
manually; none are part of the kustomization):

| File | Issuer name | Trusted? | Notes |
|---|---|---|---|
| `issuer-self-signed.example.yaml` | `zelos-selfsigned-ca` | No (distribute the root CA) | dev / internal; zero external deps |
| `issuer-letsencrypt.example.yaml` | `letsencrypt-prod` (+ `-staging`) | **Yes** | free; ACME DNS-01 on Google Cloud DNS (or HTTP-01) |
| `issuer-google-trust-services.example.yaml` | `google-trust-services` | **Yes** | free; ACME + External Account Binding; DNS-01 on Cloud DNS |

```bash
# 1) install cert-manager (once)
helm repo add jetstack https://charts.jetstack.io && helm repo update
helm install cert-manager jetstack/cert-manager -n cert-manager --create-namespace --set crds.enabled=true
# 2) edit + apply ONE issuer example (fill in email / project / EAB), e.g.:
kubectl apply -f deploy/full/issuer-letsencrypt.example.yaml
# 3) set its name in gateway-ingress.yaml -> cert-manager.io/cluster-issuer: <issuer>
# 4) (optional) automatic DNS records in Google Cloud DNS:
kubectl apply -f deploy/full/external-dns.example.yaml   # edit --google-project / --domain-filter
```

The public-CA issuers use the **DNS-01** challenge against **Google Cloud DNS**, which works for
wildcards and clusters with no public inbound — it needs a delegated managed zone + a `dns.admin`
service account (the same one external-dns reuses). Full walkthrough (incl. the Google Trust
Services EAB flow):
[docs/runbooks/full-deployment-with-tls-dns.md](../../docs/runbooks/full-deployment-with-tls-dns.md).
The mechanism mirrors the Ansible-managed beds — see
[zelos.kubernetes `docs/cluster-tls-and-dns.md`](https://github.com/ZelosAI/zelos.kubernetes/blob/develop/docs/cluster-tls-and-dns.md).

## Pod anti-affinity — recommended, but an operator gap on v0.2.0

The two gateway replicas SHOULD spread across nodes. However, the v0.2.0
operator's `PlatformComponent` knob bag does **not** surface
`affinity` / `topologySpreadConstraints` / `nodeSelector`, and the platform
controller does not pass them through to child CRs — so the spread constraint
**cannot be set through the ZelosPlatform CR**, and the rendered Deployments
don't exist offline for a kustomize patch to target. This overlay therefore
ships without it and documents the recommended constraint here. The operator
follow-up is to thread a scheduling block from `PlatformComponent` →
`WorkloadSpec` → the rendered Deployment, after which the intended constraint is:

```yaml
# desired on the gateway Deployment (operator follow-up):
topologySpreadConstraints:
- maxSkew: 1
  topologyKey: kubernetes.io/hostname
  whenUnsatisfiable: ScheduleAnyway
  labelSelector:
    matchLabels:
      app.kubernetes.io/name: zelosgateway
```

Until then, a stop-gap on a live cluster is a post-apply `kubectl patch` of the
`zelosgateway-default` Deployment, or a cluster-level default scheduling policy.

## How to apply

```bash
kubectl apply -k deploy/operator/        # operator + CRDs (once)
# fill in the deploy/minimum/ example Secrets, set your SC / Ingress host, then:
kubectl apply -k deploy/full/
kubectl -n zelos wait --for=condition=Ready pod --all --timeout=300s
```

Provision the DGX fleet separately with `zelos.dgx`, pointing each host's
`zelosclient` at the cluster's NATS LoadBalancer address (see the `split`
README for the inventory shape; in `full` the address is the LB, not a NodePort).

## Validation

- `kubectl kustomize deploy/full/` renders valid YAML offline (done).
- **Dry-apply** against any multi-node cluster validates the manifests:
  `kubectl kustomize deploy/full/ | kubectl apply --dry-run=server -f -`
  (requires the CRDs from `deploy/operator/` already installed + cluster reach).
- Actual smoke is deferred to the v0.5 `full` smoke runbook.
