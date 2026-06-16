-- ============================================================
-- Observability Platform — PostgreSQL schema
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------- Users ----------
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email         TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------- Refresh tokens ----------
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked     BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id);

-- ---------- Organizations ----------
CREATE TABLE IF NOT EXISTS organizations (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    created_by  UUID NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------- Team members (membership + role) ----------
CREATE TYPE org_role AS ENUM ('owner', 'admin', 'member');

CREATE TABLE IF NOT EXISTS team_members (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role            org_role NOT NULL DEFAULT 'member',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (organization_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_team_members_org ON team_members(organization_id);
CREATE INDEX IF NOT EXISTS idx_team_members_user ON team_members(user_id);

-- ---------- Projects ----------
CREATE TABLE IF NOT EXISTS projects (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (organization_id, slug)
);
CREATE INDEX IF NOT EXISTS idx_projects_org ON projects(organization_id);

-- ---------- Services ----------
CREATE TYPE service_status AS ENUM ('healthy', 'degraded', 'down');

CREATE TABLE IF NOT EXISTS services (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    environment TEXT NOT NULL DEFAULT 'production',
    status      service_status NOT NULL DEFAULT 'healthy',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, name, environment)
);
CREATE INDEX IF NOT EXISTS idx_services_project ON services(project_id);

-- ---------- Deployments ----------
CREATE TABLE IF NOT EXISTS deployments (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id   UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    service_id   UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    version      TEXT NOT NULL,
    commit_sha   TEXT NOT NULL DEFAULT '',
    environment  TEXT NOT NULL DEFAULT 'production',
    deployed_by  TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'success', -- success | failed | rolled_back
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_deployments_project ON deployments(project_id);
CREATE INDEX IF NOT EXISTS idx_deployments_service ON deployments(service_id);
CREATE INDEX IF NOT EXISTS idx_deployments_created ON deployments(created_at DESC);

-- ---------- Alerts ----------
CREATE TYPE alert_severity AS ENUM ('low', 'medium', 'high', 'critical');
CREATE TYPE alert_status   AS ENUM ('active', 'resolved');
CREATE TYPE alert_type     AS ENUM ('high_latency', 'high_error_rate', 'service_down');

CREATE TABLE IF NOT EXISTS alerts (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    service_id  UUID REFERENCES services(id) ON DELETE SET NULL,
    title       TEXT NOT NULL,
    type        alert_type NOT NULL,
    severity    alert_severity NOT NULL DEFAULT 'medium',
    status      alert_status NOT NULL DEFAULT 'active',
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_alerts_project ON alerts(project_id);
CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);
CREATE INDEX IF NOT EXISTS idx_alerts_created ON alerts(created_at DESC);

-- ---------- Timeline events ----------
-- A unified event store that combines deployments / alerts / metric spikes / errors.
CREATE TYPE timeline_event_type AS ENUM ('deployment', 'alert', 'metric_spike', 'error_spike', 'recovery');

CREATE TABLE IF NOT EXISTS timeline_events (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    service_id  UUID REFERENCES services(id) ON DELETE SET NULL,
    type        timeline_event_type NOT NULL,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    severity    alert_severity,
    ref_id      UUID, -- optional reference to deployment/alert id
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_timeline_project ON timeline_events(project_id);
CREATE INDEX IF NOT EXISTS idx_timeline_occurred ON timeline_events(occurred_at DESC);

-- ---------- updated_at trigger ----------
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_updated      BEFORE UPDATE ON users         FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_orgs_updated       BEFORE UPDATE ON organizations FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_projects_updated   BEFORE UPDATE ON projects      FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_services_updated   BEFORE UPDATE ON services      FOR EACH ROW EXECUTE FUNCTION set_updated_at();
