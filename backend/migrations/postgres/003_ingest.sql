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

-- Demo key for the seeded project. Plaintext: pulse_demo_ingest_key
INSERT INTO ingest_keys (project_id, name, key_hash)
VALUES (
    '33333333-3333-3333-3333-333333333333',
    'demo',
    encode(digest('pulse_demo_ingest_key', 'sha256'), 'hex')
) ON CONFLICT (key_hash) DO NOTHING;
