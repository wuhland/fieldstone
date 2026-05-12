CREATE TABLE IF NOT EXISTS audit.events (
    id             UUID PRIMARY KEY,
    occurred_at    TIMESTAMPTZ NOT NULL,
    source_service TEXT NOT NULL,
    event_type     TEXT NOT NULL,
    payload        JSONB NOT NULL,
    actor          JSONB,
    indexed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_events_occurred_at    ON audit.events(occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_event_type     ON audit.events(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_events_source_service ON audit.events(source_service);
