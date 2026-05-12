package main

import (
	"encoding/json"
	"log/slog"

	"github.com/fieldstone/fieldstone/internal/events"
	nats "github.com/nats-io/nats.go"
)

func subscribeToEvents(js nats.JetStreamContext, dispatch func(env events.EventEnvelope)) error {
	_, err := js.Subscribe(events.SubjectAll, func(msg *nats.Msg) {
		var env events.EventEnvelope
		if err := json.Unmarshal(msg.Data, &env); err != nil {
			slog.Error("failed to decode event for webhook dispatch", "error", err)
			msg.Nak()
			return
		}
		dispatch(env)
		msg.Ack()
	}, nats.Durable("webhooks-service"), nats.ManualAck())
	return err
}
