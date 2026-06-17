-- ============================================================
-- Alert rules — evaluated by the background alerting engine.
-- A rule watches a metric; when the condition holds for `for_seconds`,
-- it fires an alert (and optionally notifies Slack/webhook), then resolves
-- it on recovery.
-- ============================================================

CREATE TYPE alert_rule_operator AS ENUM ('gt', 'lt', 'gte', 'lte');

CREATE TABLE IF NOT EXISTS alert_rules (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    service_id  UUID REFERENCES services(id) ON DELETE CASCADE, -- NULL = all services
    metric      TEXT NOT NULL,                  -- e.g. error_rate, latency_p95
    operator    alert_rule_operator NOT NULL DEFAULT 'gt',
    threshold   DOUBLE PRECISION NOT NULL,
    for_seconds INT NOT NULL DEFAULT 0,         -- condition must hold this long
    severity    alert_severity NOT NULL DEFAULT 'high',
    type        alert_type NOT NULL DEFAULT 'high_error_rate',
    notify_type TEXT NOT NULL DEFAULT 'none',   -- none | slack | webhook
    notify_url  TEXT NOT NULL DEFAULT '',
    enabled     BOOLEAN NOT NULL DEFAULT true,

    -- evaluator state
    breaching_since TIMESTAMPTZ,                -- when the breach started (NULL = not breaching)
    active_alert_id UUID,                       -- the alert currently firing (NULL = not firing)

    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_alert_rules_project ON alert_rules(project_id);
CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled ON alert_rules(enabled);

CREATE TRIGGER trg_alert_rules_updated BEFORE UPDATE ON alert_rules
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
-- (demo rule is seeded separately in 006_seed_extras.sql, gated behind SEED_DEMO)
