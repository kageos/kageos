usage() {
  cat <<'EOF'
用法: bash build.sh [命令] [参数]

标准命令：
  init [--image] [--force]
                初始化 .env，并准备中间件 / MAIN_IMAGE / APP_BASE_IMAGE
  up            启动已初始化的服务（默认，不再构建镜像）
  update [--image]
                更新 main / scheduler / backup；加 --image 时直接拉取 MAIN_IMAGE
  verify        执行生产健康检查（mysql / main / scheduler / backup）
  logs [svc]    查看日志，默认 main
  status        查看 compose 服务状态
  down          停止所有服务（保留 /data/ai-agent-os 数据）
  help          显示帮助

排障 / 高级命令：
  doctor        显式执行部署预检（主路径可省略；up 已覆盖核心检查）
  pull-update   git pull --ff-only 后，仅重建并更新 main / scheduler / backup 服务
  restart-main  仅重启 main 服务，不重建镜像
  restart-scheduler
                仅重启 scheduler 服务，不重建镜像
  build-app-base [--no-cache]
                在与 main 相同的运行环境里重建用户应用基础镜像

示例：
  bash build.sh init
  bash build.sh init --image
  bash build.sh up
  bash build.sh update
  bash build.sh update --image
  bash build.sh doctor
  bash build.sh pull-update
  bash build.sh verify
  bash build.sh build-app-base --no-cache
  bash build.sh logs main
EOF
}

FIXED_STORAGE_ROOT="/data/ai-agent-os"
DEFAULT_TLS_MODE="http"
DEFAULT_MAIN_IMAGE="agentos-main:latest"
DEFAULT_APP_BASE_IMAGE="agentos-app-runtime-base:latest"
INIT_IMAGE_USAGE_HINT="bash build.sh init；如需直接拉取已发布主镜像，用 bash build.sh init --image"

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

ensure_host_dir() {
  local dir="$1"

  if [[ -d "$dir" ]]; then
    return 0
  fi

  if mkdir -p "$dir" 2>/dev/null; then
    return 0
  fi

  prepare_root_cmd
  "${ROOT_CMD[@]}" mkdir -p "$dir"
  "${ROOT_CMD[@]}" chown "$(id -u):$(id -g)" "$dir"
}

prepare_storage_layout() {
  local certs_host_dir
  certs_host_dir="$(resolve_host_path "$TLS_CERTS_HOST_DIR")"
  ensure_host_dir "$FIXED_STORAGE_ROOT/mysql"
  ensure_host_dir "$FIXED_STORAGE_ROOT/minio"
  ensure_host_dir "$FIXED_STORAGE_ROOT/podman_storage"
  ensure_host_dir "$FIXED_STORAGE_ROOT/logs"
  ensure_host_dir "$FIXED_STORAGE_ROOT/namespace"
  ensure_host_dir "$FIXED_STORAGE_ROOT/data/runtime/app-runtime"
  ensure_host_dir "$FIXED_STORAGE_ROOT/data/license"
  ensure_host_dir "$FIXED_STORAGE_ROOT/data/backup/repo"
  ensure_host_dir "$FIXED_STORAGE_ROOT/data/backup/state"
  ensure_host_dir "$FIXED_STORAGE_ROOT/data/backup/staging"
  ensure_host_dir "$FIXED_STORAGE_ROOT/data/tmp"
  ensure_host_dir "$certs_host_dir"
}

