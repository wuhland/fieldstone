# Fieldstone Extension Template

Copy this directory to create a new extension service for Fieldstone.

## What extensions are for

Extensions react to Fieldstone events without modifying core services. Use them to:
- Sync data to external systems (GIS, CRM, legacy databases)
- Send notifications via custom channels
- Generate reports
- Enforce custom business rules

## Getting started

1. Copy this directory: `cp -r extensions/template extensions/my-extension`
2. Edit `main.go` — subscribe to the events you need
3. Build and deploy alongside Fieldstone via Docker Compose

## Event subscription

Subscribe to any NATS subject using JetStream. Use a unique durable consumer name per extension:

```go
js.Subscribe("fieldstone.permits.permit.created",
    handler,
    nats.Durable("my-extension-permits"),
    nats.ManualAck(),
)
```

Common subjects:
- `fieldstone.permits.permit.created`
- `fieldstone.permits.permit.status_changed`
- `fieldstone.requests.service_request.created`
- `fieldstone.requests.service_request.closed`
- `fieldstone.records.foia_request.created`
- `fieldstone.identity.user.provisioned`
- `fieldstone.>` — all events

See [docs/extensions.md](../../docs/extensions.md) for the complete subject catalog.

## Calling core services

Extensions can call core services via HTTP. Service URLs are on the Docker network:

```go
resp, err := http.Get("http://permits:8081/v1/permits/" + permitID)
```

**Contract**: extensions must NOT write directly to core service database schemas.
Use the HTTP API only.

## Adding to Docker Compose

Add to your city's `docker-compose.yml`:

```yaml
services:
  my-extension:
    build: ./extensions/my-extension
    environment:
      NATS_URL: nats://nats:4222
      EXTENSION_ADDR: :9000
    depends_on:
      nats:
        condition: service_healthy
    restart: unless-stopped
```
