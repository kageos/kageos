#!/bin/bash

# NATS Server Podman 部署脚本

CONTAINER_NAME="nats-server"
NATS_PORT="4223"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="$SCRIPT_DIR/nats-server.conf"

echo "==> 停止并删除旧容器（如果存在）"
podman stop $CONTAINER_NAME 2>/dev/null || true
podman rm $CONTAINER_NAME 2>/dev/null || true

# 检查配置文件是否存在
if [ ! -f "$CONFIG_FILE" ]; then
    echo "错误: 配置文件不存在: $CONFIG_FILE"
    exit 1
fi

echo "==> 启动 NATS Server 容器（使用配置文件）"
podman run -d \
  --name $CONTAINER_NAME \
  -p $NATS_PORT:4222 \
  -v "$CONFIG_FILE:/etc/nats/nats-server.conf:ro" \
  --restart=always \
  docker.io/library/nats:latest \
  -c /etc/nats/nats-server.conf

echo "==> 等待 NATS Server 启动..."
sleep 3

echo "==> NATS Server 启动成功！"
echo ""
echo "NATS 地址: nats://localhost:$NATS_PORT"
echo "最大消息体大小: 10MB (10485760 bytes)"
echo "配置文件: $CONFIG_FILE"
echo ""
echo "==> 容器日志："
podman logs $CONTAINER_NAME

echo ""
echo "==> 查看容器状态："
podman ps | grep $CONTAINER_NAME

