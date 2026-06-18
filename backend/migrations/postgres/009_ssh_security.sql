-- ============================================================
-- SSH security: pinned host key (TOFU) + audit log of remote actions.
-- ============================================================

ALTER TABLE managed_servers ADD COLUMN IF NOT EXISTS host_key TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS audit_log (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    server_id   UUID,
    action      TEXT NOT NULL,         -- add_server | install | remove | status | run | delete_server
    detail      TEXT NOT NULL DEFAULT '', -- e.g. the command
    success     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_audit_project ON audit_log(project_id, created_at DESC);
