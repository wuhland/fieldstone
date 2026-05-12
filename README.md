# Fieldstone

A self-hostable civic services platform that cities can deploy to run common government services. Named for the unglamorous, load-bearing material cities are actually built on.

## What it does

Fieldstone provides the backend infrastructure for common municipal services:

- **Permit management** — applications, inspections, status tracking
- **311 service requests** — potholes, violations, streetlight outages
- **Public records (FOIA)** — request tracking and status updates
- **Staff identity** — users, roles, departments, OIDC integration
- **Audit logging** — immutable record of every state change
- **Webhooks** — HTTP delivery of events to external systems

## Design principles

- **Single-tenant** — one deployment per city, not SaaS multi-tenancy
- **Extensible** — cities customize via config and extension services, never by forking
- **Simple to operate** — Docker Compose deployment, no Kubernetes required
- **Opinionated** — Go, PostgreSQL, NATS, chi; not configurable per-language

## Quick start

```bash
cp .env.example .env
# Edit .env with your OIDC provider and database credentials
docker compose --profile core up
```

After startup:
- Gateway: http://localhost:8080
- `GET /health` on all services returns `{"status":"ok"}`

## Architecture

```
                    ┌─────────┐
         citizen ──▶│ Caddy   │◀── staff browser
                    └────┬────┘
                         │
                    ┌────▼────┐
                    │ Gateway │  JWT validation, rate limiting, routing
                    └────┬────┘
           ┌─────────────┼──────────────┐
    ┌──────▼──────┐ ┌────▼────┐ ┌───────▼──────┐
    │   Permits   │ │Requests │ │   Records    │
    └─────────────┘ └─────────┘ └──────────────┘
           │              │             │
           └──────────────▼─────────────┘
                     NATS JetStream
                    ┌──────┴──────┐
              ┌─────▼─────┐ ┌────▼────┐
              │   Audit   │ │Webhooks │
              └───────────┘ └─────────┘
```

Services communicate via:
- **Sync**: HTTP (queries between services)
- **Async**: NATS JetStream (events for audit, webhooks, extensions)

## Extending Fieldstone

Cities extend Fieldstone through four layers without forking:

1. **Metadata schemas** — add custom fields to any resource via JSON Schema
2. **Workflow config** — edit YAML to change status transitions
3. **NATS subscribers** — write extension services that react to events
4. **Webhooks** — register HTTP endpoints to receive events (no NATS required)

See [docs/extensions.md](docs/extensions.md) for the complete guide.

## Services

| Service   | Port  | Description                        |
|-----------|-------|------------------------------------|
| gateway   | 8080  | Public API gateway                 |
| identity  | 8084  | Staff users, departments, schemas  |
| permits   | 8081  | Permit applications, inspections   |
| requests  | 8082  | 311 service requests               |
| records   | 8083  | FOIA request tracking              |
| workflow  | 8085  | Configurable state machine         |
| webhooks  | 8086  | HTTP webhook delivery              |
| audit     | 8087  | Immutable audit log                |
| notify    | 8088  | Email/SMS stub (future)            |

## Development

```bash
make dev          # Start all services with dev overrides
make test         # Run unit tests
make test-race    # Run with race detector
make lint         # Run golangci-lint
make generate     # Run sqlc generate
```

Requirements: Go 1.22+, Docker, Docker Compose

## Documentation

- [Architecture](docs/architecture.md)
- [Deployment](docs/deployment.md)
- [Development](docs/development.md)
- [Extensions](docs/extensions.md)
- [Services](docs/services.md)

## License

MIT — see [LICENSE](LICENSE)
