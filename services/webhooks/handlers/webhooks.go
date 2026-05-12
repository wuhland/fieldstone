package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/fieldstone/fieldstone/internal/middleware"
)

type WebhookHandler struct{}

func NewWebhookHandler() *WebhookHandler { return &WebhookHandler{} }

func (h *WebhookHandler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

func (h *WebhookHandler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

func (h *WebhookHandler) GetWebhook(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

func (h *WebhookHandler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

func (h *WebhookHandler) TestWebhook(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

func stub(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{
		"error":      "not implemented",
		"request_id": middleware.GetRequestID(r.Context()),
	})
}
