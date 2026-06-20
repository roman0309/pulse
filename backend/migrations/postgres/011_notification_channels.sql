-- ============================================================
-- Reusable notification channels: configure a Telegram bot / Slack / webhook
-- once per project (secret stored encrypted), then reference it from rules.
-- ============================================================

CREATE TABLE IF NOT EXISTS notification_channels (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    type        TEXT NOT NULL,            -- slack | telegram | webhook
    config_enc  TEXT NOT NULL,            -- AES-256-GCM encrypted JSON (token/url/chat_id)
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_channels_project ON notification_channels(project_id);

-- Rules can target a saved channel; legacy notify_type/notify_url still work.
ALTER TABLE alert_rules ADD COLUMN IF NOT EXISTS notify_channel_id UUID;
