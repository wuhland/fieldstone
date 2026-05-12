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
# Edit .env — at minimum set OIDC_ISSUER_URL and OIDC_AUDIENCE
make dev
```

## Production deployment

1. Set all required env vars in `.env`
2. Point your domain at the server
3. Update `infra/Caddyfile` with your domain
4. Start with the core profile:

```bash
docker compose --profile core up -d
```

5. Optionally enable additional services:

```bash
docker compose --profile core --profile permits --profile records up -d
```

## Profiles

| Profile  | Services included |
|----------|-------------------|
| core     | postgres, nats, gateway, identity, requests, workflow, audit, frontend, caddy |
| permits  | permits service |
| records  | records service |
| webhooks | webhooks service |
| notify   | notify service (stub) |

## Customizing workflows

Edit `config/workflows/*.yaml`, then restart the workflow service:

```bash
docker compose restart workflow
```

No other services need to restart.

## Backups

Back up the PostgreSQL volume:

```bash
docker compose exec postgres pg_dump -U fieldstone fieldstone > backup.sql
```
