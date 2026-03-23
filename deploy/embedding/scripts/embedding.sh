#!/bin/bash
# =============================================
# AI Agent OS - Embedding 部署脚本（Linux + Podman）
#
# 在 deploy/server-deploy.sh 能力基础上，中间件改为 Podman Compose；并保留：
#   - 前端构建（Web + Hub）
#   - Nginx 反向代理（deploy/embedding/nginx/nginx-server.conf）
#   - Go 后端编译
#   - update：git pull + 编译 + 构建前端 + 重启进程 + reload nginx
#   - restart / stop / status / logs
#
# 用法见下方 usage 或: bash deploy/embedding/scripts/embedding.sh --help
#
# 须在仓库根目录执行，或保证脚本内 cd 到的 ROOT 为正确仓库根。
# =============================================

# -e: 任一命令失败即退出
# -u: 使用未定义变量时报错
# -o pipefail: 管道中任一环节失败则整个管道失败
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

BIN_DIR="$ROOT/bin"
LOG_DIR="$ROOT/logs"
PID_DIR="$ROOT/run"
WEB_DIST="$ROOT/web/dist"
HUB_DIST="$ROOT/enterprise_impl/hub/frontend/dist"

mkdir -p "$BIN_DIR" "$LOG_DIR" "$PID_DIR"

# ==================== 辅助函数（与 server-deploy.sh 对齐） ====================

info()  { echo -e "\033[32m[INFO]\033[0m $1"; }
warn()  { echo -e "\033[33m[WARN]\033[0m $1"; }
error() { echo -e "\033[31m[ERROR]\033[0m $1"; }

stop_process() {
  local name=$1
  local pid_file="$PID_DIR/${name}.pid"
  if [ -f "$pid_file" ]; then
    local pid
    pid=$(cat "$pid_file")
    if kill -0 "$pid" 2>/dev/null; then
      info "停止 $name (PID: $pid)..."
      kill -TERM "$pid" 2>/dev/null || true
      sleep 2
      kill -0 "$pid" 2>/dev/null && kill -9 "$pid" 2>/dev/null || true
    fi
    rm -f "$pid_file"
  fi
}

start_backend() {
  local name=$1
  local binary=$2
  local log_file="$LOG_DIR/${name}.log"
  local pid_file="$PID_DIR/${name}.pid"

  stop_process "$name"

  info "启动 $name..."
  cd "$ROOT"
  nohup "$binary" >> "$log_file" 2>&1 &
  echo $! > "$pid_file"
  info "$name 已启动 (PID: $(cat "$pid_file"))，日志: $log_file"
}

check_process() {
  local name=$1
  local pid_file="$PID_DIR/${name}.pid"
  if [ -f "$pid_file" ] && kill -0 "$(cat "$pid_file")" 2>/dev/null; then
    echo -e "  $name: \033[32m运行中\033[0m (PID: $(cat "$pid_file"))"
  else
    echo -e "  $name: \033[31m未运行\033[0m"
  fi
}

# ==================== 用法说明（--help） ====================

