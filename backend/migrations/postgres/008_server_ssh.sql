-- ============================================================
-- SSH connection details for managed servers. The credential (password or
-- private key) is stored ENCRYPTED (AES-GCM) in secret_enc — never in plaintext.
-- ============================================================

ALTER TABLE managed_servers ADD COLUMN IF NOT EXISTS ssh_host    TEXT NOT NULL DEFAULT '';
ALTER TABLE managed_servers ADD COLUMN IF NOT EXISTS ssh_port    INT  NOT NULL DEFAULT 22;
ALTER TABLE managed_servers ADD COLUMN IF NOT EXISTS ssh_user    TEXT NOT NULL DEFAULT 'root';
ALTER TABLE managed_servers ADD COLUMN IF NOT EXISTS auth_method TEXT NOT NULL DEFAULT 'password'; -- password | key
ALTER TABLE managed_servers ADD COLUMN IF NOT EXISTS secret_enc  TEXT NOT NULL DEFAULT '';
