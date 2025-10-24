#!/bin/bash

# 启动应用容器脚本
# 用途：启动一个包含应用代码的容器，并挂载应用目录

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 默认参数
USER_NAME="${1:-user1}"
APP_NAME="${2:-app1}"
CONTAINER_NAME="${USER_NAME}-${APP_NAME}"
IMAGE="golang:1.23-alpine"
HOST_PORT="${3:-8081}"
CONTAINER_PORT="8080"
AUTO_BUILD="${4:-true}"  # 是否自动编译

# 路径设置
WORKSPACE_ROOT="/Users/beiluo/Documents/work/code/ai-agent-os"
APP_DIR="${WORKSPACE_ROOT}/namespace/${USER_NAME}/${APP_NAME}"
MOUNT_PATH="/app"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}启动应用容器${NC}"
echo -e "${GREEN}========================================${NC}"
echo -e "用户: ${YELLOW}${USER_NAME}${NC}"
echo -e "应用: ${YELLOW}${APP_NAME}${NC}"
echo -e "容器: ${YELLOW}${CONTAINER_NAME}${NC}"
echo -e "挂载: ${YELLOW}${APP_DIR} -> ${MOUNT_PATH}${NC}"
echo -e "端口: ${YELLOW}${HOST_PORT}:${CONTAINER_PORT}${NC}"
echo ""

# 检查应用目录是否存在
if [ ! -d "${APP_DIR}" ]; then
    echo -e "${RED}错误: 应用目录不存在: ${APP_DIR}${NC}"
    exit 1
fi

# 检查 podman 是否安装
if ! command -v podman &> /dev/null; then
    echo -e "${RED}错误: podman 未安装${NC}"
    echo "请运行: brew install podman"
    exit 1
fi

# 检查 podman machine 是否运行
if ! podman machine list | grep -q "Currently running"; then
    echo -e "${YELLOW}警告: Podman machine 未运行，正在启动...${NC}"
    podman machine start
    sleep 5
fi

# 检查是否已存在同名容器
if podman ps -a --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
    echo -e "${YELLOW}警告: 容器 ${CONTAINER_NAME} 已存在${NC}"
    read -p "是否删除并重新创建？(y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "停止并删除旧容器..."
        podman stop ${CONTAINER_NAME} 2>/dev/null || true
        podman rm ${CONTAINER_NAME} 2>/dev/null || true
    else
        echo "取消操作"
        exit 0
    fi
fi

# 编译 Linux 版本的应用（如果需要）
if [ "${AUTO_BUILD}" = "true" ]; then
    echo -e "${GREEN}正在编译 Linux 版本的应用...${NC}"
    cd ${WORKSPACE_ROOT}
    GOOS=linux GOARCH=arm64 go build -o ${APP_DIR}/code/cmd/app/app_linux ./namespace/${USER_NAME}/${APP_NAME}/code/cmd/app/main.go
    echo -e "${GREEN}✅ 编译完成${NC}"
fi

# 启动容器
echo -e "${GREEN}正在启动容器...${NC}"
CONTAINER_ID=$(podman run -d \
    --name ${CONTAINER_NAME} \
    -v ${WORKSPACE_ROOT}:/workspace \
    -v ${APP_DIR}:${MOUNT_PATH} \
    -p ${HOST_PORT}:${CONTAINER_PORT} \
    -e NATS_URL="nats://host.containers.internal:4223" \
    --add-host=host.containers.internal:host-gateway \
    -w /workspace \
    ${IMAGE} \
    tail -f /dev/null)

echo -e "${GREEN}✅ 容器启动成功！${NC}"
echo -e "容器 ID: ${YELLOW}${CONTAINER_ID:0:12}${NC}"
echo ""

# 验证挂载
echo -e "${GREEN}验证挂载目录...${NC}"
podman exec ${CONTAINER_NAME} ls -la ${MOUNT_PATH}
echo ""

# 显示容器信息
echo -e "${GREEN}容器信息:${NC}"
podman ps --filter name=${CONTAINER_NAME} --format "table {{.ID}}\t{{.Names}}\t{{.Status}}\t{{.Ports}}"
echo ""

# 显示使用说明
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}使用说明${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${YELLOW}1. 进入容器:${NC}"
echo -e "   podman exec -it ${CONTAINER_NAME} sh"
echo ""
echo -e "${YELLOW}2. 在容器内运行应用:${NC}"
echo -e "   podman exec -e NATS_URL=\"nats://host.containers.internal:4223\" ${CONTAINER_NAME} ${MOUNT_PATH}/code/cmd/app/app_linux"
echo ""
echo -e "${YELLOW}3. 后台运行应用:${NC}"
echo -e "   podman exec -d -e NATS_URL=\"nats://host.containers.internal:4223\" ${CONTAINER_NAME} ${MOUNT_PATH}/code/cmd/app/app_linux"
echo ""
echo -e "${YELLOW}4. 查看应用进程:${NC}"
echo -e "   podman exec ${CONTAINER_NAME} ps aux | grep app_linux"
echo ""
echo -e "${YELLOW}5. 查看容器日志:${NC}"
echo -e "   podman logs ${CONTAINER_NAME}"
echo -e "   podman logs -f ${CONTAINER_NAME}  # 实时查看"
echo ""
echo -e "${YELLOW}6. 停止容器:${NC}"
echo -e "   podman stop ${CONTAINER_NAME}"
echo ""
echo -e "${YELLOW}7. 删除容器:${NC}"
echo -e "   podman rm ${CONTAINER_NAME}"
echo ""
echo -e "${YELLOW}8. 重新编译应用（在宿主机）:${NC}"
echo -e "   cd ${WORKSPACE_ROOT} && GOOS=linux GOARCH=arm64 go build -o ${APP_DIR}/code/cmd/app/app_linux ./namespace/${USER_NAME}/${APP_NAME}/code/cmd/app/main.go"
echo ""
echo -e "${GREEN}========================================${NC}"

