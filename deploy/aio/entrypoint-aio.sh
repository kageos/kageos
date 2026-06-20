#!/bin/bash
set -euo pipefail

source /app/entrypoint-common.sh

AIO_DATA_DIR="${KAGEOS_AIO_DATA_DIR:-/var/lib/kageos}"
AIO_INFRA_DIR="${AIO_DATA_DIR}/infra"
AIO_SECRETS_DIR="${AIO_DATA_DIR}/secrets"

MYSQL_CONTAINER_NAME="${KAGEOS_AIO_MYSQL_CONTAINER_NAME:-kageos-mysql}"
NATS_CONTAINER_NAME="${KAGEOS_AIO_NATS_CONTAINER_NAME:-kageos-nats}"
MINIO_CONTAINER_NAME="${KAGEOS_AIO_MINIO_CONTAINER_NAME:-kageos-minio}"

MYSQL_IMAGE="${KAGEOS_AIO_MYSQL_IMAGE:-docker.io/library/mysql:8.0}"
NATS_IMAGE="${KAGEOS_AIO_NATS_IMAGE:-docker.io/library/nats:2.10-alpine}"
MINIO_IMAGE="${KAGEOS_AIO_MINIO_IMAGE:-docker.io/minio/minio:RELEASE.2025-09-07T16-13-09Z}"

random_hex() {
  local bytes="${1:-32}"
  od -An -N"${bytes}" -tx1 /dev/urandom | tr -d ' \n'
}

load_or_create_secret() {
  local name="$1"
  local bytes="$2"
  local file="${AIO_SECRETS_DIR}/${name}"
  local env_value="${!name:-}"
  local value=""

  if [[ -s "$file" ]]; then
    value="$(tr -d '\r\n' < "$file")"
    if [[ -n "$env_value" && "$env_value" != "$value" ]]; then
      echo "WARN: ${name} 已在 ${file} 持久化，本次忽略不同的环境变量值"
    fi
  elif [[ -n "$env_value" ]]; then
    value="$env_value"
    printf '%s\n' "$value" > "$file"
    chmod 600 "$file"
  else
    value="$(random_hex "$bytes")"
    printf '%s\n' "$value" > "$file"
    chmod 600 "$file"
  fi

  export "${name}=${value}"
}

persist_runtime_dir() {
  local target="$1"
  local link="$2"

  mkdir -p "$target"
  mkdir -p "$(dirname "$link")"

  if [[ -L "$link" ]]; then
    return 0
  fi
  if [[ -e "$link" ]]; then
    cp -an "${link}/." "$target/" 2>/dev/null || true
    rm -rf "$link"
  fi
  ln -s "$target" "$link"
}

prepare_layout() {
  mkdir -p \
    "$AIO_DATA_DIR" \
    "$AIO_INFRA_DIR" \
    "$AIO_SECRETS_DIR" \
    "$AIO_DATA_DIR/mysql" \
    "$AIO_DATA_DIR/minio" \
    "$AIO_DATA_DIR/nats" \
    "$AIO_DATA_DIR/podman_storage" \
    "$AIO_DATA_DIR/logs" \
    "$AIO_DATA_DIR/namespace" \
    "$AIO_DATA_DIR/data/runtime/app-runtime" \
    "$AIO_DATA_DIR/data/tmp"

  chmod 700 "$AIO_SECRETS_DIR"

  persist_runtime_dir "$AIO_DATA_DIR/logs" /app/logs
  persist_runtime_dir "$AIO_DATA_DIR/namespace" /app/namespace
  persist_runtime_dir "$AIO_DATA_DIR/data" /app/data
  persist_runtime_dir "$AIO_DATA_DIR/podman_storage" /var/lib/containers/storage

  mkdir -p /run/podman /run/containers/storage
}

write_infra_files() {
  cat > "${AIO_INFRA_DIR}/mysql-init.sql" <<'SQL'
CREATE DATABASE IF NOT EXISTS `app-server` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `app-storage` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `agent-server` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `connector-server` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `hr-server` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `timer-scheduler` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `message-server` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
SQL

  cat > "${AIO_INFRA_DIR}/nats-server.conf" <<EOF
max_payload: 10485760
port: 4222
logtime: true
authorization {
  user: "${NATS_USER}"
  password: "${NATS_PASSWORD}"
}
EOF
}

