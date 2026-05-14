# Prometheus + Grafana for Observability

## Status

Accepted

## Context and Problem Statement

Fieldstone needs a way for operators to understand the health of a running deployment: are services responding? Are error rates spiking? Is latency degrading? Without metrics, the only signal available is structured logs — useful for debugging individual requests but not for detecting trends or getting paged on degradation.

What observability tooling should be bundled with the platform?

## Decision Drivers

* Operators must be able to see request rates, error rates, and latency across all services from a single dashboard.
* The tooling must be self-hostable — no SaaS dependency.
* The tooling must be embeddable in the same Docker Compose deployment as the application services.
* City IT teams should be able to use it without specialized expertise; the dashboard should work out of the box, not require manual configuration.
* The Go ecosystem has mature, first-party Prometheus client support.

## Considered Options

* **Prometheus + Grafana** — pull-based metrics collection; Grafana for dashboarding; both widely used and self-hostable; both have official Docker images.
* **OpenTelemetry Collector + Jaeger** — vendor-neutral tracing and metrics pipeline; more powerful but significantly more complex to configure.
* **Datadog / New Relic / Grafana Cloud** — managed SaaS observability; excellent product quality but adds a recurring cost and a data-exfiltration dependency that government deployments may not permit.
* **Structured logs only** — already implemented via `log/slog`; acceptable for debugging but gives no aggregate view.

## Decision Outcome

Chosen option: **Prometheus + Grafana**, because both have simple Docker deployments, a mature Go client library, and first-class support for the pull-based scrape model that works well with the existing container-per-service architecture. The Grafana dashboard is provisioned automatically from a JSON file — operators see metrics immediately after starting the `observability` profile with no manual setup.

All nine services expose `GET /metrics` (via `promhttp.Handler()`). A shared `Metrics(service)` chi middleware records two metrics per service:
- `fieldstone_http_requests_total` (counter, labels: service / method / path / status)
- `fieldstone_http_request_duration_seconds` (histogram with standard buckets)

The pre-built Fieldstone Overview dashboard shows request rate, error rate %, p99 latency, p50 latency, and HTTP status breakdown per service.

Prometheus and Grafana run under the `observability` Docker Compose profile so they are opt-in and do not add resource overhead to minimal deployments.

### Positive Consequences

* Zero manual Grafana configuration needed — datasource and dashboard are provisioned from files.
* The `fieldstone_http_requests_total` and latency histograms provide the four golden signals (latency, traffic, errors, saturation) without additional instrumentation.
* Adding new metrics to a service is a single `promauto.NewCounterVec(...)` call — no Grafana changes required to start recording.
* Prometheus query language (PromQL) is available for ad-hoc investigation and alert writing.
* Both Prometheus and Grafana have large communities and extensive documentation.

### Negative Consequences

* Prometheus stores metrics locally — no long-term retention without additional tooling (Thanos, Cortex, or external TSDB). The default retention is 30 days.
* The pull-based model requires Prometheus to reach each service; in a multi-server deployment with network segmentation, this requires firewall rules or a Prometheus federation setup.
* Distributed tracing (correlating a single request across multiple services) is not provided. OpenTelemetry would be needed for that; it is out of scope for the current deployment tier.
* Grafana's anonymous read access is enabled by default in development; production deployments should disable `GF_AUTH_ANONYMOUS_ENABLED` and set a strong `GF_SECURITY_ADMIN_PASSWORD`.

## Pros and Cons of the Options

### OpenTelemetry Collector + Jaeger

* Good, because it is vendor-neutral and provides both metrics and distributed traces in one pipeline.
* Bad, because the OTel Collector configuration (receivers, processors, exporters) has a steep learning curve.
* Bad, because adding OTel instrumentation to Go code requires replacing the existing `log/slog` structured logging with the OTel SDK.
* Neutral, because OTel is the right long-term answer for a multi-service platform at scale; it is deferred, not ruled out.

### Managed SaaS (Datadog, New Relic, Grafana Cloud)

* Good, because managed services eliminate operational burden entirely.
* Bad, because government IT policies often prohibit sending operational data to third-party services.
* Bad, because recurring SaaS costs conflict with the self-hostable, city-controlled design principle.
