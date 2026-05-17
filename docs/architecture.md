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

  ┌────────────────┐     ┌───────────┐
  │   Prometheus   │◀────│ /metrics  │  (all services scrape endpoint)
  └──────┬─────────┘     └───────────┘
  ┌────────────────┐
  │  Temporal      │  Durable workflow execution (PostgreSQL backend)
  │  (+ UI :8233)  │
  └────────────────┘
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
2. Handler opens a transaction, writes to its database, and writes an outbox row in
   the same transaction (ADR-0018)
3. Transaction commits — both the domain write and the outbox row are durable
4. Background relay reads pending outbox rows and publishes to NATS JetStream
5. Failed NATS publish is retried; the row is not deleted until publish succeeds
6. Subscribers (audit, webhooks, notify, extensions) receive events **independently**
   — the stream uses `InterestPolicy`, so each durable consumer gets every event

## Workflow execution

Every resource (permit, service request, FOIA request) has a durable Temporal workflow
execution that tracks its lifecycle from creation to terminal state.

**Staff transitions** use Temporal Updates — synchronous, validated against the YAML
config baked into the workflow's input at creation time:

```
temporalClient.UpdateWorkflow(ctx, "permit-<id>", "validate-transition", {from, to, role})
→ returns nil if allowed, error if not
```

**Resident actions** (withdrawal) use Temporal Signals — the signal fires an activity
that writes to the domain DB and publishes the event. The domain handler is not involved.

**Automated transitions** (permit expiry, FOIA deadline) use Temporal timers that fire
activities when the deadline is reached.

For resources that predate Temporal adoption, status updates fall back to HTTP
validation against the workflow-worker's `/v1/workflow/{type}/validate` endpoint.

The `workflow-worker` service also serves the `/v1/workflow/*` HTTP endpoints
(statuses, transitions, initial status) so the gateway and domain clients can query
YAML-defined workflow config without parsing it themselves.

Status transition logic lives in `config/workflows/*.yaml`, not in Go code.

## Identity model

Fieldstone uses two distinct authentication tiers, both backed by standard OIDC.
Fieldstone is an OIDC **relying party** — it validates tokens, never issues them.

| Tier | Who | Typical provider | JWT issuer env var |
|------|-----|------------------|--------------------|
| Staff | City employees | City's OIDC provider (Okta, Azure AD, etc.) | `OIDC_ISSUER_URL` |
| Residents | Citizens | Login.gov or any OIDC provider | `RESIDENT_OIDC_ISSUER_URL` |

**Staff** accounts are provisioned in the identity service and backed by the city's
enterprise OIDC provider. Staff JWTs carry `roles` claims (admin, reviewer, staff).

**Residents** authenticate through any OIDC provider the city configures. For US
municipalities, Login.gov is the recommended default — it is GSA-operated, privacy-focused,
and purpose-built for government services. Internationally, GOV.UK One Login or any
OIDC-compliant provider works. Because Fieldstone only depends on the OIDC protocol,
cities can also self-host an identity provider (Keycloak, Authentik) for full
sovereignty.

The gateway detects which tier a token belongs to by inspecting the `iss` claim and
validating against the corresponding JWKS endpoint. Resident tokens receive the
synthetic role `"resident"` regardless of the token's own claims. The gateway then
injects `X-Fieldstone-Role` and `X-Fieldstone-Sub` headers into the proxied request
so domain services can apply row-level access control without re-parsing the JWT.

**Row-level access**: residents can create and read their own submissions; all write
operations (status changes, assignment) require staff.

## Scope boundaries

Fieldstone tracks and routes civic service submissions through their lifecycle.
It deliberately excludes:

- **Document storage** — attachments and produced records (e.g., the documents
  returned in response to a FOIA request) are not stored; Fieldstone tracks the
  request, not the files.
- **Payments** — permit fees, fines, and processing charges are out of scope.
- **Scheduling/calendar** — inspection scheduling stores a `scheduled_at` timestamp;
  calendar invites and scheduling UI are not included.
- **GIS / spatial queries** — location is stored as a freetext field; spatial
  indexing and map integration are handled by an extension service consuming events.
- **Resident notification** — the `notify` service is an unimplemented stub. The
  supported pattern is to register webhooks and route events to the city's existing
  notification infrastructure (or a no-code platform such as Zapier).
- **Resident portal UI** — the bundled frontend is a staff management portal.
  A resident-facing interface can be built on top of the public API.

## Rate limiting

The gateway rate-limits resident-capable routes using a
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
