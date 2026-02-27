#!/bin/bash
set -e

# =============================================
# AI Agent OS - 服务器部署/更新脚本
# 用法:
#   首次部署: bash deploy/server-deploy.sh init
#   更新代码: bash deploy/server-deploy.sh update
#   仅重启:   bash deploy/server-deploy.sh restart
#   停止服务: bash deploy/server-deploy.sh stop
#   查看状态: bash deploy/server-deploy.sh status
# =============================================

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="$PROJECT_DIR/bin"
LOG_DIR="$PROJECT_DIR/logs"
PID_DIR="$PROJECT_DIR/run"
WEB_DIST="$PROJECT_DIR/web/dist"
HUB_DIST="$PROJECT_DIR/enterprise_impl/hub/frontend/dist"

mkdir -p "$BIN_DIR" "$LOG_DIR" "$PID_DIR"

# ==================== 辅助函数 ====================

info()  { echo -e "\033[32m[INFO]\033[0m $1"; }
warn()  { echo -e "\033[33m[WARN]\033[0m $1"; }
error() { echo -e "\033[31m[ERROR]\033[0m $1"; }

stop_process() {
    local name=$1
    local pid_file="$PID_DIR/${name}.pid"
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
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
    cd "$PROJECT_DIR"
    nohup "$binary" >> "$log_file" 2>&1 &
    echo $! > "$pid_file"
    info "$name 已启动 (PID: $(cat $pid_file))，日志: $log_file"
}

check_process() {
    local name=$1
    local pid_file="$PID_DIR/${name}.pid"
    if [ -f "$pid_file" ] && kill -0 "$(cat $pid_file)" 2>/dev/null; then
        echo -e "  $name: \033[32m运行中\033[0m (PID: $(cat $pid_file))"
    else
        echo -e "  $name: \033[31m未运行\033[0m"
    fi
}

# ==================== 基础设施（Docker 容器） ====================

