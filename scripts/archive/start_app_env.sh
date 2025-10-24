#!/bin/sh

# 应用启动脚本（支持环境变量配置）
# 在容器启动时检查并启动应用

set -e

# 默认配置
APP_DIR="${APP_DIR:-/app/workplace/bin}"
APP_NAME="${APP_NAME:-app}"
APP_ARGS="${APP_ARGS:-}"

echo "=== AI Agent OS 应用启动脚本 ==="
echo "📁 应用目录: $APP_DIR"
echo "📱 应用名称: $APP_NAME"
echo "⚙️  应用参数: $APP_ARGS"

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
echo "🚀 执行命令: $APP_DIR/$APP_NAME $APP_ARGS"

# 启动应用
exec "$APP_DIR/$APP_NAME" $APP_ARGS





