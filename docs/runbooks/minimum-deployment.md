# Runbook: Minimum scaled deployment

End-to-end install of the Zelos suite into a fresh cluster.

## Pre-requisites

- `kubectl` ≥ 1.28 pointed at the target cluster.
- `kustomize` ≥ 5.x (or `kubectl apply -k` ≥ 1.28).
- A GitHub PAT with `read:packages` scope.
- A storage class capable of `ReadWriteOnce` (cluster default is fine).

## 1. Install the operator

```bash
kubectl apply -k deploy/operator/
kubectl -n zelos-system wait --for=condition=Available deploy/zelosai-controller-manager --timeout=120s
```

This installs the CRDs, RBAC, ServiceAccount, and the controller manager
into the `zelos-system` namespace.

## 2. Prepare the workload namespace + secrets

```bash
kubectl create namespace zelos

# GHCR pull secret (read:packages PAT required)
kubectl create secret docker-registry ghcr-pull-secret \
  --namespace=zelos \
  --docker-server=ghcr.io \
  --docker-username=<gh-user> \
  --docker-password=<PAT> \
  --docker-email=<email>

# zelosmcp auth Secret
python -c "from cryptography.fernet import Fernet; print(Fernet.generate_key().decode())" \
  | xargs -I{} kubectl -n zelos create secret generic zelos-mcp-auth \
      --from-literal=auth.key={} \
      --from-literal=providers.json='{"github":{"client_id":"...","client_secret":"...","scopes":["read:user"]}}'

# MCP backend catalog ConfigMap (empty by default)
kubectl -n zelos create configmap zelos-mcp-servers --from-literal=mcpServers.json='{"mcpServers": {}}'
```

(Skip the Python step and supply your own Fernet key if `cryptography` isn't
installed locally.)

## 3. Apply the platform CR

The shipped sample at `deploy/minimum/zelosplatform.yaml` enables
gateway + backplane + mcp + telemetry. Apply it:

```bash
kubectl apply -f deploy/minimum/zelosplatform.yaml
```

(Or `kubectl apply -k deploy/minimum/` after filling in the example Secret
manifests in that directory.)

## 4. Verify

```bash
kubectl -n zelos wait --for=condition=Ready pod --all --timeout=180s
kubectl -n zelos get all
kubectl -n zelos get zelosplatform default -o yaml | yq '.status'
```

Expected Pods (all `Ready`):

| Pod | Source |
|---|---|
| `zelos-otel-collector-*` | platform telemetry |
| `zelos-backplane-nats-default-0` | platform → ZelosBackplane → NATS StatefulSet |
| `zelosgateway-default-*` (×2) | platform → ZelosGateway |
| `zelosmcp-default-0` | platform → ZelosMCP |

Verify health endpoints reach 200:

```bash
for svc in zelosgateway-default zelosmcp-default; do
  kubectl -n zelos port-forward svc/$svc 18000:8000 &
  PF=$!
  sleep 1
  curl -fsS localhost:18000/healthz && echo " ($svc healthz)"
  curl -fsS localhost:18000/readyz  && echo " ($svc readyz)"
  kill $PF
done
```

Verify telemetry flows:

```bash
kubectl -n zelos logs deploy/zelos-otel-collector | grep -E 'service.name|LogRecord'
```

## 5. Tear down

```bash
kubectl delete -k deploy/minimum/   # removes Pods, PVCs, Secrets, the platform CR
kubectl delete -k deploy/operator/  # removes the operator + CRDs (and all owned children)
```

## Common failures

- **`ImagePullBackOff`**: GHCR pull secret missing or wrong PAT scope. Re-create
  `ghcr-pull-secret` with `read:packages`.
- **`Pending` PVC**: cluster has no default storage class. Set
  `spec.persistence.storageClassName` per component or install a CSI driver.
- **`CrashLoopBackOff` on backplane-nats**: usually OOM if `persistence.size`
  is tiny. Bump to ≥ 1Gi.
- **`readyz` 503 on gateway**: backplane URL not yet reachable. Inspect
  `kubectl -n zelos describe zelosbackplane default`.