start_infra() {
    info "启动基础设施容器 (MySQL/NATS/MinIO)..."
    cd "$PROJECT_DIR"
    docker compose -f docker-compose.dev.yml up -d
    info "等待 MySQL 就绪..."
    for i in $(seq 1 30); do
        if docker exec ai-agent-os-dev-mysql mysqladmin ping -h localhost -uroot -proot --silent 2>/dev/null; then
            info "MySQL 已就绪"
            # 自动创建所有服务需要的数据库
            info "确保数据库已创建..."
            docker exec -i ai-agent-os-dev-mysql mysql -uroot -proot -e "
                CREATE DATABASE IF NOT EXISTS \`app_db\` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
                CREATE DATABASE IF NOT EXISTS \`agent-server\` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
                CREATE DATABASE IF NOT EXISTS \`hr-server\` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
                CREATE DATABASE IF NOT EXISTS \`app-storage\` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
                CREATE DATABASE IF NOT EXISTS \`hub\` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
            " && info "数据库创建/确认完成" || warn "数据库创建失败，请手动检查"
            return 0
        fi
        sleep 2
    done
    warn "MySQL 可能还未完全就绪，继续..."
}

# ==================== 编译 Go 后端 ====================

build_backend() {
    info "编译 core-server（7 个服务统一入口）..."
    cd "$PROJECT_DIR"

    # 安装编译依赖（首次需要）
    if ! dpkg -l | grep -q libgpgme-dev 2>/dev/null; then
        warn "安装编译依赖..."
        sudo apt-get update -qq && sudo apt-get install -y -qq gcc libc6-dev libgpgme-dev libdevmapper-dev pkg-config >/dev/null 2>&1 || true
    fi

    CGO_ENABLED=1 GOOS=linux go build \
        -tags "exclude_graphdriver_btrfs" \
        -ldflags="-s -w" \
        -o "$BIN_DIR/core-server" \
        ./core/cmd/main/main.go

    info "编译 hub-server..."
    cd "$PROJECT_DIR/enterprise_impl/hub/backend"
    CGO_ENABLED=0 GOOS=linux go build \
        -ldflags="-s -w" \
        -o "$BIN_DIR/hub-server" \
        ./cmd/app/main.go

    info "Go 后端编译完成"
}

# ==================== 构建前端 ====================

build_frontend() {
    info "构建 Web 前端..."
    cd "$PROJECT_DIR/web"
    npm install --silent 2>/dev/null
    # 覆盖生产环境变量：API 使用相对路径（Nginx 反向代理）
    cat > .env.production.local << 'ENVEOF'
VITE_API_BASE_URL=
VITE_APP_TITLE=AI Agent OS
VITE_APP_ENV=production
ENVEOF
    npm run build-only
    info "Web 前端构建完成 → $WEB_DIST"

    info "构建 Hub 前端..."
    cd "$PROJECT_DIR/enterprise_impl/hub/frontend"
    npm install --silent 2>/dev/null
    npm run build-only
    info "Hub 前端构建完成 → $HUB_DIST"
}

# ==================== Nginx 配置 ====================

setup_nginx() {
    info "配置 Nginx..."

    # 安装 Nginx（如未安装）
    if ! command -v nginx &>/dev/null; then
        info "安装 Nginx..."
        sudo apt-get update -qq && sudo apt-get install -y -qq nginx
    fi

    # 更新配置中的路径
    local conf="$PROJECT_DIR/deploy/nginx-server.conf"
    local target="/etc/nginx/sites-available/ai-agent-os.conf"

    sudo cp "$conf" "$target"
    sudo sed -i "s|/opt/ai-agent-os/web/dist|$WEB_DIST|g" "$target"
    sudo sed -i "s|/opt/ai-agent-os/hub-frontend/dist|$HUB_DIST|g" "$target"

    # 启用站点
    sudo ln -sf "$target" /etc/nginx/sites-enabled/ai-agent-os.conf
    # 删除默认站点（避免冲突）
    sudo rm -f /etc/nginx/sites-enabled/default 2>/dev/null || true

    sudo nginx -t && sudo systemctl reload nginx
    info "Nginx 配置完成"
}

# ==================== Podman 基础镜像 ====================

build_app_base_image() {
    if podman image exists ai-agent-os:latest 2>/dev/null; then
        info "用户应用基础镜像已存在，跳过"
        return 0
    fi

    info "构建用户应用基础镜像（首次约 10-20 分钟）..."
    cd "$PROJECT_DIR"
    podman build -t ai-agent-os:latest -f docker/Dockerfile.app-base .
    info "基础镜像构建完成"
}

# ==================== 启动 Podman 服务 ====================

ensure_podman_service() {
    if [ -S /run/podman/podman.sock ]; then
        return 0
    fi
    info "启动 Podman 系统服务..."
    podman system service --time=0 unix:///run/podman/podman.sock &
    sleep 1
}

# ==================== 命令入口 ====================

case "${1:-}" in

init)
    echo "========================================="
    echo "  AI Agent OS - 首次部署"
    echo "========================================="

    # 检查依赖
    command -v go >/dev/null    || { error "Go 未安装"; exit 1; }
    command -v node >/dev/null  || { error "Node.js 未安装"; exit 1; }
    command -v docker >/dev/null || { error "Docker 未安装"; exit 1; }
    command -v podman >/dev/null || { error "Podman 未安装"; exit 1; }

    start_infra
    build_backend
    build_frontend
    setup_nginx
    ensure_podman_service
    build_app_base_image
    start_backend "core-server" "$BIN_DIR/core-server"
    start_backend "hub-server" "$BIN_DIR/hub-server"

    echo ""
    echo "========================================="
    echo "  部署完成！"
    echo "========================================="
    echo "  Web 前端:   http://服务器IP:8999"
    echo "  Hub 前端:   http://服务器IP:8998"
    echo "  MinIO 控制台: http://服务器IP:9001"
    echo "========================================="
    ;;

update)
    echo "========================================="
    echo "  AI Agent OS - 更新部署"
    echo "========================================="

    info "拉取最新代码..."
    cd "$PROJECT_DIR"
    git pull

    build_backend
    build_frontend

    stop_process "core-server"
    stop_process "hub-server"
    ensure_podman_service
    start_backend "core-server" "$BIN_DIR/core-server"
    start_backend "hub-server" "$BIN_DIR/hub-server"

    sudo nginx -t && sudo systemctl reload nginx

    info "更新完成！"
    ;;

restart)
    stop_process "core-server"
    stop_process "hub-server"
    ensure_podman_service
    start_backend "core-server" "$BIN_DIR/core-server"
    start_backend "hub-server" "$BIN_DIR/hub-server"
    info "重启完成"
    ;;

stop)
    stop_process "core-server"
    stop_process "hub-server"
    info "所有后端服务已停止"
    ;;

status)
    echo "=== Go 后端 ==="
    check_process "core-server"
    check_process "hub-server"
    echo ""
    echo "=== 基础设施容器 ==="
    docker compose -f "$PROJECT_DIR/docker-compose.dev.yml" ps 2>/dev/null || echo "  未启动"
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
    systemctl is-active nginx 2>/dev/null && echo -e "  Nginx: \033[32m运行中\033[0m" || echo -e "  Nginx: \033[31m未运行\033[0m"
    ;;

logs)
    local service="${2:-core-server}"
    tail -f "$LOG_DIR/${service}.log"
    ;;

*)
    echo "用法: $0 {init|update|restart|stop|status|logs [service]}"
    echo ""
    echo "  init    - 首次部署（安装依赖+编译+构建+启动）"
    echo "  update  - 更新部署（git pull+编译+构建+重启）"
    echo "  restart - 仅重启后端服务"
    echo "  stop    - 停止后端服务"
    echo "  status  - 查看所有服务状态"
    echo "  logs    - 查看日志（默认 core-server，可指定 hub-server）"
    exit 1
    ;;

esac
