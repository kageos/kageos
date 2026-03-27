#!/usr/bin/env bash
# 构建用户应用基础镜像 ai-agent-os:latest（Podman）
# Canonical 位置：deploy/base/scripts/build-app-base-image.sh
# 用法：在项目根目录执行 bash deploy/base/scripts/build-app-base-image.sh

set -e
cd "$(dirname "$0")/../../.."

if podman image exists ai-agent-os:latest 2>/dev/null; then
echo "==> ai-agent-os:latest 已存在，跳过构建（如需重建请先 podman rmi ai-agent-os:latest）"
  exit 0
fi

echo "==> 构建用户应用基础镜像 ai-agent-os:latest（首次约 10–20 分钟）..."
podman build -t ai-agent-os:latest deploy/base/images/app-base
echo "==> 完成：ai-agent-os:latest"
