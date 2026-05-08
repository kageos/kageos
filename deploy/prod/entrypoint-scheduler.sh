#!/bin/bash
set -euo pipefail

source /app/entrypoint-common.sh

ensure_main_runtime_dirs

require_env MYSQL_ROOT_PASSWORD "环境变量 MYSQL_ROOT_PASSWORD 未设置或为空（须由 Compose 从宿主机 .env 注入）"
require_env MINIO_ROOT_PASSWORD "环境变量 MINIO_ROOT_PASSWORD 未设置或为空（须由 Compose 从宿主机 .env 注入）"
require_env JWT_SECRET "环境变量 JWT_SECRET 未设置或为空（须由 Compose 从宿主机 .env 注入）"
require_env CONTROL_ENC_KEY "环境变量 CONTROL_ENC_KEY 未设置或为空（须由 Compose 从宿主机 .env 注入）"

set_smtp_defaults
APP_BASE_IMAGE="${APP_BASE_IMAGE:-localhost/agentos-app-runtime-base:latest}"
MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
NATS_HOST="${NATS_HOST:-127.0.0.1}"
NATS_PORT="${NATS_PORT:-4222}"
NATS_USER="${NATS_USER:-${NATS_SEED_USER:-aos}}"
NATS_PASSWORD="${NATS_PASSWORD:-${NATS_SEED_PASSWORD:-}}"
NATS_URL="${NATS_URL:-nats://${NATS_USER}:${NATS_PASSWORD}@${NATS_HOST}:${NATS_PORT}}"
export APP_BASE_IMAGE
export MYSQL_HOST
export MYSQL_PORT
export NATS_HOST
export NATS_PORT
export NATS_USER
export NATS_PASSWORD
export NATS_URL

echo "==> 等待依赖（MySQL / NATS）..."
wait_tcp "$MYSQL_HOST" "$MYSQL_PORT" "MySQL"
wait_tcp "$NATS_HOST" "$NATS_PORT" "NATS"

PROD_TEMPLATE_VARS='${MYSQL_ROOT_PASSWORD} ${JWT_SECRET} ${CONTROL_ENC_KEY} ${MINIO_ROOT_PASSWORD} ${NATS_URL} ${SMTP_HOST} ${SMTP_PORT} ${SMTP_USERNAME} ${SMTP_PASSWORD} ${SMTP_FROM} ${SMTP_FROM_NAME} ${APP_BASE_IMAGE}'
render_runtime_templates "$PROD_TEMPLATE_VARS"

shutdown() {
  echo "==> 停止 timer-scheduler..."
  rm -f /run/timer-scheduler.pid
  kill -TERM "$SCHEDULER_PID" 2>/dev/null || true
  wait "$SCHEDULER_PID" 2>/dev/null || true
  exit 0
}
trap shutdown SIGTERM SIGINT

echo "==> 启动 timer-scheduler..."
rm -f /run/timer-scheduler.pid
/app/timer-scheduler &
SCHEDULER_PID=$!
printf '%s\n' "$SCHEDULER_PID" > /run/timer-scheduler.pid

wait -n "$SCHEDULER_PID"
echo "==> timer-scheduler 退出，关闭中..."
shutdown
