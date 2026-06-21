#!/bin/bash
set -euo pipefail

source /app/entrypoint-common.sh

ensure_main_runtime_dirs

require_env CANONICAL_BASE_URL "环境变量 CANONICAL_BASE_URL 未设置或为空（须由 Compose 从宿主机 .env 注入）"
require_env MYSQL_ROOT_PASSWORD "环境变量 MYSQL_ROOT_PASSWORD 未设置或为空（须由 Compose 从宿主机 .env 注入）"
require_env KAGEOS_APP_DB_SECRET_KEY "环境变量 KAGEOS_APP_DB_SECRET_KEY 未设置或为空（须由 Compose 从宿主机 .env 注入）"
require_env MINIO_ROOT_PASSWORD "环境变量 MINIO_ROOT_PASSWORD 未设置或为空（须由 Compose 从宿主机 .env 注入）"
require_env JWT_SECRET "环境变量 JWT_SECRET 未设置或为空（须由 Compose 从宿主机 .env 注入）"

mkdir -p /app/deploy/prod/config
set_smtp_defaults
TLS_MODE="${TLS_MODE:-http}"
HTTP_PORT="${HTTP_PORT:-80}"
HTTPS_PORT="${HTTPS_PORT:-443}"
TLS_CERT_FILE="${TLS_CERT_FILE:-/app/tls/fullchain.pem}"
TLS_KEY_FILE="${TLS_KEY_FILE:-/app/tls/privkey.pem}"
KAGEOS_APP_BASE_IMAGE="${KAGEOS_APP_BASE_IMAGE:-kagebase:latest}"
MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
NATS_HOST="${NATS_HOST:-127.0.0.1}"
NATS_PORT="${NATS_PORT:-4222}"
MINIO_HOST="${MINIO_HOST:-127.0.0.1}"
MINIO_PORT="${MINIO_PORT:-9000}"
MINIO_CONSOLE_PORT="${MINIO_CONSOLE_PORT:-9001}"
MINIO_ROOT_USER="${MINIO_ROOT_USER:-minioadmin}"
NATS_USER="${NATS_USER:-${NATS_SEED_USER:-aos}}"
NATS_PASSWORD="${NATS_PASSWORD:-${NATS_SEED_PASSWORD:-}}"
NATS_URL="${NATS_URL:-nats://${NATS_USER}:${NATS_PASSWORD}@${NATS_HOST}:${NATS_PORT}}"
export KAGEOS_APP_BASE_IMAGE
export MYSQL_HOST
export MYSQL_PORT
export NATS_HOST
export NATS_PORT
export NATS_USER
export NATS_PASSWORD
export NATS_URL
export MINIO_HOST
export MINIO_PORT
export MINIO_CONSOLE_PORT
export MINIO_ROOT_USER
export HTTP_PORT
export HTTPS_PORT

is_aio_bundle() {
  [[ -f /etc/kageos/aio-bundle ]]
}

aio_secret_value() {
  local name="$1"
  local data_dir="${KAGEOS_AIO_DATA_DIR:-/var/lib/kageos}"
  local file="${data_dir}/secrets/${name}"
  local value="${!name:-}"
  if [[ -z "$value" && -r "$file" ]]; then
    value="$(tr -d '\r\n' < "$file")"
  fi
  printf '%s' "$value"
}

wait_core_ready() {
  local url="${KAGEOS_AIO_HEALTH_URL:-http://127.0.0.1:9090/health}"
  local i=1
  while [ "$i" -le 90 ]; do
    if ! kill -0 "$CORE_PID" 2>/dev/null; then
      echo "ERROR: core-server 在就绪前已退出" >&2
      return 1
    fi
    if curl --silent --show-error --fail "$url" >/dev/null 2>&1; then
      echo "==> Kageos API (${url}) 就绪"
      return 0
    fi
    echo "    等待 Kageos API (${url}) ... ($i/90)"
    sleep 2
    i=$((i + 1))
  done
  echo "ERROR: 超时未连上 Kageos API ${url}" >&2
  return 1
}

