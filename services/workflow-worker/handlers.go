package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/fieldstone/fieldstone/internal/middleware"
	"github.com/fieldstone/fieldstone/internal/workflows"
)

type workflowHandler struct {
	configs map[string]*workflows.WorkflowConfig
}

func newWorkflowHandler(configs map[string]*workflows.WorkflowConfig) *workflowHandler {
	return &workflowHandler{configs: configs}
}

func (h *workflowHandler) GetStatuses(w http.ResponseWriter, r *http.Request) {
	cfg, ok := h.configs[chi.URLParam(r, "resource_type")]
	if !ok {
		writeError(w, r, http.StatusNotFound, "unknown resource type")
		return
	}
	writeJSON(w, http.StatusOK, cfg.Statuses)
}

func (h *workflowHandler) GetTransitions(w http.ResponseWriter, r *http.Request) {
	cfg, ok := h.configs[chi.URLParam(r, "resource_type")]
	if !ok {
		writeError(w, r, http.StatusNotFound, "unknown resource type")
		return
	}
	writeJSON(w, http.StatusOK, cfg.Transitions)
}

func (h *workflowHandler) ValidateTransition(w http.ResponseWriter, r *http.Request) {
	cfg, ok := h.configs[chi.URLParam(r, "resource_type")]
	if !ok {
		writeError(w, r, http.StatusNotFound, "unknown resource type")
		return
	}
	var req workflows.TransitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := cfg.ValidateTransition(req.From, req.To, req.Role); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error":      err.Error(),
			"request_id": middleware.GetRequestID(r.Context()),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "valid"})
}

func (h *workflowHandler) GetInitialStatus(w http.ResponseWriter, r *http.Request) {
	cfg, ok := h.configs[chi.URLParam(r, "resource_type")]
	if !ok {
		writeError(w, r, http.StatusNotFound, "unknown resource type")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"initial_status": cfg.InitialStatus})
}

// GetConfig returns the full WorkflowConfig for a resource type. Domain service
// clients fetch this once when starting a new Temporal workflow execution so the
// config is baked durably into the workflow's input history.
func (h *workflowHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, ok := h.configs[chi.URLParam(r, "resource_type")]
	if !ok {
		writeError(w, r, http.StatusNotFound, "unknown resource type")
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	writeJSON(w, status, map[string]string{
		"error":      msg,
		"request_id": middleware.GetRequestID(r.Context()),
	})
}
