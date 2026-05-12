# Single Go Module for All Services

## Status

Accepted

## Context and Problem Statement

Fieldstone is composed of nine services plus a set of shared internal packages. How should the Go module structure be organized? The choice affects how shared code is imported, how dependencies are managed, and how the build and release process works.

## Decision Drivers

* The shared `internal/` packages (events, auth, middleware, etc.) must be importable by all services without publishing to a registry.
* Operational simplicity is a core goal — the build and test pipeline should be as straightforward as possible.
* The project is early-stage; services do not yet have independent release cadences.
* A small IT contractor should be able to understand and contribute to the codebase.

## Considered Options

* **Single module at the repo root** — one `go.mod` covers all services and internal packages.
* **Multi-module workspace** — each service is its own module; a `go.work` file stitches them together locally.
* **One module per service, shared packages published** — shared code is versioned and released independently on a registry.

## Decision Outcome

Chosen option: **single module at the repo root**, because it is the simplest arrangement for an early-stage project where services have no independent versioning needs and shared code must remain tightly coupled to the services that use it.

All services and `internal/` packages share the module path `github.com/fieldstone/fieldstone`. Services import shared code as `github.com/fieldstone/fieldstone/internal/events`, etc.

### Positive Consequences

* One `go.mod` and `go.sum` to maintain.
* `go build ./services/...` and `go test ./...` work from the repo root without workspace setup.
* Changes to `internal/` packages and their callers land in the same PR and are always in sync.
* Simpler CI: a single `go vet ./...` covers the whole repo.

### Negative Consequences

* All services share the same dependency graph; one service's dependency upgrade affects all others.
* When services need to be versioned and released independently, a migration to a multi-module workspace or separate repos will be required.
* A `go build ./...` at the root builds everything, which is slower than building a single service.

## Pros and Cons of the Options

### Multi-module workspace

* Good, because services can evolve their dependencies independently.
* Good, because `go work` is now well-supported tooling.
* Bad, because `go.work` files are not committed when publishing the module, complicating `go get` for external consumers.
* Bad, because workspace setup adds friction for new contributors.
* Bad, because cross-cutting changes to shared packages require touching multiple `go.mod` files.

### One module per service, shared packages published

* Good, because services are fully decoupled at the module boundary.
* Bad, because shared packages must be versioned, tagged, and released before service changes can reference them — a significant operational overhead for an early project.
* Bad, because it requires a registry or a replace directive, both of which complicate local development.
