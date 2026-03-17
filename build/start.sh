#!/bin/sh

# 设置时区
export TZ=Asia/Shanghai

# 加载 Python 已安装包列表环境变量（用于快速检查包是否已安装）
if [ -f /etc/profile.d/python-packages.sh ]; then
    . /etc/profile.d/python-packages.sh
fi
# 如果 profile 脚本不存在，尝试从文件读取
if [ -z "$PYTHON_INSTALLED_PACKAGES" ] && [ -f /etc/python-installed-packages.txt ]; then
    export PYTHON_INSTALLED_PACKAGES=$(cat /etc/python-installed-packages.txt)
fi

# 进入应用目录
cd /app/workplace/bin

# 检查 releases 目录
if [ ! -d "releases" ]; then
    echo "错误: releases 目录不存在"
    exit 1
fi

# 检查是否有文件
if [ ! "$(ls -A releases)" ]; then
    echo "错误: releases 目录为空"
    exit 1
fi

# 检查 metadata 目录
if [ ! -d "/app/workplace/metadata" ]; then
    echo "错误: metadata 目录不存在"
    exit 1
fi

# 优先使用环境变量 APP_VERSION（新架构：每个容器对应特定版本）
# 如果没有环境变量，再读取文件（向后兼容）
if [ -n "$APP_VERSION" ]; then
    CURRENT_VERSION="$APP_VERSION"
    echo "使用环境变量 APP_VERSION: $CURRENT_VERSION"
else
    # 向后兼容：如果没有环境变量，读取文件
    if [ ! -f "/app/workplace/metadata/current_version.txt" ]; then
        echo "错误: APP_VERSION 环境变量未设置，且 current_version.txt 文件不存在"
        exit 1
    fi
    
    # 读取版本号（纯文本，无需解析）
    CURRENT_VERSION=$(cat /app/workplace/metadata/current_version.txt | tr -d '\n\r')
    
    if [ -z "$CURRENT_VERSION" ]; then
        echo "错误: APP_VERSION 环境变量未设置，且 current_version.txt 文件为空"
        exit 1
    fi
    
    echo "从文件读取版本: $CURRENT_VERSION"
fi

# 读取用户和应用名（纯文本文件）
if [ ! -f "/app/workplace/metadata/current_app.txt" ]; then
    echo "错误: current_app.txt 文件不存在"
    exit 1
fi

CURRENT_APP=$(cat /app/workplace/metadata/current_app.txt | tr -d '\n\r')

if [ -z "$CURRENT_APP" ]; then
    echo "错误: current_app.txt 文件为空"
    exit 1
fi

# 拼接二进制文件名：{user}_{app}_{current_version}
BINARY_NAME="${CURRENT_APP}_${CURRENT_VERSION}"
echo "应用: $CURRENT_APP"
echo "当前版本: $CURRENT_VERSION"
echo "二进制文件名: $BINARY_NAME"

# 检查文件是否存在
if [ ! -f "releases/$BINARY_NAME" ]; then
    echo "错误: 二进制文件 releases/$BINARY_NAME 不存在"
    echo "可用的文件:"
    ls -la releases/
    exit 1
fi

# 启动应用（直接使用 releases 目录下的版本化可执行文件）
echo "启动应用: releases/$BINARY_NAME"

# 启动当前版本（后台运行，不使用 exec）
# tini 作为 PID 1，start.sh 保持运行以便 tini 管理子进程
./"releases/$BINARY_NAME" &
APP_PID=$!

echo "应用已启动，PID: $APP_PID"
echo "容器将保持运行，支持灰度发布（多版本共存）"

# 保持脚本运行，让 tini 管理所有子进程
# 这样即使应用版本切换，容器也不会退出
while true; do
    sleep 3600
done