package clickhouse

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/acme/observability/internal/domain/entities"
)

type LogRepo struct{ conn driver.Conn }

func NewLogRepo(conn driver.Conn) *LogRepo { return &LogRepo{conn: conn} }

func (r *LogRepo) Insert(ctx context.Context, logs []entities.LogEntry) error {
	if len(logs) == 0 {
		return nil
	}
	batch, err := r.conn.PrepareBatch(ctx,
		`INSERT INTO metrics_db.logs (project_id, service_id, service_name, level, message, metadata, timestamp)`)
	if err != nil {
		return err
	}
	for _, l := range logs {
		md := l.Metadata
		if md == "" {
			md = "{}"
		}
		if err := batch.Append(l.ProjectID, l.ServiceID, l.ServiceName, l.Level, l.Message, md, l.Timestamp); err != nil {
			return err
		}
	}
	return batch.Send()
}

// DeleteService removes all logs for a service (ClickHouse lightweight delete).
func (r *LogRepo) DeleteService(ctx context.Context, projectID, serviceID string) error {
	return r.conn.Exec(ctx,
		`DELETE FROM metrics_db.logs WHERE project_id = ? AND service_id = ?`,
		projectID, serviceID)
}

// Query returns logs filtered by service, level and a full-text search term,
// newest first, with pagination via limit/offset.
func (r *LogRepo) Query(ctx context.Context, projectID, serviceID, level, search string, from, to time.Time, limit, offset int) ([]entities.LogEntry, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	query := `
		SELECT project_id, service_id, service_name, level, message, metadata, timestamp
		FROM metrics_db.logs
		WHERE project_id = ? AND timestamp BETWEEN ? AND ?`
	args := []any{projectID, from, to}

	if serviceID != "" {
		query += " AND service_id = ?"
		args = append(args, serviceID)
	}
	if level != "" {
		query += " AND level = ?"
		args = append(args, level)
	}
	if search != "" {
		query += " AND positionCaseInsensitive(message, ?) > 0"
		args = append(args, search)
	}
	query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []entities.LogEntry
	for rows.Next() {
		var l entities.LogEntry
		if err := rows.Scan(&l.ProjectID, &l.ServiceID, &l.ServiceName, &l.Level, &l.Message, &l.Metadata, &l.Timestamp); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
