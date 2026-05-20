package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"go.temporal.io/sdk/client"
	enumspb "go.temporal.io/api/enums/v1"

	"github.com/fieldstone/fieldstone/internal/events"
	webhooksdb "github.com/fieldstone/fieldstone/services/webhooks/db/generated"
	nats "github.com/nats-io/nats.go"
)

// setupDispatcher subscribes to all fieldstone events and fans them out to
// matching registered webhook endpoints via durable Temporal workflows.
func setupDispatcher(js nats.JetStreamContext, q *webhooksdb.Queries, tc client.Client) error {
	_, err := js.Subscribe(events.SubjectAll, func(msg *nats.Msg) {
		var env events.EventEnvelope
		if err := json.Unmarshal(msg.Data, &env); err != nil {
			slog.Error("decode event for webhook dispatch", "error", err)
			msg.Nak()
			return
		}
		// Ack before dispatching — workflow start is non-blocking and durable.
		msg.Ack()

		go fanOut(context.Background(), q, tc, env)
	}, nats.Durable("webhooks-service"), nats.ManualAck())
	return err
}

// fanOut loads all enabled endpoints and starts a delivery workflow for each
// whose event patterns match the incoming event subject.
func fanOut(ctx context.Context, q *webhooksdb.Queries, tc client.Client, env events.EventEnvelope) {
	endpoints, err := q.ListEnabledEndpoints(ctx)
	if err != nil {
		slog.Error("list endpoints for dispatch", "error", err)
		return
	}
	for _, ep := range endpoints {
		if matchesAny(ep.Events, env.EventType) {
			ep := ep
			startDeliveryWorkflow(ctx, tc, ep, env)
		}
	}
}

// startDeliveryWorkflow starts a WebhookDeliveryWorkflow for a single endpoint+event pair.
// The workflow ID encodes both IDs so NATS redelivery is idempotent.
func startDeliveryWorkflow(ctx context.Context, tc client.Client, ep *webhooksdb.Endpoint, env events.EventEnvelope) {
	opts := client.StartWorkflowOptions{
		ID:                    fmt.Sprintf("webhook-%s-%s", webhooksdb.UUIDToStr(ep.ID), env.ID),
		TaskQueue:             WebhookTaskQueue,
		WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
	}
	if _, err := tc.ExecuteWorkflow(ctx, opts, WebhookDeliveryWorkflow, WebhookDeliveryInput{
		EndpointID: webhooksdb.UUIDToStr(ep.ID),
		URL:        ep.URL,
		SecretHash: ep.SecretHash,
		Envelope:   env,
	}); err != nil {
		slog.Error("start webhook delivery workflow",
			"endpoint_id", webhooksdb.UUIDToStr(ep.ID),
			"event_id", env.ID,
			"error", err)
	}
}

// matchesAny returns true if subject matches any of the given NATS-style patterns.
// Supported wildcards: '*' (single token), '>' (all remaining tokens).
func matchesAny(patterns []string, subject string) bool {
	for _, p := range patterns {
		if matchPattern(p, subject) {
			return true
		}
	}
	return false
}

func matchPattern(pattern, subject string) bool {
	pTokens := strings.Split(pattern, ".")
	sTokens := strings.Split(subject, ".")

	for i, p := range pTokens {
		switch p {
		case ">":
			return true
		case "*":
			if i >= len(sTokens) {
				return false
			}
		default:
			if i >= len(sTokens) || p != sTokens[i] {
				return false
			}
		}
	}
	return len(pTokens) == len(sTokens)
}
