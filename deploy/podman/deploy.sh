#!/bin/bash
set -e

# AI Agent OS 一键部署脚本
# 用法: bash deploy/podman/deploy.sh（在仓库根执行）

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# deploy/podman → 仓库根为上两级
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$PROJECT_DIR"

COMPOSE_FILE="$PROJECT_DIR/deploy/compose/docker-compose.yml"

echo "========================================="
echo "  AI Agent OS - 一键部署"
echo "========================================="
echo ""

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo "[错误] Docker 未安装，请先安装 Docker"
    echo "  安装命令: curl -fsSL https://get.docker.com | sh"
    exit 1
fi

# 检查 Docker Compose
if ! docker compose version &> /dev/null; then
    echo "[错误] Docker Compose 未安装"
    echo "  Docker Compose V2 通常随 Docker 一起安装"
    exit 1
fi

echo "[1/3] 构建后端镜像（含内嵌 Podman 容器引擎）..."
docker compose -f "$COMPOSE_FILE" build backend

echo ""
echo "[2/3] 构建前端镜像..."
docker compose -f "$COMPOSE_FILE" build web hub-frontend

echo ""
echo "[3/3] 启动所有服务..."
docker compose -f "$COMPOSE_FILE" up -d

echo ""
echo "========================================="
echo "  部署完成！"
echo "========================================="
echo ""
echo "  Web 主前端:   http://localhost"
echo "  Hub 前端:     http://localhost:81"
echo "  API Gateway:  http://localhost:9090"
echo "  MinIO 控制台: http://localhost:9001"
echo "    用户名: minioadmin"
echo "    密码:   minioadmin123"
echo ""
echo "  查看服务状态: docker compose -f deploy/compose/docker-compose.yml ps"
echo "  查看日志:     docker compose -f deploy/compose/docker-compose.yml logs -f"
echo "  停止服务:     docker compose -f deploy/compose/docker-compose.yml down"
echo "  清除数据:     docker compose -f deploy/compose/docker-compose.yml down -v"
echo ""
echo "  ⚠️  首次启动时后端会自动构建用户应用基础镜像（约 10-20 分钟）"
echo "     可通过日志查看进度: docker compose -f deploy/compose/docker-compose.yml logs -f backend"
echo "  ⚠️  首次启动可能需要等待 MySQL 初始化完成（约 30 秒）"
echo "  ⚠️  系统账号密码见 deploy/config/compose/hr-server.yaml 中的 system_user.password"
echo ""
echo "  📋 重要：请确认 deploy/config/compose/app-storage.yaml 中的 cdn_domain"
echo "     已配置为服务器的外网域名或 IP（如 http://your-domain.com）"
echo "========================================="
