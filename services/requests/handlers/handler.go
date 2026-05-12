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
	requestsdb "github.com/fieldstone/fieldstone/services/requests/db/generated"
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

// Handler holds injected dependencies for all service request endpoints.
type Handler struct {
	queries  *requestsdb.Queries
	pub      Publisher
	workflow WorkflowClient
	schemas  SchemaValidator
}

func New(q *requestsdb.Queries, pub Publisher, wf WorkflowClient, sv SchemaValidator) *Handler {
	return &Handler{queries: q, pub: pub, workflow: wf, schemas: sv}
}

// ─── Response types ───────────────────────────────────────────────────────────

type ServiceRequestResponse struct {
	ID             string          `json:"id"`
	DepartmentID   string          `json:"department_id"`
	RequestType    string          `json:"request_type"`
	Status         string          `json:"status"`
	Description    string          `json:"description"`
	Location       json.RawMessage `json:"location"`
	SubmitterEmail *string         `json:"submitter_email"`
	AssignedTo     *string         `json:"assigned_to"`
	Metadata       json.RawMessage `json:"metadata"`
	ClosedAt       *time.Time      `json:"closed_at"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type listResponse struct {
	Requests []*ServiceRequestResponse `json:"requests"`
	Total    int64                     `json:"total"`
	Limit    int32                     `json:"limit"`
	Offset   int32                     `json:"offset"`
}

// ─── Conversion helpers ───────────────────────────────────────────────────────

func srToResponse(sr *requestsdb.ServiceRequest) *ServiceRequestResponse {
	var assignedTo *string
	if sr.AssignedTo.Valid {
		s := uuid.UUID(sr.AssignedTo.Bytes).String()
		assignedTo = &s
	}
	return &ServiceRequestResponse{
		ID:             uuidStr(sr.ID),
		DepartmentID:   uuidStr(sr.DepartmentID),
		RequestType:    sr.RequestType,
		Status:         sr.Status,
		Description:    sr.Description,
		Location:       sr.Location,
		SubmitterEmail: sr.SubmitterEmail,
		AssignedTo:     assignedTo,
		Metadata:       sr.Metadata,
		ClosedAt:       sr.ClosedAt,
		CreatedAt:      sr.CreatedAt,
		UpdatedAt:      sr.UpdatedAt,
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
	return errors.Is(err, requestsdb.ErrNotFound)
}

func defaultJSON(v json.RawMessage, fallback string) json.RawMessage {
	if len(v) == 0 || string(v) == "null" {
		return json.RawMessage(fallback)
	}
	return v
}
