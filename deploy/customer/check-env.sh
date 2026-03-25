#!/usr/bin/env bash
# 校验 deploy/customer/.env 必填项（纯 bash，无 Python）
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
ENV_FILE="${1:-$ROOT/.env}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "ERROR: 未找到 $ENV_FILE"
  echo "请执行: cp .env.example .env 并填写全部必填项"
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

fail=0
need_non_empty() {
  local k="$1"
  local v="${!k-}"
  if [[ -z "$v" ]]; then
    echo "ERROR: 必填项为空或未设置: $k"
    fail=1
  fi
}

need_non_empty CANONICAL_BASE_URL
need_non_empty MYSQL_ROOT_PASSWORD
need_non_empty MINIO_ROOT_USER
need_non_empty MINIO_ROOT_PASSWORD
need_non_empty JWT_SECRET
need_non_empty CONTROL_ENC_KEY
need_non_empty HTTP_PUBLISH_PORT
need_non_empty MAIN_IMAGE
# SMTP_PASSWORD 允许为空

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi

echo "OK: $ENV_FILE 必填项已齐（SMTP_PASSWORD 可为空）"
