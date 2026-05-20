# Architectural Decision Records

This directory contains the architectural decision records (ADRs) for Fieldstone,
written in [MADR](https://adr.github.io/madr/) (Markdown Any Decision Records) format.

## Index

| ID | Title | Status |
|----|-------|--------|
| [0001](0001-single-go-module.md) | Single Go module for all services | Accepted |
| [0002](0002-single-tenant-deployment.md) | Single-tenant deployment model | Accepted |
| [0003](0003-shared-postgres-schema-isolation.md) | Shared PostgreSQL instance with per-service schema isolation | Accepted |
| [0004](0004-nats-jetstream-event-bus.md) | NATS JetStream as the async event bus | Accepted — stream retention corrected to InterestPolicy |
| [0005](0005-http-not-grpc-inter-service.md) | HTTP for inter-service communication, not gRPC | Accepted |
| [0006](0006-sqlc-not-orm.md) | sqlc for type-safe queries, no ORM | Accepted |
| [0007](0007-yaml-driven-workflow-engine.md) | Configurable workflow engine via YAML, not hardcoded transitions | Accepted |
| [0008](0008-jsonb-metadata-extension.md) | JSONB metadata columns with JSON Schema validation for custom fields | Accepted |
| [0009](0009-extension-via-nats-not-plugins.md) | Extension services via NATS subscription, not a plugin system | Accepted |
| [0010](0010-docker-compose-not-kubernetes.md) | Docker Compose for deployment, not Kubernetes | Accepted |
| [0011](0011-caddy-reverse-proxy.md) | Caddy as the reverse proxy and TLS terminator | Accepted |
| [0012](0012-buffered-channel-event-publishing.md) | Buffered channel for decoupled NATS event publishing | Superseded by 0018 |
| [0013](0013-oidc-jwt-staff-authentication.md) | OIDC with JWT validation for staff authentication | Accepted |
| [0014](0014-nextjs-app-router-frontend.md) | Next.js 15 App Router for the frontend | Accepted |
| [0015](0015-pgbouncer-connection-pooling.md) | PgBouncer for PostgreSQL connection pooling | Accepted |
| [0016](0016-prometheus-grafana-observability.md) | Prometheus + Grafana for observability | Accepted |
| [0017](0017-raw-resp-redis-rate-limiter.md) | Zero-dependency Redis client for distributed rate limiting | Accepted |
| [0018](0018-transactional-outbox.md) | Transactional outbox for durable event publishing | Accepted — supersedes 0012 |
| [0019](0019-dynamic-parameterized-queries.md) | Dynamic parameterized queries for optional filters | Accepted |
| [0020](0020-webhook-secret-plaintext-storage.md) | Webhook secret stored as plaintext for HMAC signing | Accepted — documented spec deviation |
| [0021](0021-resident-oidc-public-submission-auth.md) | Resident identity via configurable OIDC for public submissions | Accepted |
| [0022](0022-temporal-durable-workflow.md) | Temporal as durable workflow engine | Accepted — supersedes 0007 |
| [0023](0023-temporal-timers-and-durable-webhooks.md) | Temporal timers and durable webhook delivery | Accepted — extends 0022 |
| [0024](0024-config-file-driven-service-enablement.md) | Config-file-driven service enablement | Accepted |

## Format

Each ADR follows the MADR template:

- **Status**: `Proposed` → `Accepted` (or `Deprecated` / `Superseded by NNNN`)
- **Context**: why this decision was needed
- **Considered Options**: the realistic alternatives
- **Decision Outcome**: what was chosen and why
- **Consequences**: what becomes easier or harder as a result
