-- ============================================================
-- Ingest key metadata for the onboarding UI.
-- `prefix` is a non-secret, displayable head of the key (the full key is only
-- shown once at creation). `last_used_at` powers a "last seen" indicator.
-- ============================================================

ALTER TABLE ingest_keys ADD COLUMN IF NOT EXISTS prefix       TEXT NOT NULL DEFAULT '';
ALTER TABLE ingest_keys ADD COLUMN IF NOT EXISTS last_used_at TIMESTAMPTZ;

-- Backfill a display prefix for the seeded demo key.
UPDATE ingest_keys SET prefix = 'pulse_demo_i' WHERE name = 'demo' AND prefix = '';
