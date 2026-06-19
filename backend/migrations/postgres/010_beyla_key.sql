-- ============================================================
-- Dedicated ingest key for the zero-code app-metrics agent (Beyla),
-- installed per managed server. Tracked so it can be revoked on
-- agent removal / server deletion.
-- ============================================================

ALTER TABLE managed_servers ADD COLUMN IF NOT EXISTS beyla_key_id UUID;
