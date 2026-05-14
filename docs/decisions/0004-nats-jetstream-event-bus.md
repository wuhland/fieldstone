# NATS JetStream as the Async Event Bus

## Status

Accepted — stream retention corrected from WorkQueuePolicy to InterestPolicy

## Context and Problem Statement

Fieldstone services need a reliable way to broadcast state changes (permit created, request assigned, etc.) so that other services and extension services can react without being directly coupled. What should the async messaging infrastructure be?

## Decision Drivers

* Extension services must be able to subscribe to events without being registered anywhere in the core — fire-and-forget broadcast semantics.
* The message bus must be embeddable in a single Docker Compose file without significant operational overhead.
* At-least-once delivery with acknowledgement is required for the audit log (we cannot lose events).
* The operator audience is small IT contractors, not platform engineers — the system must be operable without deep messaging expertise.
* The system should support filtering (subscribing to a subject subtree like `fieldstone.permits.>`) without a consumer group fan-out configuration step.

## Considered Options

* **NATS JetStream** — lightweight, embeddable, subject-hierarchy filtering, persistent streams with at-least-once delivery.
* **Apache Kafka** — high-throughput, durable, mature ecosystem; topic-based partitioned log.
* **RabbitMQ** — AMQP-based broker with flexible routing, exchanges, and queues.
* **Redis Streams** — lightweight, already common in stacks, append-only log with consumer groups.

## Decision Outcome

Chosen option: **NATS JetStream**, because it offers persistent, at-least-once delivery with the simplest operational footprint of any considered option, and its subject-hierarchy wildcard subscriptions (`fieldstone.>`) are a natural fit for the extension model where subscribers self-select which events they care about.

The `FIELDSTONE` JetStream stream covers `fieldstone.>`. Services publish via a buffered in-process channel (see ADR-0012). Subscribers use durable consumers for at-least-once processing.

### Positive Consequences

* A single `nats-server -js` command starts the broker; no ZooKeeper, no cluster management, no topic pre-creation.
* Subject wildcards (`fieldstone.permits.>`, `fieldstone.>`) let extension services subscribe exactly to what they need with no broker-side configuration.
* JetStream consumers persist progress; a subscriber that restarts resumes from where it left off.
* NATS client libraries are available for Go, Python, Node, Java, and Rust — extension service authors are not constrained to Go.
* Memory and CPU footprint is very low compared to Kafka or RabbitMQ, appropriate for a small-city deployment on modest hardware.

### Negative Consequences

* NATS JetStream has lower throughput than Kafka at very high message rates — not a concern at civic services scale but worth noting.
* The ecosystem of connectors and monitoring tools is smaller than Kafka's.
* ~~WorkQueuePolicy~~ **Corrected to InterestPolicy** — see note below.

## Correction: WorkQueuePolicy → InterestPolicy

The initial scaffold configured the `FIELDSTONE` stream with `WorkQueuePolicy`. This was a bug: WorkQueuePolicy delivers each message to **exactly one** consumer across all subscribers, meaning the audit service and the webhooks service would race — each receiving approximately half of all events, and the audit log would silently be missing events.

`InterestPolicy` is the correct choice for this fan-out pattern. It retains each message until **every durable consumer** has acknowledged it, so audit, webhooks, and any extension services each independently receive the full event stream.

The correction is applied in `internal/nats/connect.go`, which also upgrades existing streams on connect so previously deployed instances self-correct on restart.

## Pros and Cons of the Options

### Apache Kafka

* Good, because it is the de facto standard for event streaming with a vast ecosystem.
* Good, because it handles extremely high throughput with strong durability guarantees.
* Bad, because it requires ZooKeeper (or KRaft) coordination, significantly raising operational complexity.
* Bad, because topic configuration (partition counts, retention policies) must be done before publishing.
* Bad, because the Kafka Connect ecosystem, while large, adds yet another operational component for webhook-style fan-out.

### RabbitMQ

* Good, because AMQP routing is flexible and mature.
* Good, because the management UI is excellent for debugging.
* Bad, because the exchange/queue/binding model is significantly more complex to configure than NATS subjects.
* Bad, because fanout to multiple independent consumers requires pre-configured bindings — extension services cannot self-register without an admin step.

### Redis Streams

* Good, because Redis is often already present in a deployment stack.
* Good, because very low operational overhead.
* Bad, because Redis Streams consumer groups do not support subject-hierarchy wildcard subscriptions — each topic must be explicitly subscribed.
* Bad, because Redis is not primarily a message broker; its persistence model is less suited to guaranteed delivery than JetStream.
