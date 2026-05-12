package handlers

import (
	"net/http"
)

type DepartmentHandler struct{}

func NewDepartmentHandler() *DepartmentHandler { return &DepartmentHandler{} }

func (h *DepartmentHandler) ListDepartments(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}

func (h *DepartmentHandler) CreateDepartment(w http.ResponseWriter, r *http.Request) {
	stub(w, r)
}
