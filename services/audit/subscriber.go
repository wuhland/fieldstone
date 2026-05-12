package main

import (
	"encoding/json"
	"log/slog"

	"github.com/fieldstone/fieldstone/internal/events"
	"github.com/nats-io/nats.go"
)

func subscribeToAll(js nats.JetStreamContext) error {
	_, err := js.Subscribe(events.SubjectAll, func(msg *nats.Msg) {
		var env events.EventEnvelope
		if err := json.Unmarshal(msg.Data, &env); err != nil {
			slog.Error("failed to decode event", "error", err)
			msg.Nak()
			return
		}
		slog.Info("audit event received",
			"event_id", env.ID,
			"event_type", env.EventType,
			"source_service", env.SourceService,
		)
		// TODO(fieldstone): persist event to audit.events table
		msg.Ack()
	}, nats.Durable("audit-service"), nats.ManualAck())
	return err
}
