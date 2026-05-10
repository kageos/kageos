#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_PATH="${AOS_CONFIG:-deploy/prod/aos.yaml}"
LOG_FILE="${AOS_UP_LOG:-deploy/prod/aosctl-up.log}"
PID_FILE="${AOS_UP_PID_FILE:-deploy/prod/aosctl-up.pid}"

resolve_path() {
  local path="$1"
  if [[ "$path" = /* ]]; then
    printf '%s\n' "$path"
  else
    printf '%s/%s\n' "$ROOT_DIR" "$path"
  fi
}

CONFIG_ABS="$(resolve_path "$CONFIG_PATH")"
LOG_ABS="$(resolve_path "$LOG_FILE")"
PID_ABS="$(resolve_path "$PID_FILE")"

cd "$ROOT_DIR"

warn_rootless_podman_linger() {
  if [[ "${AOS_SKIP_LINGER_CHECK:-0}" == "1" ]]; then
    return
  fi
  if [[ "${EUID:-$(id -u)}" -eq 0 ]]; then
    return
  fi
  if ! command -v podman >/dev/null 2>&1 || ! podman compose version >/dev/null 2>&1; then
    return
  fi
  if ! command -v loginctl >/dev/null 2>&1; then
    return
  fi

  local user_name linger
  user_name="$(id -un)"
  linger="$(loginctl show-user "$user_name" -p Linger --value 2>/dev/null || true)"
  if [[ "$linger" != "yes" ]]; then
    echo "WARNING: rootless podman compose is available, but systemd linger is not enabled for $user_name."
    echo "If services stop after SSH/session logout, run: sudo loginctl enable-linger $user_name"
    echo "Skip this check with: AOS_SKIP_LINGER_CHECK=1 ./prod-up.sh"
  fi
}

if [[ ! -f "$CONFIG_ABS" ]]; then
  echo "ERROR: prod config not found: $CONFIG_PATH"
  echo "Run first: go run ./cmd/aosctl init --base-url http://your-ip-or-domain"
  exit 1
fi

mkdir -p "$(dirname "$LOG_ABS")" "$(dirname "$PID_ABS")"
warn_rootless_podman_linger

if [[ -f "$PID_ABS" ]]; then
  old_pid="$(tr -d '[:space:]' < "$PID_ABS" || true)"
  if [[ "$old_pid" =~ ^[0-9]+$ ]] && kill -0 "$old_pid" 2>/dev/null; then
    echo "prod deploy is already running in background: PID $old_pid"
    echo "log: $LOG_ABS"
    echo "stop it with: ./prod-stop.sh"
    exit 0
  fi
  rm -f "$PID_ABS"
fi

{
  printf '\n===== %s prod up start =====\n' "$(date '+%Y-%m-%d %H:%M:%S %z')"
  printf 'root: %s\n' "$ROOT_DIR"
  printf 'config: %s\n' "$CONFIG_PATH"
  printf 'args:'
  printf ' %q' "$@"
  printf '\n\n'
} >> "$LOG_ABS"

launcher='
set -Eeuo pipefail
trap "" HUP
root_dir="$1"
config_path="$2"
shift 2
cd "$root_dir"
exec </dev/null
exec go run ./cmd/aosctl up --config "$config_path" "$@"
'

if command -v setsid >/dev/null 2>&1; then
  nohup setsid bash -c "$launcher" _ "$ROOT_DIR" "$CONFIG_PATH" "$@" </dev/null >> "$LOG_ABS" 2>&1 &
else
  nohup bash -c "$launcher" _ "$ROOT_DIR" "$CONFIG_PATH" "$@" </dev/null >> "$LOG_ABS" 2>&1 &
fi

pid="$!"
printf '%s\n' "$pid" > "$PID_ABS"
disown "$pid" 2>/dev/null || true

echo "prod deploy started in background: PID $pid"
echo "log: $LOG_ABS"
echo "tail log: tail -f $LOG_FILE"
echo "stop: ./prod-stop.sh"
