-- ============================================================
-- Correlate logs with traces: store the trace id on each log line when known
-- (OTLP log records carry it; the agent best-effort parses it from the text).
-- ============================================================

ALTER TABLE metrics_db.logs ADD COLUMN IF NOT EXISTS trace_id String DEFAULT '';
