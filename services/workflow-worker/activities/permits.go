package activities

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fieldstone/fieldstone/internal/events"
	"github.com/fieldstone/fieldstone/internal/outbox"
	permitsdb "github.com/fieldstone/fieldstone/services/permits/db/generated"
)

// PermitActivities implements Temporal activities that write to the permits DB.
// These are called by workflow signal/timer handlers — not by the staff HTTP path,
// which writes to DB directly in the domain handler after a Temporal Update.
type PermitActivities struct {
	pool *pgxpool.Pool
	pub  *outbox.Publisher
}

func NewPermitActivities(pool *pgxpool.Pool) *PermitActivities {
	return &PermitActivities{pool: pool, pub: &outbox.Publisher{}}
}

type UpdatePermitStatusParams struct {
	PermitID  string
	NewStatus string
	OldStatus string
}

// UpdatePermitStatus updates a permit's status and publishes the status_changed event.
// Called by workflow signal/timer transitions (e.g., resident withdrawal, expiry).
func (a *PermitActivities) UpdatePermitStatus(ctx context.Context, params UpdatePermitStatusParams) error {
	id, err := parseUUID(params.PermitID)
	if err != nil {
		return fmt.Errorf("invalid permit id: %w", err)
	}

	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	q := permitsdb.New(tx)
	permit, err := q.UpdatePermitStatus(ctx, permitsdb.UpdatePermitStatusParams{
		ID:     id,
		Status: params.NewStatus,
	})
	if err != nil {
		return fmt.Errorf("update permit status: %w", err)
	}

	payload := map[string]any{
		"permit": permit,
		"from":   params.OldStatus,
		"to":     params.NewStatus,
	}
	if err := a.pub.PublishTx(ctx, tx, events.SubjectPermitStatusChanged, "permits", events.SubjectPermitStatusChanged, payload); err != nil {
		return fmt.Errorf("publish event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	slog.Info("permit status updated by activity", "permit_id", params.PermitID, "status", params.NewStatus)
	return nil
}
