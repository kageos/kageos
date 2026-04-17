#!/bin/bash
set -e

# AI Agent OS 旧后端大镜像启动脚本（Legacy）
# 支持两种模式:
#   1. 统一启动（默认）: 启动 core-server (7个服务) + hub-server
#   2. 单服务启动: SERVICE=hub-server 只启动 hub-server

cd /app

# ============================================
# Podman 初始化（app-runtime 容器管理引擎）
# ============================================

APP_BASE_IMAGE="${APP_BASE_IMAGE:-agentos-app-runtime-base:latest}"

echo "==> 启动 Podman 系统服务..."
podman system service --time=0 unix:///run/podman/podman.sock &
PODMAN_PID=$!
sleep 1

if [ -S /run/podman/podman.sock ]; then
    echo "    Podman 服务已就绪 (socket: /run/podman/podman.sock)"
else
    echo "    ⚠ Podman socket 未就绪，等待中..."
    sleep 3
fi

# 首次启动时构建用户应用基础镜像（后续使用 podman-storage 卷缓存）
if ! podman image exists "${APP_BASE_IMAGE}" 2>/dev/null; then
    echo "==> 首次启动: 构建用户应用基础镜像 (${APP_BASE_IMAGE})..."
    echo "    包含 Python3/LibreOffice/Tesseract/中文字体等，约需 10-20 分钟..."
    podman build -t "${APP_BASE_IMAGE}" -f /app/app-base/Dockerfile /app/app-base/
    echo "==> 基础镜像构建完成！"
else
    echo "==> 用户应用基础镜像已存在，跳过构建"
fi

# ============================================
# 服务启动
# ============================================

# 如果指定了 SERVICE 环境变量，只启动对应服务
if [ -n "$SERVICE" ]; then
    case "$SERVICE" in
        hub-server)
            echo "==> 启动 Hub Server..."
            exec /app/hub-server
            ;;
        core-server)
            echo "==> 启动 Core Server (7个服务)..."
            exec /app/core-server
            ;;
        *)
            echo "未知服务: $SERVICE"
            exit 1
            ;;
    esac
fi

# 默认: 统一启动全部服务
echo "========================================="
echo "  AI Agent OS - Docker 统一启动"
echo "========================================="

# 启动 Hub Server（后台）
echo "==> 启动 Hub Server..."
/app/hub-server &
HUB_PID=$!

# 启动 Core Server（后台）
echo "==> 启动 Core Server (7个服务)..."
/app/core-server &
CORE_PID=$!

# 信号处理: 收到 SIGTERM/SIGINT 后优雅关闭
shutdown() {
    echo "==> 收到停止信号，正在关闭..."
    kill -TERM $CORE_PID 2>/dev/null || true
    kill -TERM $HUB_PID 2>/dev/null || true
    kill -TERM $PODMAN_PID 2>/dev/null || true
    wait $CORE_PID 2>/dev/null || true
    wait $HUB_PID 2>/dev/null || true
    wait $PODMAN_PID 2>/dev/null || true
    echo "==> 所有服务已停止"
    exit 0
}

trap shutdown SIGTERM SIGINT

# 等待任一进程退出
wait -n $CORE_PID $HUB_PID
EXIT_CODE=$?

echo "==> 有服务异常退出 (exit code: $EXIT_CODE)，正在关闭其他服务..."
shutdown
