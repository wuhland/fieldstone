-- name: ListEndpoints :many
SELECT id, url, secret_hash, events, description, enabled, fail_count, created_at
FROM endpoints
ORDER BY created_at DESC;

-- name: ListEnabledEndpoints :many
SELECT id, url, secret_hash, events, description, enabled, fail_count, created_at
FROM endpoints
WHERE enabled = TRUE
ORDER BY created_at;

-- name: GetEndpointByID :one
SELECT id, url, secret_hash, events, description, enabled, fail_count, created_at
FROM endpoints
WHERE id = $1;

-- name: CreateEndpoint :one
INSERT INTO endpoints (url, secret_hash, events, description)
VALUES ($1, $2, $3, $4)
RETURNING id, url, secret_hash, events, description, enabled, fail_count, created_at;

-- name: DeleteEndpoint :exec
DELETE FROM endpoints WHERE id = $1;

-- name: DeleteEndpointDeliveries :exec
DELETE FROM deliveries WHERE endpoint_id = $1;

-- name: IncrementFailCount :one
UPDATE endpoints
SET fail_count = fail_count + 1
WHERE id = $1
RETURNING fail_count;

-- name: ResetFailCount :exec
UPDATE endpoints SET fail_count = 0 WHERE id = $1;

-- name: DisableEndpoint :exec
UPDATE endpoints SET enabled = FALSE WHERE id = $1;

-- name: InsertDelivery :one
INSERT INTO deliveries (endpoint_id, event_id, event_type, status_code, duration_ms, error)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, endpoint_id, event_id, event_type, status_code, duration_ms, error, delivered_at;

-- name: ListDeliveriesByEndpoint :many
SELECT id, endpoint_id, event_id, event_type, status_code, duration_ms, error, delivered_at
FROM deliveries
WHERE endpoint_id = $1
ORDER BY delivered_at DESC
LIMIT $2;
