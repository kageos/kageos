#!/usr/bin/env bash
set -euo pipefail

test -s /run/app-scheduler.pid
kill -0 "$(cat /run/app-scheduler.pid)" >/dev/null 2>&1
