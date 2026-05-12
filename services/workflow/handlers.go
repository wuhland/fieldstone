package main

import (
	"encoding/json"
	"net/http"

	"github.com/fieldstone/fieldstone/internal/middleware"
	"github.com/go-chi/chi/v5"
)

type WorkflowHandler struct {
	engine *Engine
}

func NewWorkflowHandler(e *Engine) *WorkflowHandler {
	return &WorkflowHandler{engine: e}
}

func (h *WorkflowHandler) GetStatuses(w http.ResponseWriter, r *http.Request) {
	resourceType := chi.URLParam(r, "resource_type")
	statuses, err := h.engine.Statuses(resourceType)
	if err != nil {
		writeNotFound(w, r, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, statuses)
}

func (h *WorkflowHandler) GetTransitions(w http.ResponseWriter, r *http.Request) {
	resourceType := chi.URLParam(r, "resource_type")
	transitions, err := h.engine.Transitions(resourceType)
	if err != nil {
		writeNotFound(w, r, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, transitions)
}

func (h *WorkflowHandler) ValidateTransition(w http.ResponseWriter, r *http.Request) {
	resourceType := chi.URLParam(r, "resource_type")
	var req ValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, r, "invalid JSON")
		return
	}
	if ve := h.engine.Validate(resourceType, req); ve != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error":      ve.Message,
			"request_id": middleware.GetRequestID(r.Context()),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "valid"})
}

func (h *WorkflowHandler) GetInitialStatus(w http.ResponseWriter, r *http.Request) {
	resourceType := chi.URLParam(r, "resource_type")
	status, err := h.engine.InitialStatus(resourceType)
	if err != nil {
		writeNotFound(w, r, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"initial_status": status})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeNotFound(w http.ResponseWriter, r *http.Request, msg string) {
	writeJSON(w, http.StatusNotFound, map[string]string{
		"error":      msg,
		"request_id": middleware.GetRequestID(r.Context()),
	})
}

func writeBadRequest(w http.ResponseWriter, r *http.Request, msg string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{
		"error":      msg,
		"request_id": middleware.GetRequestID(r.Context()),
	})
}
