#!/bin/bash

# 更新应用软链接脚本
# 用法: ./update_app_symlink.sh <user> <app> [version]

set -e

USER_NAME="$1"
APP_NAME="$2"
VERSION="$3"

if [ -z "$USER_NAME" ] || [ -z "$APP_NAME" ]; then
    echo "用法: $0 <user> <app> [version]"
    echo "示例: $0 user1 app1"
    echo "示例: $0 user1 app1 user1_app1_v3"
    exit 1
fi

# 获取脚本所在目录的父目录（项目根目录）
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
NAMESPACE_ROOT="$PROJECT_ROOT/namespace"

# 构建路径
USER_DIR="$NAMESPACE_ROOT/$USER_NAME"
BIN_DIR="$USER_DIR/$APP_NAME/workplace/bin"
RELEASES_DIR="$BIN_DIR/releases"
APP_PATH="$BIN_DIR/app"

# 检查目录是否存在
if [ ! -d "$BIN_DIR" ]; then
    echo "错误: bin 目录不存在: $BIN_DIR"
    exit 1
fi

if [ ! -d "$RELEASES_DIR" ]; then
    echo "错误: releases 目录不存在: $RELEASES_DIR"
    exit 1
fi

# 如果没有指定版本，自动选择最新版本
if [ -z "$VERSION" ]; then
    echo "查找最新版本..."
    VERSION=$(ls -t "$RELEASES_DIR" | grep "^${USER_NAME}_${APP_NAME}_" | head -1)
    if [ -z "$VERSION" ]; then
        echo "错误: 没有找到任何版本文件"
        exit 1
    fi
    echo "找到最新版本: $VERSION"
fi

# 检查版本文件是否存在
VERSION_FILE="$RELEASES_DIR/$VERSION"
if [ ! -f "$VERSION_FILE" ]; then
    echo "错误: 版本文件不存在: $VERSION_FILE"
    exit 1
fi

# 删除旧的软链接（如果存在）
if [ -L "$APP_PATH" ] || [ -f "$APP_PATH" ]; then
    echo "删除旧的软链接: $APP_PATH"
    rm -f "$APP_PATH"
fi

# 创建新的软链接
echo "创建软链接: $APP_PATH -> $VERSION"
ln -s "$VERSION" "$APP_PATH"

# 验证软链接
if [ -L "$APP_PATH" ]; then
    TARGET=$(readlink "$APP_PATH")
    echo "软链接创建成功: $APP_PATH -> $TARGET"
    
    # 检查目标文件是否存在
    if [ -f "$RELEASES_DIR/$TARGET" ]; then
        echo "目标文件存在，软链接有效"
    else
        echo "警告: 目标文件不存在: $RELEASES_DIR/$TARGET"
    fi
else
    echo "错误: 软链接创建失败"
    exit 1
fi

echo "完成！现在可以在 Dockerfile 中使用 CMD [\"./app\"] 来启动应用"





