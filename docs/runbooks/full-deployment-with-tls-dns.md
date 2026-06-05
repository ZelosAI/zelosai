# Runbook — `full` deployment with public TLS & DNS

How to put the `full` overlay's gateway Ingress behind a **publicly-trusted TLS certificate** and
(optionally) **automatic public DNS**. Covers three certificate options — self-signed (dev),
**Let's Encrypt**, and **Google Trust Services** — all via [cert-manager](https://cert-manager.io/),
plus [external-dns](https://kubernetes-sigs.github.io/external-dns/) for Google Cloud DNS.

This is the BYO-cluster counterpart to the Ansible-managed beds; the cert-manager mechanics are
identical (see zelos.kubernetes
[`docs/cluster-tls-and-dns.md`](https://github.com/ZelosAI/zelos.kubernetes/blob/develop/docs/cluster-tls-and-dns.md)
and [`docs/google-trust-services-cert-manager.md`](https://github.com/ZelosAI/zelos.kubernetes/blob/develop/docs/google-trust-services-cert-manager.md)).

## Prerequisites
- A `full`-capable cluster + Ingress controller (see [deploy/full/README.md](../../deploy/full/README.md)).
- `kubectl` admin access; `helm`; (for public CAs) `gcloud`.
- A **public domain** + a **Google Cloud DNS managed zone** delegated to it (for DNS-01).

## 1. Install cert-manager

```bash
helm repo add jetstack https://charts.jetstack.io && helm repo update
helm install cert-manager jetstack/cert-manager \
  -n cert-manager --create-namespace --set crds.enabled=true
kubectl -n cert-manager rollout status deploy/cert-manager-webhook
```

## 2. Choose a certificate option

Edit the example, apply it, then set the issuer name in `deploy/full/gateway-ingress.yaml`
(`cert-manager.io/cluster-issuer: <name>`).

### Option A — self-signed (dev / internal)
Untrusted by browsers unless you distribute the root CA. Zero external dependencies.
```bash
kubectl apply -f deploy/full/issuer-self-signed.example.yaml   # issuer: zelos-selfsigned-ca
```

### Option B — Let's Encrypt (public, free)
DNS-01 via Google Cloud DNS (works for wildcards / private clusters). HTTP-01 is a commented
alternative in the example for a simple single public host on :80.
```bash
# dns.admin service account + Secret:
gcloud iam service-accounts create dns01-solver
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member "serviceAccount:dns01-solver@${PROJECT_ID}.iam.gserviceaccount.com" --role roles/dns.admin
gcloud iam service-accounts keys create key.json \
  --iam-account "dns01-solver@${PROJECT_ID}.iam.gserviceaccount.com"
kubectl -n cert-manager create secret generic clouddns-dns01-solver-svc-acct --from-file=key.json

# edit email + project, then apply (creates letsencrypt-staging + letsencrypt-prod):
kubectl apply -f deploy/full/issuer-letsencrypt.example.yaml
```
Validate with `letsencrypt-staging` first (prod has strict rate limits), then switch to
`letsencrypt-prod`.

### Option C — Google Trust Services (public, free, EAB)
Same as B plus External Account Binding.
```bash
gcloud services enable publicca.googleapis.com dns.googleapis.com
# EAB key — single-use, expires in 7 days; record keyId + b64MacKey:
gcloud publicca external-account-keys create
# EAB HMAC Secret (b64MacKey VERBATIM — already base64url):
kubectl -n cert-manager create secret generic gts-eab --from-literal=hmac='<b64MacKey>'
# (reuse the dns.admin Secret from Option B), then edit email/project/keyID and apply:
kubectl apply -f deploy/full/issuer-google-trust-services.example.yaml   # issuer: google-trust-services
```
Start on the test directory (commented in the example), then switch to production.

## 3. Point the Ingress at the issuer + host

```bash
$EDITOR deploy/full/gateway-ingress.yaml
#   annotations: { cert-manager.io/cluster-issuer: letsencrypt-prod }   # your choice from step 2
#   tls[].hosts / rules[].host: zelos.<your-domain>
```

## 4. (Optional) Automatic public DNS

```bash
kubectl create namespace external-dns
kubectl -n external-dns create secret generic clouddns-dns01-solver-svc-acct --from-file=key.json
$EDITOR deploy/full/external-dns.example.yaml   # --google-project / --domain-filter / --txt-owner-id
kubectl apply -f deploy/full/external-dns.example.yaml
```
external-dns publishes the Ingress address. If the gateway sits behind a public WAN IP (NAT),
set `--default-targets=<WAN-IP>` and add the port-forward.

## 5. Apply `full` and verify

```bash
kubectl apply -k deploy/operator/
kubectl apply -k deploy/full/
kubectl -n zelos wait --for=condition=Ready pod --all --timeout=300s

# TLS:
kubectl describe ingress -n zelos zelosgateway
kubectl get certificate -A                 # READY=True for zelosgateway-tls
kubectl get order,challenge -A             # ACME progress (empty once issued)
openssl s_client -connect zelos.<your-domain>:443 -servername zelos.<your-domain> </dev/null 2>/dev/null \
  | openssl x509 -noout -issuer
# DNS:
kubectl -n external-dns logs deploy/external-dns | tail
dig +short zelos.<your-domain>
```

## Troubleshooting
- **Cert not Ready:** `kubectl describe certificate -n zelos zelosgateway-tls` → follow to the
  Order/Challenge. A stuck DNS-01 challenge usually means the zone isn't delegated or the SA lacks
  `dns.admin`.
- **`externalAccountRequired` (GTS):** the EAB is missing/consumed/expired — re-create it (single-use,
  7-day) and re-apply.
- **Rate-limited (LE):** you used `letsencrypt-prod` before validating on staging — wait out the limit.
- **DNS not resolving:** confirm the registrar NS records point at the Cloud DNS zone
  (`dig NS <your-domain>`), and that `--domain-filter` matches.
