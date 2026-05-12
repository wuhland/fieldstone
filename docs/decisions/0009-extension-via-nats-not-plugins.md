# Extension Services via NATS Subscription, Not a Plugin System

## Status

Accepted

## Context and Problem Statement

Cities need to integrate Fieldstone with their existing systems: GIS databases, CRM platforms, legacy vendor software, reporting tools, and no-code automation platforms. How should Fieldstone expose integration points for these external systems? What is the mechanism for adding behavior without modifying core service code?

## Decision Drivers

* Cities must be able to add integrations without forking or modifying core Fieldstone services.
* The integration mechanism should work for external systems written in any language, not just Go.
* Core services must not be aware of or coupled to the extensions that consume their events.
* The mechanism should be usable by a small vendor or city IT team with limited platform engineering resources.

## Considered Options

* **NATS subscription by independent extension services** — external services connect to the same NATS instance and subscribe to events; core services have no knowledge of them.
* **Go plugin system** — core services load `.so` plugin files at startup via `plugin.Open`.
* **Webhook delivery only** — core services call registered HTTP endpoints synchronously or via a queue.
* **Sidecar containers with shared IPC** — extension logic runs in a sidecar container alongside core services.

## Decision Outcome

Chosen option: **NATS subscription by independent extension services**, with webhooks (ADR-0012 / the webhooks service) as a secondary layer for systems that cannot run a NATS consumer.

Extension services connect to the same NATS JetStream instance and subscribe to whatever subjects they care about using a unique durable consumer name. Core services publish events after writes and have no knowledge of which extensions are running. The `extensions/template/` directory provides a starting point.

Webhooks (the fourth extension layer) bridge NATS to HTTP for systems that cannot run a long-lived NATS consumer — no-code platforms, legacy vendor APIs, Zapier.

### Positive Consequences

* Core services are completely decoupled from extensions — no registration step, no core code changes required.
* Extension services can be written in any language with a NATS client library (Go, Python, Node, Java, Rust).
* Extensions are independently deployable; they can be started and stopped without affecting core services.
* The event bus is the only contract; extensions do not depend on internal Go types or function signatures.
* A malfunctioning extension cannot crash or slow down core services.

### Negative Consequences

* Extensions are fire-and-forget recipients; if they fail to process an event, core services do not know.
* Extensions must manage their own persistence, retry logic, and error handling.
* NATS JetStream must be running for extensions to receive events; extensions that miss events must catch up using durable consumer replay.
* There is no discovery mechanism — operators must know which extensions are running.

## Pros and Cons of the Options

### Go plugin system

* Good, because plugins run in-process with core services — no network hop for integration logic.
* Bad, because Go plugins must be compiled with the same Go version and module graph as the host — effectively requiring the extension author to use the exact same toolchain version.
* Bad, because a panicking plugin can take down the core service.
* Bad, because plugins can only be written in Go.
* Bad, because the Go plugin API is notoriously fragile and rarely used in production systems.

### Webhook delivery only (no NATS access for extensions)

* Good, because the integration API is universally accessible — any HTTP endpoint qualifies.
* Good, because NATS is an internal implementation detail hidden from extension authors.
* Bad, because webhooks add latency (HTTP delivery is slower than in-process NATS consumption) and require retry infrastructure.
* Bad, because extension services that need high throughput or low latency (e.g., real-time GIS sync) are poorly served by HTTP webhooks.
* Neutral, because Fieldstone provides webhooks as a supplemental layer for exactly these use cases.
