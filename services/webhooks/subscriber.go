package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/fieldstone/fieldstone/internal/events"
	webhooksdb "github.com/fieldstone/fieldstone/services/webhooks/db/generated"
	nats "github.com/nats-io/nats.go"
)

// setupDispatcher subscribes to all fieldstone events and fans them out to
// matching registered webhook endpoints. The NATS message is acked immediately;
// HTTP delivery happens in a separate goroutine so a slow endpoint never blocks
// event processing.
func setupDispatcher(js nats.JetStreamContext, q *webhooksdb.Queries) error {
	_, err := js.Subscribe(events.SubjectAll, func(msg *nats.Msg) {
		var env events.EventEnvelope
		if err := json.Unmarshal(msg.Data, &env); err != nil {
			slog.Error("decode event for webhook dispatch", "error", err)
			msg.Nak()
			return
		}
		// Ack before dispatching — webhook delivery is best-effort and must not
		// block NATS message processing.
		msg.Ack()

		go fanOut(context.Background(), q, env)
	}, nats.Durable("webhooks-service"), nats.ManualAck())
	return err
}

// fanOut loads all enabled endpoints and dispatches to those whose event
// patterns match the incoming event subject.
func fanOut(ctx context.Context, q *webhooksdb.Queries, env events.EventEnvelope) {
	endpoints, err := q.ListEnabledEndpoints(ctx)
	if err != nil {
		slog.Error("list endpoints for dispatch", "error", err)
		return
	}
	for _, ep := range endpoints {
		if matchesAny(ep.Events, env.EventType) {
			ep := ep
			go deliverToEndpoint(ctx, q, ep, env)
		}
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
