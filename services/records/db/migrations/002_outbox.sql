CREATE TABLE IF NOT EXISTS outbox (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject    TEXT NOT NULL,
    payload    JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Partial covering index optimised for the polling SELECT ordered by created_at.
CREATE INDEX IF NOT EXISTS idx_outbox_created ON outbox (created_at);
