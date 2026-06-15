#!/bin/sh

# 继承平台部署时区；未配置时使用北京时间。
export TZ="${TZ:-Asia/Shanghai}"

# 加载 Python 已安装包列表环境变量（用于快速检查包是否已安装）
if [ -f /etc/profile.d/python-packages.sh ]; then
    . /etc/profile.d/python-packages.sh
fi
# 如果 profile 脚本不存在，尝试从文件读取
if [ -z "$PYTHON_INSTALLED_PACKAGES" ] && [ -f /etc/python-installed-packages.txt ]; then
    export PYTHON_INSTALLED_PACKAGES=$(cat /etc/python-installed-packages.txt)
fi

BIN_DIR="${KAGEOS_APP_BIN_DIR:-/app/workplace/bin/releases}"
WORK_DIR="${KAGEOS_APP_WORK_DIR:-$(dirname "$BIN_DIR")}"

# 保持旧镜像的默认工作目录语义，避免应用里相对路径行为变化。
if [ -d "$WORK_DIR" ]; then
    cd "$WORK_DIR" || exit 1
fi

# 检查 releases 目录
if [ ! -d "$BIN_DIR" ]; then
    echo "错误: releases 目录不存在: $BIN_DIR"
    exit 1
fi

# 检查是否有文件
if [ ! "$(ls -A "$BIN_DIR")" ]; then
    echo "错误: releases 目录为空: $BIN_DIR"
    exit 1
fi

# 优先使用 runtime 在创建容器时注入的精确二进制名。
# metadata 文件仅作为旧容器兼容兜底，不再是启动链路的必需输入。
if [ -n "$APP_BINARY_NAME" ]; then
    BINARY_NAME="$APP_BINARY_NAME"
    CURRENT_VERSION="${APP_VERSION:-unknown}"
    echo "使用环境变量 APP_BINARY_NAME: $BINARY_NAME"
else
    # 优先使用环境变量 APP_VERSION（新架构：每个容器对应特定版本）
    # 如果没有环境变量，再读取文件（向后兼容）
    if [ -n "$APP_VERSION" ]; then
        CURRENT_VERSION="$APP_VERSION"
        echo "使用环境变量 APP_VERSION: $CURRENT_VERSION"
    else
        if [ ! -f "/app/workplace/metadata/current_version.txt" ]; then
            echo "错误: APP_BINARY_NAME/APP_VERSION 环境变量未设置，且 current_version.txt 文件不存在"
            exit 1
        fi

        CURRENT_VERSION=$(cat /app/workplace/metadata/current_version.txt | tr -d '\n\r')

        if [ -z "$CURRENT_VERSION" ]; then
            echo "错误: APP_VERSION 环境变量未设置，且 current_version.txt 文件为空"
            exit 1
        fi

        echo "从文件读取版本: $CURRENT_VERSION"
    fi

    if [ -n "$KAGEOS_APP_USER" ] && [ -n "$KAGEOS_APP_NAME" ]; then
        CURRENT_APP="${KAGEOS_APP_USER}_${KAGEOS_APP_NAME}"
        echo "使用环境变量应用名: $CURRENT_APP"
    else
        if [ ! -f "/app/workplace/metadata/current_app.txt" ]; then
            echo "错误: KAGEOS_APP_USER/KAGEOS_APP_NAME 环境变量未设置，且 current_app.txt 文件不存在"
            exit 1
        fi

        CURRENT_APP=$(cat /app/workplace/metadata/current_app.txt | tr -d '\n\r')

        if [ -z "$CURRENT_APP" ]; then
            echo "错误: current_app.txt 文件为空"
            exit 1
        fi

        echo "从文件读取应用名: $CURRENT_APP"
    fi

    # 旧默认二进制文件名：{user}_{app}_{current_version}
    BINARY_NAME="${CURRENT_APP}_${CURRENT_VERSION}"
fi

echo "当前版本: $CURRENT_VERSION"
echo "二进制文件名: $BINARY_NAME"

# 检查文件是否存在
if [ ! -f "$BIN_DIR/$BINARY_NAME" ]; then
    echo "错误: 二进制文件 $BIN_DIR/$BINARY_NAME 不存在"
    echo "可用的文件:"
    ls -la "$BIN_DIR"
    exit 1
fi

# 启动应用（直接使用 releases 目录下的版本化可执行文件）
echo "启动应用: $BIN_DIR/$BINARY_NAME"

# 让应用进程成为容器主进程。应用 panic/退出时，容器状态必须同步变为失败/退出，
# runtime 才能及时感知启动失败。
exec "$BIN_DIR/$BINARY_NAME"
