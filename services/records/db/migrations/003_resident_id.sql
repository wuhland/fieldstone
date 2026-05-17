ALTER TABLE records.foia_requests
    ADD COLUMN IF NOT EXISTS resident_id TEXT;