ensure_podman() {
  if ! podman info >/tmp/kageos-aio-podman-info.log 2>&1; then
    echo "ERROR: AIO 镜像需要可用的容器内 Podman。请使用 --privileged 启动外层 Docker/Podman 容器。" >&2
    cat /tmp/kageos-aio-podman-info.log >&2 || true
    exit 1
  fi
}

podman_container_running() {
  local name="$1"
  [[ "$(podman inspect -f '{{.State.Running}}' "$name" 2>/dev/null || true)" == "true" ]]
}

maybe_recreate_container() {
  local name="$1"
  if [[ "${KAGEOS_AIO_RECREATE_INFRA:-0}" == "1" ]] && podman container exists "$name" 2>/dev/null; then
    echo "==> 重建 AIO 内置服务容器: ${name}"
    podman rm -f "$name" >/dev/null
  fi
}

start_mysql() {
  maybe_recreate_container "$MYSQL_CONTAINER_NAME"
  if podman_container_running "$MYSQL_CONTAINER_NAME"; then
    echo "==> MySQL 已运行: ${MYSQL_CONTAINER_NAME}"
  elif podman container exists "$MYSQL_CONTAINER_NAME" 2>/dev/null; then
    echo "==> 启动 MySQL: ${MYSQL_CONTAINER_NAME}"
    podman start "$MYSQL_CONTAINER_NAME" >/dev/null
  else
    echo "==> 创建并启动 MySQL: ${MYSQL_IMAGE}"
    podman run -d \
      --name "$MYSQL_CONTAINER_NAME" \
      --network host \
      -e MYSQL_ROOT_PASSWORD="$MYSQL_ROOT_PASSWORD" \
      -e TZ="${TZ:-Asia/Shanghai}" \
      -v "${AIO_DATA_DIR}/mysql:/var/lib/mysql" \
      -v "${AIO_INFRA_DIR}/mysql-init.sql:/docker-entrypoint-initdb.d/init.sql:ro" \
      "$MYSQL_IMAGE" \
      --character-set-server=utf8mb4 \
      --collation-server=utf8mb4_unicode_ci >/dev/null
  fi

  wait_tcp "127.0.0.1" "3306" "MySQL"
  for i in $(seq 1 90); do
    if podman exec "$MYSQL_CONTAINER_NAME" mysqladmin ping -h 127.0.0.1 -uroot -p"$MYSQL_ROOT_PASSWORD" --silent >/dev/null 2>&1; then
      echo "==> MySQL root 登录就绪"
      podman exec -i "$MYSQL_CONTAINER_NAME" mysql -uroot -p"$MYSQL_ROOT_PASSWORD" < "${AIO_INFRA_DIR}/mysql-init.sql"
      return 0
    fi
    echo "    等待 MySQL 初始化账号 ... (${i}/90)"
    sleep 2
  done
  echo "ERROR: MySQL 已监听但 root 登录未就绪" >&2
  exit 1
}

start_nats() {
  maybe_recreate_container "$NATS_CONTAINER_NAME"
  if podman_container_running "$NATS_CONTAINER_NAME"; then
    echo "==> NATS 已运行: ${NATS_CONTAINER_NAME}"
  elif podman container exists "$NATS_CONTAINER_NAME" 2>/dev/null; then
    echo "==> 启动 NATS: ${NATS_CONTAINER_NAME}"
    podman start "$NATS_CONTAINER_NAME" >/dev/null
  else
    echo "==> 创建并启动 NATS: ${NATS_IMAGE}"
    podman run -d \
      --name "$NATS_CONTAINER_NAME" \
      --network host \
      -v "${AIO_INFRA_DIR}/nats-server.conf:/etc/nats/nats-server.conf:ro" \
      "$NATS_IMAGE" \
      -c /etc/nats/nats-server.conf >/dev/null
  fi

  wait_tcp "127.0.0.1" "4222" "NATS"
}

start_minio() {
  maybe_recreate_container "$MINIO_CONTAINER_NAME"
  if podman_container_running "$MINIO_CONTAINER_NAME"; then
    echo "==> MinIO 已运行: ${MINIO_CONTAINER_NAME}"
  elif podman container exists "$MINIO_CONTAINER_NAME" 2>/dev/null; then
    echo "==> 启动 MinIO: ${MINIO_CONTAINER_NAME}"
    podman start "$MINIO_CONTAINER_NAME" >/dev/null
  else
    echo "==> 创建并启动 MinIO: ${MINIO_IMAGE}"
    podman run -d \
      --name "$MINIO_CONTAINER_NAME" \
      --network host \
      -e MINIO_ROOT_USER="${MINIO_ROOT_USER}" \
      -e MINIO_ROOT_PASSWORD="${MINIO_ROOT_PASSWORD}" \
      -e TZ="${TZ:-Asia/Shanghai}" \
      -v "${AIO_DATA_DIR}/minio:/data" \
      "$MINIO_IMAGE" \
      server /data --console-address ":9001" >/dev/null
  fi

  wait_tcp "127.0.0.1" "9000" "MinIO"
  wait_http "http://127.0.0.1:9000/minio/health/ready" "MinIO health"
}

