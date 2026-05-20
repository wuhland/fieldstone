package activities

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fieldstone/fieldstone/internal/events"
	"github.com/fieldstone/fieldstone/internal/outbox"
	recordsdb "github.com/fieldstone/fieldstone/services/records/db/generated"
)

// RecordsActivities implements Temporal activities for FOIA requests.
type RecordsActivities struct {
	pool *pgxpool.Pool
	pub  *outbox.Publisher
}

func NewRecordsActivities(pool *pgxpool.Pool) *RecordsActivities {
	return &RecordsActivities{pool: pool, pub: &outbox.Publisher{}}
}

type UpdateFOIAStatusParams struct {
	RequestID string
	NewStatus string
	OldStatus string
}

// UpdateFOIAStatus updates a FOIA request's status and publishes the event.
func (a *RecordsActivities) UpdateFOIAStatus(ctx context.Context, params UpdateFOIAStatusParams) error {
	id, err := parseUUID(params.RequestID)
	if err != nil {
		return fmt.Errorf("invalid request id: %w", err)
	}

	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	q := recordsdb.New(tx)

	terminalStatuses := map[string]bool{"fulfilled": true, "denied": true, "withdrawn": true}
	var f any
	var err2 error
	if terminalStatuses[params.NewStatus] {
		f, err2 = q.CloseFOIARequest(ctx, id, params.NewStatus)
	} else {
		f, err2 = q.UpdateFOIAStatus(ctx, recordsdb.UpdateFOIAStatusParams{
			ID:     id,
			Status: params.NewStatus,
		})
	}
	if err2 != nil {
		return fmt.Errorf("update FOIA status: %w", err2)
	}

	payload := map[string]any{"request": f, "from": params.OldStatus, "to": params.NewStatus}
	if err := a.pub.PublishTx(ctx, tx, events.SubjectFOIARequestStatusChanged, "records", events.SubjectFOIARequestStatusChanged, payload); err != nil {
		return fmt.Errorf("publish event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	slog.Info("FOIA status updated by activity", "request_id", params.RequestID, "status", params.NewStatus)
	return nil
}

// DeadlineExceededParams is the input for NotifyDeadlineExceeded.
type DeadlineExceededParams struct {
	RequestID string
}

// NotifyDeadlineExceeded publishes a deadline_exceeded event for the given FOIA
// request. It does not change the request's status — staff must still act.
func (a *RecordsActivities) NotifyDeadlineExceeded(ctx context.Context, params DeadlineExceededParams) error {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := a.pub.PublishTx(ctx, tx, events.SubjectFOIADeadlineExceeded, "records",
		events.SubjectFOIADeadlineExceeded, map[string]any{"request_id": params.RequestID}); err != nil {
		return fmt.Errorf("publish event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	slog.Info("FOIA deadline exceeded notification published", "request_id", params.RequestID)
	return nil
}
