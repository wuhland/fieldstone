# Transactional Outbox for Durable Event Publishing

## Status

Accepted — supersedes [ADR-0012](0012-buffered-channel-event-publishing.md)

## Context and Problem Statement

ADR-0012 chose a buffered in-process channel to decouple HTTP response time from NATS publishing. This worked for correctness at first but introduced a silent data loss scenario: if a service process crashes while events are queued in the channel (between DB commit and NATS publish), those events are permanently lost. For most events this is tolerable — a missed webhook delivery is recoverable. For the audit log it is a compliance problem: a government audit trail must be complete.

The audit service receives events from NATS and persists them to `audit.events`. If an event is never published to NATS, the audit service never receives it, and there is no record of the business action that produced it.

How can event publishing be made durable across process crashes without blocking the HTTP response?

## Decision Drivers

* The audit log must contain a record of every significant state change, including those that occurred during a process restart.
* HTTP response time must not be affected by NATS availability.
* The implementation must compose with the existing pgx/sqlc DB layer.
* Duplicate event delivery (from JetStream redelivery) must be handled safely by consumers.

## Considered Options

* **Keep the buffered channel** — simple, but events in the buffer are lost on crash.
* **Synchronous publish in handler** — durable if NATS is up, but blocks the HTTP response and fails the request if NATS is down.
* **Transactional outbox** — write event to an `outbox` table in the same DB transaction as the business record; a background poller publishes to NATS and deletes rows on acknowledgment.
* **Write-ahead log (WAL) tailing** — consume the PostgreSQL WAL directly for change events; operationally complex and not portable.

## Decision Outcome

Chosen option: **transactional outbox**, because it gives exactly the durability guarantee needed — the event row either commits with the business record or both roll back — while keeping the HTTP response path completely free of NATS interaction.

### How it works

Every domain service (permits, requests, records, identity) has an `outbox` table in its schema:

```sql
CREATE TABLE outbox (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject    TEXT NOT NULL,
    payload    JSONB NOT NULL,  -- full EventEnvelope JSON
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

In every write handler, the business record and the outbox row are written in a single pgx transaction:

```go
tx, _ := pool.Begin(ctx)
defer tx.Rollback(ctx)

permit, _ := queries.WithTx(tx).CreatePermit(ctx, params) // business write
pub.PublishTx(ctx, tx, subject, ...)                       // outbox write
tx.Commit(ctx)                                             // both or neither
```

A background goroutine (`internal/outbox.Run`) polls the outbox table every second, publishes pending rows to NATS JetStream using `SELECT ... FOR UPDATE SKIP LOCKED` (safe for multiple replicas), and deletes rows after acknowledgment:

```go
SELECT id, subject, payload FROM outbox ORDER BY created_at LIMIT 100 FOR UPDATE SKIP LOCKED
→ for each row: js.Publish(subject, payload)
→ DELETE FROM outbox WHERE id = $1
→ COMMIT
```

The audit service handles duplicate delivery idempotently via `ON CONFLICT (id, occurred_at) DO NOTHING`.

### The `DBTX` interface

To allow query structs to work inside or outside a transaction without changing the handler code, a shared `DBTX` interface was introduced in `internal/db/dbtx.go`:

```go
type DBTX interface {
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}
```

Both `*pgxpool.Pool` and `pgx.Tx` satisfy this interface. Each service's `Queries` struct holds a `DBTX` and gains a `WithTx(tx pgx.Tx) *Queries` method. Handlers call `queries.WithTx(tx)` to bind to the live transaction for the business write.

### Positive Consequences

* Events are atomic with business records — a crash cannot produce a DB record without a corresponding audit event.
* HTTP handlers are completely decoupled from NATS; a NATS outage has zero effect on request success.
* `FOR UPDATE SKIP LOCKED` allows multiple service replicas to run pollers concurrently without deadlocking.
* `ON CONFLICT DO NOTHING` in the audit INSERT absorbs the duplicate deliveries that JetStream's at-least-once guarantee produces.
* The outbox table is small at all times — rows are deleted after successful publish; only unpublished events accumulate.

### Negative Consequences

* Every write handler now manages an explicit `pgx.Tx` — more boilerplate compared to the previous direct `pool.QueryRow` calls.
* A sustained NATS outage will cause the outbox table to grow. The poller retries indefinitely; events are not dropped but may be delayed until NATS recovers.
* The polling interval (1 second) introduces up to 1 second of latency between a DB commit and event delivery to subscribers. Acceptable for audit and webhooks; not suitable for real-time use cases.
* The outbox poller's one-second tick means the service makes one DB round-trip per second even when idle.

## Pros and Cons of the Options

### Synchronous publish in handler

* Good, because the handler knows immediately whether the event reached NATS.
* Bad, because if NATS is down, the HTTP request fails even though the DB write succeeded — double-failure from the user's perspective.
* Bad, because NATS latency directly appears in request latency.

### WAL tailing

* Good, because it captures every write without modifying the application.
* Bad, because it requires the PostgreSQL logical decoding plugin, specific configuration, and an additional consumer process.
* Bad, because it is opaque to the application layer and difficult to test.
