#!/bin/bash
set -euo pipefail

source /app/entrypoint-common.sh

require_env BACKUP_BASIC_AUTH_PASSWORD

ensure_backup_runtime_dirs

render_runtime_template_file \
  '${MYSQL_ROOT_PASSWORD} ${MINIO_ROOT_PASSWORD} ${BACKUP_BASIC_AUTH_PASSWORD}' \
  'backup-service.yaml' \
  'backup-service.yaml'

MYSQL_HOST="${MYSQL_HOST:-mysql}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MINIO_HOST="${MINIO_HOST:-minio}"
MINIO_PORT="${MINIO_PORT:-9000}"
export MYSQL_HOST
export MYSQL_PORT
export MINIO_HOST
export MINIO_PORT

echo "==> 等待依赖（MySQL / MinIO）..."
wait_tcp "$MYSQL_HOST" "$MYSQL_PORT" "MySQL"
wait_tcp "$MINIO_HOST" "$MINIO_PORT" "MinIO"

exec /app/backup-server
