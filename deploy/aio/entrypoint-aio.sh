#!/bin/bash
set -euo pipefail

source /app/entrypoint-common.sh

AIO_DATA_DIR="${KAGEOS_AIO_DATA_DIR:-/var/lib/kageos}"
AIO_INFRA_DIR="${AIO_DATA_DIR}/infra"
AIO_SECRETS_DIR="${AIO_DATA_DIR}/secrets"

MYSQL_CONTAINER_NAME="${KAGEOS_AIO_MYSQL_CONTAINER_NAME:-kageos-mysql}"
NATS_CONTAINER_NAME="${KAGEOS_AIO_NATS_CONTAINER_NAME:-kageos-nats}"
MINIO_CONTAINER_NAME="${KAGEOS_AIO_MINIO_CONTAINER_NAME:-kageos-minio}"
KAGEOS_AIO_RECREATE_INFRA="${KAGEOS_AIO_RECREATE_INFRA:-1}"
KAGEOS_AIO_REQUIRE_BRIDGE="${KAGEOS_AIO_REQUIRE_BRIDGE:-1}"
KAGEOS_AIO_ALLOW_HOST_NETWORK="${KAGEOS_AIO_ALLOW_HOST_NETWORK:-0}"

MYSQL_IMAGE="${KAGEOS_AIO_MYSQL_IMAGE:-docker.io/library/mysql:8.0.45}"
NATS_IMAGE="${KAGEOS_AIO_NATS_IMAGE:-docker.io/library/nats:2.10.29-alpine}"
MINIO_IMAGE="${KAGEOS_AIO_MINIO_IMAGE:-docker.io/minio/minio:RELEASE.2025-04-22T22-12-26Z}"

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
port: ${NATS_PORT}
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

outer_network_looks_isolated() {
  local ifindex=""
  local iflink=""

  if [[ -r /sys/class/net/eth0/ifindex && -r /sys/class/net/eth0/iflink ]]; then
    ifindex="$(cat /sys/class/net/eth0/ifindex 2>/dev/null || true)"
    iflink="$(cat /sys/class/net/eth0/iflink 2>/dev/null || true)"
  fi

  # Docker/Podman bridge and slirp-style container networking usually expose
  # eth0 as a veth endpoint where iflink differs from ifindex. Host networking
  # normally exposes the host NIC directly, or no container eth0 at all.
  [[ -n "$ifindex" && -n "$iflink" && "$ifindex" != "$iflink" ]]
}

