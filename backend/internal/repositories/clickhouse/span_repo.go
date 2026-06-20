package clickhouse

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/acme/observability/internal/domain/entities"
)

type SpanRepo struct{ conn driver.Conn }

func NewSpanRepo(conn driver.Conn) *SpanRepo { return &SpanRepo{conn: conn} }

func (r *SpanRepo) Insert(ctx context.Context, spans []entities.Span) error {
	if len(spans) == 0 {
		return nil
	}
	batch, err := r.conn.PrepareBatch(ctx,
		`INSERT INTO metrics_db.spans (project_id, trace_id, span_id, parent_id, service_name, name, kind, status_code, start_time, duration_ms, attributes)`)
	if err != nil {
		return err
	}
	for _, s := range spans {
		attrs := s.Attributes
		if attrs == "" {
			attrs = "{}"
		}
		if err := batch.Append(s.ProjectID, s.TraceID, s.SpanID, s.ParentID, s.ServiceName,
			s.Name, s.Kind, s.StatusCode, s.StartTime, s.DurationMS, attrs); err != nil {
			return err
		}
	}
	return batch.Send()
}

// ListTraces returns one summary row per trace_id in the window, newest first.
func (r *SpanRepo) ListTraces(ctx context.Context, projectID, serviceName string, from, to time.Time, limit int) ([]entities.TraceSummary, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `
		SELECT
			trace_id,
			argMin(service_name, start_time) AS root_service,
			argMin(name, start_time)         AS root_name,
			min(start_time)                  AS started,
			max(toUnixTimestamp64Milli(start_time) + duration_ms) - toUnixTimestamp64Milli(min(start_time)) AS duration_ms,
			toInt32(count())                 AS span_count,
			toInt32(countIf(status_code = 'error')) AS error_count
		FROM metrics_db.spans
		WHERE project_id = ? AND start_time BETWEEN ? AND ?`
	args := []any{projectID, from, to}
	if serviceName != "" {
		query += " AND trace_id IN (SELECT trace_id FROM metrics_db.spans WHERE project_id = ? AND service_name = ? AND start_time BETWEEN ? AND ?)"
		args = append(args, projectID, serviceName, from, to)
	}
	query += " GROUP BY trace_id ORDER BY started DESC LIMIT ?"
	args = append(args, limit)

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []entities.TraceSummary
	for rows.Next() {
		var t entities.TraceSummary
		var spanCount, errCount int32
		if err := rows.Scan(&t.TraceID, &t.RootService, &t.RootName, &t.StartTime, &t.DurationMS, &spanCount, &errCount); err != nil {
			return nil, err
		}
		t.SpanCount = int(spanCount)
		t.ErrorCount = int(errCount)
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetTrace returns every span of one trace, ordered by start time.
func (r *SpanRepo) GetTrace(ctx context.Context, projectID, traceID string) ([]entities.Span, error) {
	rows, err := r.conn.Query(ctx, `
		SELECT project_id, trace_id, span_id, parent_id, service_name, name, kind, status_code, start_time, duration_ms, attributes
		FROM metrics_db.spans
		WHERE project_id = ? AND trace_id = ?
		ORDER BY start_time ASC
		LIMIT 2000`, projectID, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []entities.Span
	for rows.Next() {
		var s entities.Span
		if err := rows.Scan(&s.ProjectID, &s.TraceID, &s.SpanID, &s.ParentID, &s.ServiceName,
			&s.Name, &s.Kind, &s.StatusCode, &s.StartTime, &s.DurationMS, &s.Attributes); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
