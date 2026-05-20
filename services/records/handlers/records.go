package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/fieldstone/fieldstone/internal/events"
	recordsdb "github.com/fieldstone/fieldstone/services/records/db/generated"
)

// terminalStatuses are the FOIA statuses that set closed_at when reached.
var terminalStatuses = map[string]bool{
	"fulfilled": true,
	"denied":    true,
	"withdrawn": true,
}

// ─── List FOIA requests ───────────────────────────────────────────────────────

// ListFOIARequests godoc
// @Summary      List FOIA requests (staff)
// @Tags         records
// @Produce      json
// @Param        limit   query  int     false  "Max results"       default(20)
// @Param        offset  query  int     false  "Pagination offset"  default(0)
// @Param        status  query  string  false  "Filter by status (received, processing, awaiting_clarification, fulfilled, denied, withdrawn)"
// @Success      200  {object}  listResponse
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /v1/records/foia [get]
func (h *Handler) ListFOIARequests(w http.ResponseWriter, r *http.Request) {
	limit := int32(parseIntParam(r, "limit", 20))
	offset := int32(parseIntParam(r, "offset", 0))
	status := r.URL.Query().Get("status")

	var (
		reqs  []*recordsdb.FOIARequest
		total int64
		err   error
	)

	if status != "" {
		reqs, err = h.queries.ListFOIARequestsByStatus(r.Context(), status, limit, offset)
		if err == nil {
			total, err = h.queries.CountFOIARequestsByStatus(r.Context(), status)
		}
	} else {
		reqs, err = h.queries.ListFOIARequests(r.Context(), limit, offset)
		if err == nil {
			total, err = h.queries.CountFOIARequests(r.Context())
		}
	}
	if err != nil {
		slog.Error("list FOIA requests", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to list requests")
		return
	}

	out := make([]*FOIARequestResponse, len(reqs))
	for i, f := range reqs {
		out[i] = foiaToResponse(f)
	}
	writeJSON(w, http.StatusOK, listResponse{
		Requests: out,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	})
}

// ─── Create FOIA request (public endpoint) ────────────────────────────────────

type createFOIARequestInput struct {
	DepartmentID   string          `json:"department_id"`
	RequesterName  string          `json:"requester_name"`
	RequesterEmail string          `json:"requester_email"`
	Description    string          `json:"description"`
	DueDate        string          `json:"due_date"` // optional "2006-01-02"
	Metadata       json.RawMessage `json:"metadata"`
}

// CreateFOIARequest godoc
// @Summary      Submit a FOIA request
// @Description  Requires a resident or staff JWT. The authenticated resident's OIDC sub
// @Description  is stored as resident_id so they can retrieve their own request later.
// @Tags         records
// @Accept       json
// @Produce      json
// @Param        body  body  createFOIARequestInput  true  "FOIA request"
// @Success      201  {object}  FOIARequestResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      422  {object}  map[string]string  "Metadata validation failed"
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /v1/records/foia [post]
func (h *Handler) CreateFOIARequest(w http.ResponseWriter, r *http.Request) {
	var req createFOIARequestInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.DepartmentID == "" || req.RequesterName == "" || req.RequesterEmail == "" || req.Description == "" {
		writeError(w, r, http.StatusBadRequest,
			"department_id, requester_name, requester_email, and description are required")
		return
	}

	deptID, err := parseUUID(req.DepartmentID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid department_id")
		return
	}

	dueDate, err := parseDateParam(req.DueDate)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "due_date must be in YYYY-MM-DD format")
		return
	}

	initialStatus, err := h.workflow.GetInitialStatus(r.Context(), "foia_request")
	if err != nil {
		slog.Error("get initial workflow status", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to determine initial status")
		return
	}

	metadata := defaultJSON(req.Metadata, "{}")
	if string(metadata) != "{}" {
		if errs, err := h.schemas.ValidateMetadata(r.Context(), "foia_request", metadata); err != nil {
			slog.Warn("schema validation error", "error", err)
		} else if len(errs) > 0 {
			writeValidationError(w, r, errs)
			return
		}
	}

	var residentID *string
	if sub := residentSubFromRequest(r); sub != "" {
		residentID = &sub
	}

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		slog.Error("begin transaction", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to create request")
		return
	}
	defer tx.Rollback(r.Context())

	f, err := h.queries.WithTx(tx).CreateFOIARequest(r.Context(), recordsdb.CreateFOIARequestParams{
		DepartmentID:   deptID,
		Status:         initialStatus,
		RequesterName:  req.RequesterName,
		RequesterEmail: req.RequesterEmail,
		Description:    req.Description,
		DueDate:        dueDate,
		ResidentID:     residentID,
		Metadata:       metadata,
	})
	if err != nil {
		slog.Error("create FOIA request", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to create request")
		return
	}

	resp := foiaToResponse(f)
	if err := h.pub.PublishTx(r.Context(), tx, events.SubjectFOIARequestCreated, "records", events.SubjectFOIARequestCreated, resp); err != nil {
		slog.Error("write foia_request.created to outbox", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to create request")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		slog.Error("commit FOIA create", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to create request")
		return
	}

	var deadline *time.Time
	if dueDate.Valid {
		t := dueDate.Time
		deadline = &t
	}
	if err := h.workflow.StartWorkflow(r.Context(), "foia_request", resp.ID, residentID, deadline); err != nil {
		slog.Warn("failed to start foia_request workflow", "request_id", resp.ID, "error", err)
	}

	writeJSON(w, http.StatusCreated, resp)
}

// ─── Get FOIA request ─────────────────────────────────────────────────────────

// GetFOIARequest godoc
// @Summary      Get a FOIA request
// @Description  Staff can retrieve any FOIA request. Residents can only retrieve their own.
// @Tags         records
// @Produce      json
// @Param        id  path  string  true  "FOIA request UUID"
// @Success      200  {object}  FOIARequestResponse
// @Failure      400  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /v1/records/foia/{id} [get]
func (h *Handler) GetFOIARequest(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request id")
		return
	}

	f, err := h.queries.GetFOIARequest(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			writeError(w, r, http.StatusNotFound, "request not found")
			return
		}
		slog.Error("get FOIA request", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to get request")
		return
	}

	if sub := residentSubFromRequest(r); sub != "" {
		if f.ResidentID == nil || *f.ResidentID != sub {
			writeError(w, r, http.StatusForbidden, "forbidden")
			return
		}
	}

	writeJSON(w, http.StatusOK, foiaToResponse(f))
}