print_aio_success_summary() {
  local data_dir="${KAGEOS_AIO_DATA_DIR:-/var/lib/kageos}"
  local secrets_dir="${data_dir}/secrets"
  local show_secrets="${KAGEOS_AIO_PRINT_SECRETS:-1}"
  local system_password mysql_password minio_password nats_password jwt_secret app_db_secret

  if [[ "$show_secrets" == "0" || "$show_secrets" == "false" ]]; then
    system_password="(hidden; run: docker exec kageos cat ${secrets_dir}/SYSTEM_USER_PASSWORD)"
    mysql_password="(hidden; ${secrets_dir}/MYSQL_ROOT_PASSWORD)"
    minio_password="(hidden; ${secrets_dir}/MINIO_ROOT_PASSWORD)"
    nats_password="(hidden; ${secrets_dir}/NATS_PASSWORD)"
    jwt_secret="(hidden; ${secrets_dir}/JWT_SECRET)"
    app_db_secret="(hidden; ${secrets_dir}/KAGEOS_APP_DB_SECRET_KEY)"
  else
    system_password="$(aio_secret_value SYSTEM_USER_PASSWORD)"
    mysql_password="$(aio_secret_value MYSQL_ROOT_PASSWORD)"
    minio_password="$(aio_secret_value MINIO_ROOT_PASSWORD)"
    nats_password="$(aio_secret_value NATS_PASSWORD)"
    jwt_secret="$(aio_secret_value JWT_SECRET)"
    app_db_secret="$(aio_secret_value KAGEOS_APP_DB_SECRET_KEY)"
  fi

  cat <<EOF

============================================================
Kageos started successfully
============================================================
Access URL:
  ${CANONICAL_BASE_URL}

Login:
  Username: system
  Password: ${system_password}

Data:
  Data directory: ${data_dir}
  Secrets directory: ${secrets_dir}
  System password file: ${secrets_dir}/SYSTEM_USER_PASSWORD

Internal services:
  MySQL:
    Host: ${MYSQL_HOST}
    Port: ${MYSQL_PORT}
    Username: root
    Password: ${mysql_password}
  MinIO:
    API: http://${MINIO_HOST}:${MINIO_PORT}
    Console: http://${MINIO_HOST}:${MINIO_CONSOLE_PORT}
    Username: ${MINIO_ROOT_USER}
    Password: ${minio_password}
  NATS:
    URL: nats://${NATS_USER}:<password>@${NATS_HOST}:${NATS_PORT}
    Username: ${NATS_USER}
    Password: ${nats_password}

Runtime secrets:
  JWT_SECRET: ${jwt_secret}
  KAGEOS_APP_DB_SECRET_KEY: ${app_db_secret}

Useful commands:
  docker logs -f kageos
  docker exec kageos cat ${secrets_dir}/SYSTEM_USER_PASSWORD
============================================================

EOF
}

case "$HTTP_PORT" in
  ''|*[!0-9]*)
    echo "ERROR: HTTP_PORT 必须是数字端口，当前值: ${HTTP_PORT}" >&2
    exit 1
    ;;
esac
if [ "$HTTP_PORT" -lt 1 ] || [ "$HTTP_PORT" -gt 65535 ]; then
  echo "ERROR: HTTP_PORT 必须在 1-65535 之间，当前值: ${HTTP_PORT}" >&2
  exit 1
fi
case "$HTTPS_PORT" in
  ''|*[!0-9]*)
    echo "ERROR: HTTPS_PORT 必须是数字端口，当前值: ${HTTPS_PORT}" >&2
    exit 1
    ;;
esac
if [ "$HTTPS_PORT" -lt 1 ] || [ "$HTTPS_PORT" -gt 65535 ]; then
  echo "ERROR: HTTPS_PORT 必须在 1-65535 之间，当前值: ${HTTPS_PORT}" >&2
  exit 1
fi

echo "==> 等待依赖（MySQL / NATS / MinIO）..."
wait_tcp "$MYSQL_HOST" "$MYSQL_PORT" "MySQL"
wait_tcp "$NATS_HOST" "$NATS_PORT" "NATS"
wait_tcp "$MINIO_HOST" "$MINIO_PORT" "MinIO"

PROD_TEMPLATE_VARS='${CANONICAL_BASE_URL} ${MYSQL_HOST} ${MYSQL_PORT} ${MYSQL_ROOT_PASSWORD} ${KAGEOS_APP_DB_SECRET_KEY} ${KAGEOS_APP_DB_CLUSTER_KEY} ${JWT_SECRET} ${SYSTEM_USER_PASSWORD} ${MINIO_HOST} ${MINIO_PORT} ${MINIO_ROOT_USER} ${MINIO_ROOT_PASSWORD} ${NATS_URL} ${KAGEOS_REGISTRATION_MODE} ${SMTP_MODE} ${SMTP_HOST} ${SMTP_PORT} ${SMTP_USERNAME} ${SMTP_PASSWORD} ${SMTP_FROM} ${SMTP_FROM_NAME} ${KAGEOS_APP_BASE_IMAGE}'
render_runtime_templates "$PROD_TEMPLATE_VARS"

