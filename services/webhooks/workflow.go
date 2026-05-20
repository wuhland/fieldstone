package main

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/fieldstone/fieldstone/internal/events"
)

const WebhookTaskQueue = "webhooks"

// WebhookDeliveryInput is the durable input for a single endpoint+event delivery.
type WebhookDeliveryInput struct {
	EndpointID string
	URL        string
	SecretHash string
	Envelope   events.EventEnvelope
}

// WebhookDeliveryWorkflow delivers one event to one endpoint with Temporal-managed
// retries, then records the outcome and updates the endpoint's fail count.
// Workflow ID convention: "webhook-{endpointID}-{eventID}" ensures idempotent
// delivery even if the NATS message is redelivered.
func WebhookDeliveryWorkflow(ctx workflow.Context, input WebhookDeliveryInput) error {
	ao := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    5,
		},
	})

	var a *WebhookDeliveryActivities
	var result DeliverResult
	deliverErr := workflow.ExecuteActivity(ao, a.Deliver, DeliverParams{
		URL:        input.URL,
		SecretHash: input.SecretHash,
		Envelope:   input.Envelope,
	}).Get(ctx, &result)

	recordOpts := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
	})

	if deliverErr == nil {
		sc := result.StatusCode
		dm := result.DurationMs
		workflow.ExecuteActivity(recordOpts, a.RecordDelivery, RecordDeliveryParams{
			EndpointID: input.EndpointID,
			EventID:    input.Envelope.ID,
			EventType:  input.Envelope.EventType,
			StatusCode: &sc,
			DurationMs: &dm,
		}).Get(ctx, nil) //nolint:errcheck
		workflow.ExecuteActivity(recordOpts, a.ResetFailCount, input.EndpointID).Get(ctx, nil) //nolint:errcheck
	} else {
		errMsg := deliverErr.Error()
		workflow.ExecuteActivity(recordOpts, a.RecordDelivery, RecordDeliveryParams{
			EndpointID: input.EndpointID,
			EventID:    input.Envelope.ID,
			EventType:  input.Envelope.EventType,
			Error:      &errMsg,
		}).Get(ctx, nil) //nolint:errcheck
		var newCount int32
		if err := workflow.ExecuteActivity(recordOpts, a.IncrementFailCount, input.EndpointID).Get(ctx, &newCount); err == nil {
			if newCount >= maxConsecutiveFailures {
				workflow.ExecuteActivity(recordOpts, a.DisableEndpoint, input.EndpointID).Get(ctx, nil) //nolint:errcheck
			}
		}
	}

	return nil
}
