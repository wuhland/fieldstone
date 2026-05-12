-- Create per-service schemas. Each service's DSN uses search_path=<schema>,public
-- so services are isolated to their own schema.
CREATE SCHEMA IF NOT EXISTS identity;
CREATE SCHEMA IF NOT EXISTS permits;
CREATE SCHEMA IF NOT EXISTS requests;
CREATE SCHEMA IF NOT EXISTS records;
CREATE SCHEMA IF NOT EXISTS audit;
CREATE SCHEMA IF NOT EXISTS webhooks;
CREATE SCHEMA IF NOT EXISTS workflow;
