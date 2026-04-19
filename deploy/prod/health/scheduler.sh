#!/usr/bin/env bash
set -euo pipefail

PID_FILE="/run/app-scheduler.pid"
HEARTBEAT_FILE="/app/logs/app-scheduler.heartbeat"
MAX_AGE_SECONDS="${AOS_SCHEDULER_HEARTBEAT_MAX_AGE_SECONDS:-30}"

test -s "${PID_FILE}"
kill -0 "$(cat "${PID_FILE}")" >/dev/null 2>&1

test -s "${HEARTBEAT_FILE}"
last_heartbeat="$(cat "${HEARTBEAT_FILE}")"
case "${last_heartbeat}" in
  ''|*[!0-9]*)
    exit 1
    ;;
esac

now_ts="$(date +%s)"
age_seconds=$((now_ts - last_heartbeat))
if [ "${age_seconds}" -lt 0 ]; then
  age_seconds=0
fi

[ "${age_seconds}" -le "${MAX_AGE_SECONDS}" ]
