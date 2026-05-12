# Services Reference

## gateway (port 8080)

Public-facing API gateway. Validates JWTs, rate-limits public endpoints, proxies to internal services.

**Public routes** (no auth, rate-limited):
- `POST /v1/requests` — submit a 311 service request
- `GET /v1/permits/:id/status` — check permit status
- `POST /v1/records/foia` — submit a FOIA request
- `GET /health`

**Staff routes** (JWT required):
All other `/v1/*` paths.

## identity (port 8084)

Staff user management, department config, custom field schemas.

Routes: `GET/POST /v1/departments`, `GET/POST /v1/users`, `GET /v1/users/me`,
`GET/PUT /v1/config/schemas/:resource_type`

## permits (port 8081)

Permit applications and inspections.

Routes: `GET/POST /v1/permits`, `GET /v1/permits/:id`,
`PATCH /v1/permits/:id/status`, `POST /v1/permits/:id/inspections`,
`PATCH /v1/permits/:id/inspections/:iid`

## requests (port 8082)

311 service requests.

Routes: `GET/POST /v1/requests`, `GET /v1/requests/:id`,
`PATCH /v1/requests/:id/status`, `PATCH /v1/requests/:id/assign`

## records (port 8083)

FOIA request tracking.

Routes: `GET/POST /v1/records/foia`, `GET /v1/records/foia/:id`,
`PATCH /v1/records/foia/:id/status`

## workflow (port 8085)

Configurable state machine engine. Reads YAML configs. Has no database.

Routes: `GET /v1/workflow/:resource_type/statuses`,
`GET /v1/workflow/:resource_type/transitions`,
`POST /v1/workflow/:resource_type/validate`,
`GET /v1/workflow/:resource_type/initial`

## webhooks (port 8086)

HTTP webhook delivery. Subscribes to NATS, fans out to registered endpoints.

Routes: `GET/POST /v1/webhooks`, `GET/DELETE /v1/webhooks/:id`,
`POST /v1/webhooks/:id/test`

## audit (port 8087)

Subscribes to all NATS events and persists them. Immutable audit log.

Routes: `GET /v1/audit`, `GET /v1/audit/:id`

## notify (port 8088)

Stub service for future email/SMS. Currently logs all events and takes no action.
