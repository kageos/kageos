#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_PATH="${AOS_CONFIG:-deploy/prod/aos.yaml}"
PID_FILE="${AOS_UP_PID_FILE:-deploy/prod/aosctl-up.pid}"
LOG_FILE="${AOS_STOP_LOG:-deploy/prod/aosctl-stop.log}"

resolve_path() {
  local path="$1"
  if [[ "$path" = /* ]]; then
    printf '%s\n' "$path"
  else
    printf '%s/%s\n' "$ROOT_DIR" "$path"
  fi
}

CONFIG_ABS="$(resolve_path "$CONFIG_PATH")"
PID_ABS="$(resolve_path "$PID_FILE")"
LOG_ABS="$(resolve_path "$LOG_FILE")"

cd "$ROOT_DIR"
mkdir -p "$(dirname "$LOG_ABS")"
exec > >(tee -a "$LOG_ABS") 2>&1

printf '\n===== %s prod stop start =====\n' "$(date '+%Y-%m-%d %H:%M:%S %z')"

stop_background_up() {
  if [[ ! -f "$PID_ABS" ]]; then
    return
  fi

  local pid
  pid="$(tr -d '[:space:]' < "$PID_ABS" || true)"
  rm -f "$PID_ABS"

  if [[ ! "$pid" =~ ^[0-9]+$ ]]; then
    echo "ignored invalid pid file: $PID_FILE"
    return
  fi

  if ! kill -0 "$pid" 2>/dev/null; then
    echo "background prod deploy process is not running: PID $pid"
    return
  fi

  echo "stopping background prod deploy process: PID $pid"
  if ! kill -TERM "-$pid" 2>/dev/null; then
    kill -TERM "$pid" 2>/dev/null || true
  fi

  for _ in {1..10}; do
    if ! kill -0 "$pid" 2>/dev/null; then
      echo "background prod deploy process stopped"
      return
    fi
    sleep 1
  done

  echo "background prod deploy process did not stop after 10s; force killing: PID $pid"
  if ! kill -KILL "-$pid" 2>/dev/null; then
    kill -KILL "$pid" 2>/dev/null || true
  fi
}

stop_background_up

if [[ ! -f "$CONFIG_ABS" ]]; then
  echo "ERROR: prod config not found: $CONFIG_PATH"
  exit 1
fi

echo "running: go run ./cmd/aosctl down --config $CONFIG_PATH"
go run ./cmd/aosctl down --config "$CONFIG_PATH"
echo "prod services stopped"
