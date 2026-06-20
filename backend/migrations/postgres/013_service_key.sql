-- ============================================================
-- Attribute auto-created services to the ingest key that first produced them,
-- so removing an agent / Beyla (which revokes its key) can also clean up the
-- services and metrics it created. NULL = created manually or pre-dating this.
-- ============================================================

ALTER TABLE services ADD COLUMN IF NOT EXISTS ingest_key_id UUID;
