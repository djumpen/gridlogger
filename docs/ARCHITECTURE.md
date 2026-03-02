# Architecture

## Data flow

1. Device sends `POST /api/projects/{projectId}/ping` every ~30s.
2. Backend records arrival timestamp in `pings(project_id, ts)`.
3. Frontend landing (`/`) loads public project catalog via `GET /api/projects`.
4. Frontend project page (`/{slug}`) resolves project via `GET /api/project-slugs/{slug}`.
5. Frontend requests availability for a window.
6. Backend computes interval status (`available` or `outage`) at 30s steps and merges adjacent segments.
7. Backend returns intervals + stats for the same window.

## Telegram auth flow

1. Frontend loads `GET /api/auth/telegram/config`.
2. Telegram Login Widget returns signed payload in browser.
3. Frontend posts payload to `POST /api/auth/telegram/callback`.
4. Backend validates hash per Telegram spec and checks `auth_date` TTL.
5. Backend upserts `telegram_accounts`, links/creates internal `users` row, and rejects replay based on `last_auth_date`.
6. Backend issues HS256 JWT and sets `HttpOnly` session cookie.
7. Frontend reads user via `GET /api/me`.

## Why compute in backend code

Given small scale and simple rules, interval derivation in Go is easier to evolve than SQL-heavy interval stitching.
This can move to continuous aggregates later if needed.

## Outage rule

- Configurable threshold (`OUTAGE_THRESHOLD_SECONDS`, default 120).
- For a time point `t`, status is `available` if at least one ping exists in `[t-threshold, t]`, otherwise `outage`.

## Time handling

- Storage in UTC.
- API window inputs in RFC3339.
- Frontend currently presents with `Europe/Kyiv` orientation.
