#!/bin/bash

# Podman 自动启动脚本
# 用途：解决 macOS 重启后 Podman 机器不自动启动的问题
# 作者：AI Assistant
# 日期：$(date +%Y-%m-%d)

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

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

# 检查 Podman 是否安装
check_podman() {
    if ! command -v podman &> /dev/null; then
        log_error "Podman 未安装，请先安装 Podman"
        exit 1
    fi
}

# 安全警告
show_safety_warning() {
    log_warn "⚠️  安全提醒：此脚本仅用于开发环境"
    log_warn "⚠️  生产环境请谨慎使用，避免数据丢失"
    log_warn "⚠️  如有重要数据，请先备份"
}

# 检查 Podman 机器状态
check_machine_status() {
    if podman machine list 2>/dev/null | grep -q "running"; then
        return 0  # 机器正在运行
    else
        return 1  # 机器未运行
    fi
}

# 启动 Podman 机器
start_machine() {
    log_info "Podman 机器未运行，正在启动..."
    
    # 检查是否有机器存在
    if ! podman machine list 2>/dev/null | grep -q "podman-machine"; then
        log_warn "未找到 Podman 机器，正在创建..."
        log_warn "注意：新机器将不包含之前的容器和镜像"
        podman machine init
    fi
    
    # 启动机器
    if podman machine start; then
        log_info "Podman 机器启动成功"
        sleep 3
        return 0
    else
        log_error "Podman 机器启动失败"
        return 1
    fi
}

# 验证连接
verify_connection() {
    if podman ps &> /dev/null; then
        log_info "Podman 连接验证成功"
        return 0
    else
        log_error "Podman 连接验证失败"
        return 1
    fi
}

# 启动停止的容器（只启动基础设施容器）
start_stopped_containers() {
    log_info "检查停止的容器..."
    
    # 定义需要自动启动的基础设施容器
    local infrastructure_containers=("nats-server")
    
    for container_name in "${infrastructure_containers[@]}"; do
        # 检查容器是否存在且已停止
        local container_status=$(podman ps -a --filter "name=$container_name" --format "{{.Status}}" 2>/dev/null)
        
        if [[ "$container_status" == *"Exited"* ]]; then
            log_info "启动基础设施容器: $container_name"
            podman start "$container_name" 2>/dev/null || log_warn "启动容器 $container_name 失败"
        elif [[ -z "$container_status" ]]; then
            log_info "容器 $container_name 不存在，跳过"
        else
            log_info "容器 $container_name 已在运行"
        fi
    done
    
    log_info "基础设施容器检查完成"
}

# 主函数
main() {
    log_info "开始检查 Podman 状态..."
    
    # 显示安全警告
    show_safety_warning
    
    # 检查 Podman 是否安装
    check_podman
    
    # 检查机器状态
    if check_machine_status; then
        log_info "Podman 机器已在运行"
    else
        # 启动机器
        if start_machine; then
            # 验证连接
            if verify_connection; then
                log_info "Podman 机器启动完成"
            else
                log_error "Podman 连接验证失败"
                exit 1
            fi
        else
            log_error "Podman 机器启动失败"
            exit 1
        fi
    fi
    
    # 启动停止的容器
    start_stopped_containers
    
    log_info "Podman 自动启动完成"
}

# 如果直接运行此脚本
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
