#!/bin/bash
# AppArmor profile 安装脚本
# 在 AppArmor 宿主机（如 Ubuntu/Debian）上执行一次即可
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROFILE_NAME="ai-agent-os-app"
PROFILE_SRC="$SCRIPT_DIR/$PROFILE_NAME"
PROFILE_DST="/etc/apparmor.d/$PROFILE_NAME"

if [ "$(id -u)" -ne 0 ]; then
    echo "请使用 sudo 执行此脚本"
    exit 1
fi

# 检查 AppArmor 是否可用
if ! command -v apparmor_parser &>/dev/null; then
    echo "错误：未找到 apparmor_parser，请先安装 apparmor：sudo apt-get install apparmor apparmor-utils"
    exit 1
fi

# 拷贝并加载 profile
cp "$PROFILE_SRC" "$PROFILE_DST"
apparmor_parser -r "$PROFILE_DST"

echo "✅ AppArmor profile '$PROFILE_NAME' 已安装并加载"
echo "   验证：sudo aa-status | grep $PROFILE_NAME"
echo "   默认：app-runtime 会自动使用 '$PROFILE_NAME'；只有自定义 profile 名时才需要改 container.apparmor_profile"
