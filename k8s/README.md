Cluster-wide manifests were extracted to:

- `/Users/dmytro.semenchuk/projects/mylab/k8s/traefik`
- `/Users/dmytro.semenchuk/projects/mylab/k8s/logs`
- `/Users/dmytro.semenchuk/projects/mylab/k8s/infisical`
- `/Users/dmytro.semenchuk/projects/mylab/k8s/timescaledb`

This repository keeps project-scoped manifests under:

- `k8s/base`
- `k8s/overlays`
- `k8s/infisical` (project-specific `InfisicalSecret` mapping)

Database note:
- `k8s/base/timescaledb-svc.yaml` is now an `ExternalName` alias to `timescaledb.default.svc.cluster.local`.
