package postgres

import (
	"context"
	"errors"

	"github.com/acme/observability/internal/domain/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ServerRepo struct{ db *pgxpool.Pool }

func NewServerRepo(db *pgxpool.Pool) *ServerRepo { return &ServerRepo{db: db} }

const serverCols = `id, project_id, name, ssh_target, ssh_host, ssh_port, ssh_user, auth_method,
	secret_enc, host_key, status, last_result, ingest_key_id, beyla_key_id, created_at, updated_at`

func scanServer(row pgx.Row) (*entities.ManagedServer, error) {
	s := &entities.ManagedServer{}
	err := row.Scan(&s.ID, &s.ProjectID, &s.Name, &s.SSHTarget, &s.SSHHost, &s.SSHPort, &s.SSHUser,
		&s.AuthMethod, &s.SecretEnc, &s.HostKey, &s.Status, &s.LastResult, &s.IngestKeyID, &s.BeylaKeyID, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

func (r *ServerRepo) Create(ctx context.Context, s *entities.ManagedServer) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO managed_servers
		   (project_id, name, ssh_target, ssh_host, ssh_port, ssh_user, auth_method, secret_enc, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, created_at, updated_at`,
		s.ProjectID, s.Name, s.SSHTarget, s.SSHHost, s.SSHPort, s.SSHUser, s.AuthMethod, s.SecretEnc, s.Status,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

func (r *ServerRepo) ListByProject(ctx context.Context, projectID uuid.UUID) ([]entities.ManagedServer, error) {
	rows, err := r.db.Query(ctx, `SELECT `+serverCols+` FROM managed_servers WHERE project_id=$1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entities.ManagedServer
	for rows.Next() {
		s, err := scanServer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *ServerRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.ManagedServer, error) {
	s, err := scanServer(r.db.QueryRow(ctx, `SELECT `+serverCols+` FROM managed_servers WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

func (r *ServerRepo) Update(ctx context.Context, s *entities.ManagedServer) error {
	_, err := r.db.Exec(ctx,
		`UPDATE managed_servers SET name=$2, ssh_target=$3, status=$4, last_result=$5, ingest_key_id=$6, host_key=$7, beyla_key_id=$8 WHERE id=$1`,
		s.ID, s.Name, s.SSHTarget, s.Status, s.LastResult, s.IngestKeyID, s.HostKey, s.BeylaKeyID,
	)
	return err
}

func (r *ServerRepo) Delete(ctx context.Context, projectID, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM managed_servers WHERE id=$1 AND project_id=$2`, id, projectID)
	return err
}
