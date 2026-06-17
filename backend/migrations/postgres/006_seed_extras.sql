-- ============================================================
-- Demo seed extras (gated behind SEED_DEMO) — depends on tables created in
-- 003 (ingest_keys) and 005 (alert_rules), so it must run last.
-- ============================================================

-- Demo ingest key. Plaintext: pulse_demo_ingest_key
INSERT INTO ingest_keys (project_id, name, prefix, key_hash)
VALUES (
    '33333333-3333-3333-3333-333333333333',
    'demo',
    'pulse_demo_i',
    encode(digest('pulse_demo_ingest_key', 'sha256'), 'hex')
) ON CONFLICT (key_hash) DO NOTHING;

-- Example alert rule that fires on the live demo traffic.
INSERT INTO alert_rules (project_id, name, metric, operator, threshold, for_seconds, severity, type)
SELECT '33333333-3333-3333-3333-333333333333', 'Elevated error rate', 'error_rate', 'gt', 1, 0, 'high', 'high_error_rate'
WHERE NOT EXISTS (
    SELECT 1 FROM alert_rules
    WHERE project_id = '33333333-3333-3333-3333-333333333333' AND name = 'Elevated error rate'
);
