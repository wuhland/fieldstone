-- ─── Departments ─────────────────────────────────────────────────────────────

-- name: ListDepartments :many
SELECT id, name, slug, config, created_at
FROM departments
ORDER BY name;

-- name: GetDepartmentByID :one
SELECT id, name, slug, config, created_at
FROM departments
WHERE id = $1;

-- name: GetDepartmentBySlug :one
SELECT id, name, slug, config, created_at
FROM departments
WHERE slug = $1;

-- name: CreateDepartment :one
INSERT INTO departments (name, slug, config)
VALUES ($1, $2, $3)
RETURNING id, name, slug, config, created_at;

-- ─── Staff users ──────────────────────────────────────────────────────────────

-- name: ListUsers :many
SELECT id, department_id, oidc_sub, email, role, created_at
FROM staff_users
ORDER BY email;

-- name: ListUsersByDepartment :many
SELECT id, department_id, oidc_sub, email, role, created_at
FROM staff_users
WHERE department_id = $1
ORDER BY email;

-- name: GetUserByID :one
SELECT id, department_id, oidc_sub, email, role, created_at
FROM staff_users
WHERE id = $1;

-- name: GetUserByOIDCSub :one
SELECT id, department_id, oidc_sub, email, role, created_at
FROM staff_users
WHERE oidc_sub = $1;

-- name: CreateUser :one
INSERT INTO staff_users (department_id, oidc_sub, email, role)
VALUES ($1, $2, $3, $4)
RETURNING id, department_id, oidc_sub, email, role, created_at;

-- ─── Field schemas ────────────────────────────────────────────────────────────

-- name: GetFieldSchema :one
SELECT id, resource_type, schema, created_at, updated_at
FROM field_schemas
WHERE resource_type = $1;

-- name: UpsertFieldSchema :one
INSERT INTO field_schemas (resource_type, schema)
VALUES ($1, $2)
ON CONFLICT (resource_type) DO UPDATE
SET schema = EXCLUDED.schema, updated_at = NOW()
RETURNING id, resource_type, schema, created_at, updated_at;

-- name: ListFieldSchemas :many
SELECT id, resource_type, schema, created_at, updated_at
FROM field_schemas
ORDER BY resource_type;