assert_outer_network_supported() {
  if [[ "$KAGEOS_AIO_ALLOW_HOST_NETWORK" == "1" || "$KAGEOS_AIO_ALLOW_HOST_NETWORK" == "true" ]]; then
    echo "WARN: KAGEOS_AIO_ALLOW_HOST_NETWORK=1，跳过 AIO 外层网络隔离检查；内部端口可能暴露到宿主机。"
    return 0
  fi
  if [[ "$KAGEOS_AIO_REQUIRE_BRIDGE" == "0" || "$KAGEOS_AIO_REQUIRE_BRIDGE" == "false" ]]; then
    echo "WARN: KAGEOS_AIO_REQUIRE_BRIDGE=0，跳过 AIO 外层网络隔离检查。"
    return 0
  fi
  if outer_network_looks_isolated; then
    return 0
  fi

  cat >&2 <<'EOF'
ERROR: kageos AIO 不支持外层容器使用 host 网络。

原因：
  AIO 容器内部会启动 core-server、MySQL、NATS、MinIO 和用户 App 容器。
  如果外层 docker/podman run 使用 --network host，这些内部监听端口
  例如 9093、13306、14222、19000 会直接占用宿主机网络命名空间。

请改用 bridge/slirp 容器网络，并只映射入口端口：
  docker rm -f kageos
  docker run -d \
    --name kageos \
    --privileged \
    --restart unless-stopped \
    -p 8080:80 \
    -v kageos-data:/var/lib/kageos \
    -e CANONICAL_BASE_URL=http://<host-or-ip>:8080 \
    qiayanai/kageos:latest

不要加：
  --network host

如果你非常确定要承担内部端口污染宿主机的风险，可显式设置：
  -e KAGEOS_AIO_ALLOW_HOST_NETWORK=1
EOF
  exit 1
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

ensure_container_running() {
  local name="$1"
  local label="$2"
  if ! podman_container_running "$name"; then
    echo "ERROR: ${label} 容器未运行，可能是端口被占用或镜像启动失败: ${name}" >&2
    podman logs --tail 120 "$name" >&2 || true
    exit 1
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
      --restart=unless-stopped \
      -e MYSQL_ROOT_PASSWORD="$MYSQL_ROOT_PASSWORD" \
      -e TZ="${TZ:-Asia/Shanghai}" \
      -v "${AIO_DATA_DIR}/mysql:/var/lib/mysql" \
      -v "${AIO_INFRA_DIR}/mysql-init.sql:/docker-entrypoint-initdb.d/init.sql:ro" \
      "$MYSQL_IMAGE" \
      --port="${MYSQL_PORT}" \
      --character-set-server=utf8mb4 \
      --collation-server=utf8mb4_unicode_ci >/dev/null
  fi

  wait_tcp "$MYSQL_HOST" "$MYSQL_PORT" "MySQL"
  for i in $(seq 1 90); do
    ensure_container_running "$MYSQL_CONTAINER_NAME" "MySQL"
    if podman exec "$MYSQL_CONTAINER_NAME" mysql --protocol=TCP -h "$MYSQL_HOST" -P "$MYSQL_PORT" -uroot -p"$MYSQL_ROOT_PASSWORD" -e 'SELECT 1' >/dev/null 2>&1; then
      echo "==> MySQL root 登录就绪"
      podman exec -i "$MYSQL_CONTAINER_NAME" mysql --protocol=TCP -h "$MYSQL_HOST" -P "$MYSQL_PORT" -uroot -p"$MYSQL_ROOT_PASSWORD" < "${AIO_INFRA_DIR}/mysql-init.sql"
      return 0
    fi
    echo "    等待 MySQL 初始化账号 ... (${i}/90)"
    sleep 2
  done
  echo "ERROR: MySQL 已监听但 root 登录未就绪" >&2
  podman logs --tail 120 "$MYSQL_CONTAINER_NAME" >&2 || true
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

  wait_tcp "$NATS_HOST" "$NATS_PORT" "NATS"
  ensure_container_running "$NATS_CONTAINER_NAME" "NATS"
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
      server /data --address ":${MINIO_PORT}" --console-address ":${MINIO_CONSOLE_PORT}" >/dev/null
  fi

  wait_tcp "$MINIO_HOST" "$MINIO_PORT" "MinIO"
  ensure_container_running "$MINIO_CONTAINER_NAME" "MinIO"
  wait_http "http://${MINIO_HOST}:${MINIO_PORT}/minio/health/ready" "MinIO health"
}

export_defaults() {
  CANONICAL_BASE_URL="${CANONICAL_BASE_URL:-http://localhost:8080}"
  TLS_MODE="${TLS_MODE:-http}"
  HTTP_PORT="${HTTP_PORT:-80}"
  HTTPS_PORT="${HTTPS_PORT:-443}"
  KAGEOS_APP_BASE_IMAGE="${KAGEOS_APP_BASE_IMAGE:-${KAGEOS_DEFAULT_APP_BASE_IMAGE:-docker.io/qiayanai/kagebase:latest}}"

  MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
  MYSQL_PORT="${MYSQL_PORT:-13306}"
  NATS_HOST="${NATS_HOST:-127.0.0.1}"
  NATS_PORT="${NATS_PORT:-14222}"
  MINIO_HOST="${MINIO_HOST:-127.0.0.1}"
  MINIO_PORT="${MINIO_PORT:-19000}"
  MINIO_CONSOLE_PORT="${MINIO_CONSOLE_PORT:-19001}"
  MINIO_ROOT_USER="${MINIO_ROOT_USER:-minioadmin}"
  NATS_USER="${NATS_USER:-${NATS_SEED_USER:-aos}}"
  NATS_SEED_USER="${NATS_SEED_USER:-$NATS_USER}"

  KAGEOS_APP_DB_CLUSTER_KEY="${KAGEOS_APP_DB_CLUSTER_KEY:-mysql_aio}"
  KAGEOS_REGISTRATION_MODE="${KAGEOS_REGISTRATION_MODE:-admin_only}"
  SMTP_MODE="${SMTP_MODE:-log}"

  export \
    CANONICAL_BASE_URL TLS_MODE HTTP_PORT HTTPS_PORT KAGEOS_APP_BASE_IMAGE \
    MYSQL_HOST MYSQL_PORT NATS_HOST NATS_PORT MINIO_HOST MINIO_PORT MINIO_CONSOLE_PORT MINIO_ROOT_USER \
    NATS_USER NATS_SEED_USER KAGEOS_APP_DB_CLUSTER_KEY \
    KAGEOS_REGISTRATION_MODE SMTP_MODE
}

prepare_secrets() {
  load_or_create_secret MYSQL_ROOT_PASSWORD 32
  load_or_create_secret MINIO_ROOT_PASSWORD 32
  load_or_create_secret NATS_PASSWORD 24
  load_or_create_secret JWT_SECRET 32
  load_or_create_secret KAGEOS_APP_DB_SECRET_KEY 32
  load_or_create_secret SYSTEM_USER_PASSWORD 24

  NATS_SEED_PASSWORD="${NATS_SEED_PASSWORD:-$NATS_PASSWORD}"
  NATS_URL="${NATS_URL:-nats://${NATS_USER}:${NATS_PASSWORD}@${NATS_HOST}:${NATS_PORT}}"

  export NATS_PASSWORD NATS_SEED_PASSWORD NATS_URL
}

print_summary() {
  cat <<EOF
==> kageos AIO 配置
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
  assert_outer_network_supported
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
