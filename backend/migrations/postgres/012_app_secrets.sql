-- ============================================================
-- App-managed secrets: when JWT_SECRET / CREDENTIALS_KEY are not provided via
-- the environment, the server generates a strong random value once and persists
-- it here so it survives restarts (instead of a shipped default constant).
-- ============================================================

CREATE TABLE IF NOT EXISTS app_secrets (
    name        TEXT PRIMARY KEY,
    value       TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
