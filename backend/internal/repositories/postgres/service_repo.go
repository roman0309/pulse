package postgres

import (
	"context"
	"errors"

	"github.com/acme/observability/internal/domain/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ServiceRepo struct{ db *pgxpool.Pool }

func NewServiceRepo(db *pgxpool.Pool) *ServiceRepo { return &ServiceRepo{db: db} }

func (r *ServiceRepo) Create(ctx context.Context, s *entities.Service) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO services (project_id, name, environment, status)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`,
		s.ProjectID, s.Name, s.Environment, s.Status,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

func (r *ServiceRepo) ListByProject(ctx context.Context, projectID uuid.UUID) ([]entities.Service, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, name, environment, status, created_at, updated_at
		 FROM services WHERE project_id=$1 ORDER BY name`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []entities.Service
	for rows.Next() {
		var s entities.Service
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.Name, &s.Environment, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *ServiceRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Service, error) {
	s := &entities.Service{}
	err := r.db.QueryRow(ctx,
		`SELECT id, project_id, name, environment, status, created_at, updated_at
		 FROM services WHERE id=$1`,
		id,
	).Scan(&s.ID, &s.ProjectID, &s.Name, &s.Environment, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

func (r *ServiceRepo) Update(ctx context.Context, s *entities.Service) error {
	_, err := r.db.Exec(ctx,
		`UPDATE services SET name=$2, environment=$3, status=$4 WHERE id=$1`,
		s.ID, s.Name, s.Environment, s.Status,
	)
	return err
}

func (r *ServiceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM services WHERE id=$1`, id)
	return err
}

func (r *ServiceRepo) GetOrCreateByName(ctx context.Context, projectID uuid.UUID, name, env string, keyID uuid.UUID) (uuid.UUID, error) {
	if env == "" {
		env = "production"
	}
	var kid any // store NULL rather than the zero UUID when unknown
	if keyID != uuid.Nil {
		kid = keyID
	}
	var id uuid.UUID
	// ingest_key_id is set only on first creation (DO UPDATE leaves it).
	err := r.db.QueryRow(ctx,
		`INSERT INTO services (project_id, name, environment, status, ingest_key_id)
		 VALUES ($1, $2, $3, 'healthy', $4)
		 ON CONFLICT (project_id, name, environment)
		 DO UPDATE SET name = EXCLUDED.name
		 RETURNING id`,
		projectID, name, env, kid,
	).Scan(&id)
	return id, err
}

// ListByIngestKey returns services first created by the given ingest key.
func (r *ServiceRepo) ListByIngestKey(ctx context.Context, keyID uuid.UUID) ([]entities.Service, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, name, environment, status, created_at, updated_at
		 FROM services WHERE ingest_key_id=$1`,
		keyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entities.Service
	for rows.Next() {
		var s entities.Service
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.Name, &s.Environment, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
