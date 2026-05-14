// Package outbox implements the transactional outbox pattern for durable event
// publishing. Events are written to the local outbox table in the same database
// transaction as the business record. A background poller (Run) drains the table
// to NATS after commit, guaranteeing at-least-once delivery even across crashes.
package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/fieldstone/fieldstone/internal/events"
)

// Publisher writes events to the outbox table within a provided transaction.
// Events are NOT sent to NATS immediately — the background poller sends them
// after the transaction commits.
type Publisher struct{}

// PublishTx serialises the event envelope and inserts it into the outbox table
// as part of the caller's transaction. If the transaction rolls back, the event
// is discarded. If the transaction commits, the poller will deliver it to NATS.
func (p *Publisher) PublishTx(ctx context.Context, tx pgx.Tx, subject, sourceService, eventType string, payload any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	env := events.EventEnvelope{
		ID:            uuid.New().String(),
		SourceService: sourceService,
		EventType:     eventType,
		OccurredAt:    time.Now(),
		Payload:       json.RawMessage(payloadBytes),
		SchemaVersion: 1,
	}
	envBytes, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO outbox (subject, payload) VALUES ($1, $2)`,
		subject, envBytes,
	)
	return err
}
