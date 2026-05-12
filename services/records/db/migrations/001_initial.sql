CREATE TABLE IF NOT EXISTS records.foia_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    department_id   UUID NOT NULL,
    status          TEXT NOT NULL,
    requester_name  TEXT NOT NULL,
    requester_email TEXT NOT NULL,
    description     TEXT NOT NULL,
    due_date        DATE,
    metadata        JSONB NOT NULL DEFAULT '{}',
    closed_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
