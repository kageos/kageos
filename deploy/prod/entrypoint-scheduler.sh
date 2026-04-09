#!/bin/bash
set -euo pipefail

cd /app

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

require_env STORAGE_ROOT
require_env MYSQL_ROOT_PASSWORD
require_env MINIO_ROOT_USER
require_env MINIO_ROOT_PASSWORD
require_env JWT_SECRET
require_env CONTROL_ENC_KEY

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

echo "==> 等待依赖（MySQL / NATS / API Gateway）..."
wait_tcp 127.0.0.1 3306 "MySQL"
wait_tcp 127.0.0.1 4222 "NATS"
wait_tcp 127.0.0.1 9090 "API Gateway"

PROD_TEMPLATE_VARS='${STORAGE_ROOT} ${MYSQL_ROOT_PASSWORD} ${JWT_SECRET} ${CONTROL_ENC_KEY} ${MINIO_ROOT_USER} ${MINIO_ROOT_PASSWORD} ${SMTP_HOST} ${SMTP_PORT} ${SMTP_USERNAME} ${SMTP_PASSWORD} ${SMTP_FROM} ${SMTP_FROM_NAME}'

echo "==> 渲染 deploy/prod/config/runtime 模板..."
rm -rf /app/deploy/prod/config/runtime
mkdir -p /app/deploy/prod/config/runtime
for src in /app/config.prod.template/*.yaml; do
  dst="/app/deploy/prod/config/runtime/$(basename "$src")"
  envsubst "$PROD_TEMPLATE_VARS" < "$src" > "$dst"
done

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
