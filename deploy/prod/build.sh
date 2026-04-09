#!/usr/bin/env bash
# 生产主站部署脚本：直接读取 .env，统一使用 STORAGE_ROOT 宿主机目录
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"
PROJECT_ROOT="$(cd "$ROOT/../.." && pwd)"

ENV_FILE="$ROOT/.env"
COMMAND="${1:-up}"
ARG1="${2:-}"
STORAGE_ROOT=""

usage() {
  cat <<'EOF'
用法: bash build.sh [命令]

命令：
  up            首次部署 / 全量重建（默认）
  update        仅重建并更新 main / scheduler / backup 服务，不重启 MySQL / NATS / MinIO
  pull-update   git pull --ff-only 后，仅重建并更新 main / scheduler / backup 服务
  restart-main  仅重启 main 服务，不重建镜像
  restart-scheduler
                仅重启 scheduler 服务，不重建镜像
  build-app-base [--no-cache]
                在 main 容器内单独构建用户应用基础镜像 ai-agent-os:latest
  logs [svc]    查看日志，默认 main
  status        查看 compose 服务状态
  down          停止所有服务（保留 STORAGE_ROOT 数据）
  help          显示帮助

示例：
  bash build.sh
  bash build.sh update
  bash build.sh pull-update
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
  function unquote(s) {
    s = trim(s)
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

require_env_key() {
  local key="$1"
  local value
  value="$(read_env_value "$key")"
  if [[ -z "$value" ]]; then
    echo "ERROR: .env 中缺少必填项 ${key}"
    exit 1
  fi
}

prepare_storage_layout() {
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
    "$STORAGE_ROOT/data/tmp"
}

print_storage_mode() {
  echo "==> 存储模式: 宿主机固定目录 ($STORAGE_ROOT)"
}

COMPOSE_CMD=()

ensure_compose_cmd() {
  if [[ ${#COMPOSE_CMD[@]} -gt 0 ]]; then
    return 0
  fi

  if command -v podman &>/dev/null && podman compose version &>/dev/null; then
    COMPOSE_CMD=(podman compose)
  elif command -v docker &>/dev/null && docker compose version &>/dev/null; then
    COMPOSE_CMD=(docker compose)
  else
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
    echo "请复制 .env.example 为 .env，填写后重跑: bash build.sh"
    exit 1
  fi
}

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
  require_env_key MAIN_IMAGE
  STORAGE_ROOT="$(read_env_value STORAGE_ROOT)"
  if [[ "$STORAGE_ROOT" != /* ]]; then
    echo "ERROR: STORAGE_ROOT 必须是绝对路径，当前值: $STORAGE_ROOT"
    exit 1
  fi
  CANONICAL_BASE_URL="$(read_env_value CANONICAL_BASE_URL)"
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

ensure_port_80_available_for_first_up() {
  local listen80
  if main_service_running; then
    return 0
  fi

  if command -v ss &>/dev/null; then
    listen80="$(ss -tlnp 2>/dev/null | grep ':80 ' || true)"
  elif command -v netstat &>/dev/null; then
    listen80="$(netstat -tlnp 2>/dev/null | grep ':80 ' || true)"
  else
    listen80=""
  fi

  if [[ -n "$listen80" ]]; then
    echo "ERROR: 80 端口仍被占用，停止部署。"
    echo "$listen80"
    echo "请先手动释放 80 端口后再重跑 build.sh。"
    exit 1
  fi
}

print_success() {
  echo ""
  echo "=============================="
  echo "  操作完成"
  echo "=============================="
  echo "  访问地址: ${CANONICAL_BASE_URL}"
  echo "  存储根目录: ${STORAGE_ROOT}"
  echo ""
  echo "  查看日志: bash build.sh logs main"
  echo "  查看状态: bash build.sh status"
  echo "  停止服务: bash build.sh down"
  echo "  ⚠ 切勿:  rm -rf ${STORAGE_ROOT}"
  echo "=============================="
}

cmd_up() {
  validate_env
  ensure_compose_cmd
  echo "==> 使用: ${COMPOSE_CMD[*]}"
  print_storage_mode
  stop_host_nginx_if_needed
  prepare_storage_layout
  ensure_port_80_available_for_first_up
  echo "==> 全量启动并构建..."
  compose_run up -d --build
  print_success
}

cmd_update() {
  validate_env
  ensure_compose_cmd
  echo "==> 使用: ${COMPOSE_CMD[*]}"
  print_storage_mode
  stop_host_nginx_if_needed
  prepare_storage_layout
  echo "==> 仅重建并更新 main / scheduler / backup 服务（不重启中间件）..."
  compose_run up -d --build --no-deps main scheduler backup
  print_success
}

cmd_pull_update() {
  validate_env
  echo "==> 执行 git pull --ff-only ..."
  git -C "$PROJECT_ROOT" pull --ff-only
  cmd_update
}

cmd_restart_main() {
  validate_env
  ensure_compose_cmd
  echo "==> 使用: ${COMPOSE_CMD[*]}"
  print_storage_mode
  stop_host_nginx_if_needed
  prepare_storage_layout
  echo "==> 重启 main 服务..."
  compose_run restart main
  print_success
}

cmd_restart_scheduler() {
  validate_env
  ensure_compose_cmd
  echo "==> 使用: ${COMPOSE_CMD[*]}"
  print_storage_mode
  prepare_storage_layout
  echo "==> 重启 scheduler 服务..."
  compose_run restart scheduler
  print_success
}

cmd_build_app_base() {
  validate_env
  ensure_compose_cmd

  local cache_flag=""
  if [[ -n "${ARG1:-}" ]]; then
    if [[ "$ARG1" == "--no-cache" ]]; then
      cache_flag="--no-cache"
    else
      echo "ERROR: build-app-base 仅支持可选参数 --no-cache"
      exit 1
    fi
  fi

  if ! compose_run exec main bash -lc 'true' >/dev/null 2>&1; then
    echo "ERROR: main 服务未运行，无法进入容器单独构建 ai-agent-os:latest"
    echo "请先执行: bash build.sh up"
    exit 1
  fi

  echo "==> 在 main 容器内单独构建 ai-agent-os:latest ..."
  echo "==> 命令: podman build ${cache_flag} -t ai-agent-os:latest /app/app-base"
  compose_run exec main bash -lc "set -euo pipefail; podman build ${cache_flag} -t ai-agent-os:latest /app/app-base"
}

cmd_logs() {
  validate_env
  local service="${ARG1:-main}"
  echo "==> 查看日志: $service"
  compose_run logs -f "$service"
}

cmd_status() {
  validate_env
  echo "==> 服务状态"
  compose_run ps
}

cmd_down() {
  validate_env
  echo "==> 停止服务（保留数据卷）..."
  compose_run down
}

case "$COMMAND" in
  up)
    cmd_up
    ;;
  update)
    cmd_update
    ;;
  pull-update)
    cmd_pull_update
    ;;
  restart-main)
    cmd_restart_main
    ;;
  restart-scheduler)
    cmd_restart_scheduler
    ;;
  build-app-base)
    cmd_build_app_base
    ;;
  logs)
    cmd_logs
    ;;
  status|ps)
    cmd_status
    ;;
  down)
    cmd_down
    ;;
  help|-h|--help)
    usage
    ;;
  *)
    echo "ERROR: 未知命令: $COMMAND"
    echo ""
    usage
    exit 1
    ;;
esac
