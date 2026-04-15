#!/bin/bash
set -euo pipefail

cd /app

# 确保持久化目录结构存在，便于备份与恢复编排
mkdir -p \
  /app/logs \
  /app/namespace \
  /app/data/runtime/app-runtime \
  /app/data/license \
  /app/data/backup/repo \
  /app/data/backup/state \
  /app/data/backup/staging \
  /app/data/tmp

require_env() {
  local n="$1"
  if [[ -z "${!n:-}" ]]; then
    echo "ERROR: 环境变量 ${n} 未设置或为空（须由 Compose 从宿主机 .env 注入）" >&2
    exit 1
  fi
}

require_env CANONICAL_BASE_URL
require_env STORAGE_ROOT
require_env MYSQL_ROOT_PASSWORD
require_env MINIO_ROOT_USER
require_env MINIO_ROOT_PASSWORD
require_env JWT_SECRET
require_env CONTROL_ENC_KEY

echo "==> 从模板刷新 deploy/prod/config/runtime（可安全重启）..."
rm -rf /app/deploy/prod/config/runtime
mkdir -p /app/deploy/prod/config

SMTP_HOST="${SMTP_HOST:-smtp.qq.com}"
SMTP_PORT="${SMTP_PORT:-587}"
SMTP_USERNAME="${SMTP_USERNAME-}"
SMTP_PASSWORD="${SMTP_PASSWORD-}"
SMTP_FROM="${SMTP_FROM-}"
SMTP_FROM_NAME="${SMTP_FROM_NAME:-AI Agent OS}"
ENABLE_HTTPS="${ENABLE_HTTPS:-0}"
HTTPS_REDIRECT="${HTTPS_REDIRECT:-0}"
TLS_CERT_FILE="${TLS_CERT_FILE:-/app/tls/fullchain.pem}"
TLS_KEY_FILE="${TLS_KEY_FILE:-/app/tls/privkey.pem}"

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

PROD_TEMPLATE_VARS='${STORAGE_ROOT} ${MYSQL_ROOT_PASSWORD} ${JWT_SECRET} ${CONTROL_ENC_KEY} ${MINIO_ROOT_USER} ${MINIO_ROOT_PASSWORD} ${SMTP_HOST} ${SMTP_PORT} ${SMTP_USERNAME} ${SMTP_PASSWORD} ${SMTP_FROM} ${SMTP_FROM_NAME}'

echo "==> 渲染 deploy/prod/config/runtime 模板..."
mkdir -p /app/deploy/prod/config/runtime
for src in /app/config.prod.template/*.yaml; do
  dst="/app/deploy/prod/config/runtime/$(basename "$src")"
  envsubst "$PROD_TEMPLATE_VARS" < "$src" > "$dst"
done

CANONICAL_BASE_URL="${CANONICAL_BASE_URL}"
export CANONICAL_BASE_URL
export CANONICAL_SCHEME
export CANONICAL_HOST
CANONICAL_SCHEME=$(echo "$CANONICAL_BASE_URL" | sed -E 's|^(https?).*|\1|')
CANONICAL_HOST=$(echo "$CANONICAL_BASE_URL" | sed -E 's|^https?://([^/]+).*|\1|')
export CANONICAL_SCHEME
export CANONICAL_HOST

case "$ENABLE_HTTPS" in
  0|1) ;;
  *)
    echo "ERROR: ENABLE_HTTPS 仅支持 0 或 1，当前值: ${ENABLE_HTTPS}" >&2
    exit 1
    ;;
esac

case "$HTTPS_REDIRECT" in
  0|1) ;;
  *)
    echo "ERROR: HTTPS_REDIRECT 仅支持 0 或 1，当前值: ${HTTPS_REDIRECT}" >&2
    exit 1
    ;;
esac

if [[ "$HTTPS_REDIRECT" == "1" && "$ENABLE_HTTPS" != "1" ]]; then
  echo "ERROR: HTTPS_REDIRECT=1 需要同时设置 ENABLE_HTTPS=1" >&2
  exit 1
fi

NGINX_TEMPLATE="/app/deploy/prod/nginx/default.conf.template"
NGINX_MODE_DESC="80 HTTP"

if [[ "$ENABLE_HTTPS" == "1" ]]; then
  if [[ ! -f "$TLS_CERT_FILE" ]]; then
    echo "ERROR: ENABLE_HTTPS=1 但证书文件不存在: ${TLS_CERT_FILE}" >&2
    exit 1
  fi
  if [[ ! -f "$TLS_KEY_FILE" ]]; then
    echo "ERROR: ENABLE_HTTPS=1 但私钥文件不存在: ${TLS_KEY_FILE}" >&2
    exit 1
  fi

  if [[ "$CANONICAL_SCHEME" != "https" ]]; then
    echo "WARN: ENABLE_HTTPS=1 但 CANONICAL_BASE_URL 不是 https://；Nginx 会提供 HTTPS，但 canonical scheme 仍按 ${CANONICAL_SCHEME} 生成"
  fi

  if [[ "$HTTPS_REDIRECT" == "1" ]]; then
    if [[ "$CANONICAL_SCHEME" != "https" ]]; then
      echo "ERROR: HTTPS_REDIRECT=1 时 CANONICAL_BASE_URL 必须使用 https://，当前为 ${CANONICAL_BASE_URL}" >&2
      exit 1
    fi
    NGINX_TEMPLATE="/app/deploy/prod/nginx/default.https-redirect.conf.template"
    NGINX_MODE_DESC="80 -> 443 重定向 + 443 HTTPS"
  else
    NGINX_TEMPLATE="/app/deploy/prod/nginx/default.https.conf.template"
    NGINX_MODE_DESC="80 HTTP + 443 HTTPS"
  fi
fi

mkdir -p /etc/nginx/snippets
cp /app/deploy/prod/nginx/common.server.inc /etc/nginx/snippets/ai-agent-os-common.conf

export TLS_CERT_FILE
export TLS_KEY_FILE
echo "==> 生成 Nginx（${NGINX_MODE_DESC}，www → 裸域 301）canonical_host=${CANONICAL_HOST} scheme=${CANONICAL_SCHEME}"
envsubst '${CANONICAL_HOST} ${CANONICAL_SCHEME} ${TLS_CERT_FILE} ${TLS_KEY_FILE}' < "${NGINX_TEMPLATE}" > /etc/nginx/sites-enabled/default
nginx -t

echo "==> 启动 Nginx（host 网络直接监听 ${NGINX_MODE_DESC}）..."
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
