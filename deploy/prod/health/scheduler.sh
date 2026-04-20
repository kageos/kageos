#!/usr/bin/env bash
set -euo pipefail

AOS_SCHEDULER_HEALTH_PORT="${AOS_SCHEDULER_HEALTH_PORT:-9098}"
curl --silent --show-error --fail "http://127.0.0.1:${AOS_SCHEDULER_HEALTH_PORT}/health" >/dev/null
