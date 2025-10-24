#!/bin/bash

# Podman 自动启动卸载脚本
# 用途：卸载 Podman 自动启动服务
# 作者：AI Assistant
# 日期：$(date +%Y-%m-%d)

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

PLIST_NAME="com.podman.auto-start.plist"
SCRIPT_NAME="podman-auto-start.sh"

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# 卸载服务
unload_service() {
    log_step "卸载 macOS 服务..."
    
    if launchctl list | grep -q "com.podman.auto-start"; then
        launchctl unload ~/Library/LaunchAgents/$PLIST_NAME
        log_info "服务已卸载"
    else
        log_info "服务未运行"
    fi
}

# 删除服务文件
remove_plist() {
    log_step "删除服务文件..."
    
    if [[ -f ~/Library/LaunchAgents/$PLIST_NAME ]]; then
        rm ~/Library/LaunchAgents/$PLIST_NAME
        log_info "服务文件已删除"
    else
        log_info "服务文件不存在"
    fi
}

# 删除启动脚本
remove_script() {
    log_step "删除启动脚本..."
    
    if [[ -f ~/bin/$SCRIPT_NAME ]]; then
        rm ~/bin/$SCRIPT_NAME
        log_info "启动脚本已删除"
    else
        log_info "启动脚本不存在"
    fi
}

# 清理 shell 配置
cleanup_shell() {
    log_step "清理 shell 配置..."
    
    # 检测 shell 类型
    if [[ "$SHELL" == *"zsh"* ]]; then
        SHELL_RC="$HOME/.zshrc"
    elif [[ "$SHELL" == *"bash"* ]]; then
        SHELL_RC="$HOME/.bash_profile"
    else
        log_warn "未识别的 shell 类型，跳过 shell 清理"
        return
    fi
    
    if [[ -f "$SHELL_RC" ]]; then
        # 创建备份
        cp "$SHELL_RC" "$SHELL_RC.backup.$(date +%Y%m%d_%H%M%S)"
        log_info "已创建 shell 配置文件备份"
        
        # 删除相关配置
        sed -i '' '/podman-auto-start/,/^$/d' "$SHELL_RC" 2>/dev/null || true
        sed -i '' '/# Podman 便捷命令/,/^$/d' "$SHELL_RC" 2>/dev/null || true
        log_info "Shell 配置已清理"
    else
        log_info "Shell 配置文件不存在"
    fi
}

# 清理日志文件
cleanup_logs() {
    log_step "清理日志文件..."
    
    if [[ -f /tmp/podman-auto-start.log ]]; then
        rm /tmp/podman-auto-start.log
        log_info "日志文件已删除"
    fi
    
    if [[ -f /tmp/podman-auto-start.error.log ]]; then
        rm /tmp/podman-auto-start.error.log
        log_info "错误日志文件已删除"
    fi
}

# 显示卸载结果
show_result() {
    echo
    log_info "卸载完成！"
    echo
    echo "已清理的内容："
    echo "  - macOS 自动启动服务"
    echo "  - 启动脚本文件"
    echo "  - Shell 配置（已备份）"
    echo "  - 日志文件"
    echo
    log_warn "注意：Podman 机器本身不会被删除"
    log_info "如需完全清理，请手动运行：podman machine rm -f"
}

# 主函数
main() {
    echo "Podman 自动启动卸载程序"
    echo "========================"
    
    unload_service
    remove_plist
    remove_script
    cleanup_shell
    cleanup_logs
    show_result
}

# 如果直接运行此脚本
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi


