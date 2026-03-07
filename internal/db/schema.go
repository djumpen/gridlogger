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
		`CREATE TABLE IF NOT EXISTS telegram_accounts (
			telegram_id BIGINT PRIMARY KEY,
			username TEXT NOT NULL DEFAULT '',
			first_name TEXT NOT NULL DEFAULT '',
			last_name TEXT NOT NULL DEFAULT '',
			photo_url TEXT NOT NULL DEFAULT '',
			is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
			is_admin BOOLEAN NOT NULL DEFAULT FALSE,
			chat_type TEXT NOT NULL DEFAULT 'private',
			chat_title TEXT NOT NULL DEFAULT '',
			last_auth_date BIGINT NOT NULL DEFAULT 0,
			last_login_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`ALTER TABLE telegram_accounts ADD COLUMN IF NOT EXISTS chat_type TEXT`,
		`UPDATE telegram_accounts SET chat_type = 'private' WHERE chat_type IS NULL OR chat_type = ''`,
		`ALTER TABLE telegram_accounts ALTER COLUMN chat_type SET DEFAULT 'private'`,
		`ALTER TABLE telegram_accounts ALTER COLUMN chat_type SET NOT NULL`,
		`ALTER TABLE telegram_accounts ADD COLUMN IF NOT EXISTS chat_title TEXT`,
		`UPDATE telegram_accounts SET chat_title = '' WHERE chat_title IS NULL`,
		`ALTER TABLE telegram_accounts ALTER COLUMN chat_title SET DEFAULT ''`,
		`ALTER TABLE telegram_accounts ALTER COLUMN chat_title SET NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_telegram_accounts_username ON telegram_accounts(username)`,
		`CREATE INDEX IF NOT EXISTS idx_telegram_accounts_last_login_at ON telegram_accounts(last_login_at DESC)`,
		`CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			telegram_id BIGINT UNIQUE REFERENCES telegram_accounts(telegram_id) ON DELETE SET NULL,
			is_virtual BOOLEAN NOT NULL DEFAULT FALSE,
			owner_id BIGINT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS is_virtual BOOLEAN`,
		`UPDATE users SET is_virtual = FALSE WHERE is_virtual IS NULL`,
		`ALTER TABLE users ALTER COLUMN is_virtual SET DEFAULT FALSE`,
		`ALTER TABLE users ALTER COLUMN is_virtual SET NOT NULL`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS owner_id BIGINT NULL REFERENCES users(id) ON DELETE CASCADE`,
		`CREATE INDEX IF NOT EXISTS idx_users_telegram_id ON users(telegram_id)`,
		`CREATE INDEX IF NOT EXISTS idx_users_owner_id ON users(owner_id)`,
		`CREATE TABLE IF NOT EXISTS projects (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			user_id BIGINT NOT NULL REFERENCES users(id),
			city TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			secret TEXT NOT NULL DEFAULT '',
			is_public BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`ALTER TABLE projects ADD COLUMN IF NOT EXISTS secret TEXT`,
		`UPDATE projects SET secret = 'project-' || id::text WHERE secret IS NULL OR secret = ''`,
		`ALTER TABLE projects ALTER COLUMN secret SET NOT NULL`,
		`ALTER TABLE projects ADD COLUMN IF NOT EXISTS is_public BOOLEAN`,
		`UPDATE projects SET is_public = TRUE WHERE is_public IS NULL`,
		`ALTER TABLE projects ALTER COLUMN is_public SET DEFAULT TRUE`,
		`ALTER TABLE projects ALTER COLUMN is_public SET NOT NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_projects_slug ON projects(slug)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_user_id ON projects(user_id)`,
		`CREATE TABLE IF NOT EXISTS project_notification_subscriptions (
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (user_id, project_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_project_notification_subscriptions_project_active
			ON project_notification_subscriptions(project_id, is_active)`,
		`CREATE TABLE IF NOT EXISTS project_status_state (
			project_id BIGINT PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
			last_status TEXT NOT NULL DEFAULT 'unknown',
			last_changed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			last_notified_status TEXT NOT NULL DEFAULT '',
			last_notified_at TIMESTAMPTZ NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_project_status_state_last_changed_at
			ON project_status_state(last_changed_at DESC)`,
		`CREATE TABLE IF NOT EXISTS dtek_groups (
			project_id BIGINT PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
			region_id INTEGER NOT NULL,
			region_name TEXT NOT NULL DEFAULT '',
			dso_id INTEGER NOT NULL,
			dso_name TEXT NOT NULL DEFAULT '',
			street_id INTEGER NOT NULL,
			street_name TEXT NOT NULL DEFAULT '',
			house_id INTEGER NOT NULL,
			house_name TEXT NOT NULL DEFAULT '',
			group_key TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_dtek_groups_group_key ON dtek_groups(group_key)`,
	}
	for _, q := range stmts {
		if _, err := pool.Exec(ctx, q); err != nil {
			return err
		}
	}
	return nil
}
