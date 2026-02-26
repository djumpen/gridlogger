# CI/CD with GitHub Actions + GHCR + k3s

This document describes the GitHub Actions pipeline in `.github/workflows/ci-cd.yaml`.

## What this pipeline does

On every push to `main` (including merge commits), it runs:

1. `test`
   - Go tests (`go test ./...`)
   - Frontend build (`npm ci && npm run build`)
2. `build-and-push`
  - Builds backend image from `Dockerfile.backend`
  - Builds frontend image from `Dockerfile.frontend`
  - Pushes both to GHCR with tags:
    - `ghcr.io/djumpen/gridlogger-backend:<sha>` and `:latest`
    - `ghcr.io/djumpen/gridlogger-frontend:<sha>` and `:latest`
3. `deploy`
   - Applies manifests: `kubectl apply -k k8s/overlays/prod`
  - Updates backend and frontend deployment images to SHA tags
  - Waits for rollout to finish for both deployments

The deploy job is gated to `main` only.

---

## Required GitHub settings

### 1) Workflow permissions

In repository settings:

- **Settings → Actions → General → Workflow permissions**
- Set to **Read and write permissions** (required for `GITHUB_TOKEN` to push packages)
- Save

Also confirm Actions are enabled for this repository.

### 2) Package visibility

Ensure GHCR package visibility/access is appropriate for your cluster pull model:

- Private package: cluster needs image pull secret
- Public package: no pull secret needed

---

## Required repository secrets

Add these in **Settings → Secrets and variables → Actions**:

- `KUBECONFIG` (required)
  - Full kubeconfig file content (plain text, multi-line)
  - Must have permissions to `apply`, `set image`, and read rollout status in target namespace
- `K8S_NAMESPACE` (optional)
  - Defaults to `gridlogger` if not set

### Kubeconfig setup example

From your k3s server:

```bash
sudo cat /etc/rancher/k3s/k3s.yaml
```

- Replace `server:` with reachable external URL/IP if needed
- Paste full content into `KUBECONFIG` secret

---

## How image tag is updated in Kubernetes

The pipeline applies manifests first, then updates deployment image explicitly:

```bash
kubectl -n "$K8S_NAMESPACE" set image deployment/backend \
  backend="ghcr.io/djumpen/gridlogger-backend:${GITHUB_SHA}"

kubectl -n "$K8S_NAMESPACE" set image deployment/frontend \
  frontend="ghcr.io/djumpen/gridlogger-frontend:${GITHUB_SHA}"
```

Then waits for successful rollout:

```bash
kubectl -n "$K8S_NAMESPACE" rollout status deployment/backend --timeout=180s
kubectl -n "$K8S_NAMESPACE" rollout status deployment/frontend --timeout=180s
```

### Manifest example (backend deployment)

`k8s/base/backend.yaml` uses container name `backend`, so `set image` targets it directly.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend
spec:
  template:
    spec:
      containers:
        - name: backend
          image: ghcr.io/djumpen/gridlogger-backend:latest
```

At deploy time, Actions replaces backend/frontend images with SHA tags from current commit.

---

## Local run/debug guide

### Validate app tests/build locally

```bash
go test ./...
cd frontend && npm ci && npm run build
```

### Validate image build locally

```bash
docker build -f Dockerfile.backend -t ghcr.io/djumpen/gridlogger-backend:local .
docker build -f Dockerfile.frontend -t ghcr.io/djumpen/gridlogger-frontend:local .
```

### Validate k8s deploy steps locally

```bash
kubectl apply -k k8s/overlays/prod
kubectl -n gridlogger set image deployment/backend backend=ghcr.io/djumpen/gridlogger-backend:<sha>
kubectl -n gridlogger set image deployment/frontend frontend=ghcr.io/djumpen/gridlogger-frontend:<sha>
kubectl -n gridlogger rollout status deployment/backend --timeout=180s
kubectl -n gridlogger rollout status deployment/frontend --timeout=180s
```

### Optional: run GitHub Actions locally with `act`

```bash
act push -j test
```

(For full deploy emulation with `act`, provide secrets and a reachable kubeconfig.)

---

## Troubleshooting

### `denied: permission_denied` when pushing to GHCR

- Check repository **Workflow permissions** is Read/Write
- Ensure workflow has `permissions: packages: write`
- Ensure package namespace matches `ghcr.io/djumpen/gridlogger-backend` and `ghcr.io/djumpen/gridlogger-frontend`

### `kubectl` auth errors (`Unauthorized`, cert errors)

- Verify `KUBECONFIG` secret content is valid and complete
- Confirm API server endpoint in kubeconfig is reachable from GitHub-hosted runners
- Check user/service-account RBAC permissions in target namespace

### Rollout timeout

- Check deployment events:
  ```bash
  kubectl -n gridlogger describe deployment backend
  kubectl -n gridlogger get pods
  kubectl -n gridlogger logs deploy/backend --tail=100
  ```
- Common causes:
  - bad image tag
  - image pull permissions for private GHCR package
  - failing readiness/liveness probes

### Frontend rollout does not pick new image

- Verify frontend image exists in GHCR with the current SHA tag
- Check rollout status and pod events:
  ```bash
  kubectl -n gridlogger rollout status deployment/frontend --timeout=180s
  kubectl -n gridlogger describe deployment frontend
  kubectl -n gridlogger get pods
  ```

---

## Notes

- `.github/workflows/ci.yml` is kept as PR-only checks to avoid duplicate deployment runs.
- `ci-cd.yaml` includes concurrency control to cancel in-progress runs on the same branch.
