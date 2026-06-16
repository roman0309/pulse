package postgres

import (
	"context"

	"github.com/acme/observability/internal/domain/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DeploymentRepo struct{ db *pgxpool.Pool }

func NewDeploymentRepo(db *pgxpool.Pool) *DeploymentRepo { return &DeploymentRepo{db: db} }

func (r *DeploymentRepo) Create(ctx context.Context, d *entities.Deployment) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO deployments (project_id, service_id, version, commit_sha, environment, deployed_by, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at`,
		d.ProjectID, d.ServiceID, d.Version, d.CommitSHA, d.Environment, d.DeployedBy, d.Status,
	).Scan(&d.ID, &d.CreatedAt)
}

func (r *DeploymentRepo) ListByProject(ctx context.Context, projectID uuid.UUID, serviceID *uuid.UUID, limit int) ([]entities.Deployment, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.Query(ctx,
		`SELECT d.id, d.project_id, d.service_id, s.name, d.version, d.commit_sha,
		        d.environment, d.deployed_by, d.status, d.created_at
		 FROM deployments d JOIN services s ON s.id = d.service_id
		 WHERE d.project_id=$1 AND ($2::uuid IS NULL OR d.service_id=$2)
		 ORDER BY d.created_at DESC LIMIT $3`,
		projectID, serviceID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []entities.Deployment
	for rows.Next() {
		var d entities.Deployment
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.ServiceID, &d.ServiceName, &d.Version,
			&d.CommitSHA, &d.Environment, &d.DeployedBy, &d.Status, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *DeploymentRepo) CountToday(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM deployments WHERE project_id=$1 AND created_at >= date_trunc('day', now())`,
		projectID,
	).Scan(&n)
	return n, err
}
