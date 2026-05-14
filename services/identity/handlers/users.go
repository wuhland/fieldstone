package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/fieldstone/fieldstone/internal/auth"
	"github.com/fieldstone/fieldstone/internal/events"
	identitydb "github.com/fieldstone/fieldstone/services/identity/db/generated"
)

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	var (
		users []*identitydb.StaffUser
		err   error
	)

	if deptIDStr := r.URL.Query().Get("department_id"); deptIDStr != "" {
		deptID, parseErr := parseUUID(deptIDStr)
		if parseErr != nil {
			writeError(w, r, http.StatusBadRequest, "invalid department_id")
			return
		}
		users, err = h.queries.ListUsersByDepartment(r.Context(), deptID)
	} else {
		users, err = h.queries.ListUsers(r.Context())
	}
	if err != nil {
		slog.Error("list users", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to list users")
		return
	}

	out := make([]*UserResponse, len(users))
	for i, u := range users {
		out[i] = userToResponse(u)
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": out})
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		// DEV_DISABLE_AUTH mode — return a synthetic dev user so the frontend works.
		writeJSON(w, http.StatusOK, map[string]any{
			"id":            "00000000-0000-0000-0000-000000000000",
			"department_id": "00000000-0000-0000-0000-000000000000",
			"oidc_sub":      "dev",
			"email":         "dev@fieldstone.local",
			"role":          "admin",
			"dev_mode":      true,
		})
		return
	}

	user, err := h.queries.GetUserByOIDCSub(r.Context(), claims.Subject)
	if err != nil {
		if isNotFound(err) {
			writeError(w, r, http.StatusNotFound, "user not provisioned — ask an admin to create your account")
			return
		}
		slog.Error("get user by oidc sub", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to look up user")
		return
	}

	writeJSON(w, http.StatusOK, userToResponse(user))
}

type createUserInput struct {
	DepartmentID string `json:"department_id"`
	OIDCSub      string `json:"oidc_sub"`
	Email        string `json:"email"`
	Role         string `json:"role"`
}

var validRoles = map[string]bool{"admin": true, "reviewer": true, "staff": true}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.DepartmentID == "" || req.OIDCSub == "" || req.Email == "" || req.Role == "" {
		writeError(w, r, http.StatusBadRequest, "department_id, oidc_sub, email, and role are required")
		return
	}
	if !validRoles[req.Role] {
		writeError(w, r, http.StatusBadRequest, "role must be one of: admin, reviewer, staff")
		return
	}

	deptID, err := parseUUID(req.DepartmentID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid department_id")
		return
	}

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		slog.Error("begin transaction", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to create user")
		return
	}
	defer tx.Rollback(r.Context())

	user, err := h.queries.WithTx(tx).CreateUser(r.Context(), identitydb.CreateUserParams{
		DepartmentID: deptID,
		OIDCSub:      req.OIDCSub,
		Email:        req.Email,
		Role:         req.Role,
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, r, http.StatusConflict, "a user with that oidc_sub already exists")
			return
		}
		slog.Error("create user", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to create user")
		return
	}

	resp := userToResponse(user)
	if err := h.pub.PublishTx(r.Context(), tx, events.SubjectUserProvisioned, "identity", events.SubjectUserProvisioned, resp); err != nil {
		slog.Error("write user.provisioned to outbox", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to create user")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		slog.Error("commit user create", "error", err)
		writeError(w, r, http.StatusInternalServerError, "failed to create user")
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}
