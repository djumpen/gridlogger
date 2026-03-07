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
- Required header: `X-Project-Secret`.
- Missing/wrong secret returns `401 Unauthorized`.
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

- `GET /api/projects` (list for landing page, only projects with `is_public=true`)
- `GET /api/project-slugs/{slug}` (project lookup for `/{slug}` page)
- `GET /api/settings` (owner project list for `/a/settings`)
- `POST /api/settings/projects` (create project)
- `GET /api/settings/projects/{projectId}` (owner project details)
- `POST /api/settings/projects/{projectId}` (update owner project)
- `GET /api/settings/projects/{projectId}/telegram-bot/groups` (list linked Telegram groups)
- `POST /api/settings/projects/{projectId}/telegram-bot/groups` (find Telegram group and subscribe it)
- `DELETE /api/settings/projects/{projectId}/telegram-bot/groups/{virtualUserId}` (unlink Telegram group)
- `POST /api/settings/projects/{projectId}/firmware/jobs` (start ESP32-C3 firmware build)
- `GET /api/settings/projects/{projectId}/firmware/jobs/{jobId}` (poll firmware build status)
- `GET /api/settings/projects/{projectId}/firmware/jobs/{jobId}/manifest.json?token=...` (ESP Web Tools manifest)
- `GET /api/settings/projects/{projectId}/firmware/jobs/{jobId}/files/{fileName}?token=...` (artifact binary)
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
curl -i -X POST http://localhost:8080/api/projects/1/ping \
  -H 'X-Project-Secret: <project-secret>'
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
- `FIRMWARE_BUILD_ENABLED` (default `true`)
- `FIRMWARE_SERVICE_URL` (default `http://firmware:8081`)
- `FIRMWARE_SERVICE_TOKEN` (optional shared token between backend and firmware service)
- `FIRMWARE_SERVICE_TIMEOUT_SECONDS` (default `30`)
- `FIRMWARE_PING_BASE_URL` (default `https://svitlo.homes`)

Firmware service variables:
- `FIRMWARE_ARDUINO_CLI_PATH` (default `arduino-cli`)
- `FIRMWARE_BOARD_FQBN` (default `esp32:esp32:esp32c3`)
- `FIRMWARE_TEMPLATE_DIR` (default `firmware/esp32-c3`, Docker uses `/home/app/firmware/esp32-c3`)
- `FIRMWARE_WORK_DIR` (default `/tmp/gridlogger-firmware`)
- `FIRMWARE_BUILD_TIMEOUT_SECONDS` (default `300`)
- `FIRMWARE_JOB_TTL_SECONDS` (default `7200`)

Frontend assumptions:
- Timezone baseline for display is `Europe/Kyiv`.
- `/` renders landing page with project list.
- `/{slug}` renders selected project dashboard.
- `/a/settings` renders owner settings list + create flow.
- `/a/settings/project/{id}` renders project settings/integration tabs.

## Releases and rollback

- Release version is derived from Conventional Commit messages since the latest `vX.Y.Z` git tag.
- `feat:` bumps minor, `type(scope)!:` or `BREAKING CHANGE:` bumps major, everything else bumps patch.
- CI publishes Docker images with three tags: semantic version (`1.4.2`), commit SHA, and `latest`.
- OCI image labels include version, revision, and build date.

Useful commands:

```bash
make next-version
make rollout-version VERSION=1.4.2
make rollout-version VERSION=1.4.2 WITH_FIRMWARE=true
```

Migration policy:
- App rollback is supported by redeploying an older image tag.
- Database migrations are intentionally forward-only and are not rolled back automatically.
- Keep schema changes additive/backward-compatible and use expand-contract migrations when removing or renaming data structures.

## CI/CD

GitHub Actions:
- PR checks: `.github/workflows/ci.yml`
- Main branch CI/CD: `.github/workflows/ci-cd.yaml`

Main branch pipeline runs tests, builds/pushes backend/firmware/frontend images to GHCR (`sha` + `latest`), then deploys to k3s using `KUBECONFIG` secret.

See detailed setup and troubleshooting in `docs/ci-cd-gh-actions.md`.

Secrets management via Infisical is documented in `docs/INFISICAL.md`.

## Project layout

- `cmd/server`: API entrypoint
- `cmd/firmware`: firmware build service entrypoint
- `internal/httpapi`: HTTP handlers
- `internal/firmwareapi`: firmware service HTTP handlers
- `internal/service`: outage/interval logic
- `internal/db`: PostgreSQL/Timescale access
- `frontend`: Vue app
- `k8s`: manifests
- `docs`: architecture/context docs for future Codex enhancements
