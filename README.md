# GridLogger

GridLogger tracks power availability from IoT device pings (every ~30s) and visualizes outage/availability intervals in a Vue calendar-style UI.

## What is implemented

- Go (std `net/http`) REST API.
- TimescaleDB storage (`pings` hypertable).
- Outage logic: outage if no ping for `OUTAGE_THRESHOLD_SECONDS` (default 120s).
- Vue frontend with fixed views: day/week/month (week default), interval highlighting, and window stats.
- Single-tenant, trusted/open ping endpoint.
- Kubernetes manifests (k3s-ready) + `make deploy`.
- Local development via Docker Compose.

## API

### Record ping

`POST /api/projects/{projectId}/ping`

- No body.
- Uses arrival timestamp on server.
- Optional header: `X-Project-Secret`.
- Missing/wrong secret currently logs warning only (request is still accepted).
- `204 No Content` on success.

### Get availability in window

`GET /api/projects/{projectId}/availability?from=<RFC3339>&to=<RFC3339>`

Response includes:
- `intervals`: merged intervals with `status = available|outage`
- `stats`: `availabilityPercent`, `totalAvailableHours`, `totalOutageHours` (1 decimal)

### Health

- `GET /healthz`
- `GET /readyz`

### Projects catalog

- `GET /api/projects` (list for landing page)
- `GET /api/project-slugs/{slug}` (project lookup for `/{slug}` page)
- `GET /api/settings` (owner project list for `/a/settings`)
- `POST /api/settings/projects` (create project)
- `GET /api/settings/projects/{projectId}` (owner project details)
- `POST /api/settings/projects/{projectId}` (update owner project)
- `GET /api/projects/{projectId}/notifications/subscription`
- `POST /api/projects/{projectId}/notifications/subscription`

### Telegram auth

- `GET /api/auth/telegram/config`
- `POST /api/auth/telegram/callback`
- `GET /api/me`
- `POST /api/auth/logout`

Setup and security details: `docs/telegram-auth.md`

## Local run

Prerequisites:
- Docker + Docker Compose

Run:

```bash
make run
```

Open:
- Frontend: `http://localhost:5173`
- Backend API: `http://localhost:8080`

Send test ping:

```bash
curl -i -X POST http://localhost:8080/api/projects/1/ping
```

## k3s one-time setup notes

Assumes a fresh k3s server with default Traefik ingress.

1. Install k3s on server (if not already done):

```bash
curl -sfL https://get.k3s.io | sh -
```

2. Copy kubeconfig locally and point `server:` to your host/IP:

```bash
sudo cat /etc/rancher/k3s/k3s.yaml
```

3. Ensure your local `kubectl` can reach cluster:

```bash
kubectl get nodes
```

4. Push backend/frontend images to your registry and update image names in:
- `k8s/base/backend.yaml`
- `k8s/base/frontend.yaml`
- `k8s/overlays/prod/kustomization.yaml`

5. Deploy:

```bash
make deploy
```

6. Add DNS/hosts entry for ingress host (default `gridlogger.local`) to your server IP.

## Configuration

Environment variables:

- `DATABASE_URL` (required)
- `LISTEN_ADDR` (default `:8080`)
- `OUTAGE_THRESHOLD_SECONDS` (default `120`)
- `DEFAULT_PROJECT_ID` (default `1`)
- `TELEGRAM_BOT_TOKEN` (required to enable Telegram login)
- `TELEGRAM_BOT_USERNAME` (required to enable Telegram login)
- `TELEGRAM_AUTH_TTL_SECONDS` (default `86400`)
- `JWT_SECRET` (required to enable Telegram login, min 32 chars)
- `JWT_ISSUER` (default `gridlogger`)
- `SESSION_TTL_SECONDS` (default `604800`)
- `SESSION_COOKIE_NAME` (default `gridlogger_session`)
- `SESSION_COOKIE_SECURE` (default `false`, set `true` in production HTTPS)
- `NOTIFICATIONS_ENABLED` (default `true`)
- `NOTIFICATION_POLL_SECONDS` (default `30`)

Frontend assumptions:
- Timezone baseline for display is `Europe/Kyiv`.
- `/` renders landing page with project list.
- `/{slug}` renders selected project dashboard.
- `/a/settings` renders owner settings list + create flow.
- `/a/settings/project/{id}` renders project settings/integration tabs.

## CI/CD

GitHub Actions:
- PR checks: `.github/workflows/ci.yml`
- Main branch CI/CD: `.github/workflows/ci-cd.yaml`

Main branch pipeline runs tests, builds/pushes backend image to GHCR (`sha` + `latest`), then deploys to k3s using `KUBECONFIG` secret.

See detailed setup and troubleshooting in `docs/ci-cd-gh-actions.md`.

Secrets management via Infisical is documented in `docs/INFISICAL.md`.

## Project layout

- `cmd/server`: API entrypoint
- `internal/httpapi`: HTTP handlers
- `internal/service`: outage/interval logic
- `internal/db`: PostgreSQL/Timescale access
- `frontend`: Vue app
- `k8s`: manifests
- `docs`: architecture/context docs for future Codex enhancements
