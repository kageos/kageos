#!/bin/bash
set -euo pipefail

source /app/entrypoint-common.sh

require_env STORAGE_ROOT
require_env BACKUP_BASIC_AUTH_USERNAME
require_env BACKUP_BASIC_AUTH_PASSWORD

ensure_backup_runtime_dirs

render_runtime_template_file \
  '${STORAGE_ROOT} ${MYSQL_ROOT_PASSWORD} ${MINIO_ROOT_USER} ${MINIO_ROOT_PASSWORD} ${BACKUP_BASIC_AUTH_USERNAME} ${BACKUP_BASIC_AUTH_PASSWORD}' \
  'backup-service.yaml' \
  'backup-service.yaml'

exec /app/backup-server
