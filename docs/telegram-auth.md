# Telegram Login Setup

This project uses the official **Login Widget** flow from Telegram and performs server-side signature verification.

Official references:
- [Telegram Login Widget docs](https://core.telegram.org/widgets/login)
- [Checking authorization (hash verification)](https://core.telegram.org/widgets/login#checking-authorization)
- [Telegram Bot API sendMessage](https://core.telegram.org/bots/api#sendmessage)

## 1) Create and configure bot

1. Open BotFather in Telegram.
2. Run `/newbot` and create your bot.
3. Save the bot token (example format: `123456789:AA...`).
4. Set the domain for Login Widget via BotFather `/setdomain`.
   - For production, use your real public domain (example: `svitlo.homes`).
   - Telegram login widget requires a proper domain setup and HTTPS in production.

## 2) Required environment variables

Set all variables below to enable Telegram auth:

- `TELEGRAM_BOT_TOKEN` - BotFather token.
- `TELEGRAM_BOT_USERNAME` - bot username without `@`.
- `TELEGRAM_AUTH_TTL_SECONDS` - max age of `auth_date` (default `86400`, one day).
- `JWT_SECRET` - HMAC secret for session JWT (minimum 32 chars).
- `JWT_ISSUER` - JWT issuer (default `gridlogger`).
- `SESSION_TTL_SECONDS` - JWT/cookie lifetime (default `604800`, 7 days).
- `SESSION_COOKIE_NAME` - cookie name (default `gridlogger_session`).
- `SESSION_COOKIE_SECURE` - `true` in production HTTPS, `false` for local HTTP.

If only part of Telegram/JWT env vars are set, backend startup fails intentionally.

## 3) Local development

1. Add values to `.env` (or export shell vars).
2. Start stack:

```bash
make run
```

3. Open `http://localhost:5173`.
4. Use Telegram login widget in top bar.
5. Verify session endpoint:

```bash
curl -i http://localhost:8080/me
```

Notes:
- If Telegram login is not configured, frontend shows “Вхід через Telegram недоступний”.
- Localhost Telegram widget behavior depends on your BotFather domain setup.

## 4) Security checks implemented

Backend endpoint `POST /api/auth/telegram/callback` performs:

- Hash validation per Telegram spec (`HMAC-SHA256` with `sha256(bot_token)` secret).
- Data-check-string generation using all received fields except `hash`, sorted alphabetically.
- `auth_date` freshness validation (`TELEGRAM_AUTH_TTL_SECONDS`).
- Future timestamp guard (small skew tolerance).
- Replay prevention using `last_auth_date` per Telegram user.

## 5) Stored user model

Tables:

- `users`
  - `id` (internal primary key used in sessions)
  - `telegram_id` (unique, nullable, references `telegram_accounts.telegram_id`)
  - `is_virtual` (`true` only for project-owned Telegram group subscribers)
  - `owner_id` (real user that owns a virtual subscriber)
  - `created_at`
  - `updated_at`
- `telegram_accounts`
  - `telegram_id` (Telegram identity, primary key)
  - `username`
  - `first_name`
  - `last_name`
  - `photo_url`
  - `last_auth_date`
  - `last_login_at`
  - `created_at`
  - `updated_at`
  - `is_blocked` (future admin control)
  - `is_admin` (future admin control)
  - `chat_type` (`private` for real users, `group`/`supergroup` for virtual group chats)
  - `chat_title` (Telegram group title for virtual group chats)

Virtual Telegram groups are stored as ordinary Telegram identities plus `users.is_virtual=true`, which lets the existing notification subscription flow reuse the same `project_notification_subscriptions` table and dispatcher.

## 6) API summary

- `GET /api/auth/telegram/config` - frontend widget config.
- `POST /api/auth/telegram/callback` - verify payload, create/update user, issue session.
- `GET /api/me` - current logged-in user from cookie or Bearer token.
- `POST /api/auth/logout` - clear session cookie.

## 7) Common pitfalls

- **Hash mismatch**: wrong bot token or wrong data-check-string rules.
- **Stale login**: `auth_date` older than configured TTL.
- **Time drift**: server clock out of sync can reject valid login.
- **HTTP vs HTTPS**: production Telegram login should run over HTTPS.
- **Domain mismatch**: BotFather `/setdomain` does not match current site origin.
- **Partial env setup**: setting only one of Telegram/JWT vars disables startup.
