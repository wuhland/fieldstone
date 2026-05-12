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

// ListPermits returns a list of permits.
func ListPermits(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

// CreatePermit creates a new permit.
func CreatePermit(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

// GetPermit returns a single permit by ID.
func GetPermit(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

// UpdatePermitStatus updates the status of a permit.
func UpdatePermitStatus(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

// ScheduleInspection schedules an inspection for a permit.
func ScheduleInspection(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

// UpdateInspection updates an existing inspection.
func UpdateInspection(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}
