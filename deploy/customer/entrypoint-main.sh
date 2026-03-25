#!/bin/bash
set -euo pipefail

export AI_AGENT_OS_ROOT=/app
cd /app

echo "==> 从模板刷新 deploy/config/prod（可安全重启）..."
rm -rf /app/deploy/config/prod
mkdir -p /app/deploy/config
cp -a /app/config.prod.template /app/deploy/config/prod

MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-changeme}"
MINIO_ROOT_USER="${MINIO_ROOT_USER:-minioadmin}"
MINIO_ROOT_PASSWORD="${MINIO_ROOT_PASSWORD:-minioadmin123}"
JWT_SECRET="${JWT_SECRET:-change-me-jwt-secret-min-32-characters-long!!}"
CONTROL_ENC_KEY="${CONTROL_ENC_KEY:-ai-agent-os-license-key-32bytes!}"
SMTP_PASSWORD="${SMTP_PASSWORD:-}"

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
wait_tcp mysql 3306 "MySQL"
wait_tcp nats 4222 "NATS"
wait_tcp minio 9000 "MinIO"

echo "==> 注入配置占位符..."
for f in /app/deploy/config/prod/*.yaml; do
  sed -i "s|__MYSQL_ROOT_PASSWORD__|${MYSQL_ROOT_PASSWORD}|g" "$f"
  sed -i "s|__JWT_SECRET__|${JWT_SECRET}|g" "$f"
  sed -i "s|__CONTROL_ENC_KEY__|${CONTROL_ENC_KEY}|g" "$f"
  sed -i "s|__MINIO_ROOT_USER__|${MINIO_ROOT_USER}|g" "$f"
  sed -i "s|__MINIO_ROOT_PASSWORD__|${MINIO_ROOT_PASSWORD}|g" "$f"
  sed -i "s|__SMTP_PASSWORD__|${SMTP_PASSWORD}|g" "$f"
done

CANONICAL_BASE_URL="${CANONICAL_BASE_URL:-https://geeleo.com}"
export CANONICAL_BASE_URL
export CANONICAL_SCHEME
export CANONICAL_HOST
CANONICAL_SCHEME=$(echo "$CANONICAL_BASE_URL" | sed -E 's|^(https?).*|\1|')
CANONICAL_HOST=$(echo "$CANONICAL_BASE_URL" | sed -E 's|^https?://([^/]+).*|\1|')
export CANONICAL_SCHEME
export CANONICAL_HOST

echo "==> 生成 Nginx（8080，www → 裸域 301）canonical_host=${CANONICAL_HOST} scheme=${CANONICAL_SCHEME}"
envsubst '${CANONICAL_HOST} ${CANONICAL_SCHEME}' < /app/deploy/customer/nginx/default.conf.template > /etc/nginx/sites-enabled/default
nginx -t

echo "==> 启动 Nginx（对外 8080 → 映射 compose 80）..."
nginx

echo "==> 启动 Podman API..."
podman system service --time=0 unix:///run/podman/podman.sock &
PODMAN_PID=$!
sleep 1

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
