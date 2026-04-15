#!/usr/bin/env bash
set -euo pipefail

curl --silent --show-error --fail "http://127.0.0.1:19088/health" >/dev/null
