package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	identitydb "github.com/fieldstone/fieldstone/services/identity/db/generated"
)

// ListDepartments godoc
// @Summary      List departments
// @Tags         identity
// @Produce      json
// @Success      200  {object}  map[string][]DepartmentResponse
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /v1/departments [get]
func (h *Handler) ListDepartments(w http.ResponseWriter, r *http.Request) {
	depts, err := h.queries.ListDepartments(r.Context())
	if err != nil {
		slog.Error("list departments", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to list departments")
		return
	}

	out := make([]*DepartmentResponse, len(depts))
	for i, d := range depts {
		out[i] = deptToResponse(d)
	}
	writeJSON(w, http.StatusOK, map[string]any{"departments": out})
}

type createDepartmentInput struct {
	Name   string          `json:"name"`
	Slug   string          `json:"slug"`
	Config json.RawMessage `json:"config"`
}

// CreateDepartment godoc
// @Summary      Create a department
// @Tags         identity
// @Accept       json
// @Produce      json
// @Param        body  body  createDepartmentInput  true  "Department"
// @Success      201  {object}  DepartmentResponse
// @Failure      400  {object}  map[string]string
// @Failure      409  {object}  map[string]string  "Slug already exists"
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /v1/departments [post]
func (h *Handler) CreateDepartment(w http.ResponseWriter, r *http.Request) {
	var req createDepartmentInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" || req.Slug == "" {
		writeError(w, r, http.StatusBadRequest, "name and slug are required")
		return
	}
	if !isValidSlug(req.Slug) {
		writeError(w, r, http.StatusBadRequest, "slug must be lowercase letters, numbers, and hyphens only")
		return
	}

	dept, err := h.queries.CreateDepartment(r.Context(), identitydb.CreateDepartmentParams{
		Name:   req.Name,
		Slug:   req.Slug,
		Config: defaultJSON(req.Config, "{}"),
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, r, http.StatusConflict, "a department with that slug already exists")
			return
		}
		slog.Error("create department", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to create department")
		return
	}

	writeJSON(w, http.StatusCreated, deptToResponse(dept))
}

func isValidSlug(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return !strings.HasPrefix(s, "-") && !strings.HasSuffix(s, "-")
}

// isUniqueViolation detects PostgreSQL unique constraint errors (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}
