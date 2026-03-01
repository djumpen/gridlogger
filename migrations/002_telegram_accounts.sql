CREATE TABLE IF NOT EXISTS telegram_accounts (
    telegram_id BIGINT PRIMARY KEY,
    username TEXT NOT NULL DEFAULT '',
    first_name TEXT NOT NULL DEFAULT '',
    last_name TEXT NOT NULL DEFAULT '',
    photo_url TEXT NOT NULL DEFAULT '',
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    last_auth_date BIGINT NOT NULL DEFAULT 0,
    last_login_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_telegram_accounts_username ON telegram_accounts(username);
CREATE INDEX IF NOT EXISTS idx_telegram_accounts_last_login_at ON telegram_accounts(last_login_at DESC);
