#!/bin/sh

# AI Agent OS Podman 应用启动脚本
# 专门为 Podman 和交叉编译环境设计

set -e

echo "=== AI Agent OS Podman 启动脚本 ==="

# 默认配置
APP_DIR="${APP_DIR:-/app/workplace/bin}"
APP_NAME="${APP_NAME:-app}"
WAIT_FOR_APP="${WAIT_FOR_APP:-true}"
MAX_WAIT_TIME="${MAX_WAIT_TIME:-30}"

echo "📁 应用目录: $APP_DIR"
echo "📱 应用名称: $APP_NAME"

# 等待应用文件出现（交叉编译场景）
if [ "$WAIT_FOR_APP" = "true" ]; then
    echo "🔍 等待应用文件出现（交叉编译场景）..."
    wait_count=0
    while [ $wait_count -lt $MAX_WAIT_TIME ]; do
        if [ -d "$APP_DIR" ] && [ -d "$APP_DIR/releases" ]; then
            # 检查是否有版本文件
            if ls "$APP_DIR/releases"/* >/dev/null 2>&1; then
                echo "✅ 应用文件已出现"
                break
            fi
        fi
        echo "⏳ 等待中... ($((wait_count + 1))/${MAX_WAIT_TIME})"
        sleep 1
        wait_count=$((wait_count + 1))
    done
    
    if [ $wait_count -ge $MAX_WAIT_TIME ]; then
        echo "❌ 超时: 应用文件未在 ${MAX_WAIT_TIME} 秒内出现"
        echo "请检查挂载配置或应用部署状态"
        exit 1
    fi
fi

# 检查应用目录是否存在
if [ ! -d "$APP_DIR" ]; then
    echo "❌ 错误: 应用目录不存在 $APP_DIR"
    exit 1
fi

# 检查 releases 目录是否存在
if [ ! -d "$APP_DIR/releases" ]; then
    echo "❌ 错误: releases 目录不存在 $APP_DIR/releases"
    exit 1
fi

# 查找最新版本文件
LATEST_VERSION=""
LATEST_TIME=0

for version_file in "$APP_DIR/releases"/*; do
    if [ -f "$version_file" ]; then
        file_time=$(stat -c %Y "$version_file" 2>/dev/null || stat -f %m "$version_file" 2>/dev/null || echo 0)
        if [ "$file_time" -gt "$LATEST_TIME" ]; then
            LATEST_TIME=$file_time
            LATEST_VERSION=$(basename "$version_file")
        fi
    fi
done

if [ -z "$LATEST_VERSION" ]; then
    echo "❌ 错误: 没有找到任何版本文件"
    exit 1
fi

echo "📱 找到最新版本: $LATEST_VERSION"

# 创建软链接（在容器内创建，避免平台兼容性问题）
APP_PATH="$APP_DIR/$APP_NAME"
VERSION_PATH="$APP_DIR/releases/$LATEST_VERSION"

# 删除旧的软链接
if [ -L "$APP_PATH" ] || [ -f "$APP_PATH" ]; then
    rm -f "$APP_PATH"
fi

# 创建新的软链接
ln -s "$LATEST_VERSION" "$APP_PATH"

# 设置执行权限
chmod +x "$VERSION_PATH"

echo "✅ 软链接创建成功: $APP_PATH -> $LATEST_VERSION"
echo "🚀 启动应用: $APP_PATH"

# 启动应用
exec "$APP_PATH" "$@"





