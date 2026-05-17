# Services Reference

All services expose `GET /health` and `GET /metrics` (Prometheus format).

## gateway (port 8080)

Public-facing API gateway. Validates JWTs (or bypasses validation when
`DEV_DISABLE_AUTH=true`), rate-limits public endpoints using Redis-backed
sliding-window counters, and proxies to internal services.

**Truly public route** (no auth required, rate-limited):
- `GET /v1/permits/{id}/status` — check permit status without authenticating

**Resident-capable routes** (resident or staff JWT required, rate-limited):
- `POST /v1/requests` — submit a 311 service request
- `GET /v1/requests/{id}` — residents read own submission; staff read any
- `POST /v1/records/foia` — submit a FOIA request
- `GET /v1/records/foia/{id}` — residents read own submission; staff read any
- `POST /v1/permits` — submit a permit application
- `GET /v1/permits/{id}` — residents read own submission; staff read any

When `RESIDENT_OIDC_ISSUER_URL` is not configured, these routes accept only staff
JWTs — residents cannot self-submit until a resident OIDC provider is configured.

**Staff-only routes** (staff JWT required):
All other `/v1/*` paths — list endpoints, status/assignment management, identity,
audit, webhooks, workflow.

**Configuration**: `REDIS_URL`, `RATE_LIMIT_PER_MIN` (default 100), `DEV_DISABLE_AUTH`, `RESIDENT_OIDC_ISSUER_URL`

## identity (port 8084)

Staff user management, department configuration, and custom field schema registry.
This is the critical path for Layer 1 extensibility: `GET /v1/config/schemas/:resource_type`
is called by permits, requests, and records to validate metadata on every write, and
by the frontend `DynamicMetadataForm` component to render city-specific custom fields.

**Departments**
- `GET /v1/departments` — list all departments
- `POST /v1/departments` — create department `{name, slug, config?}`

**Staff users**
- `GET /v1/users` — list users; filter with `?department_id=`
- `GET /v1/users/me` — current user from JWT (returns synthetic dev user when `DEV_DISABLE_AUTH=true`)
- `POST /v1/users` — provision user `{department_id, oidc_sub, email, role}` (roles: admin/reviewer/staff)

**Field schemas** (Layer 1 extensibility)
- `GET /v1/config/schemas/{resource_type}` — returns `{id, resource_type, schema, ...}` or 404
- `PUT /v1/config/schemas/{resource_type}` — body is a raw JSON Schema document; idempotent upsert

## permits (port 8081)

Permit applications and inspections. Status transitions validated through the workflow
service before any change. Metadata validated against the registered schema for `permit`.

- `GET /v1/permits` — paginated list; filter with `?status=`
- `POST /v1/permits` — create permit `{permit_type, department_id, applicant, property_address, metadata?}`
- `GET /v1/permits/{id}` — permit detail including inspections
- `PATCH /v1/permits/{id}/status` — `{status, role}`; calls workflow/validate; sets `issued_at` on approval
- `POST /v1/permits/{id}/inspections` — schedule inspection `{inspector_id, scheduled_at}`
- `PATCH /v1/permits/{id}/inspections/{iid}` — update inspection `{completed_at?, result?, notes?}`

Events published: `permit.created`, `permit.status_changed`, `inspection.scheduled`

## requests (port 8082)

311 service requests. `POST /v1/requests` requires a resident or staff JWT; the
authenticated `sub` claim is stored as `resident_id` so residents can retrieve their
own submissions. Staff management routes require a staff JWT. Metadata validated
against the `service_request` schema.

- `GET /v1/requests` — paginated list; filter with `?status=`
- `POST /v1/requests` — resident or staff JWT required; `{request_type, department_id, description, submitter_email?, location?, metadata?}`
- `GET /v1/requests/{id}` — single request
- `PATCH /v1/requests/{id}/status` — `{status, role}`; terminal statuses set `closed_at`
- `PATCH /v1/requests/{id}/assign` — `{assigned_to, role}`; validates `open→assigned` via workflow

Events published: `service_request.created`, `service_request.assigned`, `service_request.closed`

## records (port 8083)

FOIA request tracking. `POST /v1/records/foia` requires a resident or staff JWT; the
authenticated `sub` claim is stored as `resident_id`. Metadata validated against
the `foia_request` schema. `due_date` is a `DATE` field serialized as `"YYYY-MM-DD"`.

- `GET /v1/records/foia` — paginated list; filter with `?status=`
- `POST /v1/records/foia` — resident or staff JWT required; `{department_id, requester_name, requester_email, description, due_date?, metadata?}`
- `GET /v1/records/foia/{id}` — single FOIA request
- `PATCH /v1/records/foia/{id}/status` — `{status, role}`; terminal statuses (`fulfilled`, `denied`, `withdrawn`) set `closed_at`

Events published: `foia_request.created`, `foia_request.status_changed`

## workflow-worker (port 8085)

