# Temporal as Durable Workflow Engine

## Status

Accepted — supersedes ADR-0007 (YAML-driven workflow engine)

## Context and Problem Statement

ADR-0007 introduced a stateless YAML-validator service to enforce role-based status
transitions for permits, service requests, and FOIA requests. That decision explicitly
considered and rejected Temporal on the grounds that civic workflow complexity did not
justify the operational cost.

Three requirements have since changed that calculus:

1. **Long-lived workflows** — permit applications and FOIA requests span weeks or months.
   Status transitions, inspection sequences, and clarification loops need to survive
   process crashes without losing intermediate state.

2. **External participant signals** — residents should be able to withdraw their own
   submissions, and inspectors should be able to file results, without staff manually
   entering data. The YAML validator had no concept of external parties as actors;
   only internal staff roles (`admin`, `reviewer`, `staff`, `system`) could trigger
   transitions.

3. **Automated transitions** — permit expiry and FOIA response-time enforcement require
   durable timers that fire reliably even if all services restart.

None of these can be expressed in the stateless HTTP validator model.

## Decision Drivers

* Durable execution across process restarts — the workflow's current state must survive
  crashes without re-deriving it from the DB on every transition.
* External participant support — residents and future external parties need a first-class
  way to trigger specific transitions without staff involvement.
* Preserve YAML-driven config — ADR-0007's rationale for city-editable YAML remains
  valid. Workflows must not require Go code changes for cities to customise transitions.
* Operational simplicity — Temporal must be self-hostable via Docker Compose with
  the existing PostgreSQL instance as its backend.

## Decision Outcome

**Replace the `workflow` service with Temporal as the durable execution engine.**

Fieldstone deploys `temporalio/temporal` server (PostgreSQL backend, same instance as
all other services) and a new `workflow-worker` service that:

1. **Registers workflow functions** (`PermitWorkflow`, `ServiceRequestWorkflow`,
   `FOIARequestWorkflow`) that run on Temporal's task queue.
2. **Reads YAML config at startup** (same files, same format as before) and passes
   it as input to each workflow execution at creation time — baking the active config
   durably into the workflow's history.
3. **Serves the `/v1/workflow/*` HTTP endpoints** (statuses, transitions, validate,
   initial) so the gateway proxy and domain service clients continue to work unchanged.

### How transitions work

**Staff transitions** (`PATCH /v1/permits/{id}/status`):

The domain handler sends a Temporal **Update** to the running workflow:
```
temporalClient.UpdateWorkflow(ctx, "permit-<id>", "validate-transition", {from, to, role})
```
The Update handler validates `{from, to, role}` against the YAML config baked into
the workflow's input. If valid, the transition is recorded in Temporal's execution
history and the Update returns synchronously. The domain handler then writes to the DB
as before and returns the response.

**Resident signals** (current and future):

Residents send a `withdraw` signal to the running workflow:
```
temporalClient.SignalWorkflow(ctx, "foia_request-<id>", "withdraw", {resident_id})
```
The workflow validates ownership (resident_id matches the stored sub) and executes
a Temporal **activity** that writes to the domain DB and publishes the event via the
outbox. The domain handler is not involved — the activity owns the DB write.

**Automated transitions** (timers — future):

The workflow registers a Temporal timer (e.g., `workflow.NewTimer(ctx, expiryIn)`).
When the timer fires, an activity executes the DB write atomically.

### Migration strategy

Existing in-flight records (created before Temporal adoption) have no running workflow
execution. Their status updates fall back automatically to HTTP validation against the
worker's `/v1/workflow/{type}/validate` endpoint. This fallback path remains available
until all existing open records are either closed or migrated via a one-off script that
starts synthetic workflow executions at the records' current status.

### YAML config preservation

The YAML files in `config/workflows/*.yaml` remain unchanged and are mounted into the
worker at the same path. The `WorkflowConfig` struct (now in `internal/workflows/`) is
a superset of the previous loader's struct, with JSON tags added so it serialises
cleanly into Temporal's workflow input history.

Cities still edit YAML and restart the worker. No Go code changes are needed for
transition or role customisations.

## Consequences

### Positive

* Workflow state survives process crashes — Temporal replays from history on restart.
* External participant workflows are natively supported via Signals and Updates.
* Durable timers enable permit expiry and FOIA deadline enforcement without cron jobs.
* The Temporal UI (port 8233) provides full workflow execution visibility: what
  transitions happened, in what order, by whom, and when.
* YAML config extensibility is fully preserved.

### Negative

* Temporal server and `temporal_visibility` DB are new infrastructure dependencies.
  The PostgreSQL backend reuses the existing instance (two additional databases), but
  the Temporal server is a new process to operate and monitor.
* Staff transition validation now requires Temporal to be healthy. If Temporal is
  unavailable, status updates fall back to HTTP (same as the previous workflow
  service), so the fallback maintains availability at the cost of durability.
* Workflow functions are Go code — cities cannot change the workflow *execution logic*
  (only the YAML transition config) without deploying a new worker image.
* The Temporal Go SDK adds ~40 MB of transitive dependencies to the module.

## Pros and Cons of Considered Options

### Keep the YAML validator service

* Good, because zero new infrastructure.
* Bad, because no durability, no external participant support, no timers.
* Bad, because this was already the recognised limitation that prompted this decision.

### Database-driven workflow with state machine table

* Good, because runtime-editable without restarts.
* Bad, because building a reliable state machine on top of PostgreSQL requires
  significant custom code (equivalent to re-implementing Temporal's execution model).
* Bad, because no durable timer support without a separate scheduler.

### Temporal (chosen)

* Good, because durable execution, signals, updates, and timers are all built-in.
* Good, because PostgreSQL backend means no additional storage technology.
* Good, because the Go SDK is mature and first-class.
* Good, because the Temporal UI gives immediate operational observability.
* Bad, because operational complexity increases (one more service to run and monitor).
