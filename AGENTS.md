# AGENTS.md

## Project context

GridLogger is a small single-tenant IoT power-monitoring service.

Core product behavior:
- IoT devices ping backend every 30 seconds.
- Endpoint is open and trusted.
- Server stores ping arrival timestamps in TimescaleDB.
- Outage means no ping for 4 minutes by default (configurable).
- UI shows day/week/month windows and highlights availability/outage intervals.
- Stats are calculated for the visible window only.

## Technology choices

- Backend: Go, standard library HTTP.
- Frontend: Vue 3 + Vite.
- DB: TimescaleDB.
- Infra: k3s-compatible Kubernetes manifests.
- Local dev: Docker Compose.

## Design and styling guardrails

- Keep the current visual direction: warm light background, bold green/red availability signals, rounded cards, readable contrast.
- Avoid default unstyled enterprise look.
- Preserve mobile support.

## Engineering guardrails

- Keep REST API stable unless changing docs and frontend together.
- Keep outage threshold logic configurable via environment variable.
- Keep project ID in URL path as first-class routing key.
- Preserve simple operations over premature optimization (project is small scale).
- Prefer additive migrations and backward-compatible changes.

## Useful docs

- `/Users/dmytro.semenchuk/projects/iot/gridlogger/docs/ARCHITECTURE.md`
- `/Users/dmytro.semenchuk/projects/iot/gridlogger/docs/API.md`
- `/Users/dmytro.semenchuk/projects/iot/gridlogger/docs/ROADMAP.md`

# Software factory rules

Rules for every agent working in this repository (Claude Code, Codex or any other). Existing sections above stay in force; where they conflict with the rules below, the existing section wins.

## Product

GridLogger shows whether the lights are on at a given address, based on pings from a small IoT device. People in Ukraine use it to check power status. The 3 jobs that matter:
1. See the current status and the outage history (day/week/month) of an address.
2. Get a Telegram message when the status changes.
3. Add your own address and flash a device with firmware that pings the service.

## Setup & commands

- Install: `cd frontend && npm ci` (Go 1.26 for the backend)
- Run the app (UI without Docker or a database): `make dev` — starts a demo API with sample data and the Vite dev server
- Run the full stack with the real backend and TimescaleDB: `make run` (needs Docker)
- Open it: http://localhost:5173 (landing), http://localhost:5173/sonyachna-12 (demo project dashboard)
- Unit tests: `go test ./...`
- Smoke test: `make smoke` (starts the demo stack, checks the page and API respond)
- Lint / types: `go vet ./...` (the frontend is untyped JavaScript and has no linter)
- Everything CI runs: `make ci`
- Needed services: none for `make dev`. The real backend needs TimescaleDB (PostgreSQL 16); `make run` starts it.
- Demo data: `scripts/mock-api.mjs` serves fixed sample projects. `MOCK_EMPTY=1 make dev` shows the empty-list state. Never commit real credentials.
- The demo API covers only the public read pages (landing, dashboard). Login, settings and notifications return "unauthorized" in demo mode and need the real stack.
- Agent working folder: .scratch/ (plans, logs, screenshots; gitignored)
- Spend budget per task (paid APIs, cloud): $0

## Architecture

- `cmd/server`, `internal/{config,db,service,httpapi}`: Go REST API on TimescaleDB (details in CLAUDE.md).
- `cmd/firmware`, `internal/firmwareapi`, `firmware/`: ESP32-C3 firmware build service.
- `frontend/`: Vue 3 + Vite single-page app; calls `/api/*`, which Vite proxies to the backend (`API_PROXY_TARGET`).
- `scripts/`: migrations, versioning, and the demo API / smoke test.

## Rules

Start small. Build step by step.
1. Goal = product, not green checks. Done means a user can do the thing. Write the product goal first in every plan and PR.
2. Failed? Change the approach. Never repeat an attempt that changed nothing. After two failures with the same approach, find the real cause or try a different route.
3. No invented flags or fallbacks. One direct behavior. No new feature flags, legacy and new paths side by side, silent fallbacks or extra retries unless a human asked.
4. Fix the root cause. A patch that hides the symptom is not a fix.
5. One GitHub round per PR. Find bugs locally; CI confirms finished work, it is not the tester.
6. Big file? Split it first. Over ~800 lines consider extracting; over ~1500 extract the part you touch first; over ~2000 leave it smaller than you found it.
7. Reports fit on a phone: outcome first, plain words, screenshots.

Also: write in the plan what NOT to build. Test like a user, not like the diff. New pure logic gets a focused test.

## Questions

Ask a human when information is missing that changes the result. Ask everything at once, with a suggested answer each. Decide reversible details yourself and note them.

## Stop rules

Work on your own: build, test, fix, open branches and draft PRs. Stop and ask a human before:
- merging any PR
- touching production (deploys, prod data, prod config)
- adding a new dependency (test tools the task asked for are fine)
- changing a public API, schema or contract
- anything with secrets, credentials or user data
- deleting anything you did not create in this task
- spending beyond the budget above

Never print, commit or screenshot tokens, cookies or personal data.
Blocked is a state, not a reason to stop: report what is blocked, keep going on the rest.

## Local review gate

Before any push: run the full CI command locally, then the code-review skill (3 fresh-context reviewers, 2+ models when available, 3 approvals on the same commit, max 10 fix rounds; open security findings keep the PR a draft). Push once.

## Verification

Run the real product, not only tests: click through the changed flow in a browser, check the console, reload, check desktop and phone width, screenshot before and after. Up to 10 fix rounds. Never call work verified if you did not see it work.

## Skills

Skills live in `.agents/skills/` (`.claude/skills` is a symlink to it): cto-office (plan), cto-exec (run a PR train), ship-loop (one task to a PR), code-review (local review gate), verify (prove it works), investigate (bugs), deep-qa (whole-product QA), garden-codebase (cleanup), pr-body (PR text). No screenshot, no review.
