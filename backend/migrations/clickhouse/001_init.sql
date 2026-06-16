-- ============================================================
-- Observability Platform — ClickHouse schema
-- Time-series metrics + structured logs
-- ============================================================

CREATE DATABASE IF NOT EXISTS metrics_db;

-- ---------- Metrics ----------
-- metric_name: cpu_usage | memory_usage | request_count | request_rate |
--              error_rate | latency_p50 | latency_p95 | latency_p99
CREATE TABLE IF NOT EXISTS metrics_db.metrics
(
    project_id   String,
    service_id   String,
    service_name String,
    metric_name  LowCardinality(String),
    value        Float64,
    timestamp    DateTime64(3, 'UTC')
)
ENGINE = MergeTree
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (project_id, service_id, metric_name, timestamp)
TTL toDateTime(timestamp) + INTERVAL 30 DAY;

-- ---------- Logs ----------
CREATE TABLE IF NOT EXISTS metrics_db.logs
(
    project_id   String,
    service_id   String,
    service_name String,
    level        LowCardinality(String),   -- info | warning | error
    message      String,
    metadata     String DEFAULT '{}',      -- JSON string
    timestamp    DateTime64(3, 'UTC'),
    INDEX idx_message message TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 4
)
ENGINE = MergeTree
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (project_id, service_id, timestamp)
TTL toDateTime(timestamp) + INTERVAL 30 DAY;
