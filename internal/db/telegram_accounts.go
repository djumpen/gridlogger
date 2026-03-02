package db

import (
	"context"

	"github.com/djumpen/gridlogger/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TelegramAccountRepository struct {
	pool *pgxpool.Pool
}

func NewTelegramAccountRepository(pool *pgxpool.Pool) *TelegramAccountRepository {
	return &TelegramAccountRepository{pool: pool}
}

func (r *TelegramAccountRepository) UpsertTelegramAccount(ctx context.Context, in service.TelegramAccountUpsert) (service.TelegramAccount, bool, error) {
	const q = `
		WITH upsert_tg AS (
			INSERT INTO telegram_accounts (
				telegram_id,
				username,
				first_name,
				last_name,
				photo_url,
				last_auth_date,
				last_login_at,
				updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, now())
			ON CONFLICT (telegram_id) DO UPDATE
			SET username = EXCLUDED.username,
				first_name = EXCLUDED.first_name,
				last_name = EXCLUDED.last_name,
				photo_url = EXCLUDED.photo_url,
				last_auth_date = EXCLUDED.last_auth_date,
				last_login_at = EXCLUDED.last_login_at,
				updated_at = now()
			WHERE telegram_accounts.last_auth_date IS NULL
			   OR EXCLUDED.last_auth_date > telegram_accounts.last_auth_date
			RETURNING telegram_id, username, first_name, last_name, photo_url, is_blocked, is_admin, last_auth_date, last_login_at, created_at, updated_at
		), upsert_user AS (
			INSERT INTO users (telegram_id, updated_at)
			SELECT telegram_id, now()
			FROM upsert_tg
			ON CONFLICT (telegram_id) DO UPDATE
			SET updated_at = now()
			RETURNING id, telegram_id
		)
		SELECT u.id, t.telegram_id, t.username, t.first_name, t.last_name, t.photo_url, t.is_blocked, t.is_admin, t.last_auth_date, t.last_login_at, t.created_at, t.updated_at
		FROM upsert_tg t
		INNER JOIN upsert_user u ON u.telegram_id = t.telegram_id
	`

	var account service.TelegramAccount
	err := r.pool.QueryRow(
		ctx,
		q,
		in.TelegramID,
		in.Username,
		in.FirstName,
		in.LastName,
		in.PhotoURL,
		in.LastAuthDate,
		in.LastLoginAt,
	).Scan(
		&account.UserID,
		&account.TelegramID,
		&account.Username,
		&account.FirstName,
		&account.LastName,
		&account.PhotoURL,
		&account.IsBlocked,
		&account.IsAdmin,
		&account.LastAuthDate,
		&account.LastLoginAt,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err == nil {
		return account, false, nil
	}
	if err == pgx.ErrNoRows {
		return service.TelegramAccount{}, true, nil
	}
	return service.TelegramAccount{}, false, err
}

func (r *TelegramAccountRepository) GetTelegramAccountByUserID(ctx context.Context, userID int64) (service.TelegramAccount, bool, error) {
	const q = `
		SELECT u.id, t.telegram_id, t.username, t.first_name, t.last_name, t.photo_url, t.is_blocked, t.is_admin, t.last_auth_date, t.last_login_at, t.created_at, t.updated_at
		FROM telegram_accounts t
		INNER JOIN users u ON u.telegram_id = t.telegram_id
		WHERE u.id = $1
	`

	var account service.TelegramAccount
	err := r.pool.QueryRow(ctx, q, userID).Scan(
		&account.UserID,
		&account.TelegramID,
		&account.Username,
		&account.FirstName,
		&account.LastName,
		&account.PhotoURL,
		&account.IsBlocked,
		&account.IsAdmin,
		&account.LastAuthDate,
		&account.LastLoginAt,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err == nil {
		return account, true, nil
	}
	if err == pgx.ErrNoRows {
		return service.TelegramAccount{}, false, nil
	}
	return service.TelegramAccount{}, false, err
}
