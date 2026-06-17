// Package migrate applies the embedded SQL migrations to Postgres and
// ClickHouse on startup, tracking applied versions so each runs once. Files
// whose name contains "seed" are only applied when seedDemo is true.
package migrate

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"

	chdriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/acme/observability/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres applies all pending Postgres migrations.
func Postgres(ctx context.Context, pool *pgxpool.Pool, seedDemo bool, log *slog.Logger) error {
	if _, err := pool.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ DEFAULT now())`); err != nil {
		return fmt.Errorf("pg migrations table: %w", err)
	}

	files, err := sqlFiles("postgres")
	if err != nil {
		return err
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	for _, name := range files {
		if skip(name, seedDemo) {
			continue
		}
		var applied bool
		if err := pool.QueryRow(ctx, `SELECT exists(SELECT 1 FROM schema_migrations WHERE version=$1)`, name).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		content, err := migrations.FS.ReadFile("postgres/" + name)
		if err != nil {
			return err
		}
		// Simple protocol supports multi-statement scripts (incl. $$ bodies).
		mrr := conn.Conn().PgConn().Exec(ctx, string(content))
		if _, err := mrr.ReadAll(); err != nil {
			return fmt.Errorf("pg migration %s: %w", name, err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			return err
		}
		log.Info("applied postgres migration", "version", name)
	}
	return nil
}

// ClickHouse applies all pending ClickHouse migrations.
func ClickHouse(ctx context.Context, conn chdriver.Conn, seedDemo bool, log *slog.Logger) error {
	if err := conn.Exec(ctx, `CREATE DATABASE IF NOT EXISTS metrics_db`); err != nil {
		return err
	}
	if err := conn.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS metrics_db.schema_migrations (version String, applied_at DateTime DEFAULT now()) ENGINE = MergeTree ORDER BY version`); err != nil {
		return fmt.Errorf("ch migrations table: %w", err)
	}

	files, err := sqlFiles("clickhouse")
	if err != nil {
		return err
	}
	for _, name := range files {
		if skip(name, seedDemo) {
			continue
		}
		var cnt uint64
		if err := conn.QueryRow(ctx, `SELECT count() FROM metrics_db.schema_migrations WHERE version = ?`, name).Scan(&cnt); err != nil {
			return err
		}
		if cnt > 0 {
			continue
		}
		content, err := migrations.FS.ReadFile("clickhouse/" + name)
		if err != nil {
			return err
		}
		for _, stmt := range splitStatements(string(content)) {
			if err := conn.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("ch migration %s: %w", name, err)
			}
		}
		if err := conn.Exec(ctx, `INSERT INTO metrics_db.schema_migrations (version) VALUES (?)`, name); err != nil {
			return err
		}
		log.Info("applied clickhouse migration", "version", name)
	}
	return nil
}

func sqlFiles(dir string) ([]string, error) {
	entries, err := fs.ReadDir(migrations.FS, dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func skip(name string, seedDemo bool) bool {
	return strings.Contains(name, "seed") && !seedDemo
}

// splitStatements strips line comments and splits a script into individual
// statements on ';' (safe for our ClickHouse files, which have no inline ';').
func splitStatements(script string) []string {
	var b strings.Builder
	for _, line := range strings.Split(script, "\n") {
		if i := strings.Index(line, "--"); i >= 0 {
			line = line[:i]
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	var out []string
	for _, s := range strings.Split(b.String(), ";") {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}