print_storage_mode() {
  echo "==> 存储模式: 宿主机固定目录 ($FIXED_STORAGE_ROOT)"
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

host_podman_installed() {
  command -v podman >/dev/null 2>&1
}

host_podman_compose_available() {
  host_podman_installed && podman compose version >/dev/null 2>&1
}

host_podman_ready() {
  host_podman_installed && host_podman_compose_available
}

host_podman_manual_install_hint() {
  if command -v apt-get >/dev/null 2>&1; then
    echo "手工安装可执行：sudo apt-get update && sudo apt-get install -y podman podman-compose"
  elif command -v dnf >/dev/null 2>&1; then
    echo "手工安装可执行：sudo dnf install -y podman podman-compose"
  elif command -v yum >/dev/null 2>&1; then
    echo "手工安装可执行：sudo yum install -y podman podman-compose"
  elif command -v pacman >/dev/null 2>&1; then
    echo "手工安装可执行：sudo pacman -Sy --noconfirm --needed podman podman-compose"
  else
    echo "未识别到 apt-get / dnf / yum / pacman，请手工安装 podman 与 podman-compose。"
  fi
}

apt_tencent_ubuntu_sources_present() {
  grep -R -q 'mirrors\.tencentyun\.com/ubuntu' /etc/apt /etc/apt/sources.list.d 2>/dev/null
}

rewrite_tencent_ubuntu_sources_to_official() {
  local file
  local backup_path
  local changed=0

  while IFS= read -r file; do
    [[ -z "$file" ]] && continue
    backup_path="${file}.bak-ai-agent-os"
    if [[ ! -f "$backup_path" ]]; then
      "${ROOT_CMD[@]}" cp "$file" "$backup_path"
    fi
    "${ROOT_CMD[@]}" sed -i \
      -e 's|http://mirrors.tencentyun.com/ubuntu/|http://archive.ubuntu.com/ubuntu/|g' \
      -e 's|https://mirrors.tencentyun.com/ubuntu/|http://archive.ubuntu.com/ubuntu/|g' \
      "$file"
    changed=1
  done < <(grep -R -l 'mirrors\.tencentyun\.com/ubuntu' /etc/apt /etc/apt/sources.list.d 2>/dev/null || true)

  if [[ "$changed" == "1" ]]; then
    echo "==> 已将 Ubuntu APT 源从腾讯云镜像切换到官方源，并保留备份 *.bak-ai-agent-os"
    return 0
  fi

  return 1
}

apt_install_podman_with_retry() {
  if "${ROOT_CMD[@]}" apt-get update && "${ROOT_CMD[@]}" apt-get install -y podman podman-compose; then
    return 0
  fi

  echo "WARN: apt 安装 podman 失败，尝试自动诊断镜像源问题 ..."

  if apt_tencent_ubuntu_sources_present; then
    echo "==> 检测到腾讯云 Ubuntu APT 源，尝试切换到官方源后重试 ..."
    if rewrite_tencent_ubuntu_sources_to_official; then
      "${ROOT_CMD[@]}" apt-get clean
      if "${ROOT_CMD[@]}" apt-get update && "${ROOT_CMD[@]}" apt-get install -y podman podman-compose; then
        return 0
      fi
    fi
  fi

  return 1
}

prepare_root_cmd() {
  ROOT_CMD=()
  if [[ "${EUID:-$(id -u)}" == "0" ]]; then
    return 0
  fi

  if command -v sudo >/dev/null 2>&1; then
    ROOT_CMD=(sudo)
    return 0
  fi

  echo "ERROR: 当前用户不是 root，且宿主机未安装 sudo，无法自动安装 podman。"
  host_podman_manual_install_hint
  exit 1
}

linux_distribution_id() {
  local distro="unknown"
  local like=""

  if [[ -r /etc/os-release ]]; then
    # shellcheck disable=SC1091
    . /etc/os-release
    distro="${ID:-unknown}"
    like="${ID_LIKE:-}"
  fi

  case "$distro" in
    ubuntu|debian|fedora|centos|rhel|rocky|almalinux|arch|manjaro)
      printf '%s' "$distro"
      return 0
      ;;
  esac

  case " $like " in
    *" debian "*|*" ubuntu "*)
      printf '%s' "debian"
      ;;
    *" rhel "*|*" fedora "*|*" centos "*)
      printf '%s' "rhel"
      ;;
    *" arch "*)
      printf '%s' "arch"
      ;;
    *)
      printf '%s' "$distro"
      ;;
  esac
}

install_host_podman() {
  require_linux_host

  if host_podman_ready; then
    echo "==> 宿主机 podman / podman compose 已就绪"
    return 0
  fi

  prepare_root_cmd

  local distro
  distro="$(linux_distribution_id)"

  echo "==> 检测到宿主机缺少 podman 或 podman compose，开始自动安装 ..."
  case "$distro" in
    ubuntu|debian)
      if ! apt_install_podman_with_retry; then
        echo "ERROR: apt 安装 podman / podman-compose 失败。"
        host_podman_manual_install_hint
        exit 1
      fi
      ;;
    fedora)
      "${ROOT_CMD[@]}" dnf install -y podman podman-compose
      ;;
    centos|rhel|rocky|almalinux)
      if command -v dnf >/dev/null 2>&1; then
        "${ROOT_CMD[@]}" dnf install -y podman podman-compose
      else
        "${ROOT_CMD[@]}" yum install -y podman podman-compose
      fi
      ;;
    arch|manjaro)
      "${ROOT_CMD[@]}" pacman -Sy --noconfirm --needed podman podman-compose
      ;;
    *)
      echo "ERROR: 当前发行版暂未内置自动安装逻辑: ${distro}"
      host_podman_manual_install_hint
      exit 1
      ;;
  esac

  if ! host_podman_installed; then
    echo "ERROR: podman 安装后仍不可用。"
    host_podman_manual_install_hint
    exit 1
  fi

  if ! host_podman_compose_available; then
    echo "ERROR: podman 已安装，但 podman compose 仍不可用。"
    host_podman_manual_install_hint
    exit 1
  fi

  echo "==> 宿主机 podman / podman compose 安装完成"
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
    echo "请先执行: bash build.sh init；如需直接拉取已发布主镜像，用 bash build.sh init --image"
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

delete_env_key() {
  local key="$1"
  local tmp
  tmp="$(mktemp)"

  awk -v key="$key" '
  function trim(s) {
    gsub(/^[ \t]+|[ \t]+$/, "", s)
    return s
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
    if (lhs == key) {
      next
    }
    print line
  }
  ' "$ENV_FILE" > "$tmp"

  mv "$tmp" "$ENV_FILE"
}

