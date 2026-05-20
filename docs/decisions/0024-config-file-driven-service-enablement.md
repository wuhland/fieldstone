# Config-File-Driven Service Enablement

## Status

Accepted

## Context and Problem Statement

Fieldstone is designed for city governments with varying service portfolios. A city that
processes building permits does not necessarily operate a 311 call centre, and one that
handles public records requests may not yet be ready to expose the permit module to
residents. Running every service regardless wastes resources and exposes API surface area
that has no backing operation.

The original gateway design registered all routes unconditionally and required all seven
service URLs as `env-required:"true"` fields. This meant:

1. The gateway refused to start if any service URL was absent, making it impossible to
   run a partial deployment without modifying Go source code or adding a dummy HTTP
   server at the missing URL.
2. There was no way to communicate to API clients that a route exists in the platform
   but is deliberately not available in this deployment — disabled routes either returned
   502 (if the container wasn't running) or didn't exist at the Go routing level.
3. `requests` (311 service) was hardcoded into the `core` Docker Compose profile, making
   it impossible to omit without forking the compose file.

## Decision Drivers

* **No code change required** — enabling or disabling a service module must be
  achievable by editing a config file and restarting the gateway.
* **Clear API semantics** — clients that call a disabled route should receive a response
  that distinguishes "this path does not exist" (404) from "this service is not enabled
  in this deployment" (503).
* **Future extensibility** — the same config structure should accommodate per-service
  settings beyond the on/off switch: rate limit overrides, auth requirement variations,
  custom route prefixes, etc.
* **Single source of truth for routing** — service URLs and enable flags should live
  together rather than split across `.env` and source code.

## Considered Options

### A — Null URL means disabled

Remove `env-required` from service URL env vars. Skip route registration in
`gateway/main.go` when a service URL is empty. Return 404 for routes whose service has
no URL.

Rejected because 404 is indistinguishable from "this route never existed." External
integrations built against the API cannot tell whether they hit a typo or an intentionally
disabled module.

### B — Explicit `*_ENABLED` boolean env vars

Add `PERMITS_ENABLED=true/false` per service (defaulting to `true`). Register either a
proxy handler or a 503 stub based on the flag. Keep service URLs as separate env vars.

Viable, but leaves service routing split across two env vars per service and provides no
natural path for future per-service configuration beyond the boolean.

### C — Single `ENABLED_SERVICES` list

`ENABLED_SERVICES=permits,records,webhooks` — a comma-separated allowlist. The gateway
registers real proxies for listed services and 503 stubs for the rest.

Simpler than option B but harder to read and audit than a structured file. "Empty = all
enabled" vs. "empty = all disabled" creates an ambiguous default.

### D — Deployment config file (chosen)

A YAML file (`config/services.yaml`) mounted read-only into the gateway container.
Each service entry carries `enabled: bool` and `url: string`. The gateway loads this
file at startup, validates it (enabled service without URL is a startup error), and
registers either a reverse proxy or a 503 handler for each service's routes.

## Decision Outcome

`config/services.yaml` is the single source of truth for which services are active in
a given deployment and where they are reachable. Service URLs are removed from `.env`;
the gateway reads only `SERVICES_CONFIG` (path to the file, defaulting to
`/etc/fieldstone/services.yaml`).

```yaml
services:
  permits:
    enabled: true
    url: http://permits:8081
  requests:
    enabled: false   # 311 not yet staffed — all /v1/requests routes return 503
  records:
    enabled: true
    url: http://records:8083
  # …
```

The gateway's `ServicesConfig.handler(name)` method returns the appropriate handler at
startup:

* **Enabled** — `httputil.ReverseProxy` targeting the configured URL.
* **Disabled or absent** — an `http.HandlerFunc` that writes
  `HTTP 503 {"error":"service not enabled","service":"<name>"}`.

Routes are always registered in the router. Clients that call a disabled route receive
503, not 404, so they know the route is valid but the module is off. Swagger UI
continues to document all routes regardless of the enable state.

The gateway fails fast at startup if any service has `enabled: true` but an empty `url`,
preventing silent misrouting.

### Docker Compose profile alignment

The `requests` (311) service was moved from the `core` profile to its own `requests`
profile, consistent with how `permits`, `records`, `webhooks`, and `notify` already work.
`core` is now reserved for infrastructure and platform services that every deployment
needs: the gateway, identity, audit, workflow-worker, Temporal, NATS, PostgreSQL,
PgBouncer, Redis, Caddy, and the frontend.

The gateway itself always starts with `core`. It consults `services.yaml` to decide
which routes to proxy, independently of which service containers are running. Operators
control container presence with Compose profiles and control API availability with the
config file. The two levers are intentionally decoupled:

| services.yaml | Compose profile | Outcome |
|---|---|---|
| `enabled: true` | included | Normal operation |
| `enabled: true` | excluded | Gateway routes to it; container returns 502 |
| `enabled: false` | included | Container runs; gateway returns 503 |
| `enabled: false` | excluded | Clean — container off, routes return 503 |

The decoupling is useful during maintenance windows: a service can be pulled from
Compose (container stops) while the gateway continues to return 503 rather than 502,
giving callers a more informative error without a gateway restart.

## Consequences

### Positive

* A city operator disables a module by editing two lines in `config/services.yaml` and
  restarting the gateway — no code change, no env var coordination across multiple files.
* Disabled routes return 503 with a structured body, making it practical for client code
  and integrations to detect and handle the absence of a module.
* Service URLs are co-located with their enable flag in a single auditable file, making
  deployment configuration easier to diff and review.
* The config file structure naturally accommodates future per-service settings (rate
  limits, auth overrides, custom prefixes) without further interface changes.

### Negative

* Operators must manage an additional config file alongside `.env`. For simple
  single-service deployments this is minor overhead; for automated deployments it means
  templating or mounting one more file.
* The gateway no longer validates at startup that downstream services are reachable —
  only that enabled services have non-empty URLs. A misconfigured URL is discovered at
  request time (502) rather than at boot.
* Existing deployments must migrate their `*_SERVICE_URL` env vars into `services.yaml`.
  The env vars are simply removed from the gateway config struct, so a leftover entry in
  `.env` is silently ignored rather than causing a clear error.
