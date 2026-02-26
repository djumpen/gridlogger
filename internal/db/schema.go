package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func EnsureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS pings (
			id BIGSERIAL,
			project_id INTEGER NOT NULL,
			ts TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_pings_project_ts ON pings(project_id, ts DESC)`,
		`ALTER TABLE pings DROP CONSTRAINT IF EXISTS pings_pkey`,
		`SELECT create_hypertable('pings', by_range('ts'), if_not_exists => TRUE)`,
	}
	for _, q := range stmts {
		if _, err := pool.Exec(ctx, q); err != nil {
			return err
		}
	}
	return nil
}
