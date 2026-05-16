package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	identitydb "github.com/fieldstone/fieldstone/services/identity/db/generated"
)

// GetSchema godoc
// @Summary      Get the custom field schema for a resource type
// @Description  Returns the city-registered JSON Schema (draft-07) for the given resource
// @Description  type. Used by DynamicMetadataForm and by permits/requests/records to
// @Description  validate metadata on every write.
// @Tags         config
// @Produce      json
// @Param        resource_type  path  string  true  "Resource type (permit, service_request, foia_request)"
// @Success      200  {object}  FieldSchemaResponse
// @Failure      404  {object}  map[string]string  "No schema registered for this type"
// @Failure      500  {object}  map[string]string
// @Router       /v1/config/schemas/{resource_type} [get]
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

// PutSchema godoc
// @Summary      Register or replace a custom field schema
// @Description  The request body must be a valid JSON Schema document (draft-07). This is
// @Description  an idempotent upsert — calling it again replaces the previous schema.
// @Description  Changes take effect immediately; the 60-second cache in domain services
// @Description  means active requests may use the old schema for up to one minute.
// @Tags         config
// @Accept       json
// @Produce      json
// @Param        resource_type  path  string          true  "Resource type"
// @Param        body           body  object           true  "JSON Schema document"
// @Success      200  {object}  FieldSchemaResponse
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /v1/config/schemas/{resource_type} [put]
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
