-- name: GetServiceRequest :one
SELECT id, department_id, request_type, status, description, location,
       submitter_email, assigned_to, metadata, closed_at, created_at, updated_at
FROM service_requests
WHERE id = $1;

-- name: ListServiceRequests :many
SELECT id, department_id, request_type, status, description, location,
       submitter_email, assigned_to, metadata, closed_at, created_at, updated_at
FROM service_requests
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListServiceRequestsByStatus :many
SELECT id, department_id, request_type, status, description, location,
       submitter_email, assigned_to, metadata, closed_at, created_at, updated_at
FROM service_requests
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountServiceRequests :one
SELECT COUNT(*) FROM service_requests;

-- name: CountServiceRequestsByStatus :one
SELECT COUNT(*) FROM service_requests WHERE status = $1;

-- name: CreateServiceRequest :one
INSERT INTO service_requests
    (department_id, request_type, status, description, location, submitter_email, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, department_id, request_type, status, description, location,
          submitter_email, assigned_to, metadata, closed_at, created_at, updated_at;

-- name: UpdateServiceRequestStatus :one
UPDATE service_requests
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, department_id, request_type, status, description, location,
          submitter_email, assigned_to, metadata, closed_at, created_at, updated_at;

-- name: CloseServiceRequest :one
UPDATE service_requests
SET status = $2, closed_at = NOW(), updated_at = NOW()
WHERE id = $1
RETURNING id, department_id, request_type, status, description, location,
          submitter_email, assigned_to, metadata, closed_at, created_at, updated_at;

-- name: AssignServiceRequest :one
UPDATE service_requests
SET assigned_to = $2, status = $3, updated_at = NOW()
WHERE id = $1
RETURNING id, department_id, request_type, status, description, location,
          submitter_email, assigned_to, metadata, closed_at, created_at, updated_at;
