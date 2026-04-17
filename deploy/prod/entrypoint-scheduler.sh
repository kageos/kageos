#!/bin/bash
set -euo pipefail

source /app/entrypoint-common.sh

ensure_main_runtime_dirs

require_env MYSQL_ROOT_PASSWORD "环境变量 MYSQL_ROOT_PASSWORD 未设置或为空（须由 Compose 从宿主机 .env 注入）"
require_env MINIO_ROOT_PASSWORD "环境变量 MINIO_ROOT_PASSWORD 未设置或为空（须由 Compose 从宿主机 .env 注入）"
require_env JWT_SECRET "环境变量 JWT_SECRET 未设置或为空（须由 Compose 从宿主机 .env 注入）"
require_env CONTROL_ENC_KEY "环境变量 CONTROL_ENC_KEY 未设置或为空（须由 Compose 从宿主机 .env 注入）"

set_smtp_defaults

echo "==> 等待依赖（MySQL / NATS / API Gateway）..."
wait_tcp 127.0.0.1 3306 "MySQL"
wait_tcp 127.0.0.1 4222 "NATS"
wait_tcp 127.0.0.1 9090 "API Gateway"

PROD_TEMPLATE_VARS='${MYSQL_ROOT_PASSWORD} ${JWT_SECRET} ${CONTROL_ENC_KEY} ${MINIO_ROOT_PASSWORD} ${SMTP_HOST} ${SMTP_PORT} ${SMTP_USERNAME} ${SMTP_PASSWORD} ${SMTP_FROM} ${SMTP_FROM_NAME} ${APP_BASE_IMAGE}'
render_runtime_templates "$PROD_TEMPLATE_VARS"

shutdown() {
  echo "==> 停止 scheduler..."
  kill -TERM "$SCHEDULER_PID" 2>/dev/null || true
  wait "$SCHEDULER_PID" 2>/dev/null || true
  exit 0
}
trap shutdown SIGTERM SIGINT

echo "==> 启动 app-scheduler..."
/app/app-scheduler &
SCHEDULER_PID=$!

wait -n "$SCHEDULER_PID"
echo "==> app-scheduler 退出，关闭中..."
shutdown
