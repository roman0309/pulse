package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GetOrCreateAppSecret returns the persisted value for name, inserting the
// provided value on first use. Concurrency-safe across replicas via ON CONFLICT.
func GetOrCreateAppSecret(ctx context.Context, db *pgxpool.Pool, name, value string) (string, error) {
	if _, err := db.Exec(ctx,
		`INSERT INTO app_secrets (name, value) VALUES ($1,$2) ON CONFLICT (name) DO NOTHING`,
		name, value,
	); err != nil {
		return "", err
	}
	var got string
	err := db.QueryRow(ctx, `SELECT value FROM app_secrets WHERE name=$1`, name).Scan(&got)
	return got, err
}
