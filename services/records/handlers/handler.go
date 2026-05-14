package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fieldstone/fieldstone/internal/middleware"
	recordsdb "github.com/fieldstone/fieldstone/services/records/db/generated"
)

// Publisher writes events to the outbox table within a transaction.
// Events reach NATS only after the transaction commits, guaranteeing
// at-least-once delivery even across process crashes.
type Publisher interface {
	PublishTx(ctx context.Context, tx pgx.Tx, subject, sourceService, eventType string, payload any) error
}

// WorkflowClient validates status transitions against the workflow service.
type WorkflowClient interface {
	ValidateTransition(ctx context.Context, resourceType, from, to, role string) error
	GetInitialStatus(ctx context.Context, resourceType string) (string, error)
}

// SchemaValidator fetches and validates metadata against a city-registered JSON Schema.
type SchemaValidator interface {
	ValidateMetadata(ctx context.Context, resourceType string, metadata json.RawMessage) ([]string, error)
}

// Handler holds injected dependencies for all FOIA endpoints.
type Handler struct {
	pool     *pgxpool.Pool
	queries  *recordsdb.Queries
	pub      Publisher
	workflow WorkflowClient
	schemas  SchemaValidator
}

func New(pool *pgxpool.Pool, q *recordsdb.Queries, pub Publisher, wf WorkflowClient, sv SchemaValidator) *Handler {
	return &Handler{pool: pool, queries: q, pub: pub, workflow: wf, schemas: sv}
}

// ─── Response types ───────────────────────────────────────────────────────────

type FOIARequestResponse struct {
	ID             string          `json:"id"`
	DepartmentID   string          `json:"department_id"`
	Status         string          `json:"status"`
	RequesterName  string          `json:"requester_name"`
	RequesterEmail string          `json:"requester_email"`
	Description    string          `json:"description"`
	DueDate        *string         `json:"due_date"` // "2006-01-02" or null
	Metadata       json.RawMessage `json:"metadata"`
	ClosedAt       *time.Time      `json:"closed_at"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type listResponse struct {
	Requests []*FOIARequestResponse `json:"requests"`
	Total    int64                  `json:"total"`
	Limit    int32                  `json:"limit"`
	Offset   int32                  `json:"offset"`
}

// ─── Conversion helpers ───────────────────────────────────────────────────────

func foiaToResponse(f *recordsdb.FOIARequest) *FOIARequestResponse {
	var dueDate *string
	if f.DueDate.Valid {
		s := f.DueDate.Time.Format("2006-01-02")
		dueDate = &s
	}
	return &FOIARequestResponse{
		ID:             uuidStr(f.ID),
		DepartmentID:   uuidStr(f.DepartmentID),
		Status:         f.Status,
		RequesterName:  f.RequesterName,
		RequesterEmail: f.RequesterEmail,
		Description:    f.Description,
		DueDate:        dueDate,
		Metadata:       f.Metadata,
		ClosedAt:       f.ClosedAt,
		CreatedAt:      f.CreatedAt,
		UpdatedAt:      f.UpdatedAt,
	}
}

func uuidStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

func parseUUID(s string) (pgtype.UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: u, Valid: true}, nil
}

// parseDateParam parses an optional "2006-01-02" date string into pgtype.Date.
// Returns a zero pgtype.Date (Valid=false) when s is empty.
func parseDateParam(s string) (pgtype.Date, error) {
	if s == "" {
		return pgtype.Date{}, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return pgtype.Date{}, err
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

// ─── Response writers ─────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	writeJSON(w, status, map[string]string{
		"error":      msg,
		"request_id": middleware.GetRequestID(r.Context()),
	})
}

func writeValidationError(w http.ResponseWriter, r *http.Request, fields []string) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
		"error":      "metadata validation failed",
		"fields":     fields,
		"request_id": middleware.GetRequestID(r.Context()),
	})
}

func isNotFound(err error) bool {
	return errors.Is(err, recordsdb.ErrNotFound)
}

func defaultJSON(v json.RawMessage, fallback string) json.RawMessage {
	if len(v) == 0 || string(v) == "null" {
		return json.RawMessage(fallback)
	}
	return v
}
