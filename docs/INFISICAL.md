# Infisical setup for GridLogger

This setup keeps local development manual and uses Infisical as the production source of truth for secrets.

## Desired flow

1. Update/add secrets in Infisical UI.
2. Deploy app (`make deploy`).
3. App picks up values from Kubernetes env via `gridlogger-secrets`.

`backend` and `timescaledb` already consume Kubernetes secrets via `secretKeyRef`.

## What was added

- `k8s/infisical/serviceaccount.yaml`
- `k8s/infisical/infisicalsecret-gridlogger-prod.yaml`
- `k8s/infisical/kustomization.yaml`
- `secrets.infisical.com/auto-reload: "true"` annotation on backend deployment

## One-time cluster setup

1. Install Infisical operator:

```bash
make infisical-install
```

2. Edit `k8s/infisical/infisicalsecret-gridlogger-prod.yaml` and replace:

- `REPLACE_WITH_MACHINE_IDENTITY_ID`
- `REPLACE_WITH_PROJECT_SLUG`
- `envSlug` if not `prod`
- `secretsPath` if not `/`

3. Apply Infisical resources:

```bash
make infisical-apply
```

4. Verify sync:

```bash
make infisical-status
kubectl describe infisicalsecret -n gridlogger gridlogger-prod-secrets
kubectl get secret -n gridlogger gridlogger-secrets -o yaml
```

## Required Infisical keys

Store these in your Infisical project/environment/path:

- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `POSTGRES_DB`
- `DATABASE_URL`

## Deploy flow

`make deploy` now:

1. Applies Traefik config
2. Ensures namespace exists
3. Applies Infisical CRD resources (`k8s/infisical`)
4. Applies app manifests (`k8s/overlays/prod`)

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
