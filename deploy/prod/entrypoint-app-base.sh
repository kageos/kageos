#!/bin/bash
set -euo pipefail

APP_BASE_IMAGE="${APP_BASE_IMAGE:-localhost/kagebase:latest}"
APP_BASE_ACTION="${APP_BASE_ACTION:-ensure}"
APP_BASE_BUILD_NO_CACHE="${APP_BASE_BUILD_NO_CACHE:-0}"

case "$APP_BASE_ACTION" in
  ensure|rebuild|check) ;;
  *)
    echo "ERROR: APP_BASE_ACTION 仅支持 ensure / rebuild / check，当前值: ${APP_BASE_ACTION}" >&2
    exit 1
    ;;
esac

case "$APP_BASE_BUILD_NO_CACHE" in
  0|1) ;;
  *)
    echo "ERROR: APP_BASE_BUILD_NO_CACHE 仅支持 0 或 1，当前值: ${APP_BASE_BUILD_NO_CACHE}" >&2
    exit 1
    ;;
esac

mkdir -p /run/podman /run/containers/storage

if ! podman info >/dev/null 2>&1; then
  echo "ERROR: 容器内 Podman 不可用，无法处理 ${APP_BASE_IMAGE}" >&2
  exit 1
fi

run_build() {
  if [[ "$APP_BASE_BUILD_NO_CACHE" == "1" ]]; then
    podman build --no-cache -t "${APP_BASE_IMAGE}" -f /app/app-base/Dockerfile /app/app-base/
  else
    podman build -t "${APP_BASE_IMAGE}" -f /app/app-base/Dockerfile /app/app-base/
  fi
}

case "$APP_BASE_ACTION" in
  ensure)
    if podman image exists "${APP_BASE_IMAGE}" 2>/dev/null; then
      echo "==> 用户应用基础镜像已存在，跳过构建: ${APP_BASE_IMAGE}"
      exit 0
    fi
    echo "==> 初始化用户应用基础镜像 ${APP_BASE_IMAGE}（首次约 10-20 分钟）..."
    run_build
    ;;
  rebuild)
    echo "==> 重建用户应用基础镜像 ${APP_BASE_IMAGE} ..."
    run_build
    ;;
  check)
    if podman image exists "${APP_BASE_IMAGE}" 2>/dev/null; then
      echo "==> 用户应用基础镜像已就绪: ${APP_BASE_IMAGE}"
      exit 0
    fi
    echo "ERROR: 未找到用户应用基础镜像: ${APP_BASE_IMAGE}" >&2
    exit 1
    ;;
esac

if ! podman image exists "${APP_BASE_IMAGE}" 2>/dev/null; then
  echo "ERROR: 构建完成后仍未检测到用户应用基础镜像: ${APP_BASE_IMAGE}" >&2
  exit 1
fi

echo "==> 用户应用基础镜像已就绪: ${APP_BASE_IMAGE}"
