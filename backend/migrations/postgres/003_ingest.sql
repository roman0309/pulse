-- ============================================================
-- Ingest keys — per-project API keys for OTLP / remote_write ingestion.
-- The agent/collector sends the key via the `X-Pulse-Key` header (or
-- `Authorization: Bearer <key>`). Only the SHA-256 hash is stored.
-- ============================================================

CREATE TABLE IF NOT EXISTS ingest_keys (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        TEXT NOT NULL DEFAULT 'default',
    key_hash    TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ingest_keys_project ON ingest_keys(project_id);
-- (demo key is seeded separately in 006_seed_extras.sql, gated behind SEED_DEMO)
