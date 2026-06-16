package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/acme/observability/internal/domain/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertRepo struct{ db *pgxpool.Pool }

func NewAlertRepo(db *pgxpool.Pool) *AlertRepo { return &AlertRepo{db: db} }

func (r *AlertRepo) Create(ctx context.Context, a *entities.Alert) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO alerts (project_id, service_id, title, type, severity, status, description)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at`,
		a.ProjectID, a.ServiceID, a.Title, a.Type, a.Severity, a.Status, a.Description,
	).Scan(&a.ID, &a.CreatedAt)
}

func (r *AlertRepo) ListByProject(ctx context.Context, projectID uuid.UUID, status *entities.AlertStatus) ([]entities.Alert, error) {
	rows, err := r.db.Query(ctx,
		`SELECT a.id, a.project_id, a.service_id, COALESCE(s.name, ''), a.title, a.type,
		        a.severity, a.status, a.description, a.created_at, a.resolved_at
		 FROM alerts a LEFT JOIN services s ON s.id = a.service_id
		 WHERE a.project_id=$1 AND ($2::text IS NULL OR a.status::text=$2)
		 ORDER BY a.created_at DESC`,
		projectID, status,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []entities.Alert
	for rows.Next() {
		var a entities.Alert
		if err := rows.Scan(&a.ID, &a.ProjectID, &a.ServiceID, &a.ServiceName, &a.Title, &a.Type,
			&a.Severity, &a.Status, &a.Description, &a.CreatedAt, &a.ResolvedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AlertRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Alert, error) {
	a := &entities.Alert{}
	err := r.db.QueryRow(ctx,
		`SELECT id, project_id, service_id, title, type, severity, status, description, created_at, resolved_at
		 FROM alerts WHERE id=$1`,
		id,
	).Scan(&a.ID, &a.ProjectID, &a.ServiceID, &a.Title, &a.Type, &a.Severity, &a.Status,
		&a.Description, &a.CreatedAt, &a.ResolvedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (r *AlertRepo) Resolve(ctx context.Context, id uuid.UUID, resolvedAt time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE alerts SET status='resolved', resolved_at=$2 WHERE id=$1`,
		id, resolvedAt,
	)
	return err
}

func (r *AlertRepo) CountActive(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM alerts WHERE project_id=$1 AND status='active'`,
		projectID,
	).Scan(&n)
	return n, err
}
