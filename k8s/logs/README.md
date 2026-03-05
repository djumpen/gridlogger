# VictoriaLogs (Home Lab)

This setup deploys:
- `victoria-logs-single` as the log store (6-month retention)
- `victoria-logs-collector` as a cluster-wide DaemonSet collector

## Install

```bash
helm repo add vm https://victoriametrics.github.io/helm-charts
helm repo update

kubectl create namespace logging --dry-run=client -o yaml | kubectl apply -f -
VL_AUTH_HASH="$(openssl passwd -apr1 'your-strong-password')"
kubectl -n logging create secret generic victoria-logs-basic-auth \
  --from-literal=users="admin:${VL_AUTH_HASH}" \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f k8s/logs/victoria-logs-auth.yaml

helm upgrade --install victoria-logs vm/victoria-logs-single \
  --namespace logging \
  -f k8s/logs/values-victoria-logs-single.yaml

helm upgrade --install victoria-logs-collector vm/victoria-logs-collector \
  --namespace logging \
  -f k8s/logs/values-victoria-logs-collector.yaml
```

## Verify

```bash
kubectl -n logging get pods
kubectl -n logging get ingress
kubectl -n logging logs deploy/victoria-logs-collector | tail -n 50
```

Open UI:
- `https://logs.mylab.rest`
- Basic Auth username: `admin`
- Password: configured in your runtime-created K8s secret `victoria-logs-basic-auth`

Quick query example in UI:
- `_stream:{kubernetes.namespace_name="gridlogger"}`
