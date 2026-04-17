#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
APP_BASE_IMAGE="${APP_BASE_IMAGE:-agentos-app-runtime-base:latest}"

if ! command -v go >/dev/null 2>&1; then
  echo "ERROR: 未找到 go，请先安装 Go。" >&2
  exit 1
fi

cd "$ROOT"

export APP_ENV="${APP_ENV:-dev}"

echo "==> APP_ENV=$APP_ENV"

if [[ "${AOS_SKIP_APP_BASE_BUILD:-0}" != "1" ]]; then
  if ! command -v podman >/dev/null 2>&1; then
    echo "WARN: 未找到 podman，跳过 ${APP_BASE_IMAGE} 预构建检查；如后续 app-runtime 报缺镜像，请先执行 bash deploy/dev/scripts/build-app-base.sh"
  elif ! podman image exists "${APP_BASE_IMAGE}" 2>/dev/null; then
    echo "==> 未检测到用户应用基础镜像 ${APP_BASE_IMAGE}，自动开始构建 ..."
    APP_BASE_IMAGE="${APP_BASE_IMAGE}" bash deploy/dev/scripts/build-app-base.sh
  fi
fi

exec go run ./core/cmd/main "$@"
