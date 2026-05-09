#!/usr/bin/env bash
set -euo pipefail

nginx -t >/dev/null

pid_file="/run/nginx.pid"
test -s "$pid_file"
kill -0 "$(cat "$pid_file")"

