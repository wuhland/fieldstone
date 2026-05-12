# Development

## Prerequisites

- Go 1.22+
- Docker & Docker Compose (for integration tests and local dev)
- golangci-lint
- golang-migrate CLI

## Running locally

```bash
cp .env.example .env
make dev
```

Services start with `LOG_LEVEL=debug` and internal ports exposed.

## Project structure

```
fieldstone/
├── internal/          # Shared packages (single Go module)
│   ├── auth/          # JWT validation, JWKS cache
│   ├── config/        # BaseConfig embedded by all services
│   ├── db/            # DB connection helper
│   ├── events/        # EventEnvelope, NATS subject constants
│   ├── middleware/     # chi middleware: RequestID, Logger, Recovery, AuditEmit
│   ├── nats/          # NATS+JetStream connection helper
│   └── validate/      # JSON Schema validation with TTL cache
├── services/          # One directory per service
│   ├── gateway/
│   ├── identity/
│   ├── permits/
│   └── ...
├── config/workflows/  # Workflow YAML configs (mounted into workflow service)
├── extensions/template/  # Starting point for extension services
└── frontend/          # Next.js staff portal
```

## Adding a new service

1. Create `services/<name>/` following the startup pattern in any existing service
2. Add `<NAME>_DATABASE_DSN` to `.env.example`
3. Create `services/<name>/db/migrations/001_initial.sql`
4. Add to `infra/docker-compose.yml` with appropriate profile
5. Add proxy route to `services/gateway/main.go`

## Testing

```bash
make test                  # unit tests
make test-race             # race detector
make test-integration      # needs Docker (testcontainers)
```

Integration tests use testcontainers-go to spin up real Postgres and NATS instances.
Tag them with `//go:build integration` and `// +build integration`.

## Code conventions

- sqlc only — no ORM, no fmt.Sprintf into SQL
- errors.Is / fmt.Errorf with %w for error wrapping
- slog for all logging — JSON in prod, text in dev
- cleanenv for config — fail fast on missing required vars
- No global state — dependencies passed as constructor arguments
