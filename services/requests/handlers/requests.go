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

	sr, err := h.queries.CreateServiceRequest(r.Context(), requestsdb.CreateServiceRequestParams{
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
	h.pub.Publish(events.SubjectServiceRequestCreated, "requests", events.SubjectServiceRequestCreated, resp)

	writeJSON(w, http.StatusCreated, resp)
}

// ─── Get request ──────────────────────────────────────────────────────────────

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

	var updated *requestsdb.ServiceRequest
	if terminalStatuses[req.Status] {
		updated, err = h.queries.CloseServiceRequest(r.Context(), id, req.Status)
	} else {
		updated, err = h.queries.UpdateServiceRequestStatus(r.Context(), requestsdb.UpdateServiceRequestStatusParams{
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
	h.pub.Publish(
		events.SubjectServiceRequestClosed,
		"requests",
		events.SubjectServiceRequestClosed,
		map[string]any{"request": resp, "from": oldStatus, "to": req.Status},
	)

	writeJSON(w, http.StatusOK, resp)
}

// ─── Assign request ───────────────────────────────────────────────────────────

type assignRequest struct {
	AssignedTo string `json:"assigned_to"`
	Role       string `json:"role"`
}

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

	updated, err := h.queries.AssignServiceRequest(r.Context(), requestsdb.AssignServiceRequestParams{
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
	h.pub.Publish(
		events.SubjectServiceRequestAssigned,
		"requests",
		events.SubjectServiceRequestAssigned,
		map[string]any{"request": resp, "assigned_to": req.AssignedTo},
	)

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
