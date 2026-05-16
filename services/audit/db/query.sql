-- name: InsertEvent :one
INSERT INTO events (id, occurred_at, source_service, event_type, payload, actor)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id, occurred_at) DO NOTHING
RETURNING id, occurred_at, source_service, event_type, payload, actor, indexed_at;

-- name: GetEvent :one
SELECT id, occurred_at, source_service, event_type, payload, actor, indexed_at
FROM events
WHERE id = $1
LIMIT 1;

-- ListEvents and CountEvents are generated dynamically in Go (see query.sql.go)
-- because they support optional filter parameters.
