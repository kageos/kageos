#!/usr/bin/env bash
# 生产主站部署脚本：直接读取 .env，统一使用 STORAGE_ROOT 宿主机目录
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"
PROJECT_ROOT="$(cd "$ROOT/../.." && pwd)"
SCRIPTS_DIR="$ROOT/scripts"

ENV_FILE="$ROOT/.env"
COMMAND="${1:-up}"
ARG1="${2:-}"
STORAGE_ROOT=""
ENABLE_HTTPS="0"
HTTPS_REDIRECT="0"
TLS_CERTS_HOST_DIR="./certs"
TLS_CERT_FILE="/app/tls/fullchain.pem"
TLS_KEY_FILE="/app/tls/privkey.pem"
APP_BASE_IMAGE="agentos-app-runtime-base:latest"

source "$SCRIPTS_DIR/lib.sh"
source "$SCRIPTS_DIR/validate.sh"
source "$SCRIPTS_DIR/health.sh"
source "$SCRIPTS_DIR/commands.sh"

dispatch_command "$COMMAND"
