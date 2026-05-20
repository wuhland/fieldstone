package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fieldstone/fieldstone/internal/events"
	webhooksdb "github.com/fieldstone/fieldstone/services/webhooks/db/generated"
)

// WebhookDeliveryActivities implements Temporal activities for webhook delivery.
type WebhookDeliveryActivities struct {
	q *webhooksdb.Queries
}

// DeliverParams is the input for the Deliver activity.
type DeliverParams struct {
	URL        string
	SecretHash string
	Envelope   events.EventEnvelope
}

// DeliverResult holds the HTTP outcome of a successful delivery attempt.
type DeliverResult struct {
	StatusCode int32
	DurationMs int32
}

// Deliver sends one HTTP POST to the endpoint. Non-2xx responses and transport
// errors are returned as errors so Temporal retries with its configured policy.
func (a *WebhookDeliveryActivities) Deliver(ctx context.Context, params DeliverParams) (DeliverResult, error) {
	body, err := json.Marshal(params.Envelope)
	if err != nil {
		return DeliverResult{}, fmt.Errorf("marshal event: %w", err)
	}
	sig := computeSignature(params.SecretHash, body)

	start := time.Now()
	sc, err := sendHTTP(params.URL, body, sig, params.Envelope.EventType)
	elapsed := int32(time.Since(start).Milliseconds())

	if err != nil {
		return DeliverResult{}, err
	}
	if sc < 200 || sc >= 300 {
		return DeliverResult{}, fmt.Errorf("non-2xx status: %d", sc)
	}
	return DeliverResult{StatusCode: sc, DurationMs: elapsed}, nil
}

// RecordDeliveryParams carries the delivery attempt record to persist.
type RecordDeliveryParams struct {
	EndpointID string
	EventID    string
	EventType  string
	StatusCode *int32
	DurationMs *int32
	Error      *string
}

// RecordDelivery persists the outcome of a delivery attempt to the deliveries table.
func (a *WebhookDeliveryActivities) RecordDelivery(ctx context.Context, params RecordDeliveryParams) error {
	endpointID, err := webhooksdb.ParseUUID(params.EndpointID)
	if err != nil {
		return fmt.Errorf("parse endpoint ID: %w", err)
	}
	_, err = a.q.InsertDelivery(ctx, webhooksdb.InsertDeliveryParams{
		EndpointID: endpointID,
		EventID:    params.EventID,
		EventType:  params.EventType,
		StatusCode: params.StatusCode,
		DurationMs: params.DurationMs,
		Error:      params.Error,
	})
	return err
}

// ResetFailCount resets the consecutive failure counter for an endpoint after a success.
func (a *WebhookDeliveryActivities) ResetFailCount(ctx context.Context, endpointID string) error {
	id, err := webhooksdb.ParseUUID(endpointID)
	if err != nil {
		return fmt.Errorf("parse endpoint ID: %w", err)
	}
	return a.q.ResetFailCount(ctx, id)
}

// IncrementFailCount increments the failure counter and returns the new count.
func (a *WebhookDeliveryActivities) IncrementFailCount(ctx context.Context, endpointID string) (int32, error) {
	id, err := webhooksdb.ParseUUID(endpointID)
	if err != nil {
		return 0, fmt.Errorf("parse endpoint ID: %w", err)
	}
	return a.q.IncrementFailCount(ctx, id)
}

// DisableEndpoint disables an endpoint that has exceeded the consecutive failure threshold.
func (a *WebhookDeliveryActivities) DisableEndpoint(ctx context.Context, endpointID string) error {
	id, err := webhooksdb.ParseUUID(endpointID)
	if err != nil {
		return fmt.Errorf("parse endpoint ID: %w", err)
	}
	return a.q.DisableEndpoint(ctx, id)
}
