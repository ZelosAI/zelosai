# `observability` overlay (opt-in)

The minimum deployment runs `zelos-otel-collector` with a `debug` exporter —
logs land in the collector Pod and stop there
([11-telemetry.md](../../docs/architecture/11-telemetry.md)). This overlay wires
the suite to a **recommended downstream stack** so telemetry becomes actionable:

- **Logs → Loki**
- **Metrics → Prometheus** (remote-write)
- **Traces → Tempo** (OTLP)
- **3 Grafana dashboards** (auto-discovered by the Grafana sidecar)

It is **opt-in** and **not part of any EA bundle by default** — the bundle adds
it as a flagged optional layer. It does **not** re-implement the operator's
collector, and it does **not** install Loki / Prometheus / Tempo / Grafana
themselves (commodity components; install via their own charts). It provides the
Zelos-side *downstream config + dashboards*.

## What it ships

| File | What |
|---|---|
| `downstream-collector.yaml` | a standalone OTel collector (Deployment + Service + config ConfigMap `zelos-otel-downstream`) that receives OTLP on `:4317/:4318` and fans out to Loki / Prometheus / Tempo |
| `dashboards/platform-health.json` | "Zelos Platform Health" — per-component Ready + request rate + 5xx rate |
| `dashboards/async-path-latency.json` | "Zelos Async Path Latency" — enqueue → reply p50/p95/p99, queue depth, enqueue vs reply rate |
| `dashboards/inference-throughput.json` | "Zelos Inference Throughput" — vLLM token rates, scheduler queue, TTFT p95, KV-cache util |

Each dashboard is rendered into a ConfigMap labeled `grafana_dashboard: "1"` so
the [Grafana dashboard sidecar](https://github.com/grafana/helm-charts/tree/main/charts/grafana#sidecar-for-dashboards)
imports it automatically. They use a `${DS_PROMETHEUS}` datasource template
variable, so they also import cleanly by hand into a stock **Grafana 10.x**
(Dashboards → Import → upload JSON → pick your Prometheus datasource).

## Install

```bash
# 1. Apply alongside an existing platform (minimum / solo / split / full):
kubectl apply -k deploy/observability/

# 2. Point the platform's telemetry at the downstream collector so the operator
#    skips its own debug collector and exports straight to Loki/Prom/Tempo.
#    (External endpoint mode — see 11-telemetry.md.)
kubectl -n zelos patch zelosplatform default --type merge -p \
  '{"spec":{"telemetry":{"enabled":true,"externalEndpoint":"zelos-otel-downstream.zelos.svc:4317"}}}'
```

After step 2 the operator's in-cluster `zelos-otel-collector` is suppressed and
every workload sends OTLP to `zelos-otel-downstream`, which forwards to the
backends.

### Backend endpoints

`downstream-collector.yaml` assumes the commodity stack lives in an
`observability` namespace:

- Loki push: `http://loki.observability.svc:3100/loki/api/v1/push`
- Prometheus remote-write: `http://prometheus.observability.svc:9090/api/v1/write`
  (run Prometheus with `--web.enable-remote-write-receiver`)
- Tempo OTLP: `tempo.observability.svc:4317`

Edit the exporter endpoints in the ConfigMap to match your install.

## Metric name caveat

The dashboard queries reference the **intended** metric names for the suite's
OpenTelemetry instrumentation:

- HTTP server metrics follow the OTel semantic convention
  (`http_server_request_duration_seconds_*`, label `zelos_component`).
- Async-path metrics (`zelos_async_roundtrip_seconds`,
  `zelos_gateway_enqueued_total`, `zelos_gateway_replies_total`,
  `zelos_backplane_pending_messages`) are **suite-defined** and depend on the
  gateway/backplane emitting them — they are part of the components'
  instrumentation roadmap, not guaranteed present on every build yet.
- Inference metrics use the standard **vLLM** Prometheus names (`vllm:*`),
  scraped from the host-side runtime.

Where a metric isn't emitted yet, the corresponding panel renders empty rather
than erroring. The verification target in #40 — a demo async request showing
data on the Async Path Latency dashboard within 30s — depends on the gateway
emitting the `zelos_async_roundtrip_seconds` histogram.

## Validation

`kubectl kustomize deploy/observability/` renders valid YAML offline (the three
dashboard ConfigMaps + the downstream collector). Dashboard JSON parses and
targets Grafana schemaVersion 39 (Grafana 10.x). Live import + the 30s-data
check are deferred to a cluster with the backends installed.
