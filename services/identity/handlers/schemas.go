package handlers

import (
	"net/http"
)

type SchemaHandler struct{}

func NewSchemaHandler() *SchemaHandler { return &SchemaHandler{} }

func (h *SchemaHandler) GetSchema(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

func (h *SchemaHandler) PutSchema(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}
