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

// ListRequests returns a list of service requests.
func ListRequests(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

// CreateRequest creates a new service request.
func CreateRequest(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

// GetRequest returns a single service request by ID.
func GetRequest(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

// UpdateRequestStatus updates the status of a service request.
func UpdateRequestStatus(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

// AssignRequest assigns a service request to a staff member.
func AssignRequest(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}
