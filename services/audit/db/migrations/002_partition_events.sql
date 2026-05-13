-- Convert audit.events to a range-partitioned table by occurred_at.
--
-- Partitioning is applied idempotently: if the table is already partitioned
-- this block is a no-op. Otherwise it renames the old table, creates the
-- partitioned version, migrates existing rows, and drops the old table.
--
-- Partitions cover 2025–2030 (monthly). Add future partitions before the
-- preceding year ends; automate with pg_partman in production.

DO $$
DECLARE
    yr   int;
    mo   int;
    name text;
    s    date;
    e    date;
BEGIN
    -- Skip if already partitioned
    IF EXISTS (
        SELECT 1 FROM pg_partitioned_table pt
        JOIN pg_class c ON c.oid = pt.partrelid
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = 'audit' AND c.relname = 'events'
    ) THEN
        RETURN;
    END IF;

    -- Preserve any existing rows
    ALTER TABLE audit.events RENAME TO events_v1;

    CREATE TABLE audit.events (
        id             UUID NOT NULL,
        occurred_at    TIMESTAMPTZ NOT NULL,
        source_service TEXT NOT NULL,
        event_type     TEXT NOT NULL,
        payload        JSONB NOT NULL,
        actor          JSONB,
        indexed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        PRIMARY KEY    (id, occurred_at)
    ) PARTITION BY RANGE (occurred_at);

    -- Indexes on the parent are inherited by all child partitions
    CREATE INDEX idx_audit_events_occurred_at    ON audit.events (occurred_at DESC);
    CREATE INDEX idx_audit_events_event_type     ON audit.events (event_type);
    CREATE INDEX idx_audit_events_source_service ON audit.events (source_service);

    -- Create monthly partitions for 2025–2030
    FOR yr IN 2025..2030 LOOP
        FOR mo IN 1..12 LOOP
            name := format('events_%s_%s', yr, lpad(mo::text, 2, '0'));
            s    := make_date(yr, mo, 1);
            e    := s + interval '1 month';
            EXECUTE format(
                'CREATE TABLE IF NOT EXISTS audit.%I PARTITION OF audit.events FOR VALUES FROM (%L) TO (%L)',
                name, s, e
            );
        END LOOP;
    END LOOP;

    -- Migrate existing rows (typically zero in a fresh install)
    INSERT INTO audit.events
        SELECT id, occurred_at, source_service, event_type, payload, actor, indexed_at
        FROM audit.events_v1;

    DROP TABLE audit.events_v1;
END $$;
