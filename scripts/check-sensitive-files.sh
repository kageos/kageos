#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if ! command -v rg >/dev/null 2>&1; then
  echo "Sensitive file boundary check requires ripgrep (rg)." >&2
  exit 1
fi

fail=0

tracked_sensitive="$(
  git ls-files |
    rg '(^|/)\.env(\.|$)|^\.kageos/|^deploy/prod/(aos|kage)\.yaml$|(^|/)license\.json$|(^|/)(secret|private|credential)([^/]*$|/)' |
    rg -v '\.example$' || true
)"

if [[ -n "$tracked_sensitive" ]]; then
  echo "Tracked files match sensitive/publication-boundary patterns:" >&2
  echo "$tracked_sensitive" >&2
  fail=1
fi

content_hits="$(
  rg -n --hidden -S \
    --glob '!**/.git/**' \
    --glob '!web/node_modules/**' \
    --glob '!local/**' \
    --glob '!namespace/**' \
    --glob '!.kageos/**' \
    --glob '!license.json' \
    --glob '!*.example' \
    --glob '!*.md' \
    --glob '!**/*_test.go' \
    'AKIA[0-9A-Z]{16}|BEGIN (RSA |OPENSSH |EC |DSA )?PRIVATE KEY|sk-[A-Za-z0-9_-]{20,}' || true
)"

if [[ -n "$content_hits" ]]; then
  echo "Potential secret-looking values found:" >&2
  echo "$content_hits" >&2
  fail=1
fi

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi

echo "Sensitive file boundary check passed."
