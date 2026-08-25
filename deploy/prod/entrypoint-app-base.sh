#!/bin/bash
set -euo pipefail

KAGEOS_APP_BASE_IMAGE="${KAGEOS_APP_BASE_IMAGE:-${KAGEOS_DEFAULT_APP_BASE_IMAGE:-docker.io/qiayanai/kagebase:latest}}"
KAGEOS_APP_BASE_ACTION="${KAGEOS_APP_BASE_ACTION:-ensure}"
KAGEOS_APP_BASE_BUILD_NO_CACHE="${KAGEOS_APP_BASE_BUILD_NO_CACHE:-0}"
KAGEOS_APP_BASE_PULL="${KAGEOS_APP_BASE_PULL:-1}"
KAGEOS_APP_BASE_PULL_FALLBACK_BUILD="${KAGEOS_APP_BASE_PULL_FALLBACK_BUILD:-1}"
APP_BASE_PREPARE_STARTED_AT="$(date +%s)"

report_duration() {
  local label="$1" started_at="$2" status="${3:-success}"
  local elapsed=$(( $(date +%s) - started_at ))
  echo "==> [耗时] ${label}: ${elapsed}s (${status})"
}

case "$KAGEOS_APP_BASE_ACTION" in
  ensure|rebuild|check) ;;
  *)
    echo "ERROR: KAGEOS_APP_BASE_ACTION 仅支持 ensure / rebuild / check，当前值: ${KAGEOS_APP_BASE_ACTION}" >&2
    exit 1
    ;;
esac

case "$KAGEOS_APP_BASE_BUILD_NO_CACHE" in
  0|1) ;;
  *)
    echo "ERROR: KAGEOS_APP_BASE_BUILD_NO_CACHE 仅支持 0 或 1，当前值: ${KAGEOS_APP_BASE_BUILD_NO_CACHE}" >&2
    exit 1
    ;;
esac

case "$KAGEOS_APP_BASE_PULL" in
  0|1) ;;
  *)
    echo "ERROR: KAGEOS_APP_BASE_PULL 仅支持 0 或 1，当前值: ${KAGEOS_APP_BASE_PULL}" >&2
    exit 1
    ;;
esac

case "$KAGEOS_APP_BASE_PULL_FALLBACK_BUILD" in
  0|1) ;;
  *)
    echo "ERROR: KAGEOS_APP_BASE_PULL_FALLBACK_BUILD 仅支持 0 或 1，当前值: ${KAGEOS_APP_BASE_PULL_FALLBACK_BUILD}" >&2
    exit 1
    ;;
esac

mkdir -p /run/podman /run/containers/storage

if ! podman info >/dev/null 2>&1; then
  echo "ERROR: 容器内 Podman 不可用，无法处理 ${KAGEOS_APP_BASE_IMAGE}" >&2
  exit 1
fi

run_build() {
  if [[ "$KAGEOS_APP_BASE_BUILD_NO_CACHE" == "1" ]]; then
    podman build --no-cache -t "${KAGEOS_APP_BASE_IMAGE}" -f /app/app-base/Dockerfile /app/app-base/
  else
    podman build -t "${KAGEOS_APP_BASE_IMAGE}" -f /app/app-base/Dockerfile /app/app-base/
  fi
}

pull_base_image() {
  echo "==> 拉取用户应用基础镜像: ${KAGEOS_APP_BASE_IMAGE}"
  podman pull "${KAGEOS_APP_BASE_IMAGE}"
}

case "$KAGEOS_APP_BASE_ACTION" in
  ensure)
    if podman image exists "${KAGEOS_APP_BASE_IMAGE}" 2>/dev/null; then
      echo "==> 用户应用基础镜像已存在，跳过构建: ${KAGEOS_APP_BASE_IMAGE}"
      report_duration "用户应用基础镜像准备总计" "$APP_BASE_PREPARE_STARTED_AT" cached
      exit 0
    fi
    if [[ "$KAGEOS_APP_BASE_PULL" == "1" ]]; then
      PULL_STARTED_AT="$(date +%s)"
      if pull_base_image; then
        report_duration "用户应用基础镜像拉取与入库" "$PULL_STARTED_AT"
        echo "==> 用户应用基础镜像已拉取: ${KAGEOS_APP_BASE_IMAGE}"
        report_duration "用户应用基础镜像准备总计" "$APP_BASE_PREPARE_STARTED_AT"
        exit 0
      else
        pull_status=$?
      fi
      report_duration "用户应用基础镜像拉取与入库" "$PULL_STARTED_AT" failed >&2
      if [[ "$KAGEOS_APP_BASE_PULL_FALLBACK_BUILD" == "0" ]]; then
        echo "ERROR: 拉取用户应用基础镜像失败，且已禁用本地 fallback 构建: ${KAGEOS_APP_BASE_IMAGE}" >&2
        exit "$pull_status"
      fi
      echo "WARN: 拉取用户应用基础镜像失败，改为本地构建 fallback: ${KAGEOS_APP_BASE_IMAGE}" >&2
    fi
    echo "==> 本地构建用户应用基础镜像 ${KAGEOS_APP_BASE_IMAGE}（首次约 10-20 分钟）..."
    BUILD_STARTED_AT="$(date +%s)"
    if run_build; then
      report_duration "用户应用基础镜像本地构建" "$BUILD_STARTED_AT"
    else
      build_status=$?
      report_duration "用户应用基础镜像本地构建" "$BUILD_STARTED_AT" failed >&2
      exit "$build_status"
    fi
    ;;
  rebuild)
    echo "==> 重建用户应用基础镜像 ${KAGEOS_APP_BASE_IMAGE} ..."
    BUILD_STARTED_AT="$(date +%s)"
    if run_build; then
      report_duration "用户应用基础镜像本地重建" "$BUILD_STARTED_AT"
    else
      build_status=$?
      report_duration "用户应用基础镜像本地重建" "$BUILD_STARTED_AT" failed >&2
      exit "$build_status"
    fi
    ;;
  check)
    if podman image exists "${KAGEOS_APP_BASE_IMAGE}" 2>/dev/null; then
      echo "==> 用户应用基础镜像已就绪: ${KAGEOS_APP_BASE_IMAGE}"
      exit 0
    fi
    echo "ERROR: 未找到用户应用基础镜像: ${KAGEOS_APP_BASE_IMAGE}" >&2
    exit 1
    ;;
esac

if ! podman image exists "${KAGEOS_APP_BASE_IMAGE}" 2>/dev/null; then
  echo "ERROR: 构建完成后仍未检测到用户应用基础镜像: ${KAGEOS_APP_BASE_IMAGE}" >&2
  exit 1
fi

echo "==> 用户应用基础镜像已就绪: ${KAGEOS_APP_BASE_IMAGE}"
report_duration "用户应用基础镜像准备总计" "$APP_BASE_PREPARE_STARTED_AT"
