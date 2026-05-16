package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/fieldstone/fieldstone/internal/events"
	requestsdb "github.com/fieldstone/fieldstone/services/requests/db/generated"
)

// terminalStatuses are the statuses that set closed_at when reached.
var terminalStatuses = map[string]bool{
	"resolved": true,
	"closed":   true,
}

// ─── List requests ────────────────────────────────────────────────────────────

// ListRequests godoc
// @Summary      List service requests (staff)
// @Tags         requests
// @Produce      json
// @Param        limit   query  int     false  "Max results"       default(20)
// @Param        offset  query  int     false  "Pagination offset"  default(0)
// @Param        status  query  string  false  "Filter by status (open, assigned, in_progress, resolved, closed)"
// @Success      200  {object}  listResponse
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /v1/requests [get]
func (h *Handler) ListRequests(w http.ResponseWriter, r *http.Request) {
	limit := int32(parseIntParam(r, "limit", 20))
	offset := int32(parseIntParam(r, "offset", 0))
	status := r.URL.Query().Get("status")

	var (
		reqs  []*requestsdb.ServiceRequest
		total int64
		err   error
	)

	if status != "" {
		reqs, err = h.queries.ListServiceRequestsByStatus(r.Context(), status, limit, offset)
		if err == nil {
			total, err = h.queries.CountServiceRequestsByStatus(r.Context(), status)
		}
	} else {
		reqs, err = h.queries.ListServiceRequests(r.Context(), limit, offset)
		if err == nil {
			total, err = h.queries.CountServiceRequests(r.Context())
		}
	}
	if err != nil {
		slog.Error("list service requests", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to list requests")
		return
	}

	out := make([]*ServiceRequestResponse, len(reqs))
	for i, sr := range reqs {
		out[i] = srToResponse(sr)
	}
	writeJSON(w, http.StatusOK, listResponse{
		Requests: out,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	})
}

// ─── Create request (public endpoint) ────────────────────────────────────────

type createServiceRequestRequest struct {
	RequestType    string          `json:"request_type"`
	DepartmentID   string          `json:"department_id"`
	Description    string          `json:"description"`
	Location       json.RawMessage `json:"location"`
	SubmitterEmail string          `json:"submitter_email"`
	Metadata       json.RawMessage `json:"metadata"`
}

// CreateRequest godoc
// @Summary      Submit a 311 service request (public)
// @Description  Public endpoint — no authentication required. Citizens use this to
// @Description  report issues such as potholes, broken streetlights, or code violations.
// @Tags         requests
// @Accept       json
// @Produce      json
// @Param        body  body  createServiceRequestRequest  true  "Service request"
// @Success      201  {object}  ServiceRequestResponse
// @Failure      400  {object}  map[string]string
// @Failure      422  {object}  map[string]string  "Metadata validation failed"
// @Failure      500  {object}  map[string]string
// @Router       /v1/requests [post]
func (h *Handler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	var req createServiceRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.RequestType == "" || req.DepartmentID == "" || req.Description == "" {
		writeError(w, r, http.StatusBadRequest, "request_type, department_id, and description are required")
		return
	}

	deptID, err := parseUUID(req.DepartmentID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid department_id")
		return
	}

	initialStatus, err := h.workflow.GetInitialStatus(r.Context(), "service_request")
	if err != nil {
		slog.Error("get initial workflow status", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to determine initial status")
		return
	}

	metadata := defaultJSON(req.Metadata, "{}")
	if string(metadata) != "{}" {
		if errs, err := h.schemas.ValidateMetadata(r.Context(), "service_request", metadata); err != nil {
			slog.Warn("schema validation error", "error", err)
		} else if len(errs) > 0 {
			writeValidationError(w, r, errs)
			return
		}
	}

	var submitterEmail *string
	if req.SubmitterEmail != "" {
		submitterEmail = &req.SubmitterEmail
	}

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		slog.Error("begin transaction", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to create request")
		return
	}
	defer tx.Rollback(r.Context())

	sr, err := h.queries.WithTx(tx).CreateServiceRequest(r.Context(), requestsdb.CreateServiceRequestParams{
		DepartmentID:   deptID,
		RequestType:    req.RequestType,
		Status:         initialStatus,
		Description:    req.Description,
		Location:       defaultJSON(req.Location, "{}"),
		SubmitterEmail: submitterEmail,
		Metadata:       metadata,
	})
	if err != nil {
		slog.Error("create service request", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to create request")
		return
	}

	resp := srToResponse(sr)
	if err := h.pub.PublishTx(r.Context(), tx, events.SubjectServiceRequestCreated, "requests", events.SubjectServiceRequestCreated, resp); err != nil {
		slog.Error("write service_request.created to outbox", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to create request")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		slog.Error("commit request create", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to create request")
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// ─── Get request ──────────────────────────────────────────────────────────────

// GetRequest godoc
// @Summary      Get a service request
// @Tags         requests
// @Produce      json
// @Param        id  path  string  true  "Request UUID"
// @Success      200  {object}  ServiceRequestResponse
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /v1/requests/{id} [get]
func (h *Handler) GetRequest(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request id")
		return
	}

	sr, err := h.queries.GetServiceRequest(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			writeError(w, r, http.StatusNotFound, "request not found")
			return
		}
		slog.Error("get service request", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to get request")
		return
	}

	writeJSON(w, http.StatusOK, srToResponse(sr))
}

// ─── Update status ────────────────────────────────────────────────────────────

type updateStatusRequest struct {
	Status string `json:"status"`
	Role   string `json:"role"`
}

// UpdateRequestStatus godoc
// @Summary      Update service request status
// @Description  Validates the transition via the workflow service. Terminal statuses
// @Description  (resolved, closed) also set closed_at.
// @Tags         requests
// @Accept       json
// @Produce      json
// @Param        id    path  string              true  "Request UUID"
// @Param        body  body  updateStatusRequest  true  "Status transition"
// @Success      200  {object}  ServiceRequestResponse
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      422  {object}  map[string]string  "Transition not allowed by workflow"
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /v1/requests/{id}/status [patch]
func (h *Handler) UpdateRequestStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request id")
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

	sr, err := h.queries.GetServiceRequest(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			writeError(w, r, http.StatusNotFound, "request not found")
			return
		}
		slog.Error("get service request for status update", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to get request")
		return
	}

	if sr.Status == req.Status {
		writeError(w, r, http.StatusBadRequest, "request is already in the requested status")
		return
	}

	if err := h.workflow.ValidateTransition(r.Context(), "service_request", sr.Status, req.Status, req.Role); err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err.Error())
		return
	}

	oldStatus := sr.Status

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		slog.Error("begin transaction", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to update status")
		return
	}
	defer tx.Rollback(r.Context())

	var updated *requestsdb.ServiceRequest
	if terminalStatuses[req.Status] {
		updated, err = h.queries.WithTx(tx).CloseServiceRequest(r.Context(), id, req.Status)
	} else {
		updated, err = h.queries.WithTx(tx).UpdateServiceRequestStatus(r.Context(), requestsdb.UpdateServiceRequestStatusParams{
			ID:     id,
			Status: req.Status,
		})
	}
	if err != nil {
		slog.Error("update service request status", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to update status")
		return
	}

	resp := srToResponse(updated)
	payload := map[string]any{"request": resp, "from": oldStatus, "to": req.Status}
	if err := h.pub.PublishTx(r.Context(), tx, events.SubjectServiceRequestClosed, "requests", events.SubjectServiceRequestClosed, payload); err != nil {
		slog.Error("write service_request.status_changed to outbox", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to update status")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		slog.Error("commit request status update", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to update status")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// ─── Assign request ───────────────────────────────────────────────────────────

type assignRequest struct {
	AssignedTo string `json:"assigned_to"`
	Role       string `json:"role"`
}

// AssignRequest godoc
// @Summary      Assign a request to a staff member
// @Description  Validates the open→assigned workflow transition. Updates assigned_to
// @Description  and status atomically. Publishes service_request.assigned.
// @Tags         requests
// @Accept       json
// @Produce      json
// @Param        id    path  string        true  "Request UUID"
// @Param        body  body  assignRequest  true  "Assignment"
// @Success      200  {object}  ServiceRequestResponse
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      422  {object}  map[string]string  "Transition not allowed by workflow"
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /v1/requests/{id}/assign [patch]
func (h *Handler) AssignRequest(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request id")
		return
	}

	var req assignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.AssignedTo == "" || req.Role == "" {
		writeError(w, r, http.StatusBadRequest, "assigned_to and role are required")
		return
	}

	assigneeID, err := parseUUID(req.AssignedTo)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid assigned_to uuid")
		return
	}

	sr, err := h.queries.GetServiceRequest(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			writeError(w, r, http.StatusNotFound, "request not found")
			return
		}
		slog.Error("get service request for assignment", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to get request")
		return
	}

	// Validate open→assigned transition via workflow service
	if err := h.workflow.ValidateTransition(r.Context(), "service_request", sr.Status, "assigned", req.Role); err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err.Error())
		return
	}

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		slog.Error("begin transaction", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to assign request")
		return
	}
	defer tx.Rollback(r.Context())

	updated, err := h.queries.WithTx(tx).AssignServiceRequest(r.Context(), requestsdb.AssignServiceRequestParams{
		ID:         id,
		AssignedTo: assigneeID,
		Status:     "assigned",
	})
	if err != nil {
		slog.Error("assign service request", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to assign request")
		return
	}

	resp := srToResponse(updated)
	payload := map[string]any{"request": resp, "assigned_to": req.AssignedTo}
	if err := h.pub.PublishTx(r.Context(), tx, events.SubjectServiceRequestAssigned, "requests", events.SubjectServiceRequestAssigned, payload); err != nil {
		slog.Error("write service_request.assigned to outbox", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to assign request")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		slog.Error("commit request assignment", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to assign request")
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
