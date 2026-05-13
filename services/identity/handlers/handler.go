package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fieldstone/fieldstone/internal/middleware"
	identitydb "github.com/fieldstone/fieldstone/services/identity/db/generated"
)

// Publisher publishes domain events after a successful write.
type Publisher interface {
	Publish(subject, sourceService, eventType string, payload any)
}

// Handler holds injected dependencies for all identity endpoints.
type Handler struct {
	queries *identitydb.Queries
	pub     Publisher
}

func New(q *identitydb.Queries, pub Publisher) *Handler {
	return &Handler{queries: q, pub: pub}
}

// ─── Response types ───────────────────────────────────────────────────────────

type DepartmentResponse struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Slug      string          `json:"slug"`
	Config    json.RawMessage `json:"config"`
	CreatedAt time.Time       `json:"created_at"`
}

type UserResponse struct {
	ID           string    `json:"id"`
	DepartmentID string    `json:"department_id"`
	OIDCSub      string    `json:"oidc_sub"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type FieldSchemaResponse struct {
	ID           string          `json:"id"`
	ResourceType string          `json:"resource_type"`
	Schema       json.RawMessage `json:"schema"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// ─── Conversions ──────────────────────────────────────────────────────────────

func deptToResponse(d *identitydb.Department) *DepartmentResponse {
	return &DepartmentResponse{
		ID:        uuidStr(d.ID),
		Name:      d.Name,
		Slug:      d.Slug,
		Config:    d.Config,
		CreatedAt: d.CreatedAt,
	}
}

func userToResponse(u *identitydb.StaffUser) *UserResponse {
	return &UserResponse{
		ID:           uuidStr(u.ID),
		DepartmentID: uuidStr(u.DepartmentID),
		OIDCSub:      u.OIDCSub,
		Email:        u.Email,
		Role:         u.Role,
		CreatedAt:    u.CreatedAt,
	}
}

func schemaToResponse(s *identitydb.FieldSchema) *FieldSchemaResponse {
	return &FieldSchemaResponse{
		ID:           uuidStr(s.ID),
		ResourceType: s.ResourceType,
		Schema:       s.Schema,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

// ─── UUID helpers ─────────────────────────────────────────────────────────────

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

func isNotFound(err error) bool {
	return errors.Is(err, identitydb.ErrNotFound)
}

func defaultJSON(v json.RawMessage, fallback string) json.RawMessage {
	if len(v) == 0 || string(v) == "null" {
		return json.RawMessage(fallback)
	}
	return v
}
