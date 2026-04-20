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
AOS_SCHEDULER_HEALTH_PORT="${AOS_SCHEDULER_HEALTH_PORT:-9098}"
export APP_BASE_IMAGE
export AOS_SCHEDULER_HEALTH_PORT

echo "==> 等待依赖（MySQL / NATS / app-runtime）..."
wait_tcp 127.0.0.1 3306 "MySQL"
wait_tcp 127.0.0.1 4222 "NATS"
wait_http "http://127.0.0.1:9093/health" "app-runtime"

PROD_TEMPLATE_VARS='${MYSQL_ROOT_PASSWORD} ${JWT_SECRET} ${CONTROL_ENC_KEY} ${MINIO_ROOT_PASSWORD} ${SMTP_HOST} ${SMTP_PORT} ${SMTP_USERNAME} ${SMTP_PASSWORD} ${SMTP_FROM} ${SMTP_FROM_NAME} ${APP_BASE_IMAGE} ${AOS_SCHEDULER_HEALTH_PORT}'
render_runtime_templates "$PROD_TEMPLATE_VARS"

shutdown() {
  echo "==> 停止 scheduler..."
  rm -f /run/app-scheduler.pid
  rm -f /app/logs/app-scheduler.heartbeat
  kill -TERM "$SCHEDULER_PID" 2>/dev/null || true
  wait "$SCHEDULER_PID" 2>/dev/null || true
  exit 0
}
trap shutdown SIGTERM SIGINT

echo "==> 启动 app-scheduler..."
rm -f /run/app-scheduler.pid
rm -f /app/logs/app-scheduler.heartbeat
/app/app-scheduler &
SCHEDULER_PID=$!
printf '%s\n' "$SCHEDULER_PID" > /run/app-scheduler.pid

wait -n "$SCHEDULER_PID"
echo "==> app-scheduler 退出，关闭中..."
shutdown
