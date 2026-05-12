CREATE TABLE IF NOT EXISTS requests.service_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    department_id   UUID NOT NULL,
    request_type    TEXT NOT NULL,
    status          TEXT NOT NULL,
    description     TEXT NOT NULL,
    location        JSONB NOT NULL DEFAULT '{}',
    submitter_email TEXT,
    assigned_to     UUID,
    metadata        JSONB NOT NULL DEFAULT '{}',
    closed_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