create_env_from_example() {
  local force="${1:-0}"
  if [[ -f "$ENV_FILE" && "$force" != "1" ]]; then
    return 0
  fi
  cp "$ROOT/.env.example" "$ENV_FILE"
}

bootstrap_env_defaults() {
  local derived_tls_mode
  delete_env_key STORAGE_ROOT
  delete_env_key MYSQL_IMAGE
  delete_env_key NATS_IMAGE
  delete_env_key MINIO_IMAGE
  delete_env_key MINIO_ROOT_USER
  delete_env_key BACKUP_BASIC_AUTH_USERNAME
  derived_tls_mode="$(derive_tls_mode_from_env)"
  ensure_env_value TLS_MODE "$derived_tls_mode"
  delete_env_key ENABLE_HTTPS
  delete_env_key HTTPS_REDIRECT
  ensure_env_value MYSQL_ROOT_PASSWORD "$(generate_random_hex 32)"
  ensure_env_value JWT_SECRET "$(generate_random_hex 64)"
  ensure_env_value CONTROL_ENC_KEY "$(generate_random_hex 32)"
  ensure_env_value MINIO_ROOT_PASSWORD "$(generate_random_hex 32)"
  ensure_env_value BACKUP_BASIC_AUTH_PASSWORD "$(generate_random_hex 48)"
}

derive_tls_mode_from_env() {
  local tls_mode legacy_https legacy_redirect canonical_base_url
  tls_mode="$(read_env_value TLS_MODE)"
  if [[ -n "$tls_mode" ]]; then
    printf '%s' "$tls_mode"
    return 0
  fi

  legacy_https="$(read_env_value_or_default ENABLE_HTTPS 0)"
  legacy_redirect="$(read_env_value_or_default HTTPS_REDIRECT 0)"
  canonical_base_url="$(read_env_value CANONICAL_BASE_URL)"

  case "$legacy_https" in
    0|1) ;;
    *)
      echo "ERROR: 旧配置 ENABLE_HTTPS 仅支持 0 或 1，当前值: ${legacy_https}"
      exit 1
      ;;
  esac

  case "$legacy_redirect" in
    0|1) ;;
    *)
      echo "ERROR: 旧配置 HTTPS_REDIRECT 仅支持 0 或 1，当前值: ${legacy_redirect}"
      exit 1
      ;;
  esac

  if [[ "$legacy_redirect" == "1" && "$legacy_https" != "1" ]]; then
    echo "ERROR: 旧配置 HTTPS_REDIRECT=1 需要同时设置 ENABLE_HTTPS=1"
    exit 1
  fi

  if [[ "$legacy_https" == "1" ]]; then
    if [[ "$legacy_redirect" == "1" ]]; then
      printf '%s' "redirect"
    else
      printf '%s' "https"
    fi
    return 0
  fi

  if [[ "$canonical_base_url" == https://* ]]; then
    printf '%s' "external"
    return 0
  fi

  printf '%s' "$DEFAULT_TLS_MODE"
}

tls_mode_uses_local_https() {
  local tls_mode="${1:-${TLS_MODE:-$DEFAULT_TLS_MODE}}"
  [[ "$tls_mode" == "https" || "$tls_mode" == "redirect" ]]
}

tls_mode_requires_redirect() {
  local tls_mode="${1:-${TLS_MODE:-$DEFAULT_TLS_MODE}}"
  [[ "$tls_mode" == "redirect" ]]
}

ensure_env_bootstrapped() {
  if [[ ! -f "$ENV_FILE" ]]; then
    echo "==> 未找到 .env，自动初始化 ..."
    create_env_from_example 1
  fi

  bootstrap_env_defaults
}

load_env_defaults() {
  CANONICAL_BASE_URL="$(read_env_value CANONICAL_BASE_URL)"
  TLS_MODE="$(read_env_value_or_default TLS_MODE "$DEFAULT_TLS_MODE")"
  TLS_CERTS_HOST_DIR="$(read_env_value_or_default TLS_CERTS_HOST_DIR ./certs)"
  TLS_CERT_FILE="$(read_env_value_or_default TLS_CERT_FILE /app/tls/fullchain.pem)"
  TLS_KEY_FILE="$(read_env_value_or_default TLS_KEY_FILE /app/tls/privkey.pem)"
  MAIN_IMAGE="$(read_env_value_or_default MAIN_IMAGE "$DEFAULT_MAIN_IMAGE")"
  APP_BASE_IMAGE="$(read_env_value_or_default APP_BASE_IMAGE "$DEFAULT_APP_BASE_IMAGE")"
}

require_linux_host() {
  if [[ "$(uname -s)" != "Linux" ]]; then
    echo "ERROR: prod 当前只支持 Linux 宿主机。"
    exit 1
  fi
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
