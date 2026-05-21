#!/bin/bash
set -euo pipefail

cd /app

require_env() {
  local name="$1"
  local message="${2:-环境变量 ${name} 未设置或为空}"
  if [[ -z "${!name:-}" ]]; then
    echo "ERROR: ${message}" >&2
    exit 1
  fi
}

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
  echo "ERROR: 超时未连上 ${label} ${host}:${port}" >&2
  exit 1
}

wait_http() {
  local url="$1" label="$2"
  local i=1
  while [ "$i" -le 90 ]; do
    if curl --silent --show-error --fail "$url" >/dev/null 2>&1; then
      echo "==> ${label} (${url}) 就绪"
      return 0
    fi
    echo "    等待 ${label} (${url}) ... ($i/90)"
    sleep 2
    i=$((i + 1))
  done
  echo "ERROR: 超时未连上 ${label} ${url}" >&2
  exit 1
}

ensure_main_runtime_dirs() {
  mkdir -p \
    /app/logs \
    /app/namespace \
    /app/data/runtime/app-runtime \
    /app/data/tmp
}

set_smtp_defaults() {
  SMTP_MODE="${SMTP_MODE:-smtp}"
  SMTP_HOST="${SMTP_HOST:-smtp.qq.com}"
  SMTP_PORT="${SMTP_PORT:-587}"
  SMTP_USERNAME="${SMTP_USERNAME-}"
  SMTP_PASSWORD="${SMTP_PASSWORD-}"
  SMTP_FROM="${SMTP_FROM-}"
  SMTP_FROM_NAME="${SMTP_FROM_NAME:-Kageos}"
}

render_runtime_templates() {
  local template_vars="$1"
  echo "==> 渲染 deploy/prod/config/runtime 模板..."
  rm -rf /app/deploy/prod/config/runtime
  mkdir -p /app/deploy/prod/config/runtime
  for src in /app/config.prod.template/*.yaml; do
    local dst="/app/deploy/prod/config/runtime/$(basename "$src")"
    envsubst "$template_vars" < "$src" > "$dst"
  done
}

render_runtime_template_file() {
  local template_vars="$1"
  local template_name="$2"
  local output_name="$3"
  mkdir -p /app/deploy/prod/config/runtime
  envsubst "$template_vars" < "/app/config.prod.template/${template_name}" > "/app/deploy/prod/config/runtime/${output_name}"
}
