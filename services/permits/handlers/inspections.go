package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/fieldstone/fieldstone/internal/events"
	permitsdb "github.com/fieldstone/fieldstone/services/permits/db/generated"
)

// ─── Schedule inspection ──────────────────────────────────────────────────────

type scheduleInspectionRequest struct {
	InspectorID string    `json:"inspector_id"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

func (h *Handler) ScheduleInspection(w http.ResponseWriter, r *http.Request) {
	permitID, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid permit id")
		return
	}

	// Verify permit exists
	if _, err := h.queries.GetPermit(r.Context(), permitID); err != nil {
		if isNotFound(err) {
			writeError(w, r, http.StatusNotFound, "permit not found")
			return
		}
		slog.Error("get permit for inspection", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to get permit")
		return
	}

	var req scheduleInspectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.InspectorID == "" {
		writeError(w, r, http.StatusBadRequest, "inspector_id is required")
		return
	}
	if req.ScheduledAt.IsZero() {
		writeError(w, r, http.StatusBadRequest, "scheduled_at is required")
		return
	}

	inspectorID, err := parseUUID(req.InspectorID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid inspector_id")
		return
	}

	insp, err := h.queries.CreateInspection(r.Context(), permitsdb.CreateInspectionParams{
		PermitID:    permitID,
		InspectorID: inspectorID,
		ScheduledAt: req.ScheduledAt,
	})
	if err != nil {
		slog.Error("create inspection", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to schedule inspection")
		return
	}

	resp := inspectionToResponse(insp)
	h.pub.Publish(
		events.SubjectInspectionScheduled,
		"permits",
		events.SubjectInspectionScheduled,
		resp,
	)

	writeJSON(w, http.StatusCreated, resp)
}

// ─── Update inspection ────────────────────────────────────────────────────────

type updateInspectionRequest struct {
	CompletedAt *time.Time `json:"completed_at"`
	Result      *string    `json:"result"`
	Notes       *string    `json:"notes"`
}

func (h *Handler) UpdateInspection(w http.ResponseWriter, r *http.Request) {
	permitID, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid permit id")
		return
	}

	inspID, err := parseUUID(chi.URLParam(r, "iid"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid inspection id")
		return
	}

	// Verify inspection belongs to this permit
	insp, err := h.queries.GetInspection(r.Context(), inspID)
	if err != nil {
		if isNotFound(err) {
			writeError(w, r, http.StatusNotFound, "inspection not found")
			return
		}
		slog.Error("get inspection", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to get inspection")
		return
	}
	if uuidStr(insp.PermitID) != uuidStr(permitID) {
		writeError(w, r, http.StatusNotFound, "inspection not found")
		return
	}

	var req updateInspectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON body")
		return
	}

	updated, err := h.queries.UpdateInspection(r.Context(), permitsdb.UpdateInspectionParams{
		ID:          inspID,
		CompletedAt: req.CompletedAt,
		Result:      req.Result,
		Notes:       req.Notes,
	})
	if err != nil {
		slog.Error("update inspection", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to update inspection")
		return
	}

	writeJSON(w, http.StatusOK, inspectionToResponse(updated))
}
