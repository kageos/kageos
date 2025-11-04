#!/bin/bash

# App Storage 启动脚本

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "==> 切换到项目根目录: $PROJECT_ROOT"
cd "$PROJECT_ROOT"

echo "==> 检查 MinIO 是否运行..."
if ! podman ps | grep -q minio; then
    echo "MinIO 未运行，正在启动..."
    bash scripts/podman/minio.sh
fi

echo "==> 编译 app-storage..."
go build -o bin/app-storage ./core/app-storage/cmd/app/

echo "==> 启动 app-storage..."
./bin/app-storage

