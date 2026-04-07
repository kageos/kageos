#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"

if ! command -v go >/dev/null 2>&1; then
  echo "ERROR: 未找到 go，请先安装 Go。" >&2
  exit 1
fi

cd "$ROOT"

export APP_ENV="${APP_ENV:-dev}"

echo "==> APP_ENV=$APP_ENV"

exec go run ./core/cmd/main "$@"
