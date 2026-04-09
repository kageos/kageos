#!/bin/bash
set -euo pipefail

cd /app

require_env() {
  local n="$1"
  if [[ -z "${!n:-}" ]]; then
    echo "ERROR: 环境变量 ${n} 未设置或为空" >&2
    exit 1
  fi
}

require_env STORAGE_ROOT
require_env BACKUP_BASIC_AUTH_USERNAME
require_env BACKUP_BASIC_AUTH_PASSWORD

mkdir -p \
  /app/logs \
  /app/data/backup/repo \
  /app/data/backup/state \
  /app/data/backup/staging \
  /app/data/tmp

mkdir -p /app/deploy/prod/config/runtime
envsubst '${STORAGE_ROOT} ${MYSQL_ROOT_PASSWORD} ${MINIO_ROOT_USER} ${MINIO_ROOT_PASSWORD} ${BACKUP_BASIC_AUTH_USERNAME} ${BACKUP_BASIC_AUTH_PASSWORD}' < /app/config.prod.template/backup-service.yaml > /app/deploy/prod/config/runtime/backup-service.yaml

exec /app/backup-server
