#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

targets=()
for dir in core/*/api core/*/repository core/*/service; do
  if [[ -d "$dir" ]]; then
    targets+=("$dir")
  fi
done

if [[ "${#targets[@]}" -eq 0 ]]; then
  echo "Context boundary check passed (no business-layer directories found)"
  exit 0
fi

violations="$(
  rg -n \
    --glob '*.go' \
    --glob '!**/*_test.go' \
    'context\.(Background|TODO)\(\)' \
    "${targets[@]}" || true
)"

if [[ -n "$violations" ]]; then
  echo "Business API, repository, and service code must propagate its caller context:" >&2
  echo "$violations" >&2
  echo "Create root contexts only at process or verified message boundaries, outside business layers." >&2
  exit 1
fi

echo "Context boundary check passed"
