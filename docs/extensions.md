# Fieldstone Extension Guide

Fieldstone is designed to be extended by cities without forking the core codebase.
There are four extension surfaces, each suited to different integration needs.

## Layer 1: Metadata Schema Extension

Every major entity (permits, service requests, FOIA requests) has a `metadata` field
that cities can populate with custom data. The schema for valid metadata is defined
per-resource-type and stored in the database.

### Registering a schema

```bash
curl -X PUT https://your-city.gov/v1/config/schemas/permit \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object",
    "properties": {
      "historic_district": {"type": "boolean"},
      "contractor_license": {"type": "string", "pattern": "^[A-Z]{2}-[0-9]+$"}
    }
  }'
```

When a permit is submitted, Fieldstone validates the `metadata` field against this
schema. Invalid metadata returns 422 with field-level error messages.

### Supported resource types

- `permit`
- `service_request`
- `foia_request`

## Layer 2: Configurable Workflow Engine

Status transitions for permits, service requests, and FOIA requests are defined in
YAML files mounted into the workflow service. Edit these to match your city's process.

### Workflow config location

`/etc/fieldstone/workflows/*.yaml` (mounted from `config/workflows/` in the repo)

### Format

```yaml
resource_type: permit
initial_status: submitted
statuses:
  - name: submitted
    label: "Submitted"
    terminal: false
  - name: approved
    label: "Approved"
    terminal: true
transitions:
  - from: submitted
    to: approved
    roles: [admin]
    notify_event: fieldstone.permits.permit.approved
```

Changes take effect on workflow service restart (no redeploy of domain services).

### Roles

Standard roles: `admin`, `reviewer`, `staff`, `system`

The `system` role is used for automated transitions (e.g., permit expiry).

## Layer 3: NATS Event Bus

Every significant state change publishes to NATS JetStream on the `FIELDSTONE` stream.
Extension services subscribe to the events they care about.

### Subject catalog

| Subject | Payload | Description |
|---------|---------|-------------|
| `fieldstone.permits.permit.created` | Permit object | New permit submitted |
| `fieldstone.permits.permit.status_changed` | `{permit, from, to}` | Status transition |
| `fieldstone.permits.inspection.scheduled` | Inspection object | Inspection scheduled |
| `fieldstone.requests.service_request.created` | ServiceRequest object | New 311 request |
| `fieldstone.requests.service_request.assigned` | `{request, assigned_to}` | Assigned to staff |
| `fieldstone.requests.service_request.closed` | ServiceRequest object | Request closed |
| `fieldstone.records.foia_request.created` | FOIARequest object | New FOIA request |
| `fieldstone.records.foia_request.status_changed` | `{request, from, to}` | Status transition |
| `fieldstone.identity.user.provisioned` | User object | New staff user created |

### Event envelope

All events use a common envelope:

```json
{
  "id": "uuid-v4",
  "occurred_at": "2024-01-15T10:30:00Z",
  "source_service": "permits",
  "event_type": "fieldstone.permits.permit.created",
  "payload": { ... },
  "actor": {
    "user_id": "uuid",
    "email": "staff@city.gov",
    "ip_address": "1.2.3.4",
    "request_id": "uuid"
  },
  "schema_version": 1
}
```

### Writing an extension service

See `extensions/template/` for a working example. The key steps:

1. Connect to NATS using the same `NATS_URL` as core services
2. Create a durable JetStream consumer with a unique name
3. Subscribe to subjects you care about
4. Parse the `EventEnvelope` — copy the struct, do not import from core
5. Implement your logic in the handler; call `msg.Ack()` when done

Extension services discover Fieldstone by connecting to the shared NATS instance.
They are not registered anywhere — the event bus is fire-and-forget broadcast.

## Layer 4: Webhooks

For external systems that can't run a NATS consumer, Fieldstone delivers events via HTTP.

### Registering a webhook

```bash
curl -X POST https://your-city.gov/v1/webhooks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://gis.city.gov/fieldstone-events",
    "secret": "whsec_your-random-secret-here",
    "events": ["fieldstone.permits.permit.*"],
    "description": "Sync new permits to GIS system"
  }'
```

### Verifying webhook signatures

Every delivery includes `X-Fieldstone-Signature: sha256=<hmac>` computed as:

```
HMAC-SHA256(secret, request_body)
```

Verification example (Python):
```python
import hmac, hashlib

def verify(secret: str, body: bytes, signature: str) -> bool:
    expected = hmac.new(secret.encode(), body, hashlib.sha256).hexdigest()
    return hmac.compare_digest(f"sha256={expected}", signature)
```

### Retry behavior

- Up to 5 retries with exponential backoff: 1s, 2s, 4s, 8s, 16s
- Non-2xx responses are retried
- 10 consecutive failures auto-disables the endpoint (logged at WARN)
- Delivery history available at `GET /v1/webhooks/:id`

## Contract for extension services

- **Do not** write directly to core service database schemas
- **Do not** call internal service ports directly in production (use the gateway)
- **Do** use the HTTP API for any reads or writes to core data
- **Do** use a unique durable consumer name to avoid competing with other extensions
- **Do** call `msg.Ack()` after processing — unacknowledged messages are redelivered