export_defaults() {
  CANONICAL_BASE_URL="${CANONICAL_BASE_URL:-http://localhost:8080}"
  TLS_MODE="${TLS_MODE:-http}"
  HTTP_PORT="${HTTP_PORT:-80}"
  HTTPS_PORT="${HTTPS_PORT:-443}"
  KAGEOS_APP_BASE_IMAGE="${KAGEOS_APP_BASE_IMAGE:-${KAGEOS_DEFAULT_APP_BASE_IMAGE:-docker.io/qiayanai/kagebase:latest}}"

  MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
  MYSQL_PORT="${MYSQL_PORT:-3306}"
  MINIO_HOST="${MINIO_HOST:-127.0.0.1}"
  MINIO_PORT="${MINIO_PORT:-9000}"
  MINIO_ROOT_USER="${MINIO_ROOT_USER:-minioadmin}"
  NATS_USER="${NATS_USER:-${NATS_SEED_USER:-aos}}"
  NATS_SEED_USER="${NATS_SEED_USER:-$NATS_USER}"

  KAGEOS_APP_DB_CLUSTER_KEY="${KAGEOS_APP_DB_CLUSTER_KEY:-mysql_aio}"
  KAGEOS_REGISTRATION_MODE="${KAGEOS_REGISTRATION_MODE:-admin_only}"
  SMTP_MODE="${SMTP_MODE:-log}"
  KAGEOS_COMPANY_CODE="${KAGEOS_COMPANY_CODE:-default}"
  KAGEOS_COMPANY_NAME="${KAGEOS_COMPANY_NAME:-Default}"
  KAGEOS_COMPANY_LOGO_URL="${KAGEOS_COMPANY_LOGO_URL:-}"

  export \
    CANONICAL_BASE_URL TLS_MODE HTTP_PORT HTTPS_PORT KAGEOS_APP_BASE_IMAGE \
    MYSQL_HOST MYSQL_PORT MINIO_HOST MINIO_PORT MINIO_ROOT_USER \
    NATS_USER NATS_SEED_USER KAGEOS_APP_DB_CLUSTER_KEY \
    KAGEOS_REGISTRATION_MODE SMTP_MODE \
    KAGEOS_COMPANY_CODE KAGEOS_COMPANY_NAME KAGEOS_COMPANY_LOGO_URL
}

prepare_secrets() {
  load_or_create_secret MYSQL_ROOT_PASSWORD 32
  load_or_create_secret MINIO_ROOT_PASSWORD 32
  load_or_create_secret NATS_PASSWORD 24
  load_or_create_secret JWT_SECRET 32
  load_or_create_secret KAGEOS_APP_DB_SECRET_KEY 32
  load_or_create_secret SYSTEM_USER_PASSWORD 24

  NATS_SEED_PASSWORD="${NATS_SEED_PASSWORD:-$NATS_PASSWORD}"
  NATS_URL="${NATS_URL:-nats://${NATS_USER}:${NATS_PASSWORD}@127.0.0.1:4222}"

  export NATS_PASSWORD NATS_SEED_PASSWORD NATS_URL
}

print_summary() {
  cat <<EOF
==> Kageos AIO 配置
    访问地址: ${CANONICAL_BASE_URL}
    system 初始用户: system
    system 初始密码文件: ${AIO_SECRETS_DIR}/SYSTEM_USER_PASSWORD
    数据目录: ${AIO_DATA_DIR}
EOF
}

main() {
  prepare_layout
  export_defaults
  prepare_secrets
  write_infra_files
  ensure_podman

  start_mysql
  start_nats
  start_minio

  echo "==> 初始化用户应用基础镜像（首次会比较久）..."
  KAGEOS_APP_BASE_ACTION="${KAGEOS_APP_BASE_ACTION:-ensure}" /app/entrypoint-app-base.sh

  print_summary
  exec /app/entrypoint-main.sh
}

main "$@"
