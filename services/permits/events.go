package main

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/fieldstone/fieldstone/internal/events"
	"github.com/nats-io/nats.go"
)

// publisher sends events to NATS via a buffered channel. Failed publishes are logged, not returned.
type publisher struct {
	js nats.JetStreamContext
	ch chan publishRequest
}

type publishRequest struct {
	subject string
	env     events.EventEnvelope
}

func newPublisher(js nats.JetStreamContext) *publisher {
	p := &publisher{js: js, ch: make(chan publishRequest, 1000)}
	go p.drain()
	return p
}

func (p *publisher) drain() {
	for req := range p.ch {
		data, err := json.Marshal(req.env)
		if err != nil {
			slog.Error("failed to marshal event", "error", err)
			continue
		}
		if _, err := p.js.Publish(req.subject, data); err != nil {
			slog.Error("failed to publish event", "subject", req.subject, "error", err)
		}
	}
}

func (p *publisher) publish(subject string, sourceService string, eventType string, payload interface{}) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal event payload", "error", err)
		return
	}
	env := events.EventEnvelope{
		ID:            nats.NewInbox(),
		SourceService: sourceService,
		EventType:     eventType,
		OccurredAt:    time.Now(),
		Payload:       json.RawMessage(payloadBytes),
		SchemaVersion: 1,
	}
	select {
	case p.ch <- publishRequest{subject: subject, env: env}:
	default:
		slog.Error("event publish channel full, dropping event", "subject", subject)
	}
}
