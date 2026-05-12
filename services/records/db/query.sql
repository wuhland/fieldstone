-- name: GetFOIARequest :one
SELECT id, department_id, status, requester_name, requester_email, description,
       due_date, metadata, closed_at, created_at, updated_at
FROM foia_requests
WHERE id = $1;

-- name: ListFOIARequests :many
SELECT id, department_id, status, requester_name, requester_email, description,
       due_date, metadata, closed_at, created_at, updated_at
FROM foia_requests
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListFOIARequestsByStatus :many
SELECT id, department_id, status, requester_name, requester_email, description,
       due_date, metadata, closed_at, created_at, updated_at
FROM foia_requests
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountFOIARequests :one
SELECT COUNT(*) FROM foia_requests;

-- name: CountFOIARequestsByStatus :one
SELECT COUNT(*) FROM foia_requests WHERE status = $1;

-- name: CreateFOIARequest :one
INSERT INTO foia_requests
    (department_id, status, requester_name, requester_email, description, due_date, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, department_id, status, requester_name, requester_email, description,
          due_date, metadata, closed_at, created_at, updated_at;

-- name: UpdateFOIAStatus :one
UPDATE foia_requests
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, department_id, status, requester_name, requester_email, description,
          due_date, metadata, closed_at, created_at, updated_at;

-- name: CloseFOIARequest :one
UPDATE foia_requests
SET status = $2, closed_at = NOW(), updated_at = NOW()
WHERE id = $1
RETURNING id, department_id, status, requester_name, requester_email, description,
          due_date, metadata, closed_at, created_at, updated_at;
