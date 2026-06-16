package postgres

import (
	"context"
	"time"

	"github.com/acme/observability/internal/domain/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TimelineRepo struct{ db *pgxpool.Pool }

func NewTimelineRepo(db *pgxpool.Pool) *TimelineRepo { return &TimelineRepo{db: db} }

func (r *TimelineRepo) Create(ctx context.Context, e *entities.TimelineEvent) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO timeline_events (project_id, service_id, type, title, description, severity, ref_id, occurred_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, created_at`,
		e.ProjectID, e.ServiceID, e.Type, e.Title, e.Description, e.Severity, e.RefID, e.OccurredAt,
	).Scan(&e.ID, &e.CreatedAt)
}

func (r *TimelineRepo) ListByProject(ctx context.Context, projectID uuid.UUID, from, to time.Time) ([]entities.TimelineEvent, error) {
	rows, err := r.db.Query(ctx,
		`SELECT t.id, t.project_id, t.service_id, COALESCE(s.name, ''), t.type, t.title,
		        t.description, t.severity, t.ref_id, t.occurred_at, t.created_at
		 FROM timeline_events t LEFT JOIN services s ON s.id = t.service_id
		 WHERE t.project_id=$1 AND t.occurred_at BETWEEN $2 AND $3
		 ORDER BY t.occurred_at ASC`,
		projectID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []entities.TimelineEvent
	for rows.Next() {
		var e entities.TimelineEvent
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.ServiceID, &e.ServiceName, &e.Type, &e.Title,
			&e.Description, &e.Severity, &e.RefID, &e.OccurredAt, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
