#!/bin/sh

# AI Agent OS 容器入口脚本
# 支持多种启动模式和配置

set -e

# 默认配置
APP_DIR="${APP_DIR:-/app/workplace/bin}"
APP_NAME="${APP_NAME:-app}"
WAIT_FOR_APP="${WAIT_FOR_APP:-true}"
MAX_WAIT_TIME="${MAX_WAIT_TIME:-30}"

echo "=== AI Agent OS 容器启动 ==="
echo "📁 应用目录: $APP_DIR"
echo "📱 应用名称: $APP_NAME"
echo "⏳ 等待应用: $WAIT_FOR_APP"
echo "⏰ 最大等待时间: ${MAX_WAIT_TIME}秒"

# 等待应用文件出现
if [ "$WAIT_FOR_APP" = "true" ]; then
    echo "🔍 等待应用文件出现..."
    wait_count=0
    while [ $wait_count -lt $MAX_WAIT_TIME ]; do
        if [ -d "$APP_DIR" ] && [ -L "$APP_DIR/$APP_NAME" ]; then
            echo "✅ 应用文件已出现"
            break
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
    echo "请确保正确挂载了应用目录或设置正确的 APP_DIR 环境变量"
    exit 1
fi

# 检查软链接是否存在
if [ ! -L "$APP_DIR/$APP_NAME" ]; then
    echo "❌ 错误: 应用软链接不存在 $APP_DIR/$APP_NAME"
    echo "请确保应用已正确部署"
    exit 1
fi

# 检查目标文件是否存在
TARGET=$(readlink "$APP_DIR/$APP_NAME")
if [ ! -f "$APP_DIR/releases/$TARGET" ]; then
    echo "❌ 错误: 目标文件不存在 $APP_DIR/releases/$TARGET"
    exit 1
fi

# 检查执行权限
if [ ! -x "$APP_DIR/releases/$TARGET" ]; then
    echo "⚠️  警告: 目标文件不可执行，尝试设置执行权限"
    chmod +x "$APP_DIR/releases/$TARGET"
fi

echo "✅ 应用检查通过"
echo "📱 启动应用: $TARGET"
echo "🚀 执行命令: $APP_DIR/$APP_NAME"

# 如果提供了自定义命令，执行它
if [ $# -gt 0 ]; then
    echo "🔧 执行自定义命令: $@"
    exec "$@"
else
    # 启动应用
    exec "$APP_DIR/$APP_NAME"
fi





