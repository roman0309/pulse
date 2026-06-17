-- ============================================================
-- Managed servers — remote hosts Pulse can install/remove the agent on via
-- Tailscale SSH. No credentials are stored: access is by tailnet identity.
-- ============================================================

CREATE TABLE IF NOT EXISTS managed_servers (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,                 -- service name the agent reports as
    ssh_target  TEXT NOT NULL,                 -- user@host (tailnet name), no secrets
    status      TEXT NOT NULL DEFAULT 'pending', -- pending | installed | removed | error
    last_result TEXT NOT NULL DEFAULT '',      -- last action output / error (truncated)
    ingest_key_id UUID REFERENCES ingest_keys(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_managed_servers_project ON managed_servers(project_id);

DROP TRIGGER IF EXISTS trg_managed_servers_updated ON managed_servers;
CREATE TRIGGER trg_managed_servers_updated BEFORE UPDATE ON managed_servers
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
