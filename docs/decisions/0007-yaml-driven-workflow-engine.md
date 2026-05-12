# Configurable Workflow Engine via YAML, Not Hardcoded Transitions

## Status

Accepted

## Context and Problem Statement

Permits, service requests, and FOIA requests all move through a series of statuses (submitted → under review → approved, etc.). Different cities have different processes — some add intermediate review steps, some have different role assignments, some have additional terminal states. How should status transition logic be defined and enforced?

## Decision Drivers

* Cities must be able to change their workflow without modifying or redeploying core service code.
* Domain services (permits, requests, records) must not contain hardcoded state machine logic that varies by jurisdiction.
* The workflow definition must be auditable and reviewable by non-engineers (city administrators should understand what transitions are allowed).
* Changes to workflow should take effect quickly — not require a full Docker image rebuild and redeploy.

## Considered Options

* **Hardcoded state machine in each domain service** — transitions defined in Go constants or switch statements.
* **YAML config files loaded by a dedicated workflow service** — domain services call the workflow service to validate transitions; cities edit YAML.
* **Database-driven workflow** — transitions stored in a database table, editable via an admin UI.
* **External workflow engine** — integrate Temporal, Cadence, or a BPMN engine.

## Decision Outcome

Chosen option: **YAML config files loaded by a dedicated workflow service**, because YAML files are human-readable, version-controllable, and can be changed by editing a mounted volume and restarting the workflow service — without rebuilding or redeploying any domain service.

Domain services call `POST /v1/workflow/:resource_type/validate` before any status change. The workflow service reads `*.yaml` from `/etc/fieldstone/workflows/` (mounted via Docker Compose) at startup and logs each loaded config. Cities fork the config files, not the code.

### Positive Consequences

* Workflow changes are code changes in YAML files — they go through version control and review like any other change.
* Domain service code contains zero hardcoded status strings for transitions; the only source of truth is the YAML.
* Changes take effect on workflow service restart (no redeploy of permits, requests, or records).
* Default workflow configs ship with Fieldstone as examples that cities edit, not as immutable behavior.
* YAML is auditable by city administrators who cannot read Go code.

### Negative Consequences

* Every status change requires an HTTP round-trip to the workflow service — an additional network call in the hot path.
* The workflow service is a new dependency; if it is unavailable, domain services cannot change statuses.
* YAML validation only catches syntax errors; semantic errors (referencing a non-existent status in a transition) are caught at startup, not at edit time.
* There is no admin UI for editing workflows — city administrators must edit files and restart the service.

## Pros and Cons of the Options

### Hardcoded state machine in each domain service

* Good, because there is no network call or external dependency for transitions.
* Good, because the transition logic is in the same codebase as the handler, easy to trace.
* Bad, because every jurisdiction customization requires forking and recompiling the service.
* Bad, because transition logic is scattered across three or more services with no single source of truth.

### Database-driven workflow

* Good, because changes can be made at runtime without any service restart.
* Good, because an admin UI can expose workflow configuration to non-technical city staff.
* Bad, because the workflow service now has its own database dependency with migrations to manage.
* Bad, because live editing of production workflows without a restart window carries risk of inconsistent state.
* Bad, because adds significant scope for a feature that YAML covers adequately.

### External workflow engine (Temporal, Cadence, BPMN)

* Good, because these engines are purpose-built for complex, long-running workflows with durable execution.
* Bad, because they are significant operational additions (Temporal requires its own server cluster).
* Bad, because the complexity far exceeds what civic services workflows require.
* Bad, because they are difficult to operate without dedicated platform expertise.
