# Deployment

## Prerequisites

- Docker Engine 24+
- Docker Compose v2
- A domain name with DNS pointing to your server
- An OIDC provider (Azure AD, Keycloak, Auth0, Okta, etc.)

## Quick start (development)

```bash
git clone https://github.com/fieldstone/fieldstone
cd fieldstone
cp .env.example .env
# Edit .env — fill in OIDC values or leave DEV_DISABLE_AUTH=true for local dev
make dev
```

After startup, check all services are healthy:
```bash
curl http://localhost:8080/health   # gateway
curl http://localhost:8085/health   # workflow (reads YAML configs at startup)
```

## Production deployment

1. Set all required env vars in `.env` (remove `DEV_DISABLE_AUTH=true`, set real OIDC values)
2. Point your domain at the server
3. Update `infra/Caddyfile` with your domain
4. Start with the core profile:

```bash
docker compose --env-file .env -f infra/docker-compose.yml --profile core up -d
```

5. Optionally enable additional services:

```bash
docker compose --env-file .env -f infra/docker-compose.yml \
  --profile core --profile permits --profile records --profile webhooks up -d
```

6. Enable observability (Prometheus + Grafana):

```bash
docker compose --env-file .env -f infra/docker-compose.yml \
  --profile core --profile observability up -d
```

## Profiles

| Profile       | Services included |
|---------------|-------------------|
| core          | postgres, pgbouncer, nats, redis, gateway, identity, requests, workflow, audit, frontend, caddy |
| permits       | permits service |
| records       | records service |
| webhooks      | webhooks service |
| notify        | notify service (stub) |
| observability | prometheus, grafana |

## Infrastructure services

**PgBouncer** sits between application services and PostgreSQL, providing
transaction-mode connection pooling. Services connect to `pgbouncer:5432/{service}`
rather than PostgreSQL directly. In development, PgBouncer is accessible on host
port 5434 for direct DB inspection:

```bash
psql -h localhost -p 5434 -U fieldstone -d identity
```

**Redis** provides distributed rate limiting state shared across gateway replicas.
In development, Redis is accessible on host port 6379.

**NATS JetStream** stores events in a persistent volume (`nats_data`). Events
survive container restarts. The `FIELDSTONE` stream uses `InterestPolicy` — each
durable consumer (audit, webhooks) independently receives every event.

## Observability

Once the observability profile is running:
- **Prometheus**: http://localhost:9090 (in dev)
- **Grafana**: http://localhost:3001 (in dev) — login admin / `GRAFANA_PASSWORD` (default: `fieldstone`)

The Fieldstone Overview dashboard is auto-provisioned. No manual Grafana setup needed.

Set `GRAFANA_PASSWORD` in `.env` before production deployment.

## Customizing workflows

Edit `config/workflows/*.yaml`, then restart the workflow service:

```bash
docker compose --env-file .env -f infra/docker-compose.yml restart workflow
```

No other services need to restart.

## First-time database setup

The `audit.events` table requires PostgreSQL partitioning. If you are upgrading
from an earlier version that created the table without partitioning, the
`002_partition_events.sql` migration handles the conversion automatically on
the next audit service startup. No manual intervention is needed for fresh installs.

If you need to reset the database volume entirely (e.g., during development):

```bash
docker volume rm infra_postgres_data
make dev
```

## Backups

Back up the PostgreSQL volume:

```bash
docker compose --env-file .env -f infra/docker-compose.yml \
  exec postgres pg_dump -U fieldstone fieldstone > backup.sql
```

For production, automate this with a scheduled job and test restores regularly.
PostgreSQL's point-in-time recovery (PITR) via WAL archiving is recommended for
any deployment that cannot tolerate data loss between daily backups.
