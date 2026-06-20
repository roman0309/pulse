package postgres

import (
	"context"
	"errors"

	"github.com/acme/observability/internal/domain/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IngestKeyRepo struct{ db *pgxpool.Pool }

func NewIngestKeyRepo(db *pgxpool.Pool) *IngestKeyRepo { return &IngestKeyRepo{db: db} }

func (r *IngestKeyRepo) ResolveProject(ctx context.Context, keyHash string) (uuid.UUID, uuid.UUID, error) {
	var projectID, keyID uuid.UUID
	err := r.db.QueryRow(ctx,
		`UPDATE ingest_keys SET last_used_at = now() WHERE key_hash=$1 RETURNING project_id, id`, keyHash,
	).Scan(&projectID, &keyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, uuid.Nil, ErrNotFound
	}
	return projectID, keyID, err
}

func (r *IngestKeyRepo) Create(ctx context.Context, k *entities.IngestKey) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO ingest_keys (project_id, name, prefix, key_hash)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		k.ProjectID, k.Name, k.Prefix, k.KeyHash,
	).Scan(&k.ID, &k.CreatedAt)
}

func (r *IngestKeyRepo) ListByProject(ctx context.Context, projectID uuid.UUID) ([]entities.IngestKey, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, name, prefix, created_at, last_used_at
		 FROM ingest_keys WHERE project_id=$1 ORDER BY created_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []entities.IngestKey
	for rows.Next() {
		var k entities.IngestKey
		if err := rows.Scan(&k.ID, &k.ProjectID, &k.Name, &k.Prefix, &k.CreatedAt, &k.LastUsed); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (r *IngestKeyRepo) Delete(ctx context.Context, projectID, keyID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM ingest_keys WHERE id=$1 AND project_id=$2`, keyID, projectID)
	return err
}
