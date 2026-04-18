#!/bin/bash
set -euo pipefail

source /app/entrypoint-common.sh

ensure_main_runtime_dirs

require_env CANONICAL_BASE_URL "环境变量 CANONICAL_BASE_URL 未设置或为空（须由 Compose 从宿主机 .env 注入）"
require_env MYSQL_ROOT_PASSWORD "环境变量 MYSQL_ROOT_PASSWORD 未设置或为空（须由 Compose 从宿主机 .env 注入）"
require_env MINIO_ROOT_PASSWORD "环境变量 MINIO_ROOT_PASSWORD 未设置或为空（须由 Compose 从宿主机 .env 注入）"
require_env JWT_SECRET "环境变量 JWT_SECRET 未设置或为空（须由 Compose 从宿主机 .env 注入）"
require_env CONTROL_ENC_KEY "环境变量 CONTROL_ENC_KEY 未设置或为空（须由 Compose 从宿主机 .env 注入）"

mkdir -p /app/deploy/prod/config
set_smtp_defaults
TLS_MODE="${TLS_MODE:-http}"
TLS_CERT_FILE="${TLS_CERT_FILE:-/app/tls/fullchain.pem}"
TLS_KEY_FILE="${TLS_KEY_FILE:-/app/tls/privkey.pem}"
APP_BASE_IMAGE="${APP_BASE_IMAGE:-localhost/agentos-app-runtime-base:latest}"

echo "==> 等待依赖（MySQL / NATS / MinIO）..."
wait_tcp 127.0.0.1 3306 "MySQL"
wait_tcp 127.0.0.1 4222 "NATS"
wait_tcp 127.0.0.1 9000 "MinIO"

PROD_TEMPLATE_VARS='${MYSQL_ROOT_PASSWORD} ${JWT_SECRET} ${CONTROL_ENC_KEY} ${MINIO_ROOT_PASSWORD} ${SMTP_HOST} ${SMTP_PORT} ${SMTP_USERNAME} ${SMTP_PASSWORD} ${SMTP_FROM} ${SMTP_FROM_NAME} ${APP_BASE_IMAGE}'
render_runtime_templates "$PROD_TEMPLATE_VARS"

CANONICAL_BASE_URL="${CANONICAL_BASE_URL}"
export CANONICAL_BASE_URL
export CANONICAL_SCHEME
export CANONICAL_HOST
CANONICAL_SCHEME=$(echo "$CANONICAL_BASE_URL" | sed -E 's|^(https?).*|\1|')
CANONICAL_HOST=$(echo "$CANONICAL_BASE_URL" | sed -E 's|^https?://([^/]+).*|\1|')
export CANONICAL_SCHEME
export CANONICAL_HOST

case "$TLS_MODE" in
  http|https|redirect|external) ;;
  *)
    echo "ERROR: TLS_MODE 仅支持 http / https / redirect / external，当前值: ${TLS_MODE}" >&2
    exit 1
    ;;
esac

NGINX_TEMPLATE="/app/deploy/prod/nginx/default.conf.template"
NGINX_MODE_DESC="80 HTTP"

if [[ "$TLS_MODE" == "https" || "$TLS_MODE" == "redirect" ]]; then
  if [[ ! -f "$TLS_CERT_FILE" ]]; then
    echo "ERROR: TLS_MODE=${TLS_MODE} 但证书文件不存在: ${TLS_CERT_FILE}" >&2
    exit 1
  fi
  if [[ ! -f "$TLS_KEY_FILE" ]]; then
    echo "ERROR: TLS_MODE=${TLS_MODE} 但私钥文件不存在: ${TLS_KEY_FILE}" >&2
    exit 1
  fi

  if [[ "$CANONICAL_SCHEME" != "https" ]]; then
    echo "WARN: TLS_MODE=${TLS_MODE} 会在本机提供 HTTPS，但 CANONICAL_BASE_URL 不是 https://；canonical scheme 仍按 ${CANONICAL_SCHEME} 生成"
  fi

  if [[ "$TLS_MODE" == "redirect" ]]; then
    if [[ "$CANONICAL_SCHEME" != "https" ]]; then
      echo "ERROR: TLS_MODE=redirect 时 CANONICAL_BASE_URL 必须使用 https://，当前为 ${CANONICAL_BASE_URL}" >&2
      exit 1
    fi
    NGINX_TEMPLATE="/app/deploy/prod/nginx/default.https-redirect.conf.template"
    NGINX_MODE_DESC="80 -> 443 重定向 + 443 HTTPS"
  else
    NGINX_TEMPLATE="/app/deploy/prod/nginx/default.https.conf.template"
    NGINX_MODE_DESC="80 HTTP + 443 HTTPS"
  fi
elif [[ "$TLS_MODE" == "external" ]]; then
  if [[ "$CANONICAL_SCHEME" != "https" ]]; then
    echo "WARN: TLS_MODE=external 通常建议配合 https:// 的 CANONICAL_BASE_URL；当前为 ${CANONICAL_BASE_URL}"
  fi
  NGINX_MODE_DESC="80 HTTP（外部 TLS 终止）"
elif [[ "$CANONICAL_SCHEME" == "https" ]]; then
  echo "WARN: TLS_MODE=http 但 CANONICAL_BASE_URL 使用 https://；如果前面有 TLS 终止，请改成 TLS_MODE=external"
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

if ! podman image exists "${APP_BASE_IMAGE}" 2>/dev/null; then
  echo "ERROR: 未找到用户应用基础镜像 ${APP_BASE_IMAGE}" >&2
  echo "ERROR: 请先在宿主机执行 bash build.sh init；如需直接拉取已发布主镜像，用 bash build.sh init --image" >&2
  exit 1
fi
echo "==> 用户应用基础镜像已就绪: ${APP_BASE_IMAGE}"

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
