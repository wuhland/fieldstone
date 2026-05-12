CREATE TABLE IF NOT EXISTS webhooks.endpoints (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url         TEXT NOT NULL,
    secret_hash TEXT NOT NULL,
    events      TEXT[] NOT NULL,
    description TEXT,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    fail_count  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS webhooks.deliveries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id  UUID NOT NULL REFERENCES webhooks.endpoints(id),
    event_id     TEXT NOT NULL,
    event_type   TEXT NOT NULL,
    status_code  INT,
    duration_ms  INT,
    error        TEXT,
    delivered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_deliveries_endpoint_id ON webhooks.deliveries(endpoint_id, delivered_at DESC);
