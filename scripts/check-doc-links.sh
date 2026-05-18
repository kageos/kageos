#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

fail=0

while IFS=: read -r file link; do
  [[ -z "${file:-}" || -z "${link:-}" ]] && continue
  [[ "$link" =~ ^https?:// ]] && continue
  [[ "$link" =~ ^mailto: ]] && continue
  [[ "$link" =~ ^# ]] && continue

  target="${link%%#*}"
  [[ -z "$target" ]] && continue

  if [[ "$target" == /* ]]; then
    resolved=".$target"
  else
    resolved="$(dirname "$file")/$target"
  fi

  if [[ ! -e "$resolved" ]]; then
    echo "Broken local markdown link: $file -> $link" >&2
    fail=1
  fi
done < <(
  rg -n --no-heading -o '\[[^]]+\]\(([^)]+)\)' --glob '*.md' |
    sed -E 's/^([^:]+):[0-9]+:\[[^]]+\]\(([^)]+)\)$/\1:\2/'
)

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi

echo "Markdown local link check passed."
