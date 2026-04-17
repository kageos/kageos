#!/bin/bash
set -euo pipefail

source /app/entrypoint-common.sh

require_env BACKUP_BASIC_AUTH_PASSWORD

ensure_backup_runtime_dirs

render_runtime_template_file \
  '${MYSQL_ROOT_PASSWORD} ${MINIO_ROOT_PASSWORD} ${BACKUP_BASIC_AUTH_PASSWORD}' \
  'backup-service.yaml' \
  'backup-service.yaml'

exec /app/backup-server
