# Architecture

Fieldstone is a set of independently deployable Go services sharing a single PostgreSQL
instance and NATS JetStream message bus.

## Service map

```
Public internet
       │
  ┌────▼────┐
  │  Caddy  │  TLS termination, reverse proxy
  └────┬────┘
       │
  ┌────▼────┐
  │ Gateway │  JWT validation, Redis rate limiting, routing
  └────┬────┘
       │  HTTP (sync)
   ┌───┴──────────────────┐
   │                      │
┌──▼──┐  ┌────────┐  ┌────▼───┐
│Permits│ │Requests│  │Records │  Domain services
└──┬──┘  └───┬────┘  └───┬────┘
   └──────────┴───────────┘
              │ NATS JetStream (async, InterestPolicy)
        ┌─────┴──────────────┐
   ┌────▼────┐  ┌────▼────┐  ┌──────────┐  ┌────────┐
   │  Audit  │  │Webhooks │  │ Identity │  │Workflow│
   └─────────┘  └─────────┘  └──────────┘  └────────┘
```

## Infrastructure layer

```
All services
     │
  ┌──▼──────────┐
  │  PgBouncer  │  Transaction pool mode — 20 server conns per service pool
  └──┬──────────┘
     │
  ┌──▼──────────┐     ┌────────────┐     ┌──────────────┐
  │ PostgreSQL  │     │   Redis    │     │     NATS     │
  │  (schemas)  │     │ (rate lmt) │     │  JetStream   │
  └─────────────┘     └────────────┘     └──────────────┘

  ┌──────────────┐     ┌───────────┐
  │  Prometheus  │◀────│ /metrics  │  (all 9 services scrape endpoint)
  └──────┬───────┘     └───────────┘
         │
  ┌──────▼───────┐
  │   Grafana    │  Fieldstone Overview dashboard (auto-provisioned)
  └──────────────┘
```

## Data isolation

Services connect to PostgreSQL through PgBouncer. Each service uses a distinct
PgBouncer "database" name that maps to the real database with the correct `search_path`
set server-side. Services never reference another service's schema in queries.

| Service  | PgBouncer DB name | PostgreSQL search_path |
|----------|--------------------|------------------------|
| identity | identity           | identity,public        |
| permits  | permits            | permits,public         |
| requests | requests           | requests,public        |
| records  | records            | records,public         |
| audit    | audit              | audit,public           |
| webhooks | webhooks           | webhooks,public        |

## Event flow

1. HTTP request arrives at a domain service handler
2. Handler writes to its database (commits transaction)
3. After commit, handler puts event on a buffered channel (size 1000)
4. Background goroutine drains the channel, publishing to NATS JetStream
5. Failed NATS publish is logged at ERROR but does NOT fail the HTTP request
6. Subscribers (audit, webhooks, notify, extensions) receive events **independently**
   — the stream uses `InterestPolicy`, so each durable consumer gets every event

> **Upcoming**: the buffered channel will be replaced with a transactional outbox
> pattern so audit events are durable even across process crashes.

## Workflow validation

Before any status change, domain services call the workflow service:

```
POST /v1/workflow/:resource_type/validate
{"from": "submitted", "to": "under_review", "role": "reviewer"}
→ 200 if allowed, 422 if not
```

Status transition logic lives in YAML config, not in domain service code.

## Rate limiting

The gateway rate-limits all public endpoints (citizen-facing routes) using a
Redis-backed sliding-window algorithm. Each gateway replica shares state via Redis,
so the limit is consistent regardless of replica count. If Redis is unavailable,
the limiter fails open (requests are allowed through) and logs an error.

## Observability

Every service exposes `GET /metrics` (Prometheus format). The `observability`
Docker Compose profile starts Prometheus (scrapes all services every 15s) and
Grafana (auto-provisioned Fieldstone Overview dashboard). In development, Grafana
is on host port 3001 and Prometheus on 9090.

Key metrics:
- `fieldstone_http_requests_total` — request count by service/method/path/status
- `fieldstone_http_request_duration_seconds` — latency histogram (p50, p99 visible in Grafana)
