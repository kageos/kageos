#!/usr/bin/env bash
# 兼容入口。Canonical 脚本已迁到 deploy/base/scripts/build-app-base-image.sh

set -e
cd "$(dirname "$0")/.."
exec bash deploy/base/scripts/build-app-base-image.sh
