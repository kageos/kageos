#!/bin/sh

# 应用启动脚本
# 在容器启动时检查并启动应用

set -e

echo "=== AI Agent OS 应用启动脚本 ==="

# 检查应用目录是否存在
if [ ! -d "/app/workplace/bin" ]; then
    echo "❌ 错误: 应用目录不存在 /app/workplace/bin"
    echo "请确保正确挂载了应用目录"
    exit 1
fi

# 检查软链接是否存在
if [ ! -L "/app/workplace/bin/app" ]; then
    echo "❌ 错误: 应用软链接不存在 /app/workplace/bin/app"
    echo "请确保应用已正确部署"
    exit 1
fi

# 检查目标文件是否存在
TARGET=$(readlink /app/workplace/bin/app)
if [ ! -f "/app/workplace/bin/releases/$TARGET" ]; then
    echo "❌ 错误: 目标文件不存在 /app/workplace/bin/releases/$TARGET"
    exit 1
fi

# 检查执行权限
if [ ! -x "/app/workplace/bin/releases/$TARGET" ]; then
    echo "⚠️  警告: 目标文件不可执行，尝试设置执行权限"
    chmod +x "/app/workplace/bin/releases/$TARGET"
fi

echo "✅ 应用检查通过"
echo "📱 启动应用: $TARGET"
echo "🚀 执行命令: /app/workplace/bin/app"

# 启动应用
exec /app/workplace/bin/app "$@"





