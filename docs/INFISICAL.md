# Infisical setup for GridLogger

This setup keeps local development manual and uses Infisical as the production source of truth for secrets.

Infisical resources are split:

- Project-specific `InfisicalSecret` stays in this repo:
  - `k8s/infisical/infisicalsecret-gridlogger-prod.yaml`
  - `k8s/infisical/kustomization.yaml`
- Shared/service-account bootstrap can be managed centrally in:
  - `/Users/dmytro.semenchuk/projects/mylab/k8s/infisical/gridlogger`

## Desired flow

1. Update/add secrets in Infisical UI.
2. Deploy app (`make deploy`).
3. App picks up values from Kubernetes env via `gridlogger-secrets`.

`backend` and `timescaledb` already consume Kubernetes secrets via `secretKeyRef`.

## Current manifest location

- `k8s/infisical/infisicalsecret-gridlogger-prod.yaml`
- `k8s/infisical/kustomization.yaml`
- Optional centralized companions:
  - `/Users/dmytro.semenchuk/projects/mylab/k8s/infisical/gridlogger/serviceaccount.yaml`
  - `/Users/dmytro.semenchuk/projects/mylab/k8s/infisical/gridlogger/serviceaccount-token.yaml`
- `secrets.infisical.com/auto-reload: "true"` annotation on backend deployment

## One-time cluster setup

1. Install Infisical operator:

```bash
helm repo add infisical-helm-charts https://dl.cloudsmith.io/public/infisical/helm-charts/helm/charts/
helm repo update
helm upgrade --install infisical-secrets-operator infisical-helm-charts/secrets-operator \
  --namespace infisical-secrets \
  --create-namespace
```

2. Edit `k8s/infisical/infisicalsecret-gridlogger-prod.yaml` and replace:

- `REPLACE_WITH_MACHINE_IDENTITY_ID`
- `REPLACE_WITH_PROJECT_SLUG`
- `envSlug` if not `prod`
- `secretsPath` if not `/`

3. Apply Infisical resources:

```bash
kubectl apply -k k8s/infisical
```

4. Verify sync:

```bash
kubectl describe infisicalsecret -n gridlogger gridlogger-prod-secrets
kubectl get secret -n gridlogger gridlogger-secrets -o yaml
```

## Troubleshooting: "service account token has expired"

If Infisical logs show `invalid bearer token, service account token has expired`, make sure:

- `k8s/infisical/infisicalsecret-gridlogger-prod.yaml` has `autoCreateServiceAccountToken: false`
- serviceaccount token secret is applied (from local or centralized manifests)
- Re-apply `k8s/infisical` and wait for `InfisicalSecret` status to become ready

## Required Infisical keys

Store these in your Infisical project/environment/path:

- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `POSTGRES_DB`
- `DATABASE_URL`

## Deploy flow

Deploy flow:

1. Apply optional centralized serviceaccount/token manifests (if you manage them in `mylab`).
2. Apply project `InfisicalSecret` (`k8s/infisical`).
3. Apply app manifests (`k8s/overlays/prod`) from this repo.

## Local development

Local dev stays manual as requested (`docker-compose.yml` values and/or local env files).

## Managing multiple apps/environments in one cluster

Pattern:

- one `InfisicalSecret` resource per `{app, environment}`
- one target Kubernetes secret per app/env (for example: `payments-prod-secrets`, `gridlogger-staging-secrets`)
- one namespace-scoped service account per app/env if you want stronger isolation

You can copy `k8s/infisical/infisicalsecret-gridlogger-prod.yaml` and create additional resources, changing:

- `metadata.name`
- `authentication.kubernetesAuth.secretsScope.projectSlug`
- `authentication.kubernetesAuth.secretsScope.envSlug`
- `managedKubeSecretReferences[].secretName`
- `managedKubeSecretReferences[].secretNamespace`

Then point each app deployment to its own Kubernetes secret via `secretKeyRef`.
