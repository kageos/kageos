validate_env() {
  ensure_env_file
  require_env_key CANONICAL_BASE_URL
  require_env_key STORAGE_ROOT
  require_env_key MYSQL_ROOT_PASSWORD
  require_env_key JWT_SECRET
  require_env_key CONTROL_ENC_KEY
  require_env_key MINIO_ROOT_USER
  require_env_key MINIO_ROOT_PASSWORD
  require_env_key BACKUP_BASIC_AUTH_USERNAME
  require_env_key BACKUP_BASIC_AUTH_PASSWORD
  require_env_key MYSQL_IMAGE
  require_env_key NATS_IMAGE
  require_env_key MINIO_IMAGE
  require_env_key MAIN_IMAGE
  require_env_key_min_length JWT_SECRET 32
  require_env_key_exact_length CONTROL_ENC_KEY 32
  require_env_key_min_length BACKUP_BASIC_AUTH_PASSWORD 16
  STORAGE_ROOT="$(read_env_value STORAGE_ROOT)"
  if [[ "$STORAGE_ROOT" != /* ]]; then
    echo "ERROR: STORAGE_ROOT 必须是绝对路径，当前值: $STORAGE_ROOT"
    exit 1
  fi
  CANONICAL_BASE_URL="$(read_env_value CANONICAL_BASE_URL)"
  ENABLE_HTTPS="$(read_env_value_or_default ENABLE_HTTPS 0)"
  HTTPS_REDIRECT="$(read_env_value_or_default HTTPS_REDIRECT 0)"
  TLS_CERTS_HOST_DIR="$(read_env_value_or_default TLS_CERTS_HOST_DIR ./certs)"
  TLS_CERT_FILE="$(read_env_value_or_default TLS_CERT_FILE /app/tls/fullchain.pem)"
  TLS_KEY_FILE="$(read_env_value_or_default TLS_KEY_FILE /app/tls/privkey.pem)"
  MYSQL_IMAGE="$(read_env_value_or_default MYSQL_IMAGE mysql:8.0)"
  NATS_IMAGE="$(read_env_value_or_default NATS_IMAGE nats:2.10-alpine)"
  MINIO_IMAGE="$(read_env_value_or_default MINIO_IMAGE minio/minio:latest)"
  MAIN_IMAGE="$(read_env_value_or_default MAIN_IMAGE agentos-main:latest)"
  APP_BASE_IMAGE="$(read_env_value_or_default APP_BASE_IMAGE agentos-app-runtime-base:latest)"

  case "$ENABLE_HTTPS" in
    0|1) ;;
    *)
      echo "ERROR: ENABLE_HTTPS 仅支持 0 或 1，当前值: $ENABLE_HTTPS"
      exit 1
      ;;
  esac

  case "$HTTPS_REDIRECT" in
    0|1) ;;
    *)
      echo "ERROR: HTTPS_REDIRECT 仅支持 0 或 1，当前值: $HTTPS_REDIRECT"
      exit 1
      ;;
  esac

  if [[ "$HTTPS_REDIRECT" == "1" && "$ENABLE_HTTPS" != "1" ]]; then
    echo "ERROR: HTTPS_REDIRECT=1 需要同时设置 ENABLE_HTTPS=1"
    exit 1
  fi

  if [[ "$ENABLE_HTTPS" == "1" ]]; then
    if [[ "$TLS_CERT_FILE" != /app/tls/* ]]; then
      echo "ERROR: TLS_CERT_FILE 必须位于 /app/tls/ 下，当前值: $TLS_CERT_FILE"
      exit 1
    fi
    if [[ "$TLS_KEY_FILE" != /app/tls/* ]]; then
      echo "ERROR: TLS_KEY_FILE 必须位于 /app/tls/ 下，当前值: $TLS_KEY_FILE"
      exit 1
    fi

    local certs_host_dir cert_host_path key_host_path cert_rel key_rel
    certs_host_dir="$(resolve_host_path "$TLS_CERTS_HOST_DIR")"
    cert_rel="${TLS_CERT_FILE#/app/tls/}"
    key_rel="${TLS_KEY_FILE#/app/tls/}"
    cert_host_path="${certs_host_dir}/${cert_rel}"
    key_host_path="${certs_host_dir}/${key_rel}"

    if [[ ! -f "$cert_host_path" ]]; then
      echo "ERROR: ENABLE_HTTPS=1 但证书文件不存在: $cert_host_path"
      exit 1
    fi
    if [[ ! -f "$key_host_path" ]]; then
      echo "ERROR: ENABLE_HTTPS=1 但私钥文件不存在: $key_host_path"
      exit 1
    fi

    if [[ "$HTTPS_REDIRECT" == "1" && "$CANONICAL_BASE_URL" != https://* ]]; then
      echo "ERROR: HTTPS_REDIRECT=1 时 CANONICAL_BASE_URL 必须使用 https://，当前值: $CANONICAL_BASE_URL"
      exit 1
    fi

    if [[ "$CANONICAL_BASE_URL" != https://* ]]; then
      echo "WARN: ENABLE_HTTPS=1 但 CANONICAL_BASE_URL 不是 https://；将同时提供 HTTPS，但 canonical scheme 仍按当前值生成。"
    fi
  elif [[ "$CANONICAL_BASE_URL" == https://* ]]; then
    echo "WARN: CANONICAL_BASE_URL 当前是 https:// 但 ENABLE_HTTPS=0；如果不是外部 TLS 终止场景，请先补齐证书配置。"
  fi
}

stop_host_nginx_if_needed() {
  if command -v systemctl &>/dev/null && systemctl is-active --quiet nginx 2>/dev/null; then
    echo "==> 检测到宿主机 nginx 正在运行，停止并禁用..."
    sudo systemctl stop nginx
    sudo systemctl disable nginx
    echo "    宿主机 nginx 已停止"
  fi
}

main_service_running() {
  local cid
  cid="$(compose_run ps -q main 2>/dev/null || true)"
  [[ -n "$cid" ]]
}

port_listener_snapshot() {
  local port="$1"
  if command -v ss &>/dev/null; then
    ss -tlnp 2>/dev/null | grep ":${port} " || true
  elif command -v netstat &>/dev/null; then
    netstat -tlnp 2>/dev/null | grep ":${port} " || true
  else
    true
  fi
}

ensure_required_ports_available_for_first_up() {
  local listen80 listen443
  if main_service_running; then
    return 0
  fi

  listen80="$(port_listener_snapshot 80)"
  if [[ -n "$listen80" ]]; then
    echo "ERROR: 80 端口仍被占用，停止部署。"
    echo "$listen80"
    echo "请先手动释放 80 端口后再重跑 build.sh。"
    exit 1
  fi

  if [[ "$ENABLE_HTTPS" == "1" ]]; then
    listen443="$(port_listener_snapshot 443)"
    if [[ -n "$listen443" ]]; then
      echo "ERROR: 443 端口仍被占用，停止部署。"
      echo "$listen443"
      echo "请先手动释放 443 端口后再重跑 build.sh。"
      exit 1
    fi
  fi
}
