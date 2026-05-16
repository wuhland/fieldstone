package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fieldstone/fieldstone/internal/events"
	"github.com/fieldstone/fieldstone/internal/middleware"
	webhooksdb "github.com/fieldstone/fieldstone/services/webhooks/db/generated"
)

type WebhookHandler struct {
	pool    *pgxpool.Pool
	queries *webhooksdb.Queries
	// dispatch is called by TestWebhook to send a live test event.
	dispatch func(ep *webhooksdb.Endpoint, env events.EventEnvelope)
}

func NewWebhookHandler(pool *pgxpool.Pool, q *webhooksdb.Queries, dispatch func(*webhooksdb.Endpoint, events.EventEnvelope)) *WebhookHandler {
	return &WebhookHandler{pool: pool, queries: q, dispatch: dispatch}
}

// ─── Response types ───────────────────────────────────────────────────────────

type EndpointResponse struct {
	ID          string     `json:"id"`
	URL         string     `json:"url"`
	Events      []string   `json:"events"`
	Description *string    `json:"description"`
	Enabled     bool       `json:"enabled"`
	FailCount   int32      `json:"fail_count"`
	CreatedAt   time.Time  `json:"created_at"`
}

type EndpointCreatedResponse struct {
	EndpointResponse
	Secret string `json:"secret"` // shown only at creation time
}

type DeliveryResponse struct {
	ID          string     `json:"id"`
	EventID     string     `json:"event_id"`
	EventType   string     `json:"event_type"`
	StatusCode  *int32     `json:"status_code"`
	DurationMs  *int32     `json:"duration_ms"`
	Error       *string    `json:"error"`
	DeliveredAt time.Time  `json:"delivered_at"`
}

type EndpointDetailResponse struct {
	EndpointResponse
	RecentDeliveries []DeliveryResponse `json:"recent_deliveries"`
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func toResponse(e *webhooksdb.Endpoint) EndpointResponse {
	return EndpointResponse{
		ID:          webhooksdb.UUIDToStr(e.ID),
		URL:         e.URL,
		Events:      e.Events,
		Description: e.Description,
		Enabled:     e.Enabled,
		FailCount:   e.FailCount,
		CreatedAt:   e.CreatedAt,
	}
}

func toDeliveryResponse(d *webhooksdb.Delivery) DeliveryResponse {
	return DeliveryResponse{
		ID:          webhooksdb.UUIDToStr(d.ID),
		EventID:     d.EventID,
		EventType:   d.EventType,
		StatusCode:  d.StatusCode,
		DurationMs:  d.DurationMs,
		Error:       d.Error,
		DeliveredAt: d.DeliveredAt,
	}
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

func parseID(r *http.Request) (pgtype.UUID, error) {
	return webhooksdb.ParseUUID(chi.URLParam(r, "id"))
}

// ─── List webhooks ────────────────────────────────────────────────────────────

func (h *WebhookHandler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	endpoints, err := h.queries.ListEndpoints(r.Context())
	if err != nil {
		slog.Error("list webhooks", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to list webhooks")
		return
	}

	out := make([]EndpointResponse, len(endpoints))
	for i, ep := range endpoints {
		out[i] = toResponse(ep)
	}
	writeJSON(w, http.StatusOK, map[string]any{"webhooks": out})
}

// ─── Create webhook ───────────────────────────────────────────────────────────

type createWebhookRequest struct {
	URL         string   `json:"url"`
	Secret      string   `json:"secret"`
	Events      []string `json:"events"`
	Description string   `json:"description"`
}

func (h *WebhookHandler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	var req createWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.URL == "" || req.Secret == "" || len(req.Events) == 0 {
		writeError(w, r, http.StatusBadRequest, "url, secret, and events are required")
		return
	}

	// Validate event patterns — each must start with "fieldstone."
	for _, pattern := range req.Events {
		if len(pattern) < 12 || pattern[:11] != "fieldstone." {
			writeError(w, r, http.StatusBadRequest, "event patterns must start with 'fieldstone.'")
			return
		}
	}

	var desc *string
	if req.Description != "" {
		desc = &req.Description
	}

	// Store the secret in plaintext for HMAC signing.
	// TODO(fieldstone): encrypt at rest using AES-256-GCM with a key from env.
	ep, err := h.queries.CreateEndpoint(r.Context(), webhooksdb.CreateEndpointParams{
		URL:         req.URL,
		SecretHash:  req.Secret,
		Events:      req.Events,
		Description: desc,
	})
	if err != nil {
		slog.Error("create webhook", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to create webhook")
		return
	}

	// Return secret only at creation time — it will never be returned again.
	writeJSON(w, http.StatusCreated, EndpointCreatedResponse{
		EndpointResponse: toResponse(ep),
		Secret:           req.Secret,
	})
}

// ─── Get webhook ──────────────────────────────────────────────────────────────

func (h *WebhookHandler) GetWebhook(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid webhook id")
		return
	}

	ep, err := h.queries.GetEndpointByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, webhooksdb.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "webhook not found")
			return
		}
		slog.Error("get webhook", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to get webhook")
		return
	}

	deliveries, err := h.queries.ListDeliveriesByEndpoint(r.Context(), id, 100)
	if err != nil {
		slog.Error("list deliveries", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to load delivery log")
		return
	}

	drs := make([]DeliveryResponse, len(deliveries))
	for i, d := range deliveries {
		drs[i] = toDeliveryResponse(d)
	}
	writeJSON(w, http.StatusOK, EndpointDetailResponse{
		EndpointResponse: toResponse(ep),
		RecentDeliveries: drs,
	})
}

// ─── Delete webhook ───────────────────────────────────────────────────────────

func (h *WebhookHandler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid webhook id")
		return
	}

	// Verify it exists first
	if _, err := h.queries.GetEndpointByID(r.Context(), id); err != nil {
		if errors.Is(err, webhooksdb.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "webhook not found")
			return
		}
		slog.Error("get webhook for delete", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to get webhook")
		return
	}

	// Delete deliveries then endpoint within a transaction.
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "failed to delete webhook")
		return
	}
	defer tx.Rollback(r.Context())

	q := h.queries.WithTx(tx)
	if err := q.DeleteEndpointDeliveries(r.Context(), id); err != nil {
		slog.Error("delete deliveries", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to delete webhook")
		return
	}
	if err := q.DeleteEndpoint(r.Context(), id); err != nil {
		slog.Error("delete endpoint", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to delete webhook")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, r, http.StatusInternalServerError, "failed to delete webhook")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ─── Test webhook ─────────────────────────────────────────────────────────────

func (h *WebhookHandler) TestWebhook(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid webhook id")
		return
	}

	ep, err := h.queries.GetEndpointByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, webhooksdb.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "webhook not found")
			return
		}
		slog.Error("get webhook for test", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to get webhook")
		return
	}

	testEnv := events.EventEnvelope{
		ID:            uuid.New().String(),
		OccurredAt:    time.Now(),
		SourceService: "webhooks",
		EventType:     "fieldstone.webhooks.test",
		SchemaVersion: 1,
	}

	// Deliver synchronously so we can return the result.
	h.dispatch(ep, testEnv)

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "test dispatched",
		"event_id": testEnv.ID,
	})
}
