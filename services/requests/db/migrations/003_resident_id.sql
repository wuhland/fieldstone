ALTER TABLE requests.service_requests
    ADD COLUMN IF NOT EXISTS resident_id TEXT;
