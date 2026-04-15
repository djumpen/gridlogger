# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project overview

GridLogger is a small IoT power-monitoring service. IoT devices ping the backend every 30 seconds; the backend stores timestamps in TimescaleDB and the frontend visualizes availability/outage intervals. See `docs/ARCHITECTURE.md` for detailed data flows and `docs/API.md` for endpoint specs.

## Commands

### Backend (Go)

```bash
go test ./...              # run all tests
go test ./internal/...     # run tests in a specific package subtree
make run                   # docker compose up --build (full local stack)
make migrate-local         # run DB migrations against local Postgres
```

### Frontend (Vue 3 + Vite)

```bash
cd frontend && npm run dev      # dev server (port 5173)
cd frontend && npm run build    # production build to frontend/dist/
```

### Versioning & deployment

```bash
make next-version                            # calculate next semver from git commits
make rollout-version VERSION=X.Y.Z           # deploy image version to k3s
make rollout-version VERSION=X.Y.Z WITH_FIRMWARE=true  # include firmware service
```

CI runs `go test ./...` and `npm run build` on every PR, then builds/pushes Docker images and deploys to k3s on merge to `main`.

## Architecture

### Services

| Service | Entry point | Port | Purpose |
|---|---|---|---|
| Backend API | `cmd/server/main.go` | 8080 | REST API, session management, Telegram auth, notifications |
| Firmware service | `cmd/firmware/main.go` | 8081 | Arduino CLI wrapper, compiles ESP32-C3 firmware on demand |
| Frontend | `frontend/` | 5173 | Vue 3 SPA (three views: landing, project dashboard, settings) |

### Backend layers

```
cmd/server/main.go
  → internal/config/      environment-driven config with validation
  → internal/db/          repository layer (pgx v5 direct queries, no ORM)
  → internal/service/     business logic (availability calc, notifications, auth)
  → internal/httpapi/     HTTP handlers (stdlib net/http, no framework)
```

Services are constructed in `main.go` with explicit dependency injection. There is no DI container.

### Key data flow

1. Device → `POST /api/projects/{projectId}/ping` (unauthenticated, secret-based)
2. Backend writes timestamp to `pings` hypertable in TimescaleDB
3. Frontend → `GET /api/projects/{projectId}/availability?from=&to=`
4. Backend computes intervals in-memory at 30-second granularity; outage threshold is configurable via env var (default 240s)
5. Telegram notification polling loop (goroutine, 5s interval) dispatches alerts on status changes

### Database

PostgreSQL 16 + TimescaleDB. Migrations in `migrations/` are numbered sequentially and applied forward-only. Prefer additive, backward-compatible changes. Key tables: `pings` (hypertable on `ts`), `projects`, `users`, `telegram_accounts`, `project_notification_subscriptions`, `project_status_state`, `dtek_groups`.

### Authentication

Telegram Login Widget validates signed payload in `internal/service/telegram_auth.go`, then issues an HS256 JWT stored as a session cookie. Virtual users are created for Telegram group notifications.

### Frontend timezone

All timestamps stored as UTC; frontend displays in `Europe/Kyiv` (hardcoded).

## Engineering guardrails

- Keep REST API stable — change `docs/API.md` and frontend together.
- Keep outage threshold configurable via environment variable, not hardcoded.
- Keep project ID in URL path as the primary routing key.
- Prefer simple, direct solutions over premature abstraction (this is a small-scale project).
- DB migrations must be additive and backward-compatible.
- Visual style: warm light background, bold green/red availability signals, rounded cards, mobile-responsive. Do not introduce an unstyled enterprise look.
