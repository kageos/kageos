usage() {
  cat <<'EOF'
用法: bash build.sh [命令]

命令：
  init [--force]
                初始化 .env；为空的密钥会自动生成
  doctor        执行部署前环境预检（compose / 端口 / 路径 / 证书等）
  up            首次部署 / 全量重建（默认）
  up-image      基于 MAIN_IMAGE 拉取镜像并启动，不在目标机本地构建
  update        仅重建并更新 main / scheduler / backup 服务，不重启 MySQL / NATS / MinIO
  update-image  仅拉取 MAIN_IMAGE 并更新 main / scheduler / backup，不在目标机本地构建
  pull-update   git pull --ff-only 后，仅重建并更新 main / scheduler / backup 服务
  restart-main  仅重启 main 服务，不重建镜像
  restart-scheduler
                仅重启 scheduler 服务，不重建镜像
  verify        执行生产健康检查（mysql / main / scheduler / backup）
  build-app-base [--no-cache]
                在 main 容器内单独构建用户应用基础镜像（默认 agentos-app-runtime-base:latest）
  logs [svc]    查看日志，默认 main
  status        查看 compose 服务状态
  down          停止所有服务（保留 STORAGE_ROOT 数据）
  help          显示帮助

示例：
  bash build.sh init
  bash build.sh doctor
  bash build.sh
  bash build.sh up-image
  bash build.sh update
  bash build.sh update-image
  bash build.sh pull-update
  bash build.sh verify
  bash build.sh build-app-base --no-cache
  bash build.sh logs main
EOF
}

read_env_value() {
  local key="$1"
  awk -v key="$key" '
  function trim(s) {
    gsub(/^[ \t]+|[ \t]+$/, "", s)
    return s
  }
  function strip_comment(s,    i, ch, out, in_single, in_double) {
    out = ""
    in_single = 0
    in_double = 0
    for (i = 1; i <= length(s); i++) {
      ch = substr(s, i, 1)
      if (ch == "\"" && !in_single) {
        in_double = !in_double
        out = out ch
        continue
      }
      if (ch == "'"'"'" && !in_double) {
        in_single = !in_single
        out = out ch
        continue
      }
      if (ch == "#" && !in_single && !in_double) {
        break
      }
      out = out ch
    }
    return trim(out)
  }
  function unquote(s) {
    s = strip_comment(s)
    if (length(s) >= 2) {
      first = substr(s, 1, 1)
      last = substr(s, length(s), 1)
      if ((first == "\"" && last == "\"") || (first == "'"'"'" && last == "'"'"'")) {
        return substr(s, 2, length(s) - 2)
      }
    }
    return s
  }
  /^[ \t]*#/ { next }
  {
    line = $0
    sub(/^[ \t]*/, "", line)
    if (line == "" || index(line, "=") == 0) next
    split(line, pair, "=")
    if (trim(pair[1]) != key) next
    sub(/^[^=]+=[ \t]*/, "", line)
    print unquote(line)
    exit
  }
  ' "$ENV_FILE"
}

read_env_value_or_default() {
  local key="$1"
  local default_value="$2"
  local value
  value="$(read_env_value "$key")"
  if [[ -z "$value" ]]; then
    printf '%s' "$default_value"
  else
    printf '%s' "$value"
  fi
}

require_env_key() {
  local key="$1"
  local value
  value="$(read_env_value "$key")"
  if [[ -z "$value" ]]; then
    echo "ERROR: .env 中缺少必填项 ${key}"
    exit 1
  fi
}

