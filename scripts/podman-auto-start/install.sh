#!/bin/bash

# Podman 自动启动安装脚本
# 用途：在 macOS 上安装 Podman 自动启动服务
# 作者：AI Assistant
# 日期：$(date +%Y-%m-%d)

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_NAME="podman-auto-start.sh"
PLIST_NAME="com.podman.auto-start.plist"

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

# 检查是否为 macOS
check_macos() {
    if [[ "$OSTYPE" != "darwin"* ]]; then
        log_error "此脚本仅支持 macOS 系统"
        exit 1
    fi
}

# 检查 Podman 是否安装
check_podman() {
    if ! command -v podman &> /dev/null; then
        log_error "Podman 未安装，请先安装 Podman"
        log_info "安装命令：brew install podman"
        exit 1
    fi
    log_info "Podman 已安装：$(podman --version)"
}

# 创建必要的目录
create_directories() {
    log_step "创建必要的目录..."
    
    # 创建 bin 目录
    mkdir -p ~/bin
    log_info "创建目录：~/bin"
    
    # 创建 LaunchAgents 目录
    mkdir -p ~/Library/LaunchAgents
    log_info "创建目录：~/Library/LaunchAgents"
}

# 复制启动脚本
install_script() {
    log_step "安装启动脚本..."
    
    # 复制脚本到 ~/bin
    cp "$SCRIPT_DIR/$SCRIPT_NAME" ~/bin/
    chmod +x ~/bin/$SCRIPT_NAME
    log_info "启动脚本已安装到：~/bin/$SCRIPT_NAME"
}

# 创建 plist 文件
create_plist() {
    log_step "创建 macOS 服务文件..."
    
    # 获取当前用户名
    USERNAME=$(whoami)
    
    cat > ~/Library/LaunchAgents/$PLIST_NAME << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.podman.auto-start</string>
    <key>ProgramArguments</key>
    <array>
        <string>/bin/bash</string>
        <string>-c</string>
        <string>sleep 10 && /Users/$USERNAME/bin/$SCRIPT_NAME</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
    <key>StandardOutPath</key>
    <string>/tmp/podman-auto-start.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/podman-auto-start.error.log</string>
</dict>
</plist>
EOF
    
    log_info "服务文件已创建：~/Library/LaunchAgents/$PLIST_NAME"
}

# 加载服务
load_service() {
    log_step "加载 macOS 服务..."
    
    # 先卸载可能存在的服务
    launchctl unload ~/Library/LaunchAgents/$PLIST_NAME 2>/dev/null || true
    
    # 加载服务
    launchctl load ~/Library/LaunchAgents/$PLIST_NAME
    log_info "服务已加载"
}

# 配置 shell
configure_shell() {
    log_step "配置 shell 自动检查..."
    
    # 检测 shell 类型
    if [[ "$SHELL" == *"zsh"* ]]; then
        SHELL_RC="$HOME/.zshrc"
    elif [[ "$SHELL" == *"bash"* ]]; then
        SHELL_RC="$HOME/.bash_profile"
    else
        log_warn "未识别的 shell 类型，跳过 shell 配置"
        return
    fi
    
    # 检查是否已经配置过
    if grep -q "podman-auto-start" "$SHELL_RC" 2>/dev/null; then
        log_info "Shell 配置已存在，跳过"
        return
    fi
    
    # 添加配置
    cat >> "$SHELL_RC" << 'EOF'

# Podman 自动启动检查
if command -v podman &> /dev/null; then
    # 检查 Podman 机器是否运行，如果没有运行则启动
    if ! podman machine list 2>/dev/null | grep -q "running"; then
        echo "正在启动 Podman 机器..."
        podman machine start 2>/dev/null
    fi
fi

# Podman 便捷命令
alias pps='podman ps'
alias pstart='podman machine start'
alias pstop='podman machine stop'
alias pstatus='podman machine list'
alias prestart='podman machine stop && podman machine start'
EOF
    
    log_info "Shell 配置已添加到：$SHELL_RC"
}

# 测试安装
test_installation() {
    log_step "测试安装..."
    
    # 测试脚本
    if ~/bin/$SCRIPT_NAME; then
        log_info "启动脚本测试成功"
    else
        log_warn "启动脚本测试失败"
    fi
    
    # 检查服务状态
    if launchctl list | grep -q "com.podman.auto-start"; then
        log_info "服务已加载"
    else
        log_warn "服务未加载"
    fi
}

# 显示使用说明
show_usage() {
    echo
    log_info "安装完成！"
    echo
    echo "使用方法："
    echo "  手动启动：~/bin/$SCRIPT_NAME"
    echo "  查看状态：launchctl list | grep podman"
    echo "  查看日志：tail -f /tmp/podman-auto-start.log"
    echo "  卸载服务：launchctl unload ~/Library/LaunchAgents/$PLIST_NAME"
    echo
    echo "便捷命令："
    echo "  pps      - podman ps"
    echo "  pstart   - podman machine start"
    echo "  pstop    - podman machine stop"
    echo "  pstatus  - podman machine list"
    echo "  prestart - 重启 Podman 机器"
    echo
    log_info "重启电脑后，Podman 将自动启动"
}

# 主函数
main() {
    echo "Podman 自动启动安装程序"
    echo "=========================="
    
    check_macos
    check_podman
    create_directories
    install_script
    create_plist
    load_service
    configure_shell
    test_installation
    show_usage
}

# 如果直接运行此脚本
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi


