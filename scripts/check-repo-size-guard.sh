#!/usr/bin/env bash
set -euo pipefail

max_bytes="${MAX_TRACKED_FILE_BYTES:-5242880}"
failed=0
tmp_objects="$(mktemp)"
tmp_sizes="$(mktemp)"
trap 'rm -f "$tmp_objects" "$tmp_sizes"' EXIT

human_size() {
  local bytes="$1"
  awk -v bytes="$bytes" 'BEGIN {
    split("B KiB MiB GiB", units, " ")
    value = bytes
    unit = 1
    while (value >= 1024 && unit < 4) {
      value = value / 1024
      unit++
    }
    printf "%.2f %s", value, units[unit]
  }'
}

is_forbidden_path() {
  case "$1" in
    node_modules/*|*/node_modules/*) return 0 ;;
    dist/*|*/dist/*) return 0 ;;
    logs/*|*/logs/*) return 0 ;;
    data/*|*/data/*) return 0 ;;
    local/*|*/local/*) return 0 ;;
    temp/*|*/temp/*) return 0 ;;
    coverage/*|*/coverage/*) return 0 ;;
    playwright-report/*|*/playwright-report/*) return 0 ;;
    test-results/*|*/test-results/*) return 0 ;;
    *.log|*.log.*|*.gz|*.db|*.sqlite|*.sqlite3|*.exe|*.test) return 0 ;;
    app|main|embed_case_code) return 0 ;;
    core/app-runtime/app) return 0 ;;
  esac
  return 1
}

git ls-files -s |
  awk '{
    oid = $2
    sub(/^[0-9]+ [0-9a-f]+ [0-9]+\t/, "")
    print oid "\t" $0
  }' > "$tmp_objects"

cut -f1 "$tmp_objects" | git cat-file --batch-check='%(objectsize)' > "$tmp_sizes"

while IFS=$'\t' read -r size _oid path; do
  if is_forbidden_path "$path"; then
    printf 'forbidden tracked artifact: %s\n' "$path" >&2
    failed=1
  fi

  if [ "$size" -gt "$max_bytes" ]; then
    printf 'oversized tracked file: %s (%s, limit %s)\n' \
      "$path" "$(human_size "$size")" "$(human_size "$max_bytes")" >&2
    failed=1
  fi
done < <(paste "$tmp_sizes" "$tmp_objects")

if [ "$failed" -ne 0 ]; then
  cat >&2 <<'MSG'

Repository size guard failed.
Move generated binaries, logs, runtime data, and dependency folders out of Git.
If a large asset is intentional, raise MAX_TRACKED_FILE_BYTES for that command
and document why the file belongs in the repository.
MSG
  exit 1
fi

echo "Repository size guard passed."
