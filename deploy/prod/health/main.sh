#!/usr/bin/env bash
set -euo pipefail

check_http() {
  local port="$1"
  curl --silent --show-error --fail "http://127.0.0.1:${port}/health" >/dev/null
}

check_http 9090
check_http 9091
check_http 9092
check_http 9095
check_http 9096
check_http 9097
check_http 9109

curl --silent --show-error --fail "http://127.0.0.1:9093/health" >/dev/null
test -S /run/podman/podman.sock
podman info >/dev/null 2>&1
podman image exists "${APP_BASE_IMAGE:-localhost/agentos-app-runtime-base:latest}" >/dev/null 2>&1
