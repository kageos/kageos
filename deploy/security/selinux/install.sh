#!/bin/bash
# SELinux 策略模块安装脚本
# 在 SELinux 宿主机（如 RHEL/Fedora/CentOS）上执行一次即可
#
# 用法：sudo bash install.sh [namespace_base_path]
#   namespace_base_path: 应用目录基础路径（默认为当前目录下的 namespace）
#   例如：sudo bash install.sh /data/kageos/namespace
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MODULE_NAME="kageos_app"
TE_FILE="$SCRIPT_DIR/kageos-app.te"
NAMESPACE_BASE="${1:-namespace}"

if [ "$(id -u)" -ne 0 ]; then
    echo "请使用 sudo 执行此脚本"
    exit 1
fi

# 检查依赖工具
for cmd in checkmodule semodule_package semodule semanage restorecon; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "错误：未找到 $cmd"
        echo "请安装：sudo dnf install policycoreutils-devel selinux-policy-devel policycoreutils-python-utils"
        exit 1
    fi
done

echo "=== 1/3 编译并安装 SELinux 策略模块 ==="
WORK_DIR=$(mktemp -d)
checkmodule -M -m -o "$WORK_DIR/$MODULE_NAME.mod" "$TE_FILE"
semodule_package -o "$WORK_DIR/$MODULE_NAME.pp" -m "$WORK_DIR/$MODULE_NAME.mod"
semodule -i "$WORK_DIR/$MODULE_NAME.pp"
rm -rf "$WORK_DIR"
echo "策略模块 '$MODULE_NAME' 已安装"

echo "=== 2/3 设置文件上下文规则 ==="
# 对 namespace 下所有应用的 code/ 和 workplace/ 目录及其内容打标签
ABS_BASE="$(cd "$NAMESPACE_BASE" 2>/dev/null && pwd || echo "$NAMESPACE_BASE")"
semanage fcontext -a -t kageos_data_t "$ABS_BASE/.*/code(/.*)?" 2>/dev/null || \
    semanage fcontext -m -t kageos_data_t "$ABS_BASE/.*/code(/.*)?"
semanage fcontext -a -t kageos_data_t "$ABS_BASE/.*/workplace(/.*)?" 2>/dev/null || \
    semanage fcontext -m -t kageos_data_t "$ABS_BASE/.*/workplace(/.*)?"
echo "文件上下文规则已设置：$ABS_BASE/.*/code 和 .../workplace"

echo "=== 3/3 应用标签到现有目录 ==="
if [ -d "$ABS_BASE" ]; then
    restorecon -Rv "$ABS_BASE"
    echo "标签已应用到 $ABS_BASE"
else
    echo "目录 $ABS_BASE 不存在，将在首次创建应用时自动继承标签"
fi

echo ""
echo "✅ SELinux 策略模块安装完成"
echo "   验证：semodule -l | grep $MODULE_NAME"
echo "   查看标签：ls -lZ $ABS_BASE/*/code/ （应显示 kageos_data_t）"
echo "   注意：新创建的应用目录会自动继承标签规则，无需重复执行"
