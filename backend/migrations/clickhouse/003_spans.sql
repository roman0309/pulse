-- ============================================================
-- Distributed tracing spans (OTLP). Higher volume + shorter retention than
-- metrics/logs. One row per span; a trace is all rows sharing trace_id.
-- ============================================================

CREATE TABLE IF NOT EXISTS metrics_db.spans
(
    project_id   String,
    trace_id     String,
    span_id      String,
    parent_id    String,
    service_name String,
    name         String,
    kind         LowCardinality(String),   -- server | client | producer | consumer | internal
    status_code  LowCardinality(String),   -- unset | ok | error
    start_time   DateTime64(3, 'UTC'),
    duration_ms  Float64,
    attributes   String DEFAULT '{}'
)
ENGINE = MergeTree
PARTITION BY toYYYYMMDD(start_time)
ORDER BY (project_id, trace_id, start_time)
TTL toDateTime(start_time) + INTERVAL 7 DAY;
