#!/usr/bin/env bash
set -euo pipefail

/app/health/edge.sh
/app/health/platform.sh
/app/health/runtime.sh
