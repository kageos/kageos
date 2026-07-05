#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

pattern='github\.com/kageos/kageos/(sdk/agent-app|pkg/(logger|gormx/query)|dto|core)'
targets=(
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

SDK_ROOT="${KAGEOS_SDK_ROOT:-"$ROOT_DIR/../kageos-sdk"}"
if [[ ! -d "$SDK_ROOT" ]]; then
  echo "SDK boundary check passed (sibling SDK checkout not found, drift check skipped)"
  exit 0
fi

normalize_module_refs() {
  sed \
    -e 's#github\.com/kageos/kageos-sdk#github.com/kageos/__SDK_CONTRACT__#g' \
    -e 's#github\.com/kageos/kageos#github.com/kageos/__SDK_CONTRACT__#g'
}

check_shared_file() {
  local rel="$1"
  local main_file="$ROOT_DIR/$rel"
  local sdk_file="$SDK_ROOT/$rel"

  if [[ ! -f "$main_file" || ! -f "$sdk_file" ]]; then
    echo "SDK drift: shared file missing: $rel" >&2
    return 1
  fi

  if ! diff -u \
    <(normalize_module_refs < "$sdk_file") \
    <(normalize_module_refs < "$main_file") \
    >/tmp/kageos-sdk-drift.diff; then
    echo "SDK drift after module import normalization: $rel" >&2
    cat /tmp/kageos-sdk-drift.diff >&2
    return 1
  fi
}

check_shared_dir() {
  local rel="$1"
  local main_dir="$ROOT_DIR/$rel"
  local sdk_dir="$SDK_ROOT/$rel"

  if [[ ! -d "$main_dir" || ! -d "$sdk_dir" ]]; then
    echo "SDK drift: shared directory missing: $rel" >&2
    return 1
  fi

  local status=0
  while IFS= read -r file_rel; do
    if ! check_shared_file "$rel/$file_rel"; then
      status=1
    fi
  done < <(
    {
      (cd "$main_dir" && find . -type f | sed 's#^\./##')
      (cd "$sdk_dir" && find . -type f | sed 's#^\./##')
    } | sort -u
  )
  return "$status"
}

shared_contracts=(
  "pkg/apicall"
  "pkg/contextx/context_info.go"
  "pkg/functionschema"
  "pkg/msgx/msgs.go"
  "pkg/natsx/connect.go"
  "pkg/natsx/connect_test.go"
  "pkg/scheduledsdk"
  "pkg/storage"
)

drift_status=0
for rel in "${shared_contracts[@]}"; do
  if [[ -d "$ROOT_DIR/$rel" || -d "$SDK_ROOT/$rel" ]]; then
    if ! check_shared_dir "$rel"; then
      drift_status=1
    fi
  else
    if ! check_shared_file "$rel"; then
      drift_status=1
    fi
  fi
done

if [[ "$drift_status" -ne 0 ]]; then
  exit "$drift_status"
fi

if command -v go >/dev/null 2>&1 && [[ "${KAGEOS_SKIP_SDK_COMPILE:-}" != "1" ]]; then
  (
    cd "$SDK_ROOT"
    go test ./pkg/apicall ./pkg/contextx ./pkg/functionschema ./pkg/msgx ./pkg/natsx ./pkg/scheduledsdk ./pkg/storage
  )
fi

rm -f /tmp/kageos-sdk-drift.diff
echo "SDK boundary and drift checks passed"
