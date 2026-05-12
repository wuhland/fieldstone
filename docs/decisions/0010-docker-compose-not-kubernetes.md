# Docker Compose for Deployment, Not Kubernetes

## Status

Accepted

## Context and Problem Statement

Fieldstone needs a deployment mechanism that a small IT contractor — not a dedicated DevOps team — can operate for a mid-sized city. The deployment must support optional service profiles (not every city runs permits), configuration via environment variables, and health-check-based startup ordering.

## Decision Drivers

* The target operator is a small IT contractor, not a platform engineering team.
* Operational complexity is a first-class cost: every additional concept an operator must learn is a barrier to adoption.
* The full stack (9 services, Postgres, NATS, Caddy, frontend) must run on a single server for budget-conscious municipalities.
* Service profiles allow cities to enable only the services they use.
* Self-hostable means the operator controls the infrastructure — managed Kubernetes (EKS, GKE) should not be required.

## Considered Options

* **Docker Compose** — multi-container orchestration with a single YAML file, native support for profiles and health checks.
* **Kubernetes** — production-grade container orchestration with Deployments, Services, ConfigMaps, Secrets, Ingress.
* **Nomad** — HashiCorp's lightweight orchestrator; simpler than Kubernetes but more capable than Compose.
* **Bare metal / systemd** — run each service as a systemd unit without containers.

## Decision Outcome

Chosen option: **Docker Compose**, because it matches the skill set and resources of the target operator audience, requires no cluster management, and natively supports the profile-based optional services model. A `docker compose --profile core up` is the complete getting-started command.

Docker Compose profiles group services into named sets (core, permits, records, webhooks, notify). Postgres and NATS health checks gate dependent service startup. The workflow config is mounted as a read-only volume. Dev overrides in `docker-compose.dev.yml` expose internal ports and enable debug logging.

### Positive Consequences

* A contractor with Docker experience can deploy and operate Fieldstone without learning Kubernetes.
* The entire stack runs on one Linux server — appropriate for small and mid-sized municipalities.
* `docker compose restart workflow` is the full upgrade path for workflow config changes.
* Service profiles give cities a simple toggle for optional services without editing service-level config.
* Development and production use the same `docker compose` tooling, reducing environment drift.

### Negative Consequences

* Docker Compose does not support horizontal scaling of individual services without additional tooling (Docker Swarm or an external load balancer).
* There is no built-in rolling update, health-check-gated rollout, or automatic self-healing beyond `restart: unless-stopped`.
* Large-city deployments that outgrow a single server will require migrating to Kubernetes or another orchestrator — this migration is non-trivial.
* Secret management is limited to `.env` files or Docker secrets, which are less sophisticated than Kubernetes Secrets or Vault.

## Pros and Cons of the Options

### Kubernetes

* Good, because it handles horizontal scaling, rolling updates, self-healing, and resource limits natively.
* Good, because it is the industry standard for production container orchestration.
* Bad, because even a minimal cluster (kubeadm, k3s) requires significantly more operator knowledge than Docker Compose.
* Bad, because Kubernetes adds dozens of resource types (Deployments, Services, ConfigMaps, Ingress, PersistentVolumeClaims) that must be defined and maintained.
* Bad, because managed Kubernetes services (EKS, GKE, AKS) are expensive and add a cloud provider dependency for a self-hostable product.

### Nomad

* Good, because it is operationally simpler than Kubernetes while supporting multi-node scheduling.
* Bad, because it is less well-known than Docker Compose or Kubernetes — fewer contractors have Nomad experience.
* Bad, because it adds a Nomad server process to the operational dependency list.
