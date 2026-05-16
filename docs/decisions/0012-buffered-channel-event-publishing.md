# Buffered Channel for Decoupled NATS Event Publishing

## Status

Superseded by [ADR-0018](0018-transactional-outbox.md)

The buffered channel was replaced by the transactional outbox pattern after it was identified as a correctness problem: events in the channel are lost on process crash, meaning the audit log could silently have gaps. The outbox writes the event row to the database in the same transaction as the business record, guaranteeing at-least-once delivery even across restarts.

The original rationale and analysis below is preserved for context.

## Context and Problem Statement

Domain services must publish events to NATS JetStream after each significant state change. How should the publishing be integrated into HTTP handlers? A NATS publish that blocks, fails, or is slow must not affect the HTTP response the client receives.

## Decision Drivers

* A failed NATS publish must never cause an HTTP request to fail — the user's action (creating a permit, closing a request) must be durable regardless of messaging infrastructure state.
* Publishing must happen after the database transaction commits, not before — events represent facts that have occurred.
* The solution must be simple enough to reason about and maintain without a distributed systems background.
* At-most-once delivery loss on NATS failure is acceptable for the current stage; the audit log is an eventual consistency concern.

## Considered Options

* **Direct synchronous publish** — call `js.Publish()` in the HTTP handler after the DB commit; return an error if it fails.
* **Buffered in-process channel** — after DB commit, send the event to a `chan EventEnvelope` (size 1000); a background goroutine drains to NATS.
* **Transactional outbox pattern** — write events to an `outbox` table in the same DB transaction; a separate process polls and publishes.
* **Fire-and-forget goroutine** — launch `go js.Publish(...)` in the handler; ignore errors.

## Decision Outcome

Chosen option: **buffered in-process channel**, because it fully decouples the HTTP handler's response time from NATS latency, absorbs brief NATS unavailability without dropping events (up to the channel buffer), and is straightforward to implement and reason about.

Each service that publishes events has a `publisher` struct with a `chan publishRequest` of size 1000. A background goroutine drains the channel, marshals events, and calls `js.Publish()`. Failed publishes are logged at ERROR but do not propagate back to the HTTP caller. The channel being full (NATS down for an extended period) causes events to be dropped with an ERROR log.

### Positive Consequences

* HTTP handlers return immediately after a successful DB write; NATS latency has zero effect on client-facing response time.
* Brief NATS outages (restart, transient failure) are absorbed by the buffer without dropping events.
* The publisher goroutine is a single, auditable place for all event publication logic and error handling.
* No additional database table or polling process is required.

### Negative Consequences

* If the service process crashes with events in the channel, those events are lost — the buffer is in-memory only.
* If NATS is unavailable for longer than it takes to fill 1000 events, events are silently dropped.
* There is no delivery confirmation back to the HTTP handler — the handler cannot know whether the event was ultimately published.
* This is effectively at-most-once delivery from the service's perspective; reliable delivery requires the transactional outbox pattern.

## Pros and Cons of the Options

### Direct synchronous publish

* Good, because the handler knows immediately whether the event was published.
* Good, because no background goroutine or channel to manage.
* Bad, because a slow or unavailable NATS server directly increases HTTP response latency.
* Bad, because a NATS failure causes the HTTP request to fail, even though the DB write succeeded — the user's action would appear to have failed when it actually succeeded.

### Transactional outbox pattern

* Good, because events are durable even if the service crashes between the DB write and the NATS publish.
* Good, because exactly-once or at-least-once delivery can be guaranteed.
* Bad, because it requires an `outbox` table in each service's schema, a polling process, and careful handling of duplicate deliveries.
* Bad, because it adds significant complexity for a feature that the buffered channel handles adequately for the current reliability tier.
* Neutral, because this is the right long-term answer for services where event loss is unacceptable — documented here so the upgrade path is clear.

### Fire-and-forget goroutine

* Good, because it is the simplest possible implementation.
* Bad, because goroutine leaks are possible if the NATS client blocks.
* Bad, because there is no backpressure — if NATS is slow, unbounded goroutines accumulate.
* Bad, because errors are silently discarded with no visibility.
