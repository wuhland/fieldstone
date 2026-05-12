# HTTP for Inter-Service Communication, Not gRPC

## Status

Accepted

## Context and Problem Statement

Fieldstone services need to call each other synchronously in some cases: domain services call the workflow service to validate transitions, and the gateway proxies all inbound requests to upstream services. What protocol should be used for this synchronous communication?

## Decision Drivers

* Simplicity of implementation — the project prioritizes being understandable by a small team.
* Extension services need to call core services (e.g., look up a permit by ID); they should be able to do so with a plain HTTP client in any language.
* Debugging and observability should be accessible with standard tools (curl, browser devtools, Caddy logs).
* gRPC imposes a protobuf contract that must be compiled and kept in sync across service boundaries.

## Considered Options

* **HTTP with JSON** — standard REST-style HTTP calls using `net/http` and `encoding/json`.
* **gRPC** — binary protocol with protobuf schemas, code generation for typed clients and servers.
* **gRPC-Gateway** — gRPC internally, with an HTTP/JSON transcoding layer for external callers.

## Decision Outcome

Chosen option: **HTTP with JSON**, because it matches the skill set of the target audience (city IT contractors and small vendors), requires no protobuf toolchain, and is debuggable with universally available tools. The performance advantage of gRPC is not needed at civic services traffic volumes.

All synchronous inter-service calls are plain HTTP using `net/http`. The gateway uses `httputil.ReverseProxy` to forward requests to upstream services. Service URLs are configured via environment variables.

### Positive Consequences

* Any language can call any service with a standard HTTP library — no protobuf compilation step.
* `curl https://city.gov/v1/permits` is the full debugging workflow.
* OpenAPI specs document the API contract without requiring a separate IDL.
* No code generation step in the build pipeline for service contracts.
* Caddy, logging middleware, and standard HTTP tracing all work without gRPC-specific plugins.

### Negative Consequences

* JSON serialization/deserialization is slower than protobuf — acceptable at this scale.
* API contracts are enforced by convention and OpenAPI linting rather than compiler errors from generated code.
* Streaming RPCs (e.g., real-time event streaming) are awkward over plain HTTP/1.1 and would require SSE or WebSockets.

## Pros and Cons of the Options

### gRPC

* Good, because the protobuf schema is the contract — clients and servers generated from the same `.proto` file are guaranteed to be compatible.
* Good, because binary encoding is significantly faster and smaller than JSON.
* Good, because bidirectional streaming is a first-class feature.
* Bad, because protobuf tooling (`protoc`, plugins) must be installed and version-pinned in CI.
* Bad, because extension service authors must generate client stubs in their language or use reflection — a significant barrier.
* Bad, because gRPC is not inspectable with standard HTTP tools without a transcoding layer.

### gRPC-Gateway

* Good, because it provides both a gRPC and HTTP/JSON interface.
* Bad, because it doubles the surface area: proto files, gRPC handlers, and HTTP annotations must all be maintained.
* Bad, because it adds complexity without a clear benefit given the traffic profile.