usage() {
  cat <<'EOF'
用法: bash deploy/embedding/scripts/embedding.sh <命令> [参数]

【栈与构建】
  all              一键：infra → local（若存在目录）→ dbs → build → runtime（不含 Nginx/前端/启进程）
  infra            Podman Compose 启动 MySQL / NATS / MinIO
  local            将 deploy/config/local/*.yaml 覆盖到 deploy/config/prod/
  dbs              幂等创建 embedding 所需 MySQL 库
  build            编译 bin/core-server、bin/hub-server
  runtime          podman build → 镜像 ai-agent-os:latest（build/Dockerfile）
  frontend         构建 Web + Hub 前端（npm）
  nginx            安装/更新 Nginx 站点配置并 reload（需 sudo）

【首次完整上线（参考 server-deploy.sh init）】
  init             infra → local? → dbs → build → frontend → nginx → podman.sock → runtime → 启动 core/hub

【运维（参考 server-deploy.sh）】
  update           git pull → build → frontend → 重启 core/hub → reload nginx（不重建 runtime 镜像，节省时间）
  restart          仅重启 core-server、hub-server（需已编译出 bin）
  stop             停止上述进程
  status           后端 PID、Podman 中间件、Podman 镜像摘要、Nginx
  logs [服务名]    tail -f 日志，默认 core-server，可 hub-server

示例:
  bash deploy/embedding/scripts/embedding.sh init
  bash deploy/embedding/scripts/embedding.sh update
  bash deploy/embedding/scripts/embedding.sh status
  bash deploy/embedding/scripts/embedding.sh logs hub-server
EOF
}

require_root_cd() {
  cd "$ROOT"
}

# Podman 的 `podman compose` 是外包给「compose 提供者」的；若系统里装了 docker-compose，
# 默认会优先用它，镜像由 Docker 拉取（国内常直连 registry-1.docker.io 超时），
# 不会走你在 /etc/containers/registries.conf 里配的镜像加速。
# 强制使用 podman-compose 后，拉镜像由 Podman 完成，可走腾讯云等 mirror。
ensure_podman_compose_provider() {
  if command -v podman-compose &>/dev/null; then
    export PODMAN_COMPOSE_PROVIDER="$(command -v podman-compose)"
    # 减少 “Executing external compose provider …” 刷屏（可选）
    export PODMAN_COMPOSE_WARNING_LOGS="${PODMAN_COMPOSE_WARNING_LOGS:-false}"
    return 0
  fi
  warn "未找到 podman-compose：当前 podman compose 可能改用 docker-compose，拉镜像不走 Podman 镜像加速（易超时）。"
  warn "建议安装: sudo apt-get install -y podman-compose   （或 pip install podman-compose）"
  return 1
}

# ==================== 基础设施（Podman Compose） ====================

cmd_infra() {
  require_root_cd

  if ! command -v podman &>/dev/null; then
    error "未找到 podman，请先安装 Podman。"
    exit 1
  fi
  ensure_podman_compose_provider || true

  if ! podman compose version &>/dev/null && ! podman-compose version &>/dev/null; then
    error "需要 podman compose（Podman 4+）或 podman-compose 插件。"
    exit 1
  fi

  info "在 $ROOT 启动基础设施..."
  podman compose -f docker-compose.dev.yml up -d

  info "等待 MySQL 就绪（最多约 60s）..."
  for _ in $(seq 1 30); do
    if podman exec ai-agent-os-dev-mysql mysqladmin ping -h localhost -uroot -proot --silent 2>/dev/null; then
      info "MySQL 已就绪"
      return 0
    fi
    sleep 2
  done

  warn "MySQL 可能尚未就绪，请稍后执行: bash deploy/embedding/scripts/embedding.sh dbs"
}

# ==================== 本机配置覆盖 ====================

cmd_local() {
  require_root_cd
  local LOCAL="$ROOT/deploy/config/local"
  local TARGET="$ROOT/deploy/config/prod"

  if [ ! -d "$LOCAL" ]; then
    error "目录不存在: $LOCAL"
    exit 1
  fi
  if [ ! -d "$TARGET" ]; then
    error "目录不存在: $TARGET"
    exit 1
  fi

  local count=0
  shopt -s nullglob
  for f in "$LOCAL"/*.yaml "$LOCAL"/*.yml; do
    local base
    base="$(basename "$f")"
    cp -f "$f" "$TARGET/$base"
    info "已应用: $base -> deploy/config/prod/"
    count=$((count + 1))
  done
  shopt -u nullglob

  if [ "$count" -eq 0 ]; then
    info "local/ 下无 .yaml/.yml，跳过"
  else
    info "共应用 $count 个文件。请勿将敏感内容提交到 Git。"
  fi
}

# ==================== MySQL 数据库 ====================

cmd_dbs() {
  require_root_cd
  local CT="ai-agent-os-dev-mysql"

  if ! podman ps --format '{{.Names}}' | grep -qx "$CT"; then
    error "容器未运行: $CT ，请先执行: bash deploy/embedding/scripts/embedding.sh infra"
    exit 1
  fi

  info "创建/确认数据库..."
  podman exec -i "$CT" mysql -uroot -proot <<'SQL'
CREATE DATABASE IF NOT EXISTS `app_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
CREATE DATABASE IF NOT EXISTS `agent-server` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
CREATE DATABASE IF NOT EXISTS `hr-server` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
CREATE DATABASE IF NOT EXISTS `app-storage` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
CREATE DATABASE IF NOT EXISTS `hub` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
SQL

  info "数据库已就绪（幂等）"
}

# ==================== Debian/Ubuntu：CGO 编译依赖（与 server-deploy.sh 一致） ====================

maybe_install_go_cgo_deps() {
  # 非 Debian/Ubuntu（无 dpkg）则跳过，由本机自行保证 CGO 依赖
  if command -v dpkg >/dev/null 2>&1 && ! dpkg -l 2>/dev/null | grep -q libgpgme-dev; then
    warn "尝试安装 core-server 编译依赖（需 sudo），下载可能较慢，请稍候（勿误以为卡住）..."
    # 勿将 install 输出丢到 /dev/null，否则网络慢时长时间无日志像死机
    sudo apt-get update -qq && sudo apt-get install -y gcc libc6-dev libgpgme-dev libdevmapper-dev pkg-config || true
  fi
}

# ==================== 编译 Go 后端 ====================

cmd_build() {
  require_root_cd
  mkdir -p "$BIN_DIR"

  if ! command -v go &>/dev/null; then
    error "未找到 go"
    exit 1
  fi

  maybe_install_go_cgo_deps

  info "编译 core-server -> $BIN_DIR/core-server"
  CGO_ENABLED=1 GOOS=linux go build \
    -tags "exclude_graphdriver_btrfs" \
    -ldflags="-s -w" \
    -o "$BIN_DIR/core-server" \
    ./core/cmd/main/main.go

  info "编译 hub-server -> $BIN_DIR/hub-server"
  cd "$ROOT/enterprise_impl/hub/backend"
  CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o "$BIN_DIR/hub-server" \
    ./cmd/app/main.go

  cd "$ROOT"
  info "Go 后端编译完成"
}

# ==================== 构建前端（与 server-deploy.sh build_frontend 一致） ====================

cmd_frontend() {
  require_root_cd

  if ! command -v node &>/dev/null; then
    error "未找到 Node.js"
    exit 1
  fi

  info "构建 Web 前端..."
  cd "$ROOT/web"
  npm install --silent 2>/dev/null || npm install
  # 生产环境：API 走相对路径（由 Nginx 反代）
  cat > .env.production.local << 'ENVEOF'
VITE_API_BASE_URL=
VITE_APP_TITLE=AI Agent OS
VITE_APP_ENV=production
ENVEOF
  npm run build-only
  info "Web 前端构建完成 → $WEB_DIST"

  info "构建 Hub 前端..."
  cd "$ROOT/enterprise_impl/hub/frontend"
  npm install --silent 2>/dev/null || npm install
  npm run build-only
  info "Hub 前端构建完成 → $HUB_DIST"
}

# ==================== Nginx（与 server-deploy.sh setup_nginx 一致） ====================

cmd_nginx() {
  require_root_cd

  if ! command -v nginx &>/dev/null; then
    info "安装 Nginx（需 sudo）..."
    sudo apt-get update -qq && sudo apt-get install -y -qq nginx
  fi

  if [ ! -f "$WEB_DIST/index.html" ]; then
    error "未找到 $WEB_DIST/index.html，请先执行: bash deploy/embedding/scripts/embedding.sh frontend"
    exit 1
  fi
  if [ ! -f "$HUB_DIST/index.html" ]; then
    error "未找到 $HUB_DIST/index.html，请先执行: bash deploy/embedding/scripts/embedding.sh frontend"
    exit 1
  fi

  # 仓库在 /root 等目录时，Nginx（www-data）无法 stat /root/... → 500 + 重定向死循环。
  # 同步到 /opt 并开放读权限，配置模板保持 deploy/embedding/nginx 中的 /opt/ai-agent-os/... 路径即可。
  local deploy_root="/opt/ai-agent-os"
  info "同步静态资源到 $deploy_root（供 Nginx 读取）..."
  sudo mkdir -p "$deploy_root/web/dist" "$deploy_root/hub-frontend/dist"
  sudo rsync -a --delete "$WEB_DIST/" "$deploy_root/web/dist/"
  sudo rsync -a --delete "$HUB_DIST/" "$deploy_root/hub-frontend/dist/"
  sudo chmod -R a+rX "$deploy_root"

  local conf="$ROOT/deploy/embedding/nginx/nginx-server.conf"
  local target="/etc/nginx/sites-available/ai-agent-os.conf"

  sudo cp "$conf" "$target"

  sudo ln -sf "$target" /etc/nginx/sites-enabled/ai-agent-os.conf
  sudo rm -f /etc/nginx/sites-enabled/default 2>/dev/null || true

  sudo nginx -t && sudo systemctl reload nginx
  info "Nginx 配置完成"
}

# ==================== Podman API Socket（app-runtime 需要） ====================

ensure_podman_service() {
  if [ -S /run/podman/podman.sock ]; then
    return 0
  fi
  info "启动 Podman 系统服务（后台）..."
  podman system service --time=0 unix:///run/podman/podman.sock &
  sleep 1
}

# ==================== 用户应用运行时镜像 ====================

cmd_runtime() {
  require_root_cd
  if ! command -v podman &>/dev/null; then
    error "未找到 podman"
    exit 1
  fi

  info "podman build -f build/Dockerfile -t ai-agent-os:latest（耗时较长）..."
  podman build -f build/Dockerfile -t ai-agent-os:latest .

  info "镜像 ai-agent-os:latest 已构建"
}

# ==================== 一键：仅栈（无 Nginx/前端/进程） ====================

cmd_all() {
  cmd_infra
  if [ -d "$ROOT/deploy/config/local" ]; then
    cmd_local
  else
    info "跳过 local（无 deploy/config/local/ 目录）"
  fi
  cmd_dbs
  cmd_build
  cmd_runtime
}

# ==================== 首次完整部署（对齐 server-deploy.sh init） ====================

cmd_init() {
  echo "========================================="
  echo "  AI Agent OS - Embedding 首次部署"
  echo "========================================="

  command -v go >/dev/null    || { error "Go 未安装"; exit 1; }
  command -v node >/dev/null  || { error "Node.js 未安装"; exit 1; }
  command -v podman >/dev/null || { error "Podman 未安装"; exit 1; }

  cmd_infra
  if [ -d "$ROOT/deploy/config/local" ]; then
    cmd_local
  else
    info "跳过 local（无 deploy/config/local/ 目录）"
  fi
  cmd_dbs
  cmd_build
  cmd_frontend
  cmd_nginx
  ensure_podman_service
  cmd_runtime
  start_backend "core-server" "$BIN_DIR/core-server"
  start_backend "hub-server" "$BIN_DIR/hub-server"

  echo ""
  echo "========================================="
  echo "  部署完成（Embedding + Nginx）"
  echo "========================================="
  echo "  Web 前端:     http://服务器IP:8999"
  echo "  Hub 前端:     http://服务器IP:8998"
  echo "  MinIO 控制台: http://服务器IP:9001"
  echo "========================================="
}

# ==================== 更新部署（对齐 server-deploy.sh update） ====================

cmd_update() {
  echo "========================================="
  echo "  AI Agent OS - Embedding 更新部署"
  echo "========================================="

  if [ ! -d "$ROOT/.git" ]; then
    error "非 git 仓库，无法 pull: $ROOT"
    exit 1
  fi

  info "拉取最新代码..."
  cd "$ROOT"
  git pull

  cmd_build
  cmd_frontend
  if command -v nginx &>/dev/null; then
    cmd_nginx
  else
    warn "未检测到 nginx，跳过静态同步与站点配置"
  fi

  stop_process "core-server"
  stop_process "hub-server"
  ensure_podman_service
  start_backend "core-server" "$BIN_DIR/core-server"
  start_backend "hub-server" "$BIN_DIR/hub-server"

  info "更新完成！"
}

cmd_restart() {
  stop_process "core-server"
  stop_process "hub-server"
  ensure_podman_service
  start_backend "core-server" "$BIN_DIR/core-server"
  start_backend "hub-server" "$BIN_DIR/hub-server"
  info "重启完成"
}

cmd_stop() {
  stop_process "core-server"
  stop_process "hub-server"
  info "所有后端服务已停止"
}

cmd_status() {
  echo "=== Go 后端 ==="
  check_process "core-server"
  check_process "hub-server"
  echo ""
  echo "=== 基础设施容器（Podman） ==="
  (
    cd "$ROOT"
    ensure_podman_compose_provider || true
    podman compose -f docker-compose.dev.yml ps 2>/dev/null
  ) || echo "  未启动或 compose 失败"
  echo ""
  echo "=== Podman ==="
  if [ -S /run/podman/podman.sock ]; then
    echo -e "  Podman 服务: \033[32m运行中\033[0m"
    podman images --format "  镜像: {{.Repository}}:{{.Tag}} ({{.Size}})" 2>/dev/null | head -5
  else
    echo -e "  Podman 服务: \033[31m未运行\033[0m"
  fi
  echo ""
  echo "=== Nginx ==="
  if systemctl is-active nginx &>/dev/null; then
    echo -e "  Nginx: \033[32m运行中\033[0m"
  else
    echo -e "  Nginx: \033[31m未运行\033[0m"
  fi
}

cmd_logs() {
  local service="${1:-core-server}"
  tail -f "$LOG_DIR/${service}.log"
}

# ==================== 命令入口 ====================

main() {
  local sub="${1:-}"
  case "$sub" in
    infra)     cmd_infra ;;
    local)     cmd_local ;;
    dbs)       cmd_dbs ;;
    build)     cmd_build ;;
    runtime)   cmd_runtime ;;
    frontend)  cmd_frontend ;;
    nginx)     cmd_nginx ;;
    all)       cmd_all ;;
    init)      cmd_init ;;
    update)    cmd_update ;;
    restart)   cmd_restart ;;
    stop)      cmd_stop ;;
    status)    cmd_status ;;
    logs)      cmd_logs "${2:-core-server}" ;;
    -h|--help|help)
      usage
      exit 0
      ;;
    "")
      usage
      exit 1
      ;;
    *)
      error "未知命令: $sub"
      usage
      exit 1
      ;;
  esac
}

main "$@"
