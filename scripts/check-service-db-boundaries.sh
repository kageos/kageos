#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

violations=""

cross_service_imports="$({
  rg -n \
    --glob '*.go' \
    --glob '!**/*_test.go' \
    'github\.com/kageos/kageos/core/[^/]+/(model|repository)' \
    core || true
} | while IFS=: read -r file line source_line; do
  source_service="$(printf '%s' "$file" | cut -d/ -f2)"
  target_service="$(printf '%s' "$source_line" | sed -nE 's#.*core/([^/]+)/(model|repository).*#\1#p')"
  if [[ -n "$target_service" && "$source_service" != "$target_service" ]]; then
    printf '%s:%s imports %s data layer from %s\n' "$file" "$line" "$target_service" "$source_service"
  fi
done)"

if [[ -n "$cross_service_imports" ]]; then
  violations+="$cross_service_imports"$'\n'
fi

table_owners="$(mktemp)"
trap 'rm -f "$table_owners"' EXIT

while IFS= read -r model_file; do
  service="$(printf '%s' "$model_file" | cut -d/ -f2)"
  awk -v service="$service" '
    /TableName\(\) string/ { inside = 1; next }
    inside && match($0, /return "[^"]+"/) {
      table_name = substr($0, RSTART + 8, RLENGTH - 9)
      print table_name "\t" service
      inside = 0
    }
    inside && /}/ { inside = 0 }
  ' "$model_file"
done < <(rg --files core | rg '/model/[^/]+\.go$') | sort -u > "$table_owners"

while IFS=$'\t' read -r table_name owner_service; do
  [[ -n "$table_name" && -n "$owner_service" ]] || continue
  query_pattern="Table\\([[:space:]]*\"${table_name}([[:space:]]|\")|Joins\\(\"[^\"]*(JOIN|FROM)[[:space:]]+${table_name}([[:space:]]|\")"
  matches="$(rg -n -i \
    --glob '*.go' \
    --glob '!**/*_test.go' \
    "$query_pattern" \
    core/*/model core/*/repository core/*/service 2>/dev/null || true)"
  [[ -n "$matches" ]] || continue

  while IFS=: read -r file line source_line; do
    source_service="$(printf '%s' "$file" | cut -d/ -f2)"
    if [[ "$source_service" != "$owner_service" ]]; then
      violations+="${file}:${line} queries ${table_name} owned by ${owner_service} from ${source_service}: ${source_line}"$'\n'
    fi
  done <<< "$matches"
done < "$table_owners"

if [[ -n "$violations" ]]; then
  echo "Core services must access another service's data through its API, not its database tables:" >&2
  printf '%s' "$violations" >&2
  exit 1
fi

echo "Service database ownership check passed"
