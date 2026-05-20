# Temporal Timers and Durable Webhook Delivery

## Status

Accepted — extends ADR-0022 (Temporal as durable workflow engine)

## Context and Problem Statement

ADR-0022 adopted Temporal for state machine validation but explicitly deferred three
patterns to future work:

1. **Automated transitions via timers** — permit expiry and FOIA response-time
   enforcement were listed as motivating requirements but not implemented. The
   `approved → expired (system)` transition existed in `permits.yaml` with no mechanism
   to fire it. FOIA requests had a `due_date` column with no enforcement.

2. **Webhook delivery reliability** — the webhooks service used a manual retry loop
   (`time.Sleep` over `[1, 2, 4, 8, 16]` seconds) with un-tracked goroutines. An
   in-flight delivery was silently lost on service restart. Retry timing was rigid.
   There was no visibility into the delivery pipeline.

Both problems are natural fits for Temporal's durable execution model, which was already
running in the stack. This ADR documents the decision to wire up timers and migrate
webhook delivery to Temporal activities rather than building custom solutions for each.

## Decision Drivers

* **Permit lifecycle completeness** — city staff should not have to manually expire
  permits; the platform should handle it automatically once a permit is approved.
* **FOIA compliance** — FOIA requests have legally mandated response deadlines.
  Silent misses are a compliance risk. Staff need a reliable notification when a deadline
  passes without the request reaching a terminal state.
* **Webhook durability** — HTTP delivery to external integrations must survive service
  restarts without message loss. The goroutine-per-delivery model provided no such
  guarantee.
* **Prefer existing infrastructure** — Temporal is already deployed. Solving both
  problems with it avoids introducing a separate cron system, job queue, or retry
  framework.
* **Preserve the outbox** — the transactional outbox (ADR-0018) remains the sole bridge
  from DB writes to NATS. Temporal activities call the outbox; they do not bypass it.

## Decision Outcome

### Permit auto-expiry

`config/workflows/permits.yaml` gained an `auto_expire_after: "8760h"` field on the
`approved` status (the `Status` struct in `internal/workflows/types.go` carries this as
`AutoExpireAfter string`). The `approved` status is now `terminal: false` — it is a
stable resting state but not the final one, since the permit will eventually expire.

When `PermitWorkflow` processes a validated transition into a status that carries
`auto_expire_after`, it launches a `workflow.Go` coroutine that calls `workflow.Sleep`
for the configured duration, then executes `PermitActivities.UpdatePermitStatus` to
write `expired` to the permits DB and publish the `permit.status_changed` event via the
outbox. If the workflow is already complete when the timer fires (e.g., a staff admin
manually set the permit to `expired`), the coroutine exits without writing.

The permit workflow execution timeout was extended from 1 year to 3 years to accommodate
permits that spend months under review before approval and then run a full 1-year expiry
timer.

### FOIA deadline notification

`WorkflowInput` gained an optional `Deadline *time.Time` field. The records service
handler extracts `due_date` from the FOIA request at creation time and passes it to
`StartWorkflow`. `FOIARequestWorkflow` uses `workflow.Sleep` until the deadline, then
(if the request is not yet terminal) executes `RecordsActivities.NotifyDeadlineExceeded`,
which publishes a `fieldstone.records.foia_request.deadline_exceeded` event via the
outbox.

The notification does not change the FOIA request's status — staff retain control over
the actual resolution. Consumers (the notify service stub, future email/SMS channels)
subscribe to `deadline_exceeded` events to alert the responsible department.

### Durable webhook delivery

The webhooks service was extended with its own Temporal worker running on a dedicated
`"webhooks"` task queue, keeping webhook domain logic co-located with the webhooks DB
rather than adding it to the `workflow-worker` service.

The NATS subscriber (`setupDispatcher`) now starts a `WebhookDeliveryWorkflow` per
`(event, endpoint)` pair instead of spawning a goroutine. Temporal's retry policy
replaces the manual `time.Sleep` loop:

```
InitialInterval: 1s, BackoffCoefficient: 2.0, MaximumInterval: 30s, MaximumAttempts: 5
```

Workflow ID `webhook-{endpointID}-{eventID}` makes NATS redelivery idempotent — a
duplicate message triggers `WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE` and is silently
dropped rather than causing a duplicate delivery.

The workflow calls four activities after delivery:
- `Deliver` — single HTTP attempt; non-2xx and transport errors are returned as errors
  so Temporal retries with the configured policy.
- `RecordDelivery` — inserts one row into `webhooks.deliveries` with the final outcome.
- `ResetFailCount` / `IncrementFailCount` + `DisableEndpoint` — update the endpoint's
  consecutive failure counter, disabling the endpoint after 10 failures.

The manual `attemptDelivery` function and its retry loop were removed entirely.
`sendHTTP` and `computeSignature` are retained as utility functions used by the
`Deliver` activity.

## Consequences

### Positive

* Permit expiry is automatic and durable — no cron job, no manual staff action.
* FOIA deadlines surface as observable events in the audit log and can drive
  notifications without polling the DB.
* Webhook delivery survives service restarts. In-flight deliveries resume when the
  webhooks service comes back up.
* Delivery retries with exponential backoff and idempotency are Temporal built-ins,
  not custom code. The old 90-line retry loop is gone.
* All webhook deliveries are visible in the Temporal UI with full retry history.

### Negative

* `approved` permits now have a running workflow execution for up to 1 year after
  approval. The workflow count in Temporal grows proportionally to the number of
  approved permits. This is expected and manageable but must be monitored as permit
  volume scales.
* The webhooks service now has a hard startup dependency on Temporal. Previously it
  only needed NATS and PostgreSQL.
* `TestWebhook` (`POST /v1/webhooks/{id}/test`) changed from synchronous to
  asynchronous — it starts a workflow and returns immediately rather than blocking until
  the first attempt completes.

## Pros and Cons of Considered Options

### Cron job for permit expiry / FOIA deadlines

* Good, because no new Temporal code.
* Bad, because cron jobs are stateless — a missed fire due to downtime is silently
  skipped rather than retried.
* Bad, because a cron must scan the DB on every tick; Temporal timers fire exactly once
  per workflow execution with no DB scan.

### Dedicated job queue (e.g., River, Asynq)

* Good, because a purpose-built job queue has a smaller footprint than Temporal.
* Bad, because Temporal is already running. Adding another scheduler is infrastructure
  duplication.
* Bad, because job queues typically do not provide the durable signal/update model
  needed for the state machine use cases already in Temporal.

### Retain goroutine-per-delivery with improved error handling

* Good, because zero new infrastructure.
* Bad, because goroutines are still lost on restart — the fundamental reliability
  problem is unsolved.
* Bad, because visibility into in-flight deliveries remains zero.

### Temporal (chosen)

* Good, because durable timers, retries, and workflow IDs for idempotency are all
  first-class primitives.
* Good, because the infrastructure is already deployed and monitored.
* Good, because the Temporal UI provides immediate delivery observability.
* Bad, because the webhooks service now has a Temporal dependency at startup.
