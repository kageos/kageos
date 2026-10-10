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

kageos_now_seconds() {
  date +%s
}

kageos_report_duration() {
  local label="$1"
  local started_at="$2"
  local status="${3:-success}"
  local finished_at elapsed
  finished_at="$(kageos_now_seconds)"
  elapsed=$((finished_at - started_at))
  echo "==> [耗时] ${label}: ${elapsed}s (${status})"
}

kageos_run_timed_stage() {
  local label="$1"
  local started_at status
  shift
  started_at="$(kageos_now_seconds)"
  if "$@"; then
    kageos_report_duration "$label" "$started_at" success
    return 0
  else
    status=$?
    kageos_report_duration "$label" "$started_at" failed >&2
    return "$status"
  fi
}

# Load an installer-exported archive into this instance's isolated store.
kageos_load_cached_image() {
  local image="$1" key archive
  podman image exists "$image" 2>/dev/null && return 0
  # DaoCloud changes the transport prefix, not the upstream image/version.
  if [[ "$image" == m.daocloud.io/* ]] && podman image exists "${image#m.daocloud.io/}" 2>/dev/null; then
    podman tag "${image#m.daocloud.io/}" "$image" || return $?
    return 0
  fi
  key="$(printf '%s' "$image" | sha256sum | awk '{print $1}')"
  archive="${KAGEOS_AIO_IMAGE_CACHE_DIR:-/var/cache/kageos-images}/${key}.tar"
  [[ -s "$archive" ]] || return 1
  echo "==> 加载共享镜像缓存: ${image}"
  podman load -i "$archive" || return $?
  if [[ "$image" == m.daocloud.io/* ]] && ! podman image exists "$image" 2>/dev/null; then
    podman image exists "${image#m.daocloud.io/}" 2>/dev/null || return 1
    podman tag "${image#m.daocloud.io/}" "$image" || return $?
  fi
  podman image exists "$image"
}

kageos_pull_with_progress() {
  local command
  printf -v command 'podman pull %q' "$1"
  # util-linux script preserves the pull exit status and enables byte bars.
  script -q -e -c "$command" /dev/null </dev/null
}

kageos_ensure_image() {
  local image="$1"
  if kageos_load_cached_image "$image"; then
    echo "==> 复用本地镜像缓存: ${image}"
    return 0
  fi
  echo "==> 下载镜像: ${image}"
  kageos_pull_with_progress "$image" || return $?
}

wait_endpoint() {
  local label="$1" timeout_seconds="${KAGEOS_DEPENDENCY_READY_TIMEOUT:-180}" started=$SECONDS elapsed next_report=0
  shift
  [[ "$timeout_seconds" =~ ^[1-9][0-9]*$ ]] || {
    echo "ERROR: KAGEOS_DEPENDENCY_READY_TIMEOUT 必须为正整数秒" >&2
    return 1
  }
  while (( SECONDS - started < timeout_seconds )); do
    if "$@" >/dev/null 2>&1; then
      echo "==> ${label} 就绪"
      return 0
    fi
    elapsed=$((SECONDS - started))
    if (( elapsed >= next_report )); then
      echo "==> 等待 ${label} ... (${elapsed}/${timeout_seconds}s)"
      next_report=$((elapsed + 10))
    fi
    sleep 2
  done
  echo "ERROR: ${timeout_seconds}s 内未连上 ${label}" >&2
  return 1
}

wait_tcp() {
  local host="$1" port="$2" label="$3"
  wait_endpoint "${label} (${host}:${port})" timeout 4 nc -z -w 3 "$host" "$port"
}

wait_http() {
  local url="$1" label="$2"
  wait_endpoint "${label} (${url})" curl --silent --fail --connect-timeout 2 --max-time 5 "$url"
}

ensure_main_runtime_dirs() {
  mkdir -p \
    /app/logs \
    /app/namespace \
    /app/data/runtime/app-runtime \
    /app/data/tmp
}

set_smtp_defaults() {
  KAGEOS_REGISTRATION_MODE="${KAGEOS_REGISTRATION_MODE:-admin_only}"
  SMTP_MODE="${SMTP_MODE:-smtp}"
  SMTP_HOST="${SMTP_HOST:-smtp.qq.com}"
  SMTP_PORT="${SMTP_PORT:-587}"
  SMTP_USERNAME="${SMTP_USERNAME-}"
  SMTP_PASSWORD="${SMTP_PASSWORD-}"
  SMTP_FROM="${SMTP_FROM-}"
  SMTP_FROM_NAME="${SMTP_FROM_NAME:-kageos}"
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