// ─── Update FOIA status ───────────────────────────────────────────────────────

type updateStatusInput struct {
	Status string `json:"status"`
	Role   string `json:"role"`
}

// UpdateFOIAStatus godoc
// @Summary      Update FOIA request status
// @Description  Validates the transition via the workflow service. Terminal statuses
// @Description  (fulfilled, denied, withdrawn) also set closed_at.
// @Tags         records
// @Accept       json
// @Produce      json
// @Param        id    path  string             true  "FOIA request UUID"
// @Param        body  body  updateStatusInput   true  "Status transition"
// @Success      200  {object}  FOIARequestResponse
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      422  {object}  map[string]string  "Transition not allowed by workflow"
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /v1/records/foia/{id}/status [patch]
func (h *Handler) UpdateFOIAStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request id")
		return
	}

	var req updateStatusInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Status == "" || req.Role == "" {
		writeError(w, r, http.StatusBadRequest, "status and role are required")
		return
	}

	f, err := h.queries.GetFOIARequest(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			writeError(w, r, http.StatusNotFound, "request not found")
			return
		}
		slog.Error("get FOIA request for status update", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to get request")
		return
	}

	if f.Status == req.Status {
		writeError(w, r, http.StatusBadRequest, "request is already in the requested status")
		return
	}

	if err := h.workflow.ValidateTransition(r.Context(), "foia_request", chi.URLParam(r, "id"), f.Status, req.Status, req.Role); err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err.Error())
		return
	}

	oldStatus := f.Status

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		slog.Error("begin transaction", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to update status")
		return
	}
	defer tx.Rollback(r.Context())

	var updated *recordsdb.FOIARequest
	if terminalStatuses[req.Status] {
		updated, err = h.queries.WithTx(tx).CloseFOIARequest(r.Context(), id, req.Status)
	} else {
		updated, err = h.queries.WithTx(tx).UpdateFOIAStatus(r.Context(), recordsdb.UpdateFOIAStatusParams{
			ID:     id,
			Status: req.Status,
		})
	}
	if err != nil {
		slog.Error("update FOIA status", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to update status")
		return
	}

	resp := foiaToResponse(updated)
	payload := map[string]any{"request": resp, "from": oldStatus, "to": req.Status}
	if err := h.pub.PublishTx(r.Context(), tx, events.SubjectFOIARequestStatusChanged, "records", events.SubjectFOIARequestStatusChanged, payload); err != nil {
		slog.Error("write foia_request.status_changed to outbox", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to update status")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		slog.Error("commit FOIA status update", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to update status")
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