Temporal worker and workflow configuration HTTP server. Reads YAML configs from
`/etc/fieldstone/workflows/` at startup. Connects to the Temporal server to run
workflow and activity functions.

**Temporal responsibilities**
- Registers `PermitWorkflow`, `ServiceRequestWorkflow`, `FOIARequestWorkflow` on
  the `fieldstone` task queue.
- Each workflow function validates staff transitions via the `validate-transition`
  Update handler, handles resident `withdraw` signals, and tracks lifecycle until
  a terminal state is reached.
- Activities (called from signal/timer handlers) write to domain DBs and publish
  events via the outbox. Activities are retried automatically on failure.

**HTTP endpoints** (same interface as the retired `workflow` service; used by the
gateway proxy and domain service clients for config queries and fallback validation):
- `GET /v1/workflow/{resource_type}/statuses`
- `GET /v1/workflow/{resource_type}/transitions`
- `GET /v1/workflow/{resource_type}/config` — full WorkflowConfig (used by domain
  clients when starting new workflow executions)
- `POST /v1/workflow/{resource_type}/validate` — fallback for pre-Temporal records
- `GET /v1/workflow/{resource_type}/initial`

Default resource types: `permit`, `service_request`, `foia_request`

**Configuration**: `TEMPORAL_HOST`, `WORKFLOWS_DIR`, `PERMITS_DATABASE_DSN`,
`REQUESTS_DATABASE_DSN`, `RECORDS_DATABASE_DSN`

## webhooks (port 8086)

HTTP webhook delivery service. Subscribes to every `fieldstone.>` NATS event and
fans out to registered endpoints whose event patterns match the incoming subject.

**Dispatch behaviour**
- The NATS message is acked immediately; delivery happens in a goroutine so a slow
  or retrying endpoint never blocks event processing.
- Up to 5 retries with exponential backoff (1 s, 2 s, 4 s, 8 s, 16 s) on non-2xx.
- Every POST carries `X-Fieldstone-Signature: sha256=<hmac>` and `X-Fieldstone-Event`.
- Each delivery attempt is recorded in `webhooks.deliveries` (status code, duration,
  error). `GET /v1/webhooks/{id}` returns the last 100 delivery records.
- `fail_count` is incremented on failure and reset to 0 on success. An endpoint
  is automatically disabled and logged at WARN when `fail_count` reaches 10.

**Event pattern matching**
Patterns support NATS-style wildcards:
- `*` matches exactly one subject token: `fieldstone.permits.permit.*` matches
  `fieldstone.permits.permit.created` but not `fieldstone.permits.permit.created.sub`
- `>` matches all remaining tokens: `fieldstone.permits.>` matches everything under
  `fieldstone.permits.`

**Routes**
- `GET /v1/webhooks` — list registered endpoints (secret never returned after creation)
- `POST /v1/webhooks` — register `{url, secret, events[], description?}`; secret
  shown once in the 201 response
- `GET /v1/webhooks/{id}` — endpoint detail + last 100 delivery records
- `DELETE /v1/webhooks/{id}` — remove endpoint and its delivery history; 204
- `POST /v1/webhooks/{id}/test` — dispatch a synthetic `fieldstone.webhooks.test`
  event synchronously and return the event ID

**Secret storage note**: the signing secret is currently stored as plaintext in the
database for HMAC computation (bcrypt is one-way and cannot be used for signing).
The production upgrade path is AES-256-GCM encryption at rest; see ADR-0020.

## audit (port 8087)

Immutable audit log. Subscribes to every `fieldstone.>` NATS event (durable consumer
`audit-service`) and persists each one to `audit.events`. The table is range-partitioned
by `occurred_at` with monthly partitions (2025–2030).

**Persistence guarantees**
- Nak on decode or DB error causes JetStream to redeliver — at-least-once persistence.
- `INSERT ... ON CONFLICT (id, occurred_at) DO NOTHING` makes re-delivery idempotent.
- Combined with the transactional outbox in domain services (ADR-0018), the audit log
  contains a record of every business action even if a service crashed immediately
  after its DB commit.

**Routes**
- `GET /v1/audit` — paginated event list; all filters are optional and combinable:
  - `?event_type=fieldstone.permits.permit.created`
  - `?source_service=permits`
  - `?from=2026-05-01T00:00:00Z` (RFC3339)
  - `?to=2026-05-31T23:59:59Z` (RFC3339)
  - `?limit=50&offset=0`
- `GET /v1/audit/{id}` — single event by UUID (scans all partitions)

## notify (port 8088)

Unimplemented stub. Subscribes to all events and logs them; no messages are sent.

Resident notification (email on permit approval, FOIA fulfillment, etc.) is not
built into the core platform. The supported pattern today is to register a webhook
and route events to the city's existing notification infrastructure or a no-code
integration tool (Zapier, Make). See [docs/extensions.md](extensions.md) for webhook
setup and event subject patterns.
