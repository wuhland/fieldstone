-- name: GetPermit :one
SELECT id, department_id, permit_type, status, applicant, property_address, metadata,
       submitted_at, issued_at, expires_at, created_at, updated_at
FROM permits
WHERE id = $1;

-- name: ListPermits :many
SELECT id, department_id, permit_type, status, applicant, property_address, metadata,
       submitted_at, issued_at, expires_at, created_at, updated_at
FROM permits
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListPermitsByStatus :many
SELECT id, department_id, permit_type, status, applicant, property_address, metadata,
       submitted_at, issued_at, expires_at, created_at, updated_at
FROM permits
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountPermits :one
SELECT COUNT(*) FROM permits;

-- name: CountPermitsByStatus :one
SELECT COUNT(*) FROM permits WHERE status = $1;

-- name: CreatePermit :one
INSERT INTO permits (department_id, permit_type, status, applicant, property_address, metadata)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, department_id, permit_type, status, applicant, property_address, metadata,
          submitted_at, issued_at, expires_at, created_at, updated_at;

-- name: UpdatePermitStatus :one
UPDATE permits
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, department_id, permit_type, status, applicant, property_address, metadata,
          submitted_at, issued_at, expires_at, created_at, updated_at;

-- name: SetPermitIssuedAt :one
UPDATE permits
SET issued_at = NOW(), status = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, department_id, permit_type, status, applicant, property_address, metadata,
          submitted_at, issued_at, expires_at, created_at, updated_at;

-- name: GetInspection :one
SELECT id, permit_id, inspector_id, scheduled_at, completed_at, result, notes, created_at
FROM inspections
WHERE id = $1;

-- name: ListInspectionsByPermit :many
SELECT id, permit_id, inspector_id, scheduled_at, completed_at, result, notes, created_at
FROM inspections
WHERE permit_id = $1
ORDER BY scheduled_at;

-- name: CreateInspection :one
INSERT INTO inspections (permit_id, inspector_id, scheduled_at)
VALUES ($1, $2, $3)
RETURNING id, permit_id, inspector_id, scheduled_at, completed_at, result, notes, created_at;

-- name: UpdateInspection :one
UPDATE inspections
SET completed_at = $2, result = $3, notes = $4
WHERE id = $1
RETURNING id, permit_id, inspector_id, scheduled_at, completed_at, result, notes, created_at;
