package activities

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fieldstone/fieldstone/internal/events"
	"github.com/fieldstone/fieldstone/internal/outbox"
	requestsdb "github.com/fieldstone/fieldstone/services/requests/db/generated"
)

// RequestActivities implements Temporal activities for service requests.
type RequestActivities struct {
	pool *pgxpool.Pool
	pub  *outbox.Publisher
}

func NewRequestActivities(pool *pgxpool.Pool) *RequestActivities {
	return &RequestActivities{pool: pool, pub: &outbox.Publisher{}}
}

type UpdateRequestStatusParams struct {
	RequestID string
	NewStatus string
	OldStatus string
}

// UpdateRequestStatus updates a service request's status and publishes the event.
func (a *RequestActivities) UpdateRequestStatus(ctx context.Context, params UpdateRequestStatusParams) error {
	id, err := parseUUID(params.RequestID)
	if err != nil {
		return fmt.Errorf("invalid request id: %w", err)
	}

	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	q := requestsdb.New(tx)

	terminalStatuses := map[string]bool{"resolved": true, "closed": true}
	var sr any
	var err2 error
	if terminalStatuses[params.NewStatus] {
		sr, err2 = q.CloseServiceRequest(ctx, id, params.NewStatus)
	} else {
		sr, err2 = q.UpdateServiceRequestStatus(ctx, requestsdb.UpdateServiceRequestStatusParams{
			ID:     id,
			Status: params.NewStatus,
		})
	}
	if err2 != nil {
		return fmt.Errorf("update request status: %w", err2)
	}

	subject := events.SubjectServiceRequestClosed
	payload := map[string]any{"request": sr, "from": params.OldStatus, "to": params.NewStatus}
	if err := a.pub.PublishTx(ctx, tx, subject, "requests", subject, payload); err != nil {
		return fmt.Errorf("publish event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	slog.Info("request status updated by activity", "request_id", params.RequestID, "status", params.NewStatus)
	return nil
}
