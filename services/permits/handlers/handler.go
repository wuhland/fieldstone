package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fieldstone/fieldstone/internal/middleware"
	permitsdb "github.com/fieldstone/fieldstone/services/permits/db/generated"
)

// Publisher publishes domain events to NATS after a successful write.
type Publisher interface {
	Publish(subject, sourceService, eventType string, payload any)
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

// Handler holds injected dependencies for all permit endpoints.
type Handler struct {
	queries  *permitsdb.Queries
	pub      Publisher
	workflow WorkflowClient
	schemas  SchemaValidator
}

func New(q *permitsdb.Queries, pub Publisher, wf WorkflowClient, sv SchemaValidator) *Handler {
	return &Handler{queries: q, pub: pub, workflow: wf, schemas: sv}
}

// ─── Response types ───────────────────────────────────────────────────────────

type PermitResponse struct {
	ID              string          `json:"id"`
	DepartmentID    string          `json:"department_id"`
	PermitType      string          `json:"permit_type"`
	Status          string          `json:"status"`
	Applicant       json.RawMessage `json:"applicant"`
	PropertyAddress string          `json:"property_address"`
	Metadata        json.RawMessage `json:"metadata"`
	SubmittedAt     time.Time       `json:"submitted_at"`
	IssuedAt        *time.Time      `json:"issued_at"`
	ExpiresAt       *time.Time      `json:"expires_at"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type PermitDetailResponse struct {
	PermitResponse
	Inspections []InspectionResponse `json:"inspections"`
}

type InspectionResponse struct {
	ID          string     `json:"id"`
	PermitID    string     `json:"permit_id"`
	InspectorID string     `json:"inspector_id"`
	ScheduledAt time.Time  `json:"scheduled_at"`
	CompletedAt *time.Time `json:"completed_at"`
	Result      *string    `json:"result"`
	Notes       *string    `json:"notes"`
	CreatedAt   time.Time  `json:"created_at"`
}

type listResponse struct {
	Permits []*PermitResponse `json:"permits"`
	Total   int64             `json:"total"`
	Limit   int32             `json:"limit"`
	Offset  int32             `json:"offset"`
}

type errorResponse struct {
	Error     string `json:"error"`
	RequestID string `json:"request_id"`
}

type validationErrorResponse struct {
	Error     string   `json:"error"`
	Fields    []string `json:"fields"`
	RequestID string   `json:"request_id"`
}

// ─── Conversion helpers ───────────────────────────────────────────────────────

func permitToResponse(p *permitsdb.Permit) *PermitResponse {
	return &PermitResponse{
		ID:              uuidStr(p.ID),
		DepartmentID:    uuidStr(p.DepartmentID),
		PermitType:      p.PermitType,
		Status:          p.Status,
		Applicant:       p.Applicant,
		PropertyAddress: p.PropertyAddress,
		Metadata:        p.Metadata,
		SubmittedAt:     p.SubmittedAt,
		IssuedAt:        p.IssuedAt,
		ExpiresAt:       p.ExpiresAt,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}

func inspectionToResponse(i *permitsdb.Inspection) *InspectionResponse {
	return &InspectionResponse{
		ID:          uuidStr(i.ID),
		PermitID:    uuidStr(i.PermitID),
		InspectorID: uuidStr(i.InspectorID),
		ScheduledAt: i.ScheduledAt,
		CompletedAt: i.CompletedAt,
		Result:      i.Result,
		Notes:       i.Notes,
		CreatedAt:   i.CreatedAt,
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

// ─── Response writers ─────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	writeJSON(w, status, errorResponse{
		Error:     msg,
		RequestID: middleware.GetRequestID(r.Context()),
	})
}

func writeValidationError(w http.ResponseWriter, r *http.Request, fields []string) {
	writeJSON(w, http.StatusUnprocessableEntity, validationErrorResponse{
		Error:     "metadata validation failed",
		Fields:    fields,
		RequestID: middleware.GetRequestID(r.Context()),
	})
}

func isNotFound(err error) bool {
	return errors.Is(err, permitsdb.ErrNotFound)
}

func defaultJSON(v json.RawMessage, fallback string) json.RawMessage {
	if len(v) == 0 || string(v) == "null" {
		return json.RawMessage(fallback)
	}
	return v
}
