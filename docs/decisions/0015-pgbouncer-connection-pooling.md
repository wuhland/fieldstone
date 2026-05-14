# PgBouncer for PostgreSQL Connection Pooling

## Status

Accepted

## Context and Problem Statement

Each Fieldstone service maintains a `pgxpool` connection pool to PostgreSQL. With the default pool size of a few connections per service, a single-server deployment is fine. But when services are scaled to multiple replicas — or when the number of deployed services grows — the total number of PostgreSQL connections can exhaust the server's `max_connections` limit (typically 100–200). PostgreSQL forks a process per connection; too many connections degrades performance before they're even used.

How should connection exhaustion be prevented without requiring per-service configuration changes or a complete re-architecture of the data layer?

## Decision Drivers

* Connection exhaustion must be prevented when running multiple service replicas.
* The solution must require minimal changes to service code — ideally none.
* Each service's schema isolation (via `search_path`) must be preserved.
* The pooler must be operationally simple: no separate cluster, no custom protocol.
* Works with pgx v5's `pgxpool`, which already pools connections per service instance.

## Considered Options

* **PgBouncer** — lightweight connection pooler; transaction pool mode multiplexes many client connections onto few server connections; widely deployed.
* **pgpool-II** — more capable (load balancing, replication); significantly more complex to configure.
* **No pooler; reduce pgxpool size** — tune down `DefaultPoolSize` per service; doesn't help when the number of replicas grows.
* **Pgx connection reuse without a pooler** — already doing this; doesn't solve multi-replica exhaustion.

## Decision Outcome

Chosen option: **PgBouncer in transaction pool mode**, because it is the industry standard for this specific problem, requires zero changes to application SQL, and adds only one container and a small config file to the deployment.

PgBouncer sits between services and PostgreSQL. Each service uses a distinct PgBouncer "database" name (e.g., `pgbouncer:5432/permits`) that maps to the real database with the correct `search_path` override set as a server-side connection option. Services no longer carry `?search_path=...` in their DSNs — the pooler enforces it.

```
Service replica A ─┐
Service replica B ─┼─▶ PgBouncer (transaction pool) ─▶ PostgreSQL
Service replica C ─┘
```

Transaction pool mode is appropriate because Fieldstone services do not use session-level features (prepared statements, advisory locks, `SET LOCAL` outside transactions) that would break under connection sharing.

### Positive Consequences

* 1,000 client connections map to 20 server connections per service pool — PostgreSQL sees at most `20 × N_services` connections regardless of replica count.
* Zero application code changes: services connect to `pgbouncer:5432/{service}` instead of `postgres:5432/fieldstone?search_path=...`.
* `search_path` isolation is now enforced by PgBouncer server-side rather than the DSN query parameter, which is more reliable in transaction pool mode where server connections are shared.
* Auth credentials stay in `.env`; PgBouncer generates `userlist.txt` at startup from env vars.

### Negative Consequences

* PostgreSQL session-level features (e.g., `SET` commands that persist across transactions, `LISTEN/NOTIFY`) do not work in transaction pool mode. Fieldstone does not currently use these, but future features must avoid them or switch to session pool mode.
* PgBouncer is an additional container in the dependency chain. Services now depend on PgBouncer health, which depends on PostgreSQL health.
* PgBouncer is a single point of failure unless it is run in HA mode (two instances with a virtual IP or a load balancer). Single-instance PgBouncer is acceptable for a single-server deployment; HA requires additional infrastructure.

## Pros and Cons of the Options

### pgpool-II

* Good, because it also supports read replica load balancing.
* Bad, because its configuration syntax is significantly more complex than PgBouncer's.
* Bad, because it is overkill for a deployment with one PostgreSQL primary and no replicas.

### No pooler; reduce pool sizes

* Good, because it requires no new infrastructure component.
* Bad, because it trades connection count for latency — smaller pools mean more queued requests under load.
* Bad, because it doesn't scale: every new service replica increases the connection count regardless of pool size.
