package main

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	nats "github.com/nats-io/nats.go"

	"github.com/fieldstone/fieldstone/internal/events"
	auditdb "github.com/fieldstone/fieldstone/services/audit/db/generated"
)

// subscribeToAll subscribes to every fieldstone event and persists each one
// to audit.events. The NATS message is nak'd on decode or DB errors so that
// JetStream redelivers it — ensuring at-least-once persistence.
func subscribeToAll(js nats.JetStreamContext, q *auditdb.Queries) error {
	_, err := js.Subscribe(events.SubjectAll, func(msg *nats.Msg) {
		var env events.EventEnvelope
		if err := json.Unmarshal(msg.Data, &env); err != nil {
			slog.Error("audit: decode event", "error", err)
			msg.Nak()
			return
		}

		if err := persistEvent(context.Background(), q, env); err != nil {
			slog.Error("audit: persist event",
				"event_id", env.ID,
				"event_type", env.EventType,
				"error", err,
			)
			msg.Nak()
			return
		}

		slog.Debug("audit: event persisted",
			"event_id", env.ID,
			"event_type", env.EventType,
			"source_service", env.SourceService,
		)
		msg.Ack()
	}, nats.Durable("audit-service"), nats.ManualAck())
	return err
}

func persistEvent(ctx context.Context, q *auditdb.Queries, env events.EventEnvelope) error {
	id, err := uuid.Parse(env.ID)
	if err != nil {
		return err
	}

	var actorJSON json.RawMessage
	if env.Actor != nil {
		b, err := json.Marshal(env.Actor)
		if err != nil {
			return err
		}
		actorJSON = b
	}

	_, err = q.InsertEvent(ctx, auditdb.InsertEventParams{
		ID:            pgtype.UUID{Bytes: [16]byte(id), Valid: true},
		OccurredAt:    env.OccurredAt,
		SourceService: env.SourceService,
		EventType:     env.EventType,
		Payload:       env.Payload,
		Actor:         actorJSON,
	})
	return err
}
