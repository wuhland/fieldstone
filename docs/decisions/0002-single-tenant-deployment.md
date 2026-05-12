# Single-Tenant Deployment Model

## Status

Accepted

## Context and Problem Statement

Fieldstone provides software for city governments to run civic services. The deployment model — whether one shared instance serves many cities or each city gets its own isolated instance — is one of the most consequential architectural choices. It affects data isolation, compliance, operational complexity, and the extensibility model.

## Decision Drivers

* City governments have strict data sovereignty requirements; one city's data must never be accessible by another.
* Cities need to customize workflows, metadata schemas, and potentially replace entire services — customization that would be much harder in a shared multi-tenant system.
* The target operator is a small IT contractor, not a SaaS platform team with dedicated reliability engineers.
* Self-hostability is a core goal: cities must be able to run Fieldstone on infrastructure they control.
* Open-source sustainability: a product that cities fully own is easier to adopt than one they must pay to access.

## Considered Options

* **Single-tenant** — each city deploys its own isolated instance (its own database, NATS cluster, services).
* **Multi-tenant with row-level isolation** — one shared deployment, with every table row tagged by `city_id`; queries filtered by tenant context.
* **Multi-tenant with schema-per-tenant** — one deployment, one schema per city within a shared PostgreSQL instance.

## Decision Outcome

Chosen option: **single-tenant**, because it provides the strongest data isolation guarantee with the simplest possible data model (no tenant discriminators anywhere in the schema or query layer), and it matches the operational and compliance expectations of city governments.

Each city runs the full Fieldstone stack: their own Postgres, their own NATS, their own container instances. Fieldstone provides Docker Compose configs to make this tractable.

### Positive Consequences

* Zero risk of cross-tenant data leakage; the database simply does not contain other cities' data.
* No tenant context to thread through every query, middleware, and event.
* Cities can customize aggressively (workflows, schemas, even replace services) without affecting anyone else.
* Compliance and audit requirements are scoped to a single jurisdiction.
* Upgrades are city-controlled; no forced migrations affecting all tenants simultaneously.

### Negative Consequences

* Fieldstone cannot offer a managed SaaS product without a significant orchestration layer on top.
* Each city's instance must be operated independently; there is no central fleet management.
* Bug fixes and security patches must be deployed per-city rather than once.
* Economies of scale from shared infrastructure are not available to cities.
