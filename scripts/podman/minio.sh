#!/bin/bash

# MinIO Podman 部署脚本
#
# 若遇到「客户端与 MinIO 服务器的时间差过大」错误，多半是 Podman Mac 上 VM 时间未与宿主机同步。
# 解决：重启 podman machine 后再启动 MinIO：
#   podman machine stop && podman machine start
#   podman start minio

CONTAINER_NAME="minio"
DATA_DIR="${HOME}/minio-data"
MINIO_PORT="9000"
CONSOLE_PORT="9001"
ROOT_USER="minioadmin"
ROOT_PASSWORD="minioadmin123"

echo "==> 停止并删除旧容器（如果存在）"
podman stop $CONTAINER_NAME 2>/dev/null || true
podman rm $CONTAINER_NAME 2>/dev/null || true

echo "==> 创建数据目录"
mkdir -p $DATA_DIR

echo "==> 启动 MinIO 容器"
# 构建挂载参数（如果 /etc/timezone 存在则挂载）
MOUNT_ARGS="-v $DATA_DIR:/data:z -v /etc/localtime:/etc/localtime:ro"
if [ -f /etc/timezone ]; then
  MOUNT_ARGS="$MOUNT_ARGS -v /etc/timezone:/etc/timezone:ro"
fi

podman run -d \
  --name $CONTAINER_NAME \
  --tz=local \
  -p $MINIO_PORT:9000 \
  -p $CONSOLE_PORT:9001 \
  -e "MINIO_ROOT_USER=$ROOT_USER" \
  -e "MINIO_ROOT_PASSWORD=$ROOT_PASSWORD" \
  -e "TZ=Asia/Shanghai" \
  $MOUNT_ARGS \
  docker.io/minio/minio:latest \
  server /data --console-address ":9001"

echo "==> 等待 MinIO 启动..."
sleep 5

echo "==> MinIO 启动成功！"
echo ""
echo "API 地址: http://localhost:$MINIO_PORT"
echo "控制台: http://localhost:$CONSOLE_PORT"
echo "用户名: $ROOT_USER"
echo "密码: $ROOT_PASSWORD"
echo ""
echo "==> 容器日志："
podman logs $CONTAINER_NAME

echo ""
echo "==> 查看容器状态："
podman ps | grep $CONTAINER_NAME

