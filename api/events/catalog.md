# Fieldstone Event Catalog

All events use the subject format: `fieldstone.<service>.<resource>.<action>`

Events are published to the `FIELDSTONE` JetStream stream on subject `fieldstone.>`.

## Envelope

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "occurred_at": "2024-01-15T10:30:00Z",
  "source_service": "permits",
  "event_type": "fieldstone.permits.permit.created",
  "payload": { ... },
  "actor": {
    "user_id": "uuid",
    "email": "staff@city.gov",
    "ip_address": "10.0.0.1",
    "request_id": "uuid"
  },
  "schema_version": 1
}
```

## Permit events

### `fieldstone.permits.permit.created`
Published when a new permit application is submitted.
Payload: full permit object (id, permit_type, status, applicant, property_address, metadata, submitted_at)

### `fieldstone.permits.permit.status_changed`
Published when a permit's status changes.
Payload: `{"permit": {...}, "from": "submitted", "to": "under_review"}`

### `fieldstone.permits.inspection.scheduled`
Published when an inspection is scheduled.
Payload: full inspection object (id, permit_id, inspector_id, scheduled_at)

## Service request events

### `fieldstone.requests.service_request.created`
Published when a new 311 request is submitted.
Payload: full service_request object

### `fieldstone.requests.service_request.assigned`
Published when a request is assigned to a staff member.
Payload: `{"request": {...}, "assigned_to": "uuid"}`

### `fieldstone.requests.service_request.closed`
Published when a request reaches a terminal state.
Payload: full service_request object

## FOIA request events

### `fieldstone.records.foia_request.created`
Published when a new FOIA request is received.
Payload: full foia_request object

### `fieldstone.records.foia_request.status_changed`
Published on any FOIA status transition.
Payload: `{"request": {...}, "from": "received", "to": "processing"}`

## Identity events

### `fieldstone.identity.user.provisioned`
Published when a new staff user is created.
Payload: `{"id": "uuid", "email": "...", "role": "...", "department_id": "..."}`
