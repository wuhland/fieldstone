package outbox

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	nats "github.com/nats-io/nats.go"
)

// Run polls the outbox table every second, publishing pending events to NATS
// and deleting them after acknowledgement. It exits when ctx is cancelled.
// Run should be called in a goroutine; pass the same pool the service uses so
// the poller operates within the same schema (search_path) as the service.
func Run(ctx context.Context, pool *pgxpool.Pool, js nats.JetStreamContext) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			// Drain once more before exiting to minimise latency at shutdown
			if err := drain(context.Background(), pool, js); err != nil {
				slog.Error("outbox: final drain on shutdown", "error", err)
			}
			return
		case <-ticker.C:
			if err := drain(ctx, pool, js); err != nil {
				slog.Error("outbox: drain error", "error", err)
			}
		}
	}
}

func drain(ctx context.Context, pool *pgxpool.Pool, js nats.JetStreamContext) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// FOR UPDATE SKIP LOCKED lets multiple instances run without deadlocking.
	rows, err := tx.Query(ctx, `
		SELECT id, subject, payload
		FROM outbox
		ORDER BY created_at
		LIMIT 100
		FOR UPDATE SKIP LOCKED
	`)
	if err != nil {
		return err
	}

	type entry struct {
		id      pgtype.UUID
		subject string
		payload []byte
	}

	var pending []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.id, &e.subject, &e.payload); err != nil {
			rows.Close()
			return err
		}
		pending = append(pending, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(pending) == 0 {
		return tx.Rollback(ctx)
	}

	for _, e := range pending {
		if _, err := js.Publish(e.subject, e.payload); err != nil {
			// Return without committing — the row stays for the next tick.
			slog.Error("outbox: NATS publish failed", "subject", e.subject, "error", err)
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM outbox WHERE id = $1`, e.id); err != nil {
			return err
		}
	}

	slog.Debug("outbox: published events", "count", len(pending))
	return tx.Commit(ctx)
}
