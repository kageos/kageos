#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

pattern='github\.com/kageos/kageos/(sdk/agent-app|pkg/(logger|gormx/query)|dto|core)'
targets=(
  "deploy/aio/namespace-seed"
  "core/app-server/system-seed"
  "core/agent-server/prompt/system/prompt"
)

violations="$(
  rg -n --glob '*.go' --glob '*.md' --glob '*.json' "$pattern" "${targets[@]}" || true
)"

if [[ -n "$violations" ]]; then
  echo "User-facing workspace code must import github.com/kageos/kageos-sdk, not main repo internals:" >&2
  echo "$violations" >&2
  exit 1
fi

echo "SDK boundary check passed"
