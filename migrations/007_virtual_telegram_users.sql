ALTER TABLE telegram_accounts
    ADD COLUMN IF NOT EXISTS chat_type TEXT NOT NULL DEFAULT 'private',
    ADD COLUMN IF NOT EXISTS chat_title TEXT NOT NULL DEFAULT '';

UPDATE telegram_accounts
SET chat_type = 'private'
WHERE chat_type IS NULL OR chat_type = '';

UPDATE telegram_accounts
SET chat_title = ''
WHERE chat_title IS NULL;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_virtual BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS owner_id BIGINT NULL REFERENCES users(id) ON DELETE CASCADE;

UPDATE users
SET is_virtual = FALSE
WHERE is_virtual IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_owner_id ON users(owner_id);
