package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	identitydb "github.com/fieldstone/fieldstone/services/identity/db/generated"
)

func (h *Handler) GetSchema(w http.ResponseWriter, r *http.Request) {
	resourceType := chi.URLParam(r, "resource_type")

	schema, err := h.queries.GetFieldSchema(r.Context(), resourceType)
	if err != nil {
		if isNotFound(err) {
			writeError(w, r, http.StatusNotFound, "no schema registered for resource type: "+resourceType)
			return
		}
		slog.Error("get field schema", "resource_type", resourceType, "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to get schema")
		return
	}

	writeJSON(w, http.StatusOK, schemaToResponse(schema))
}

func (h *Handler) PutSchema(w http.ResponseWriter, r *http.Request) {
	resourceType := chi.URLParam(r, "resource_type")

	// The request body IS the JSON Schema document — not a wrapper object.
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "failed to read request body")
		return
	}
	if len(body) == 0 {
		writeError(w, r, http.StatusBadRequest, "schema body is required")
		return
	}
	if !json.Valid(body) {
		writeError(w, r, http.StatusBadRequest, "request body must be a valid JSON document")
		return
	}

	schema, err := h.queries.UpsertFieldSchema(r.Context(), identitydb.UpsertFieldSchemaParams{
		ResourceType: resourceType,
		Schema:       json.RawMessage(body),
	})
	if err != nil {
		slog.Error("upsert field schema", "resource_type", resourceType, "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to save schema")
		return
	}

	writeJSON(w, http.StatusOK, schemaToResponse(schema))
}
