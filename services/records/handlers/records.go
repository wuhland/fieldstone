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

// ListFOIARequests returns a list of FOIA requests.
func ListFOIARequests(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

// CreateFOIARequest creates a new FOIA request.
func CreateFOIARequest(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

// GetFOIARequest returns a single FOIA request by ID.
func GetFOIARequest(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

// UpdateFOIAStatus updates the status of a FOIA request.
func UpdateFOIAStatus(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}
