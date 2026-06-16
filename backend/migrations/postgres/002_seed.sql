-- ============================================================
-- Seed data
-- Demo login:  demo@metrics.dev  /  demo123456
-- Passwords are hashed with bcrypt (pgcrypto bf) — compatible with golang.org/x/crypto/bcrypt
-- ============================================================

-- Demo user
INSERT INTO users (id, email, name, password_hash)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    'demo@metrics.dev',
    'Demo User',
    crypt('demo123456', gen_salt('bf', 10))
) ON CONFLICT (email) DO NOTHING;

-- Organization
INSERT INTO organizations (id, name, slug, created_by)
VALUES (
    '22222222-2222-2222-2222-222222222222',
    'Acme Inc',
    'acme',
    '11111111-1111-1111-1111-111111111111'
) ON CONFLICT (slug) DO NOTHING;

-- Membership (owner)
INSERT INTO team_members (organization_id, user_id, role)
VALUES (
    '22222222-2222-2222-2222-222222222222',
    '11111111-1111-1111-1111-111111111111',
    'owner'
) ON CONFLICT (organization_id, user_id) DO NOTHING;

-- Project
INSERT INTO projects (id, organization_id, name, slug, description)
VALUES (
    '33333333-3333-3333-3333-333333333333',
    '22222222-2222-2222-2222-222222222222',
    'Production Platform',
    'production',
    'Main production environment'
) ON CONFLICT (organization_id, slug) DO NOTHING;

-- Services
INSERT INTO services (id, project_id, name, environment, status) VALUES
    ('44444444-0000-0000-0000-000000000001', '33333333-3333-3333-3333-333333333333', 'payment-api', 'production', 'degraded'),
    ('44444444-0000-0000-0000-000000000002', '33333333-3333-3333-3333-333333333333', 'auth-api',    'production', 'healthy'),
    ('44444444-0000-0000-0000-000000000003', '33333333-3333-3333-3333-333333333333', 'frontend',    'production', 'healthy'),
    ('44444444-0000-0000-0000-000000000004', '33333333-3333-3333-3333-333333333333', 'worker',      'production', 'healthy')
ON CONFLICT (project_id, name, environment) DO NOTHING;

-- Deployments
INSERT INTO deployments (project_id, service_id, version, commit_sha, environment, deployed_by, status, created_at) VALUES
    ('33333333-3333-3333-3333-333333333333', '44444444-0000-0000-0000-000000000001', 'v1.8.2', 'a1b2c3d', 'production', 'alice', 'success',     now() - interval '1 hour 28 minutes'),
    ('33333333-3333-3333-3333-333333333333', '44444444-0000-0000-0000-000000000001', 'v1.8.1', '9f8e7d6', 'production', 'alice', 'rolled_back', now() - interval '3 hours'),
    ('33333333-3333-3333-3333-333333333333', '44444444-0000-0000-0000-000000000002', 'v2.3.0', 'c4d5e6f', 'production', 'bob',   'success',     now() - interval '5 hours'),
    ('33333333-3333-3333-3333-333333333333', '44444444-0000-0000-0000-000000000003', 'v4.1.0', '1a2b3c4', 'production', 'carol', 'success',     now() - interval '8 hours');

-- Alerts
INSERT INTO alerts (project_id, service_id, title, type, severity, status, description, created_at, resolved_at) VALUES
    ('33333333-3333-3333-3333-333333333333', '44444444-0000-0000-0000-000000000001', 'High latency on payment-api',   'high_latency',    'critical', 'active',   'P95 latency exceeded 800ms threshold', now() - interval '1 hour 25 minutes', NULL),
    ('33333333-3333-3333-3333-333333333333', '44444444-0000-0000-0000-000000000001', 'Error rate spike on payment-api','high_error_rate', 'high',     'active',   'Error rate exceeded 5%',              now() - interval '1 hour 26 minutes', NULL),
    ('33333333-3333-3333-3333-333333333333', '44444444-0000-0000-0000-000000000004', 'Worker service down',            'service_down',    'medium',   'resolved', 'Worker stopped responding to health checks', now() - interval '6 hours', now() - interval '5 hours 40 minutes');

-- Timeline events (the famous payment-api incident)
INSERT INTO timeline_events (project_id, service_id, type, title, description, severity, occurred_at) VALUES
    ('33333333-3333-3333-3333-333333333333', '44444444-0000-0000-0000-000000000001', 'deployment',   'Deployment v1.8.2',      'payment-api deployed by alice',                NULL,       now() - interval '1 hour 28 minutes'),
    ('33333333-3333-3333-3333-333333333333', '44444444-0000-0000-0000-000000000001', 'metric_spike', 'Latency increased',      'P95 latency jumped from 120ms to 800ms (+230%)','high',     now() - interval '1 hour 27 minutes'),
    ('33333333-3333-3333-3333-333333333333', '44444444-0000-0000-0000-000000000001', 'error_spike',  'Error rate increased',   'Database timeout exceptions detected',         'high',     now() - interval '1 hour 26 minutes'),
    ('33333333-3333-3333-3333-333333333333', '44444444-0000-0000-0000-000000000001', 'alert',        'Alert triggered',        'High latency alert fired (critical)',          'critical', now() - interval '1 hour 25 minutes');
