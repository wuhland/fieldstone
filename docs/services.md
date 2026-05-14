# Services Reference

All services expose `GET /health` and `GET /metrics` (Prometheus format).

## gateway (port 8080)

Public-facing API gateway. Validates JWTs (or bypasses validation when
`DEV_DISABLE_AUTH=true`), rate-limits public endpoints using Redis-backed
sliding-window counters, and proxies to internal services.

**Public routes** (no auth, rate-limited to `RATE_LIMIT_PER_MIN` per IP):
- `POST /v1/requests` — submit a 311 service request
- `GET /v1/permits/{id}/status` — check permit status
- `POST /v1/records/foia` — submit a FOIA request

**Staff routes** (JWT required in production):
All other `/v1/*` paths.

**Configuration**: `REDIS_URL`, `RATE_LIMIT_PER_MIN` (default 100), `DEV_DISABLE_AUTH`

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

311 service requests. `POST /v1/requests` is a public endpoint (no auth); all other
routes require staff authentication. Metadata validated against the `service_request` schema.

- `GET /v1/requests` — paginated list; filter with `?status=`
- `POST /v1/requests` — **public**; `{request_type, department_id, description, submitter_email?, location?, metadata?}`
- `GET /v1/requests/{id}` — single request
- `PATCH /v1/requests/{id}/status` — `{status, role}`; terminal statuses set `closed_at`
- `PATCH /v1/requests/{id}/assign` — `{assigned_to, role}`; validates `open→assigned` via workflow

Events published: `service_request.created`, `service_request.assigned`, `service_request.closed`

## records (port 8083)

FOIA request tracking. `POST /v1/records/foia` is public. Metadata validated against
the `foia_request` schema. `due_date` is a `DATE` field serialized as `"YYYY-MM-DD"`.

- `GET /v1/records/foia` — paginated list; filter with `?status=`
- `POST /v1/records/foia` — **public**; `{department_id, requester_name, requester_email, description, due_date?, metadata?}`
- `GET /v1/records/foia/{id}` — single FOIA request
- `PATCH /v1/records/foia/{id}/status` — `{status, role}`; terminal statuses (`fulfilled`, `denied`, `withdrawn`) set `closed_at`

Events published: `foia_request.created`, `foia_request.status_changed`

## workflow (port 8085)

Configurable state machine engine. Reads YAML configs from `/etc/fieldstone/workflows/`
at startup and logs each loaded config. Has no database.

- `GET /v1/workflow/{resource_type}/statuses` — valid statuses
- `GET /v1/workflow/{resource_type}/transitions` — valid transitions with role requirements
- `POST /v1/workflow/{resource_type}/validate` — `{from, to, role}` → 200 or 422
- `GET /v1/workflow/{resource_type}/initial` — initial status for new resources

Default resource types: `permit`, `service_request`, `foia_request`

## webhooks (port 8086)

HTTP webhook delivery. Subscribes to NATS, fans out to registered endpoints with
HMAC-SHA256 signatures and exponential backoff retry.

- `GET/POST /v1/webhooks` — list / register webhook
- `GET/DELETE /v1/webhooks/{id}` — get details + delivery log / remove
- `POST /v1/webhooks/{id}/test` — send a test payload

## audit (port 8087)

Subscribes to all NATS events (`fieldstone.>`) and persists them. The `audit.events`
table is range-partitioned by `occurred_at` (monthly partitions, 2025–2030).

- `GET /v1/audit` — paginated events; filter with `?event_type=`, `?source_service=`
- `GET /v1/audit/{id}` — single event

## notify (port 8088)

Stub service for future email/SMS. Currently subscribes to all events, logs them,
and takes no action.
