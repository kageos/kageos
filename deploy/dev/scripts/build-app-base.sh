#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"

cd "$ROOT"
exec bash deploy/base/scripts/build-app-base-image.sh
