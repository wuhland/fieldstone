# Shared PostgreSQL Instance with Per-Service Schema Isolation

## Status

Accepted

## Context and Problem Statement

Each Fieldstone service owns its data and must not read or write another service's tables directly. Within a single-tenant deployment, how should this isolation be enforced at the database level? Separate databases per service provide the strongest isolation but require more operational overhead. PostgreSQL schemas offer a middle ground.

## Decision Drivers

* A small IT contractor should be able to deploy and operate Fieldstone without managing multiple database instances.
* Services should not be able to accidentally query each other's tables.
* Each service's migrations should be independently runnable without affecting other services.
* Backup, monitoring, and connection pooling should remain simple.

## Considered Options

* **One PostgreSQL database per service** — full isolation, each service connects to its own database.
* **Shared PostgreSQL with per-service schemas** — one database, one schema per service, each service's DSN sets `search_path=<service_schema>,public`.
* **Single shared schema** — all services use the same schema; isolation is enforced only by convention in application code.

## Decision Outcome

Chosen option: **shared PostgreSQL with per-service schemas**, because it provides meaningful structural isolation — services cannot reference each other's tables in sqlc queries without explicitly naming the schema — while requiring only one PostgreSQL instance to operate and back up.

Each service's `DATABASE_DSN` includes `search_path=<service>,public`. Services never write cross-schema queries. The `infra/postgres/init.sql` creates all schemas at startup.

### Positive Consequences

* One Postgres instance to provision, monitor, and back up.
* A service's `go test` integration tests can target a single schema without starting the whole stack.
* sqlc enforces schema isolation at code generation time: queries in `services/permits/db/query.sql` only see the `permits` schema tables.
* Adding or removing a service only requires creating or dropping one schema.

### Negative Consequences

* Schema-level isolation is not the same as database-level isolation; a sufficiently privileged user can still cross schema boundaries.
* All services share connection limits of one Postgres instance; a misbehaving service can starve others.
* When a service needs to be extracted to a separate deployment, it also needs its data extracted — a harder migration than if it had always had its own database.

## Pros and Cons of the Options

### One PostgreSQL database per service

* Good, because isolation is enforced at the connection level — the credential for the permits service literally cannot reach identity tables.
* Good, because services can use different Postgres versions or extensions independently.
* Bad, because the operator must provision, monitor, tune, and back up N databases instead of one.
* Bad, because Docker Compose would need N postgres containers, significantly increasing baseline resource usage for small deployments.

### Single shared schema

* Good, because zero configuration — one DSN, one schema.
* Bad, because there is nothing preventing `services/permits` from accidentally querying `staff_users` in application code.
* Bad, because migration files from all services are mixed together, making per-service rollback impossible.
