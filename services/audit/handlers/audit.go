package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fieldstone/fieldstone/internal/middleware"
	auditdb "github.com/fieldstone/fieldstone/services/audit/db/generated"
)

// Handler holds the DB queries for audit endpoints.
type Handler struct {
	queries *auditdb.Queries
}

func New(q *auditdb.Queries) *Handler {
	return &Handler{queries: q}
}

// ─── Response types ───────────────────────────────────────────────────────────

type EventResponse struct {
	ID            string          `json:"id"`
	OccurredAt    time.Time       `json:"occurred_at"`
	SourceService string          `json:"source_service"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	Actor         json.RawMessage `json:"actor"`
	IndexedAt     time.Time       `json:"indexed_at"`
}

type listResponse struct {
	Events []*EventResponse `json:"events"`
	Total  int64            `json:"total"`
	Limit  int32            `json:"limit"`
	Offset int32            `json:"offset"`
}

func toResponse(e *auditdb.Event) *EventResponse {
	actor := e.Actor
	if len(actor) == 0 {
		actor = json.RawMessage("null")
	}
	return &EventResponse{
		ID:            uuidStr(e.ID),
		OccurredAt:    e.OccurredAt,
		SourceService: e.SourceService,
		EventType:     e.EventType,
		Payload:       e.Payload,
		Actor:         actor,
		IndexedAt:     e.IndexedAt,
	}
}

// ─── List events ──────────────────────────────────────────────────────────────

// ListEvents handles GET /v1/audit
// Filters: limit, offset, event_type, source_service, from (RFC3339), to (RFC3339)
// ListEvents godoc
// @Summary      List audit events
// @Description  Returns a paginated, filterable view of the immutable audit log.
// @Description  All filter parameters are optional and combinable.
// @Tags         audit
// @Produce      json
// @Param        limit           query  int     false  "Max results"           default(50)
// @Param        offset          query  int     false  "Pagination offset"      default(0)
// @Param        event_type      query  string  false  "e.g. fieldstone.permits.permit.created"
// @Param        source_service  query  string  false  "e.g. permits"
// @Param        from            query  string  false  "RFC3339 start time e.g. 2026-05-01T00:00:00Z"
// @Param        to              query  string  false  "RFC3339 end time"
// @Success      200  {object}  listResponse
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /v1/audit [get]
func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	params := auditdb.ListEventsParams{
		Limit:  int32(intParam(r, "limit", 50)),
		Offset: int32(intParam(r, "offset", 0)),
	}
	if v := r.URL.Query().Get("event_type"); v != "" {
		params.EventType = &v
	}
	if v := r.URL.Query().Get("source_service"); v != "" {
		params.SourceService = &v
	}
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			params.From = &t
		} else {
			writeError(w, r, http.StatusBadRequest, "from must be RFC3339 (e.g. 2026-05-01T00:00:00Z)")
			return
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			params.To = &t
		} else {
			writeError(w, r, http.StatusBadRequest, "to must be RFC3339")
			return
		}
	}

	evts, err := h.queries.ListEvents(r.Context(), params)
	if err != nil {
		slog.Error("list audit events", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to list events")
		return
	}

	total, err := h.queries.CountEvents(r.Context(), params)
	if err != nil {
		slog.Error("count audit events", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to count events")
		return
	}

	out := make([]*EventResponse, len(evts))
	for i, e := range evts {
		out[i] = toResponse(e)
	}
	writeJSON(w, http.StatusOK, listResponse{
		Events: out,
		Total:  total,
		Limit:  params.Limit,
		Offset: params.Offset,
	})
}

// ─── Get event ────────────────────────────────────────────────────────────────

// GetEvent godoc
// @Summary      Get a single audit event by ID
// @Tags         audit
// @Produce      json
// @Param        id  path  string  true  "Event UUID"
// @Success      200  {object}  EventResponse
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /v1/audit/{id} [get]
func (h *Handler) GetEvent(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	u, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid event id")
		return
	}

	e, err := h.queries.GetEvent(r.Context(), pgtype.UUID{Bytes: [16]byte(u), Valid: true})
	if err != nil {
		if errors.Is(err, auditdb.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "event not found")
			return
		}
		slog.Error("get audit event", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to get event")
		return
	}

	writeJSON(w, http.StatusOK, toResponse(e))
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func uuidStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "error", err)
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	writeJSON(w, status, map[string]string{
		"error":      msg,
		"request_id": middleware.GetRequestID(r.Context()),
	})
}

func intParam(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return def
	}
	return v
}