CANONICAL_BASE_URL="${CANONICAL_BASE_URL}"
export CANONICAL_BASE_URL
export CANONICAL_SCHEME
export CANONICAL_HOST
export CANONICAL_SERVER_NAME
CANONICAL_SCHEME=$(echo "$CANONICAL_BASE_URL" | sed -E 's|^(https?).*|\1|')
CANONICAL_HOST=$(echo "$CANONICAL_BASE_URL" | sed -E 's|^https?://([^/]+).*|\1|')
CANONICAL_SERVER_NAME=$(echo "$CANONICAL_HOST" | sed -E 's|:[0-9]+$||')
export CANONICAL_SCHEME
export CANONICAL_HOST
export CANONICAL_SERVER_NAME

case "$TLS_MODE" in
  http|https|redirect|external) ;;
  *)
    echo "ERROR: TLS_MODE 仅支持 http / https / redirect / external，当前值: ${TLS_MODE}" >&2
    exit 1
    ;;
esac

NGINX_TEMPLATE="/app/deploy/prod/nginx/default.conf.template"
NGINX_MODE_DESC="${HTTP_PORT} HTTP"

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
    NGINX_MODE_DESC="${HTTP_PORT} -> ${HTTPS_PORT} 重定向 + ${HTTPS_PORT} HTTPS"
  else
    NGINX_TEMPLATE="/app/deploy/prod/nginx/default.https.conf.template"
    NGINX_MODE_DESC="${HTTP_PORT} HTTP + ${HTTPS_PORT} HTTPS"
  fi
elif [[ "$TLS_MODE" == "external" ]]; then
  if [[ "$CANONICAL_SCHEME" != "https" ]]; then
    echo "WARN: TLS_MODE=external 通常建议配合 https:// 的 CANONICAL_BASE_URL；当前为 ${CANONICAL_BASE_URL}"
  fi
  NGINX_MODE_DESC="${HTTP_PORT} HTTP（外部 TLS 终止）"
elif [[ "$CANONICAL_SCHEME" == "https" ]]; then
  echo "WARN: TLS_MODE=http 但 CANONICAL_BASE_URL 使用 https://；如果前面有 TLS 终止，请改成 TLS_MODE=external"
fi

mkdir -p /etc/nginx/snippets
envsubst '${MINIO_HOST} ${MINIO_PORT}' < /app/deploy/prod/nginx/common.server.inc > /etc/nginx/snippets/kageos-common.conf

export TLS_CERT_FILE
export TLS_KEY_FILE
echo "==> 生成 Nginx（${NGINX_MODE_DESC}，www → 裸域 301）canonical_host=${CANONICAL_HOST} server_name=${CANONICAL_SERVER_NAME} scheme=${CANONICAL_SCHEME}"
envsubst '${CANONICAL_HOST} ${CANONICAL_SERVER_NAME} ${CANONICAL_SCHEME} ${HTTP_PORT} ${HTTPS_PORT} ${TLS_CERT_FILE} ${TLS_KEY_FILE}' < "${NGINX_TEMPLATE}" > /etc/nginx/sites-enabled/default
nginx -t

echo "==> 启动 Nginx（监听 ${NGINX_MODE_DESC}）..."
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

if ! podman image exists "${KAGEOS_APP_BASE_IMAGE}" 2>/dev/null; then
  echo "ERROR: 未找到用户应用基础镜像 ${KAGEOS_APP_BASE_IMAGE}" >&2
  echo "ERROR: 请先在宿主机执行 go run ./cmd/kagectl up" >&2
  exit 1
fi
echo "==> 用户应用基础镜像已就绪: ${KAGEOS_APP_BASE_IMAGE}"

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

if is_aio_bundle; then
  if ! wait_core_ready; then
    kill -TERM "$CORE_PID" 2>/dev/null || true
    kill -TERM "$PODMAN_PID" 2>/dev/null || true
    nginx -s quit 2>/dev/null || true
    wait "$CORE_PID" 2>/dev/null || true
    exit 1
  fi
  print_aio_success_summary
fi

wait -n "$CORE_PID"
echo "==> core-server 退出，关闭中..."
shutdown
