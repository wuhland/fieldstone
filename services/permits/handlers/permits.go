package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/fieldstone/fieldstone/internal/events"
	permitsdb "github.com/fieldstone/fieldstone/services/permits/db/generated"
)

// ─── List permits ─────────────────────────────────────────────────────────────

func (h *Handler) ListPermits(w http.ResponseWriter, r *http.Request) {
	limit := int32(parseIntParam(r, "limit", 20))
	offset := int32(parseIntParam(r, "offset", 0))
	status := r.URL.Query().Get("status")

	var (
		permits []*permitsdb.Permit
		total   int64
		err     error
	)

	if status != "" {
		permits, err = h.queries.ListPermitsByStatus(r.Context(), status, limit, offset)
		if err == nil {
			total, err = h.queries.CountPermitsByStatus(r.Context(), status)
		}
	} else {
		permits, err = h.queries.ListPermits(r.Context(), limit, offset)
		if err == nil {
			total, err = h.queries.CountPermits(r.Context())
		}
	}
	if err != nil {
		slog.Error("list permits", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to list permits")
		return
	}

	out := make([]*PermitResponse, len(permits))
	for i, p := range permits {
		out[i] = permitToResponse(p)
	}
	writeJSON(w, http.StatusOK, listResponse{
		Permits: out,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	})
}

// ─── Create permit ────────────────────────────────────────────────────────────

type createPermitRequest struct {
	PermitType      string          `json:"permit_type"`
	DepartmentID    string          `json:"department_id"`
	Applicant       json.RawMessage `json:"applicant"`
	PropertyAddress string          `json:"property_address"`
	Metadata        json.RawMessage `json:"metadata"`
}

func (h *Handler) CreatePermit(w http.ResponseWriter, r *http.Request) {
	var req createPermitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.PermitType == "" || req.DepartmentID == "" || req.PropertyAddress == "" {
		writeError(w, r, http.StatusBadRequest, "permit_type, department_id, and property_address are required")
		return
	}

	deptID, err := parseUUID(req.DepartmentID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid department_id")
		return
	}

	// Get initial status from workflow service
	initialStatus, err := h.workflow.GetInitialStatus(r.Context(), "permit")
	if err != nil {
		slog.Error("get initial workflow status", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to determine initial status")
		return
	}

	// Validate metadata against city-registered schema
	metadata := defaultJSON(req.Metadata, "{}")
	if string(metadata) != "{}" {
		if errs, err := h.schemas.ValidateMetadata(r.Context(), "permit", metadata); err != nil {
			slog.Warn("schema validation error", "error", err)
			// schema fetch failure is non-fatal — log and continue
		} else if len(errs) > 0 {
			writeValidationError(w, r, errs)
			return
		}
	}

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		slog.Error("begin transaction", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to create permit")
		return
	}
	defer tx.Rollback(r.Context())

	permit, err := h.queries.WithTx(tx).CreatePermit(r.Context(), permitsdb.CreatePermitParams{
		DepartmentID:    deptID,
		PermitType:      req.PermitType,
		Status:          initialStatus,
		Applicant:       defaultJSON(req.Applicant, "{}"),
		PropertyAddress: req.PropertyAddress,
		Metadata:        metadata,
	})
	if err != nil {
		slog.Error("create permit", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to create permit")
		return
	}

	resp := permitToResponse(permit)
	if err := h.pub.PublishTx(r.Context(), tx, events.SubjectPermitCreated, "permits", events.SubjectPermitCreated, resp); err != nil {
		slog.Error("write permit.created to outbox", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to create permit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		slog.Error("commit permit create", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to create permit")
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// ─── Get permit ───────────────────────────────────────────────────────────────

func (h *Handler) GetPermit(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid permit id")
		return
	}

	permit, err := h.queries.GetPermit(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			writeError(w, r, http.StatusNotFound, "permit not found")
			return
		}
		slog.Error("get permit", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to get permit")
		return
	}

	inspections, err := h.queries.ListInspectionsByPermit(r.Context(), id)
	if err != nil {
		slog.Error("list inspections", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to load inspections")
		return
	}

	inspResp := make([]InspectionResponse, len(inspections))
	for i, insp := range inspections {
		inspResp[i] = *inspectionToResponse(insp)
	}

	writeJSON(w, http.StatusOK, PermitDetailResponse{
		PermitResponse: *permitToResponse(permit),
		Inspections:    inspResp,
	})
}

// ─── Update permit status ─────────────────────────────────────────────────────

type updateStatusRequest struct {
	Status string `json:"status"`
	Role   string `json:"role"`
}

func (h *Handler) UpdatePermitStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid permit id")
		return
	}

	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Status == "" || req.Role == "" {
		writeError(w, r, http.StatusBadRequest, "status and role are required")
		return
	}

	permit, err := h.queries.GetPermit(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			writeError(w, r, http.StatusNotFound, "permit not found")
			return
		}
		slog.Error("get permit for status update", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to get permit")
		return
	}

	if permit.Status == req.Status {
		writeError(w, r, http.StatusBadRequest, "permit is already in the requested status")
		return
	}

	// Validate transition with workflow service
	if err := h.workflow.ValidateTransition(r.Context(), "permit", permit.Status, req.Status, req.Role); err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err.Error())
		return
	}

	oldStatus := permit.Status

	// For the "approved" status, also set issued_at
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		slog.Error("begin transaction", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to update permit status")
		return
	}
	defer tx.Rollback(r.Context())

	var updated *permitsdb.Permit
	if req.Status == "approved" {
		updated, err = h.queries.WithTx(tx).SetPermitIssuedAt(r.Context(), id, req.Status)
	} else {
		updated, err = h.queries.WithTx(tx).UpdatePermitStatus(r.Context(), permitsdb.UpdatePermitStatusParams{
			ID:     id,
			Status: req.Status,
		})
	}
	if err != nil {
		slog.Error("update permit status", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to update permit status")
		return
	}

	resp := permitToResponse(updated)
	payload := map[string]any{"permit": resp, "from": oldStatus, "to": req.Status}
	if err := h.pub.PublishTx(r.Context(), tx, events.SubjectPermitStatusChanged, "permits", events.SubjectPermitStatusChanged, payload); err != nil {
		slog.Error("write permit.status_changed to outbox", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to update permit status")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		slog.Error("commit permit status update", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to update permit status")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func parseIntParam(r *http.Request, key string, def int) int {
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
