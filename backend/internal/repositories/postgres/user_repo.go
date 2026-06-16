package postgres

import (
	"context"
	"errors"

	"github.com/acme/observability/internal/domain/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type UserRepo struct{ db *pgxpool.Pool }

func NewUserRepo(db *pgxpool.Pool) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) CreateUser(ctx context.Context, u *entities.User) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO users (email, name, password_hash)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at, updated_at`,
		u.Email, u.Name, u.PasswordHash,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	u := &entities.User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, email, name, password_hash, created_at, updated_at FROM users WHERE email=$1`,
		email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	u := &entities.User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, email, name, password_hash, created_at, updated_at FROM users WHERE id=$1`,
		id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (r *UserRepo) SaveRefreshToken(ctx context.Context, t *entities.RefreshToken) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3) RETURNING id, created_at`,
		t.UserID, t.TokenHash, t.ExpiresAt,
	).Scan(&t.ID, &t.CreatedAt)
}

func (r *UserRepo) GetRefreshToken(ctx context.Context, tokenHash string) (*entities.RefreshToken, error) {
	t := &entities.RefreshToken{}
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, revoked, created_at
		 FROM refresh_tokens WHERE token_hash=$1`,
		tokenHash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.Revoked, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (r *UserRepo) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked=true WHERE id=$1`, id)
	return err
}
