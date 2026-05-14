# Zero-Dependency Redis Client for Distributed Rate Limiting

## Status

Accepted

## Context and Problem Statement

The gateway's in-memory rate limiter works correctly for a single replica but does not share state between replicas. When the gateway is scaled to multiple instances behind a load balancer, each instance maintains its own per-IP counter — the effective rate limit becomes `N × limit` where `N` is the number of replicas. A city running two gateway replicas effectively has no rate limiting.

Rate limit state must be shared across replicas. Redis is the standard tool for this. However, all maintained Go Redis client libraries (`redis/go-redis/v9`, `go-redis/redis/v8`) now require Go 1.23 or 1.24, and Fieldstone targets Go 1.22 to align with the current Dockerfile base image.

## Decision Drivers

* Rate limit counters must be consistent across all gateway replicas.
* No dependency on a Go Redis client library that requires a newer Go version than the project uses.
* The implementation must be simple enough to understand and audit without Redis expertise.
* Failure of the Redis connection must not fail legitimate requests — the system should fail open.

## Considered Options

* **External Redis client library** (`go-redis/redis/v8` or `redis/go-redis/v9`) — idiomatic; feature-rich; requires Go ≥ 1.23 or 1.24.
* **Raw RESP protocol over TCP** — implement only the 4 commands needed for a sliding-window counter; no external dependency; works on any Go version.
* **Upgrade Go to 1.24** — unblocks using any Redis library; requires updating Dockerfiles, CI, and potentially other dependency pins.
* **Accept per-replica rate limiting** — document the limitation; adequate for single-replica deployments.

## Decision Outcome

Chosen option: **Raw RESP protocol over TCP**, because it requires no external dependency, adds approximately 120 lines of focused code, and implements exactly the commands needed — no more. The Go version constraint is a real deployment concern for city IT teams who may be running infrastructure with specific OS/tool versions.

The implementation uses a sliding-window algorithm:
1. `ZREMRANGEBYSCORE rl:{ip} 0 {now - window}` — remove requests older than the window
2. `ZADD rl:{ip} {now} {uuid}` — record this request with a unique member
3. `ZCARD rl:{ip}` — count requests in the current window
4. `EXPIRE rl:{ip} {window_seconds}` — set TTL for automatic cleanup

These four commands are pipelined in a single TCP write. A small connection pool (16 connections) avoids reconnect overhead. If Redis is unreachable, the limiter **fails open** — requests are allowed through and an ERROR is logged. This is the correct behavior: a Redis outage should not take down the public-facing portal.

The `newRateLimiter(redisURL, limit, window)` constructor checks `REDIS_URL` at startup. If empty or unreachable, it falls back to the in-memory implementation with a warning log.

### Positive Consequences

* Zero new runtime dependencies — the rate limiter is entirely standard library.
* The RESP protocol implementation is ~80 lines and handles only the subset needed; it is easier to audit than a full Redis client.
* Works identically on Go 1.22, 1.23, and 1.24 — no future version-pin churn for this component.
* Fail-open behavior means Redis downtime does not create a customer-facing outage.
* The `RateLimiter` interface makes it straightforward to swap in a library-based implementation when the Go version constraint lifts.

### Negative Consequences

* The raw RESP implementation only handles integer (`:`) and simple string (`+`) responses — it would need extension for bulk strings or arrays if new commands were added.
* No support for Redis AUTH, TLS, or Sentinel — adequate for a containerized deployment where Redis is on the internal Docker network, but production deployments that expose Redis outside the Docker network should add TLS termination at the Caddy or network layer.
* The connection pool is naive (no health checking of pooled connections); a connection broken by Redis restart may cause one failed request before being replaced.

## Note on Go version management

The right long-term fix is to upgrade the project's Go toolchain directive to 1.24 when the ecosystem has stabilized and city IT departments have had time to update their container runtimes. At that point, this implementation can be replaced with `go-redis/v9` or similar, and this ADR should be superseded.
