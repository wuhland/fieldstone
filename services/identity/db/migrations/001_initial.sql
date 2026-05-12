CREATE TABLE IF NOT EXISTS identity.departments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    config      JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS identity.staff_users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    department_id UUID NOT NULL REFERENCES identity.departments(id),
    oidc_sub      TEXT NOT NULL UNIQUE,
    email         TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('admin', 'reviewer', 'staff')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS identity.field_schemas (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_type TEXT NOT NULL UNIQUE,
    schema        JSONB NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
