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

func (r *TelegramAccountRepository) GetTelegramAccountByID(ctx context.Context, telegramID int64) (service.TelegramAccount, bool, error) {
	const q = `
		SELECT telegram_id, username, first_name, last_name, photo_url, is_blocked, is_admin, last_auth_date, last_login_at, created_at, updated_at
		FROM telegram_accounts
		WHERE telegram_id = $1
	`

	var account service.TelegramAccount
	err := r.pool.QueryRow(ctx, q, telegramID).Scan(
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
