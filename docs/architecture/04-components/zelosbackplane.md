# zelosbackplane

- **Repo:** [ZelosAI/zelosbackplane](https://github.com/ZelosAI/zelosbackplane)
- **Image:** `ghcr.io/zelosai/zelosbackplane`
- **Status:** Scaffold — v0.1.0, connector interfaces + schema skeleton, no
  substrate chosen yet.

## Role in the suite

The async fabric. A message bus / event stream that decouples request-issuing
components (`zelosgateway`) from request-processing components (`zelosclient`
workers). Carries:

- `inference.requests.*` — work-queue topics; one of N subscribed workers claims
  each message.
- `inference.responses.<corrId>` — request/reply correlation.
- `provisioning.events` — events emitted by Ansible Jobs and other lifecycle work.
- `metrics.*` — optional fan-out for cluster-wide metrics.

See [06-naming-conventions.md](../06-naming-conventions.md#backplane-topics) for
the topic-naming rules and
[01-async-path.md](../01-async-path.md) for how requests flow through here.

```mermaid
flowchart LR
  subgraph publishers["Publishers"]
    direction TB
    gw["zelosgateway"]
    op["zelosai operator<br/><i>(future)</i>"]
    job["Ansible Job"]
    metrics_src["any component<br/><i>(heartbeats)</i>"]
  end

  subgraph bp_runtime["<b>zelosbackplane</b><br/>(NATS / Redis / Kafka — connector-agnostic)"]
    direction TB
    req["<b>inference.requests.&lt;kind&gt;</b><br/><i>work-queue retention</i>"]
    resp["<b>inference.responses.&lt;corrId&gt;</b><br/><i>limits retention</i>"]
    ev["<b>provisioning.events</b><br/><i>limits retention</i>"]
    metrics_topic["<b>metrics.*</b><br/><i>fan-out</i>"]
  end

  subgraph subscribers["Subscribers"]
    direction TB
    workers["zelosclient<br/>workers"]
    gw_sub["zelosgateway<br/><i>(response handler)</i>"]
    obs["observability<br/><i>(future zelosserver?)</i>"]
  end

  gw --> req
  op --> ev
  job --> ev
  metrics_src --> metrics_topic
  req --> workers
  workers --> resp
  resp --> gw_sub
  ev --> obs
  metrics_topic --> obs
```

## Substrate

**Not pinned in v1.** The repo defines a substrate-agnostic connector interface
(`src/zelosbackplane/connectors/`) with skeleton implementations for NATS, Redis
Streams, and Kafka. NATS is the most likely first choice (lightweight, native
work-queue semantics via JetStream, fits single-node deployments well), but the
abstraction stays so the suite can swap later without rewriting consumers.

## Schemas

The repo holds the canonical **envelope schema** (`schemas/envelopes/v1/`) and
the topic catalog (`schemas/topics.yaml`). Every publisher and subscriber in
the suite validates against the envelope. The envelope:

```
{
  "id":       "<uuid>",
  "ts":       "<rfc3339>",
  "source":   "<component-name>",
  "kind":     "<event/request kind>",
  "traceId":  "<distributed-trace-id>",
  "payload":  { ... }
}
```

Bumps to the schema use semver inside `schemas/VERSION`; consumers pin to a
version and CI runs contract tests on bumps.

## Deployment shape (future)

When `zelosai`'s operator lands, a `Backplane` CR will own the runtime:

- `dgx-single` profile → NATS StatefulSet, `replicas: 1`, local-path PVC.
- `k8s-multi` profile → NATS cluster, `replicas: 3`, JetStream + cluster storage.

For now, the repo just ships the connector library + a topic-bootstrap sidecar.

## See also

- [01-async-path.md](../01-async-path.md)
- [00-overview.md](../00-overview.md)
- [06-naming-conventions.md](../06-naming-conventions.md)
