package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/fieldstone/fieldstone/internal/middleware"
)

func stub(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{
		"error":      "not implemented",
		"request_id": middleware.GetRequestID(r.Context()),
	})
}

// ListEvents returns a paginated list of audit events.
func ListEvents(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

// GetEvent returns a single audit event by ID.
func GetEvent(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}
