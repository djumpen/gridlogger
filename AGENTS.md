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
