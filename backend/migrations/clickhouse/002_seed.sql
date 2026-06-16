-- ============================================================
-- ClickHouse seed data — synthetic time series for last 2 hours
-- payment-api shows a latency/error spike ~88 min ago (after v1.8.2)
-- ============================================================

-- ---------- payment-api metrics (degraded after deploy) ----------
INSERT INTO metrics_db.metrics (project_id, service_id, service_name, metric_name, value, timestamp)
SELECT
    '33333333-3333-3333-3333-333333333333' AS project_id,
    '44444444-0000-0000-0000-000000000001' AS service_id,
    'payment-api' AS service_name,
    m.name AS metric_name,
    -- spike begins at minute_ago < 88
    multiIf(
        m.name = 'cpu_usage',     if(number < 88, 35 + rand() % 15, 30 + rand() % 10),
        m.name = 'memory_usage',  if(number < 88, 70 + rand() % 10, 55 + rand() % 8),
        m.name = 'request_count', 100 + rand() % 50,
        m.name = 'request_rate',  20 + rand() % 10,
        m.name = 'error_rate',    if(number < 88, 6 + rand() % 4, rand() % 1),
        m.name = 'latency_p50',   if(number < 88, 220 + rand() % 80, 60 + rand() % 30),
        m.name = 'latency_p95',   if(number < 88, 800 + rand() % 200, 120 + rand() % 40),
        m.name = 'latency_p99',   if(number < 88, 1200 + rand() % 300, 180 + rand() % 50),
        0
    ) AS value,
    now() - INTERVAL number MINUTE AS timestamp
FROM numbers(120) AS n
CROSS JOIN (
    SELECT arrayJoin(['cpu_usage','memory_usage','request_count','request_rate','error_rate','latency_p50','latency_p95','latency_p99']) AS name
) AS m;

-- ---------- auth-api metrics (healthy) ----------
INSERT INTO metrics_db.metrics (project_id, service_id, service_name, metric_name, value, timestamp)
SELECT
    '33333333-3333-3333-3333-333333333333',
    '44444444-0000-0000-0000-000000000002',
    'auth-api',
    m.name,
    multiIf(
        m.name = 'cpu_usage',     25 + rand() % 10,
        m.name = 'memory_usage',  50 + rand() % 8,
        m.name = 'request_count', 200 + rand() % 60,
        m.name = 'request_rate',  40 + rand() % 12,
        m.name = 'error_rate',    rand() % 1,
        m.name = 'latency_p50',   40 + rand() % 20,
        m.name = 'latency_p95',   90 + rand() % 30,
        m.name = 'latency_p99',   140 + rand() % 40,
        0
    ),
    now() - INTERVAL number MINUTE
FROM numbers(120) AS n
CROSS JOIN (
    SELECT arrayJoin(['cpu_usage','memory_usage','request_count','request_rate','error_rate','latency_p50','latency_p95','latency_p99']) AS name
) AS m;

-- ---------- frontend metrics (healthy) ----------
INSERT INTO metrics_db.metrics (project_id, service_id, service_name, metric_name, value, timestamp)
SELECT
    '33333333-3333-3333-3333-333333333333',
    '44444444-0000-0000-0000-000000000003',
    'frontend',
    m.name,
    multiIf(
        m.name = 'cpu_usage',     15 + rand() % 8,
        m.name = 'memory_usage',  40 + rand() % 6,
        m.name = 'request_count', 500 + rand() % 100,
        m.name = 'request_rate',  90 + rand() % 20,
        m.name = 'error_rate',    rand() % 1,
        m.name = 'latency_p50',   20 + rand() % 10,
        m.name = 'latency_p95',   50 + rand() % 15,
        m.name = 'latency_p99',   80 + rand() % 20,
        0
    ),
    now() - INTERVAL number MINUTE
FROM numbers(120) AS n
CROSS JOIN (
    SELECT arrayJoin(['cpu_usage','memory_usage','request_count','request_rate','error_rate','latency_p50','latency_p95','latency_p99']) AS name
) AS m;

-- ---------- Logs ----------
-- payment-api error logs during the incident window
INSERT INTO metrics_db.logs (project_id, service_id, service_name, level, message, metadata, timestamp)
SELECT
    '33333333-3333-3333-3333-333333333333',
    '44444444-0000-0000-0000-000000000001',
    'payment-api',
    if(number % 3 = 0, 'error', 'warning'),
    if(number % 3 = 0,
       'database connection timeout after 5000ms while processing payment',
       'slow query detected on payments table (>500ms)'),
    '{"trace_id":"abc123","db":"payments"}',
    now() - INTERVAL (number * 2) MINUTE
FROM numbers(40);

-- general info logs across services
INSERT INTO metrics_db.logs (project_id, service_id, service_name, level, message, metadata, timestamp)
SELECT
    '33333333-3333-3333-3333-333333333333',
    '44444444-0000-0000-0000-000000000002',
    'auth-api',
    'info',
    concat('request handled GET /api/v1/session ', toString(200)),
    '{"method":"GET","status":200}',
    now() - INTERVAL number MINUTE
FROM numbers(60);
