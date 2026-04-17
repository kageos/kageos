validate_env() {
  ensure_env_bootstrapped
  CANONICAL_BASE_URL="$(read_env_value CANONICAL_BASE_URL)"
  if [[ -z "$CANONICAL_BASE_URL" ]]; then
    echo "ERROR: 请先在 .env 中填写 CANONICAL_BASE_URL，例如 http://your-domain-or-ip 或 https://your-domain"
    exit 1
  fi
  require_env_key MYSQL_ROOT_PASSWORD
  require_env_key JWT_SECRET
  require_env_key CONTROL_ENC_KEY
  require_env_key MINIO_ROOT_PASSWORD
  require_env_key BACKUP_BASIC_AUTH_PASSWORD
  require_env_key_min_length JWT_SECRET 32
  require_env_key_exact_length CONTROL_ENC_KEY 32
  require_env_key_min_length BACKUP_BASIC_AUTH_PASSWORD 16
  if [[ "$FIXED_STORAGE_ROOT" != /* ]]; then
    echo "ERROR: 固定存储目录必须是绝对路径，当前值: $FIXED_STORAGE_ROOT"
    exit 1
  fi
  case "$CANONICAL_BASE_URL" in
    http://*|https://*) ;;
    *)
      echo "ERROR: CANONICAL_BASE_URL 必须以 http:// 或 https:// 开头，当前值: $CANONICAL_BASE_URL"
      exit 1
      ;;
  esac
  TLS_MODE="$(read_env_value_or_default TLS_MODE "$DEFAULT_TLS_MODE")"
  TLS_CERTS_HOST_DIR="$(read_env_value_or_default TLS_CERTS_HOST_DIR ./certs)"
  TLS_CERT_FILE="$(read_env_value_or_default TLS_CERT_FILE /app/tls/fullchain.pem)"
  TLS_KEY_FILE="$(read_env_value_or_default TLS_KEY_FILE /app/tls/privkey.pem)"
  MYSQL_IMAGE="$(read_env_value_or_default MYSQL_IMAGE "$DEFAULT_MYSQL_IMAGE")"
  NATS_IMAGE="$(read_env_value_or_default NATS_IMAGE "$DEFAULT_NATS_IMAGE")"
  MINIO_IMAGE="$(read_env_value_or_default MINIO_IMAGE "$DEFAULT_MINIO_IMAGE")"
  MAIN_IMAGE="$(read_env_value_or_default MAIN_IMAGE "$DEFAULT_MAIN_IMAGE")"
  APP_BASE_IMAGE="$(read_env_value_or_default APP_BASE_IMAGE "$DEFAULT_APP_BASE_IMAGE")"

  case "$TLS_MODE" in
    http|https|redirect|external) ;;
    *)
      echo "ERROR: TLS_MODE 仅支持 http / https / redirect / external，当前值: $TLS_MODE"
    exit 1
    ;;
  esac

  if tls_mode_uses_local_https "$TLS_MODE"; then
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
      echo "ERROR: TLS_MODE=${TLS_MODE} 但证书文件不存在: $cert_host_path"
      exit 1
    fi
    if [[ ! -f "$key_host_path" ]]; then
      echo "ERROR: TLS_MODE=${TLS_MODE} 但私钥文件不存在: $key_host_path"
      exit 1
    fi

    if tls_mode_requires_redirect "$TLS_MODE" && [[ "$CANONICAL_BASE_URL" != https://* ]]; then
      echo "ERROR: TLS_MODE=redirect 时 CANONICAL_BASE_URL 必须使用 https://，当前值: $CANONICAL_BASE_URL"
      exit 1
    fi

    if [[ "$CANONICAL_BASE_URL" != https://* ]]; then
      echo "WARN: TLS_MODE=${TLS_MODE} 会在本机提供 HTTPS，但 CANONICAL_BASE_URL 不是 https://；canonical scheme 仍按当前值生成。"
    fi
  elif [[ "$TLS_MODE" == "http" && "$CANONICAL_BASE_URL" == https://* ]]; then
    echo "WARN: CANONICAL_BASE_URL 当前是 https:// 但 TLS_MODE=http；如果前面有 TLS 终止，请改成 TLS_MODE=external。"
  elif [[ "$TLS_MODE" == "external" && "$CANONICAL_BASE_URL" != https://* ]]; then
    echo "WARN: TLS_MODE=external 通常应配合 https:// 的 CANONICAL_BASE_URL；如果只想跑明文 HTTP，请改成 TLS_MODE=http。"
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

  if tls_mode_uses_local_https "$TLS_MODE"; then
    listen443="$(port_listener_snapshot 443)"
    if [[ -n "$listen443" ]]; then
      echo "ERROR: 443 端口仍被占用，停止部署。"
      echo "$listen443"
      echo "请先手动释放 443 端口后再重跑 build.sh。"
      exit 1
    fi
  fi
}
