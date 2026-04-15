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

nc -z 127.0.0.1 9093
test -S /run/podman/podman.sock
podman info >/dev/null 2>&1
