CREATE TABLE IF NOT EXISTS permits.permits (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    department_id    UUID NOT NULL,
    permit_type      TEXT NOT NULL,
    status           TEXT NOT NULL,
    applicant        JSONB NOT NULL,
    property_address TEXT NOT NULL,
    metadata         JSONB NOT NULL DEFAULT '{}',
    submitted_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    issued_at        TIMESTAMPTZ,
    expires_at       TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS permits.inspections (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    permit_id     UUID NOT NULL REFERENCES permits.permits(id),
    inspector_id  UUID NOT NULL,
    scheduled_at  TIMESTAMPTZ NOT NULL,
    completed_at  TIMESTAMPTZ,
    result        TEXT,
    notes         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
