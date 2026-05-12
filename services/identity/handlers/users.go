package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/fieldstone/fieldstone/internal/middleware"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler { return &UserHandler{} }

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

func stub(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
		"error":      "not implemented",
		"request_id": middleware.GetRequestID(r.Context()),
	})
}
