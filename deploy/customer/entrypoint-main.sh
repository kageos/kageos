#!/bin/bash
set -euo pipefail

export AI_AGENT_OS_ROOT=/app
cd /app

require_env() {
  local n="$1"
  if [[ -z "${!n:-}" ]]; then
    echo "ERROR: 环境变量 ${n} 未设置或为空（须由 Compose 从宿主机 .env 注入）" >&2
    exit 1
  fi
}

require_env CANONICAL_BASE_URL
require_env MYSQL_ROOT_PASSWORD
require_env MINIO_ROOT_USER
require_env MINIO_ROOT_PASSWORD
require_env JWT_SECRET
require_env CONTROL_ENC_KEY

echo "==> 从模板刷新 deploy/config/prod（可安全重启）..."
rm -rf /app/deploy/config/prod
mkdir -p /app/deploy/config
cp -a /app/config.prod.template /app/deploy/config/prod

SMTP_HOST="${SMTP_HOST:-smtp.qq.com}"
SMTP_PORT="${SMTP_PORT:-587}"
SMTP_USERNAME="${SMTP_USERNAME-}"
SMTP_PASSWORD="${SMTP_PASSWORD-}"
SMTP_FROM="${SMTP_FROM-}"
SMTP_FROM_NAME="${SMTP_FROM_NAME:-AI Agent OS}"

wait_tcp() {
  local host="$1" port="$2" label="$3"
  local i=1
  while [ "$i" -le 90 ]; do
    if nc -z "$host" "$port" 2>/dev/null; then
      echo "==> ${label} (${host}:${port}) 就绪"
      return 0
    fi
    echo "    等待 ${label} (${host}:${port}) ... ($i/90)"
    sleep 2
    i=$((i + 1))
  done
  echo "ERROR: 超时未连上 ${label} ${host}:${port}"
  exit 1
}

echo "==> 等待依赖（MySQL / NATS / MinIO）..."
wait_tcp 127.0.0.1 3306 "MySQL"
wait_tcp 127.0.0.1 4222 "NATS"
wait_tcp 127.0.0.1 9000 "MinIO"

echo "==> 注入配置占位符..."
for f in /app/deploy/config/prod/*.yaml; do
  sed -i "s|__MYSQL_ROOT_PASSWORD__|${MYSQL_ROOT_PASSWORD}|g" "$f"
  sed -i "s|__JWT_SECRET__|${JWT_SECRET}|g" "$f"
  sed -i "s|__CONTROL_ENC_KEY__|${CONTROL_ENC_KEY}|g" "$f"
  sed -i "s|__MINIO_ROOT_USER__|${MINIO_ROOT_USER}|g" "$f"
  sed -i "s|__MINIO_ROOT_PASSWORD__|${MINIO_ROOT_PASSWORD}|g" "$f"
  sed -i "s|__SMTP_HOST__|${SMTP_HOST}|g" "$f"
  sed -i "s|__SMTP_PORT__|${SMTP_PORT}|g" "$f"
  sed -i "s|__SMTP_USERNAME__|${SMTP_USERNAME}|g" "$f"
  sed -i "s|__SMTP_PASSWORD__|${SMTP_PASSWORD}|g" "$f"
  sed -i "s|__SMTP_FROM__|${SMTP_FROM}|g" "$f"
  sed -i "s|__SMTP_FROM_NAME__|${SMTP_FROM_NAME}|g" "$f"
done

CANONICAL_BASE_URL="${CANONICAL_BASE_URL}"
export CANONICAL_BASE_URL
export CANONICAL_SCHEME
export CANONICAL_HOST
CANONICAL_SCHEME=$(echo "$CANONICAL_BASE_URL" | sed -E 's|^(https?).*|\1|')
CANONICAL_HOST=$(echo "$CANONICAL_BASE_URL" | sed -E 's|^https?://([^/]+).*|\1|')
export CANONICAL_SCHEME
export CANONICAL_HOST

echo "==> 生成 Nginx（80，www → 裸域 301）canonical_host=${CANONICAL_HOST} scheme=${CANONICAL_SCHEME}"
envsubst '${CANONICAL_HOST} ${CANONICAL_SCHEME}' < /app/deploy/customer/nginx/default.conf.template > /etc/nginx/sites-enabled/default
nginx -t

echo "==> 启动 Nginx（host 网络直接监听 80）..."
nginx

# Podman 需要明确的 runroot/graphroot（见 /etc/containers/storage.conf）；/run 每次启动需重建
mkdir -p /run/podman /run/containers/storage

echo "==> 启动 Podman API..."
podman system service --time=0 unix:///run/podman/podman.sock &
PODMAN_PID=$!
for _i in $(seq 1 30); do
  if [ -S /run/podman/podman.sock ]; then
    echo "==> Podman socket 就绪"
    break
  fi
  sleep 1
done
if [ ! -S /run/podman/podman.sock ]; then
  echo "WARN: /run/podman/podman.sock 未出现，app-runtime 可能仍失败"
fi

if ! podman image exists ai-agent-os:latest 2>/dev/null; then
  echo "==> 首次构建用户应用基础镜像 ai-agent-os:latest（较久）..."
  podman build -t ai-agent-os:latest -f /app/app-base/Dockerfile /app/app-base/ || {
    echo "WARN: 基础镜像构建失败，app-runtime 可能不可用"
  }
else
  echo "==> 已存在 ai-agent-os:latest，跳过构建"
fi

shutdown() {
  echo "==> 停止..."
  kill -TERM "$CORE_PID" 2>/dev/null || true
  kill -TERM "$PODMAN_PID" 2>/dev/null || true
  nginx -s quit 2>/dev/null || true
  wait "$CORE_PID" 2>/dev/null || true
  exit 0
}
trap shutdown SIGTERM SIGINT

echo "==> 启动 core-server（全服务）..."
/app/core-server &
CORE_PID=$!

wait -n "$CORE_PID"
echo "==> core-server 退出，关闭中..."
shutdown
