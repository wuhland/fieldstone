# Caddy as the Reverse Proxy and TLS Terminator

## Status

Accepted

## Context and Problem Statement

Fieldstone needs a public-facing reverse proxy to terminate TLS, route traffic between the frontend and the API gateway, and serve the application over HTTPS. The proxy must be operable by a small IT contractor without deep web server expertise.

## Decision Drivers

* TLS configuration must be automatic — the operator should not need to manually generate, install, or renew certificates.
* The reverse proxy config must be readable and maintainable without extensive documentation.
* The proxy must handle routing between two upstreams: the API gateway (`/v1/*`) and the frontend (everything else).
* Operational burden must be minimal; certificate renewal should not be a recurring task.

## Considered Options

* **Caddy** — modern web server with automatic HTTPS via Let's Encrypt, simple Caddyfile syntax.
* **nginx** — high-performance, mature, widely understood; TLS configuration is manual.
* **Traefik** — container-native reverse proxy with automatic service discovery via Docker labels.
* **HAProxy** — high-performance load balancer; more complex configuration syntax.

## Decision Outcome

Chosen option: **Caddy**, because automatic HTTPS provisioning and renewal via Let's Encrypt eliminates the most error-prone operational task (certificate management) and the Caddyfile syntax is far simpler than nginx config for a two-upstream routing rule.

The Caddyfile routes `/v1/*` to `gateway:8080` and all other requests to `frontend:3000`. Caddy handles TLS termination, certificate renewal, and HTTP→HTTPS redirect automatically.

### Positive Consequences

* TLS certificates are provisioned and renewed automatically via ACME/Let's Encrypt — no cron jobs, no manual `certbot` runs.
* The Caddyfile for Fieldstone's routing needs is under 20 lines.
* HTTP/2 and HTTP/3 (QUIC) are supported out of the box.
* The Docker image is small and the process requires no special privileges when binding to ports 80/443 via Docker networking.

### Negative Consequences

* Caddy's automatic HTTPS requires the server to be reachable on port 80 from the internet for the ACME challenge; air-gapped or strictly firewalled deployments need manual certificate configuration.
* Caddy is less widely known among system administrators than nginx — some operators may need to learn new tooling.
* Advanced features (rate limiting, WAF, complex header manipulation) that are plugins in nginx are not always available in the same form in Caddy.

## Pros and Cons of the Options

### nginx

* Good, because it is extremely well-documented and nearly every sysadmin knows it.
* Good, because its performance characteristics under high load are well-understood.
* Bad, because TLS configuration requires manual certificate provisioning (certbot) and a separate renewal cron job.
* Bad, because nginx configuration syntax for even simple routing can be surprisingly cryptic.

### Traefik

* Good, because it integrates deeply with Docker — service routing can be configured via container labels, requiring no separate config file.
* Good, because it also supports automatic HTTPS.
* Bad, because label-based configuration scatters routing logic across `docker-compose.yml` rather than centralizing it.
* Bad, because Traefik's configuration model (providers, entrypoints, routers, middlewares) has a steeper learning curve than a Caddyfile.