require_env_key_min_length() {
  local key="$1"
  local min_len="$2"
  local value
  value="$(read_env_value "$key")"
  if (( ${#value} < min_len )); then
    echo "ERROR: .env 中 ${key} 长度不足，至少需要 ${min_len} 个字符，当前为 ${#value}"
    exit 1
  fi
}

require_env_key_exact_length() {
  local key="$1"
  local exact_len="$2"
  local value
  value="$(read_env_value "$key")"
  if (( ${#value} != exact_len )); then
    echo "ERROR: .env 中 ${key} 长度必须为 ${exact_len} 个字符，当前为 ${#value}"
    exit 1
  fi
}

resolve_host_path() {
  local path="$1"
  if [[ "$path" == /* ]]; then
    printf '%s' "$path"
  else
    printf '%s' "$ROOT/$path"
  fi
}

prepare_storage_layout() {
  local certs_host_dir
  certs_host_dir="$(resolve_host_path "$TLS_CERTS_HOST_DIR")"
  mkdir -p \
    "$STORAGE_ROOT/mysql" \
    "$STORAGE_ROOT/minio" \
    "$STORAGE_ROOT/podman_storage" \
    "$STORAGE_ROOT/logs" \
    "$STORAGE_ROOT/namespace" \
    "$STORAGE_ROOT/data/runtime/app-runtime" \
    "$STORAGE_ROOT/data/license" \
    "$STORAGE_ROOT/data/backup/repo" \
    "$STORAGE_ROOT/data/backup/state" \
    "$STORAGE_ROOT/data/backup/staging" \
    "$STORAGE_ROOT/data/tmp" \
    "$certs_host_dir"
}

print_storage_mode() {
  echo "==> 存储模式: 宿主机固定目录 ($STORAGE_ROOT)"
}

COMPOSE_CMD=()

detect_compose_cmd() {
  if command -v podman &>/dev/null && podman compose version &>/dev/null; then
    COMPOSE_CMD=(podman compose)
    return 0
  fi

  if command -v docker &>/dev/null && docker compose version &>/dev/null; then
    COMPOSE_CMD=(docker compose)
    return 0
  fi

  COMPOSE_CMD=()
  return 1
}

ensure_compose_cmd() {
  if [[ ${#COMPOSE_CMD[@]} -gt 0 ]]; then
    return 0
  fi

  if ! detect_compose_cmd; then
    echo "ERROR: 未找到 podman compose 或 docker compose，请先安装。"
    exit 1
  fi
}

compose_run() {
  ensure_compose_cmd
  "${COMPOSE_CMD[@]}" --env-file "$ENV_FILE" "$@"
}

ensure_env_file() {
  if [[ ! -f "$ENV_FILE" ]]; then
    echo "ERROR: 未找到 $ENV_FILE"
    echo "请先执行: bash build.sh init"
    exit 1
  fi
}

generate_random_hex() {
  local hex_len="$1"
  local byte_len=$(( (hex_len + 1) / 2 ))
  local value=""

  if command -v openssl >/dev/null 2>&1; then
    value="$(openssl rand -hex "$byte_len" 2>/dev/null | tr -d '\n' | cut -c1-"$hex_len")"
  fi

  if [[ -z "$value" ]]; then
    value="$(od -An -N "$byte_len" -tx1 /dev/urandom 2>/dev/null | tr -d ' \n' | cut -c1-"$hex_len")"
  fi

  if [[ ${#value} -ne hex_len ]]; then
    echo "ERROR: 无法生成长度为 ${hex_len} 的随机十六进制字符串" >&2
    exit 1
  fi

  printf '%s' "$value"
}

write_env_value() {
  local key="$1"
  local value="$2"
  local tmp
  tmp="$(mktemp)"

  awk -v key="$key" -v value="$value" '
  function trim(s) {
    gsub(/^[ \t]+|[ \t]+$/, "", s)
    return s
  }
  function escape(s) {
    gsub(/\\/,"\\\\", s)
    gsub(/"/,"\\\"", s)
    return s
  }
  BEGIN {
    written = 0
    escaped = escape(value)
  }
  {
    line = $0
    raw = line
    sub(/^[ \t]*/, "", raw)
    if (raw == "" || raw ~ /^[ \t]*#/) {
      print line
      next
    }
    pos = index(raw, "=")
    if (pos == 0) {
      print line
      next
    }
    lhs = trim(substr(raw, 1, pos - 1))
    if (lhs != key) {
      print line
      next
    }
    print key "=\"" escaped "\""
    written = 1
  }
  END {
    if (!written) {
      print key "=\"" escaped "\""
    }
  }
  ' "$ENV_FILE" > "$tmp"

  mv "$tmp" "$ENV_FILE"
}

ensure_env_value() {
  local key="$1"
  local value="$2"
  local current
  current="$(read_env_value "$key")"
  if [[ -z "$current" ]]; then
    write_env_value "$key" "$value"
  fi
}

create_env_from_example() {
  local force="${1:-0}"
  if [[ -f "$ENV_FILE" && "$force" != "1" ]]; then
    return 0
  fi
  cp "$ROOT/.env.example" "$ENV_FILE"
}

doctor_ok() {
  echo "[OK]  $1"
}

doctor_warn() {
  echo "[WARN] $1"
}

doctor_fail() {
  echo "[FAIL] $1"
  return 1
}
