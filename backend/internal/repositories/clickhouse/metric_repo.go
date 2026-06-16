package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/acme/observability/internal/domain/entities"
)

type MetricRepo struct{ conn driver.Conn }

func NewMetricRepo(conn driver.Conn) *MetricRepo { return &MetricRepo{conn: conn} }

func (r *MetricRepo) Insert(ctx context.Context, points []entities.MetricPoint) error {
	if len(points) == 0 {
		return nil
	}
	batch, err := r.conn.PrepareBatch(ctx,
		`INSERT INTO metrics_db.metrics (project_id, service_id, service_name, metric_name, value, timestamp)`)
	if err != nil {
		return err
	}
	for _, p := range points {
		if err := batch.Append(p.ProjectID, p.ServiceID, p.ServiceName, p.MetricName, p.Value, p.Timestamp); err != nil {
			return err
		}
	}
	return batch.Send()
}

// Query returns one MetricSeries per service, with values averaged into
// time buckets of stepSeconds for charting.
func (r *MetricRepo) Query(ctx context.Context, projectID, serviceID, metricName string, from, to time.Time, stepSeconds int) ([]entities.MetricSeries, error) {
	if stepSeconds <= 0 {
		stepSeconds = 60
	}
	query := `
		SELECT service_id,
		       toStartOfInterval(timestamp, INTERVAL ? second) AS bucket,
		       avg(value) AS v
		FROM metrics_db.metrics
		WHERE project_id = ? AND metric_name = ?
		  AND timestamp BETWEEN ? AND ?`
	args := []any{stepSeconds, projectID, metricName, from, to}
	if serviceID != "" {
		query += " AND service_id = ?"
		args = append(args, serviceID)
	}
	query += " GROUP BY service_id, bucket ORDER BY service_id, bucket"

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seriesMap := map[string]*entities.MetricSeries{}
	var order []string
	for rows.Next() {
		var sid string
		var bucket time.Time
		var v float64
		if err := rows.Scan(&sid, &bucket, &v); err != nil {
			return nil, err
		}
		s, ok := seriesMap[sid]
		if !ok {
			s = &entities.MetricSeries{MetricName: metricName, ServiceID: sid}
			seriesMap[sid] = s
			order = append(order, sid)
		}
		s.Points = append(s.Points, entities.SeriesPoint{Timestamp: bucket, Value: v})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]entities.MetricSeries, 0, len(order))
	for _, sid := range order {
		out = append(out, *seriesMap[sid])
	}
	return out, nil
}

// EvalValue averages a metric over [since, now] for the alerting engine.
func (r *MetricRepo) EvalValue(ctx context.Context, projectID, serviceID, metricName string, since time.Time) (float64, bool, error) {
	query := `SELECT avg(value), count() FROM metrics_db.metrics
	          WHERE project_id = ? AND metric_name = ? AND timestamp >= ?`
	args := []any{projectID, metricName, since}
	if serviceID != "" {
		query += " AND service_id = ?"
		args = append(args, serviceID)
	}
	var avg float64
	var cnt uint64
	if err := r.conn.QueryRow(ctx, query, args...).Scan(&avg, &cnt); err != nil {
		return 0, false, err
	}
	if cnt == 0 {
		return 0, false, nil
	}
	return avg, true, nil
}

func (r *MetricRepo) Latest(ctx context.Context, projectID, serviceID, metricName string) (float64, error) {
	query := fmt.Sprintf(`
		SELECT value FROM metrics_db.metrics
		WHERE project_id = ? AND metric_name = ? %s
		ORDER BY timestamp DESC LIMIT 1`,
		map[bool]string{true: "AND service_id = ?", false: ""}[serviceID != ""])

	args := []any{projectID, metricName}
	if serviceID != "" {
		args = append(args, serviceID)
	}
	var v float64
	row := r.conn.QueryRow(ctx, query, args...)
	if err := row.Scan(&v); err != nil {
		return 0, nil // no data yet — treat as zero
	}
	return v, nil
}
