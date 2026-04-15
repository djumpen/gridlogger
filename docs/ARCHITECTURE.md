# Architecture

## Data flow

1. Device sends `POST /api/projects/{projectId}/ping` every ~30s.
2. Backend records arrival timestamp in `pings(project_id, ts)`.
3. Frontend landing (`/`) loads public project catalog via `GET /api/projects`.
4. Frontend project page (`/{slug}`) resolves project via `GET /api/project-slugs/{slug}`.
5. Owner settings page (`/a/settings`) loads user projects via `GET /api/settings`.
6. Owner can create/edit projects via `/api/settings/projects` and `/api/settings/projects/{projectId}`.
7. Owner project settings include a Telegram bot tab that can link a Telegram group to project notifications.
8. Backend resolves Telegram group chat IDs from bot `getUpdates`, stores them as project-owned virtual users, and subscribes them like regular Telegram users.
9. Frontend requests availability for a window.
10. Backend computes interval status (`available` or `outage`) at 30s steps and merges adjacent segments.
11. Backend returns intervals + stats for the same window.

## Telegram auth flow

1. Frontend loads `GET /api/auth/telegram/config`.
2. Telegram Login Widget returns signed payload in browser.
3. Frontend posts payload to `POST /api/auth/telegram/callback`.
4. Backend validates hash per Telegram spec and checks `auth_date` TTL.
5. Backend upserts `telegram_accounts`, links/creates internal `users` row, and rejects replay based on `last_auth_date`.
6. Backend issues HS256 JWT and sets `HttpOnly` session cookie.
7. Frontend reads user via `GET /api/me`.

## Telegram group notification flow

1. Owner opens project settings and chooses `Телеграм бот`.
2. Owner adds the bot to a Telegram group, then submits the exact group title.
3. Backend calls `getUpdates`, scans group/supergroup chats from recent updates, and matches by title.
4. When a match is found, backend upserts a virtual `users` row owned by the real owner and links it to the Telegram chat identity.
5. Backend creates/activates a `project_notification_subscriptions` row for that virtual user and project.
6. Notification dispatcher sends the same project status messages to both normal Telegram users and linked group chats.

## Why compute in backend code

Given small scale and simple rules, interval derivation in Go is easier to evolve than SQL-heavy interval stitching.
This can move to continuous aggregates later if needed.

## Outage rule

- Configurable threshold (`OUTAGE_THRESHOLD_SECONDS`, default 240).
- For a time point `t`, status is `available` if at least one ping exists in `[t-threshold, t]`, otherwise `outage`.
- Ping endpoint reads optional `X-Project-Secret` and currently logs warning on mismatch without rejecting.

## Time handling

- Storage in UTC.
- API window inputs in RFC3339.
- Frontend currently presents with `Europe/Kyiv` orientation.
