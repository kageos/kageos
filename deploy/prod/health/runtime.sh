#!/usr/bin/env bash
set -euo pipefail

curl --silent --show-error --fail "http://127.0.0.1:9093/health" >/dev/null
test -S /run/podman/podman.sock
podman info >/dev/null 2>&1
podman image exists "${KAGEOS_APP_BASE_IMAGE:-kagebase:latest}" >/dev/null 2>&1
