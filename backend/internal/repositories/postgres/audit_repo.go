package postgres

import (
	"context"

	"github.com/acme/observability/internal/domain/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditRepo struct{ db *pgxpool.Pool }

func NewAuditRepo(db *pgxpool.Pool) *AuditRepo { return &AuditRepo{db: db} }

func (r *AuditRepo) Create(ctx context.Context, e *entities.AuditEntry) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO audit_log (project_id, user_id, server_id, action, detail, success)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, created_at`,
		e.ProjectID, e.UserID, e.ServerID, e.Action, e.Detail, e.Success,
	).Scan(&e.ID, &e.CreatedAt)
}

func (r *AuditRepo) ListByProject(ctx context.Context, projectID uuid.UUID, limit int) ([]entities.AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, user_id, server_id, action, detail, success, created_at
		 FROM audit_log WHERE project_id=$1 ORDER BY created_at DESC LIMIT $2`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entities.AuditEntry
	for rows.Next() {
		var e entities.AuditEntry
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.UserID, &e.ServerID, &e.Action, &e.Detail, &e.Success, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
