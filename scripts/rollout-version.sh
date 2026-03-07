#!/usr/bin/env sh
set -eu

usage() {
  cat <<'EOF'
Usage: sh scripts/rollout-version.sh <version> [--with-firmware]

Examples:
  sh scripts/rollout-version.sh 1.4.2
  sh scripts/rollout-version.sh v1.4.2 --with-firmware

Notes:
  - This script only rolls Kubernetes deployments back to a previously published image tag.
  - Database migrations are intentionally not rolled back here. Keep migrations forward-only and additive.
EOF
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

if [ $# -lt 1 ]; then
  usage >&2
  exit 1
fi

version_raw="$1"
shift

with_firmware="false"
while [ $# -gt 0 ]; do
  case "$1" in
    --with-firmware)
      with_firmware="true"
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
  shift
done

version="${version_raw#v}"
if [ -z "$version" ]; then
  echo "version is required" >&2
  exit 1
fi

namespace="${K8S_NAMESPACE:-gridlogger}"
backend_image="${BACKEND_IMAGE:-ghcr.io/djumpen/gridlogger-backend}"
frontend_image="${FRONTEND_IMAGE:-ghcr.io/djumpen/gridlogger-frontend}"
firmware_image="${FIRMWARE_IMAGE:-ghcr.io/djumpen/gridlogger-firmware}"
backend_timeout="${BACKEND_ROLLOUT_TIMEOUT:-180s}"
frontend_timeout="${FRONTEND_ROLLOUT_TIMEOUT:-180s}"
firmware_timeout="${FIRMWARE_ROLLOUT_TIMEOUT:-300s}"

kubectl -n "$namespace" set image deployment/backend "backend=${backend_image}:${version}"
kubectl -n "$namespace" annotate deployment/backend app.kubernetes.io/version="$version" --overwrite

kubectl -n "$namespace" set image deployment/frontend "frontend=${frontend_image}:${version}"
kubectl -n "$namespace" annotate deployment/frontend app.kubernetes.io/version="$version" --overwrite

if [ "$with_firmware" = "true" ]; then
  kubectl -n "$namespace" set image deployment/firmware "firmware=${firmware_image}:${version}"
  kubectl -n "$namespace" annotate deployment/firmware app.kubernetes.io/version="$version" --overwrite
fi

kubectl -n "$namespace" rollout status deployment/backend --timeout="$backend_timeout"
kubectl -n "$namespace" rollout status deployment/frontend --timeout="$frontend_timeout"

if [ "$with_firmware" = "true" ]; then
  kubectl -n "$namespace" rollout status deployment/firmware --timeout="$firmware_timeout"
fi

echo "Rolled out version ${version} in namespace ${namespace}."
