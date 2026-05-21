#!/usr/bin/env bash
# 构建用户应用基础镜像（默认 kagebase:latest，可由 KAGEOS_APP_BASE_IMAGE 覆盖）
# Canonical 位置：deploy/base/scripts/build-app-base-image.sh
# 用法：在项目根目录执行 bash deploy/base/scripts/build-app-base-image.sh [--force] [--no-cache]

set -euo pipefail
cd "$(dirname "$0")/../../.."
KAGEOS_APP_BASE_IMAGE="${KAGEOS_APP_BASE_IMAGE:-kagebase:latest}"
KAGEOS_APP_BASE_APT_CHECK_DATE="${KAGEOS_APP_BASE_APT_CHECK_DATE:-0}"
force=0
no_cache=0

usage() {
  cat <<'EOF'
用法: bash deploy/base/scripts/build-app-base-image.sh [--force] [--no-cache]

选项：
  --force     即使镜像 tag 已存在，也继续重建
  --no-cache  构建时不使用 layer 缓存
EOF
}

for arg in "$@"; do
  case "$arg" in
    --force)
      force=1
      ;;
    --no-cache)
      no_cache=1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "ERROR: 不支持的参数: $arg"
      usage
      exit 1
      ;;
  esac
done

if [[ "$force" != "1" ]] && podman image exists "${KAGEOS_APP_BASE_IMAGE}" 2>/dev/null; then
  echo "==> ${KAGEOS_APP_BASE_IMAGE} 已存在，跳过构建（如需重建请先 podman rmi ${KAGEOS_APP_BASE_IMAGE}）"
  exit 0
fi

echo "==> 构建用户应用基础镜像 ${KAGEOS_APP_BASE_IMAGE}（首次约 10–20 分钟）..."
if [[ "$force" == "1" ]]; then
  echo "==> 已启用 --force：即使 tag 已存在也会继续构建"
fi
if [[ "$no_cache" == "1" ]]; then
  echo "==> 已启用 --no-cache：本次不复用 layer 缓存"
fi
if [[ "$no_cache" == "1" ]]; then
  podman build \
    --no-cache \
    --build-arg APT_CHECK_DATE="${KAGEOS_APP_BASE_APT_CHECK_DATE}" \
    -t "${KAGEOS_APP_BASE_IMAGE}" \
    deploy/base/images/app-base
else
  podman build \
    --build-arg APT_CHECK_DATE="${KAGEOS_APP_BASE_APT_CHECK_DATE}" \
    -t "${KAGEOS_APP_BASE_IMAGE}" \
    deploy/base/images/app-base
fi
echo "==> 完成：${KAGEOS_APP_BASE_IMAGE}"
