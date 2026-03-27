#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"

if ! command -v go >/dev/null 2>&1; then
  echo "ERROR: 未找到 go，请先安装 Go。" >&2
  exit 1
fi

cd "$ROOT"

export AI_AGENT_OS_ROOT="${AI_AGENT_OS_ROOT:-$ROOT}"
export APP_ENV="${APP_ENV:-dev}"
export AI_AGENT_OS_DEV_SKIP_EMBEDDING_INFRA="${AI_AGENT_OS_DEV_SKIP_EMBEDDING_INFRA:-1}"

echo "==> AI_AGENT_OS_ROOT=$AI_AGENT_OS_ROOT"
echo "==> APP_ENV=$APP_ENV"
echo "==> AI_AGENT_OS_DEV_SKIP_EMBEDDING_INFRA=$AI_AGENT_OS_DEV_SKIP_EMBEDDING_INFRA"

exec go run ./core/cmd/main "$@"
