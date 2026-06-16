package postgres

import (
	"context"
	"errors"

	"github.com/acme/observability/internal/domain/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProjectRepo struct{ db *pgxpool.Pool }

func NewProjectRepo(db *pgxpool.Pool) *ProjectRepo { return &ProjectRepo{db: db} }

func (r *ProjectRepo) Create(ctx context.Context, p *entities.Project) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO projects (organization_id, name, slug, description)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`,
		p.OrganizationID, p.Name, p.Slug, p.Description,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *ProjectRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]entities.Project, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, organization_id, name, slug, description, created_at, updated_at
		 FROM projects WHERE organization_id=$1 ORDER BY created_at DESC`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []entities.Project
	for rows.Next() {
		var p entities.Project
		if err := rows.Scan(&p.ID, &p.OrganizationID, &p.Name, &p.Slug, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *ProjectRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Project, error) {
	p := &entities.Project{}
	err := r.db.QueryRow(ctx,
		`SELECT id, organization_id, name, slug, description, created_at, updated_at
		 FROM projects WHERE id=$1`,
		id,
	).Scan(&p.ID, &p.OrganizationID, &p.Name, &p.Slug, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func (r *ProjectRepo) Update(ctx context.Context, p *entities.Project) error {
	_, err := r.db.Exec(ctx,
		`UPDATE projects SET name=$2, description=$3 WHERE id=$1`,
		p.ID, p.Name, p.Description,
	)
	return err
}

func (r *ProjectRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM projects WHERE id=$1`, id)
	return err
}
