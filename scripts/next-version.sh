#!/usr/bin/env sh
set -eu

mode="version"
if [ "${1:-}" = "--tag" ]; then
  mode="tag"
fi

latest_tag="$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | head -n1 || true)"
base_version="0.0.0"
range="HEAD"

if [ -n "$latest_tag" ]; then
  base_version="${latest_tag#v}"
  range="${latest_tag}..HEAD"
fi

log_output="$(git log --format='%s%n%b' "$range" 2>/dev/null || true)"

if [ -z "$log_output" ]; then
  next_version="$base_version"
else
  bump="patch"

  if printf '%s\n' "$log_output" | grep -Eq '(^|[[:space:]])BREAKING CHANGE:'; then
    bump="major"
  elif git log --format='%s' "$range" 2>/dev/null | grep -Eq '^[a-z]+(\([^)]+\))?!:'; then
    bump="major"
  elif git log --format='%s' "$range" 2>/dev/null | grep -Eq '^feat(\([^)]+\))?:'; then
    bump="minor"
  fi

  IFS=. read -r major minor patch <<EOF_VERSION
$base_version
EOF_VERSION

  case "$bump" in
    major)
      major=$((major + 1))
      minor=0
      patch=0
      ;;
    minor)
      minor=$((minor + 1))
      patch=0
      ;;
    patch)
      patch=$((patch + 1))
      ;;
    *)
      echo "unsupported bump type: $bump" >&2
      exit 1
      ;;
  esac

  next_version="${major}.${minor}.${patch}"
fi

if [ "$mode" = "tag" ]; then
  printf 'v%s\n' "$next_version"
else
  printf '%s\n' "$next_version"
fi
