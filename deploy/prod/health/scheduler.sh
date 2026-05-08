#!/usr/bin/env bash
set -euo pipefail

TIMER_SCHEDULER_PORT="${TIMER_SCHEDULER_PORT:-9108}"
curl --silent --show-error --fail "http://127.0.0.1:${TIMER_SCHEDULER_PORT}/health" >/dev/null
