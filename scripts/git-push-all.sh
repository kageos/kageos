#!/bin/bash

# Git 一键提交脚本
# 功能：提交所有代码（包括 Submodule）并推送到 GitHub 和 Gitee

set -e  # 遇到错误立即退出

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查是否在 Git 仓库中
check_git_repo() {
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        print_error "当前目录不是 Git 仓库"
        exit 1
    fi
}

# 检查远程仓库配置
check_remotes() {
    print_info "检查远程仓库配置..."
    
    # 检查 GitHub
    if ! git remote get-url origin > /dev/null 2>&1; then
        print_error "未找到 origin 远程仓库"
        exit 1
    fi
    
    ORIGIN_URL=$(git remote get-url origin)
    print_info "Origin: $ORIGIN_URL"
    
    # 检查 Gitee
    if git remote get-url gitee > /dev/null 2>&1; then
        GITEE_URL=$(git remote get-url gitee)
        print_info "Gitee: $GITEE_URL"
    else
        print_warning "未找到 gitee 远程仓库，将自动添加"
        
        # 尝试从 origin 推断 Gitee URL
        if [[ "$ORIGIN_URL" == *"github.com"* ]]; then
            # 从 GitHub URL 推断 Gitee URL
            REPO_NAME=$(echo "$ORIGIN_URL" | sed -E 's|.*github.com[:/]([^/]+/[^/]+)(\.git)?$|\1|' | sed 's/\.git$//')
            GITEE_URL="https://gitee.com/$REPO_NAME.git"
            print_info "推断 Gitee URL: $GITEE_URL"
            
            read -p "是否添加此 Gitee 仓库？(y/n): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                git remote add gitee "$GITEE_URL"
                print_success "已添加 Gitee 远程仓库: $GITEE_URL"
            else
                read -p "请输入 Gitee 仓库 URL: " GITEE_URL
                git remote add gitee "$GITEE_URL"
                print_success "已添加 Gitee 远程仓库: $GITEE_URL"
            fi
        else
            read -p "请输入 Gitee 仓库 URL: " GITEE_URL
            git remote add gitee "$GITEE_URL"
            print_success "已添加 Gitee 远程仓库: $GITEE_URL"
        fi
    fi
}

# 检查 Submodule 状态
check_submodules() {
    if [ -f .gitmodules ]; then
        print_info "检测到 Submodule，检查状态..."
        
        # 检查是否有未提交的 Submodule 更改
        SUBMODULE_STATUS=$(git submodule status)
        if echo "$SUBMODULE_STATUS" | grep -q "^+"; then
            print_warning "检测到 Submodule 有未提交的更改"
            echo "$SUBMODULE_STATUS"
            
            read -p "是否先提交 Submodule 的更改？(y/n): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                commit_submodules
            fi
        else
            print_success "Submodule 状态正常"
        fi
    fi
}

# 提交 Submodule 更改
commit_submodules() {
    print_info "提交 Submodule 更改..."
    
    # 获取所有 Submodule 路径
    SUBMODULES=$(git config --file .gitmodules --get-regexp path | awk '{print $2}')
    
    for SUBMODULE in $SUBMODULES; do
        if [ -d "$SUBMODULE" ]; then
            print_info "处理 Submodule: $SUBMODULE"
            
            cd "$SUBMODULE"
            
            # 检查是否有未提交的更改
            if [ -n "$(git status --porcelain)" ]; then
                print_warning "$SUBMODULE 有未提交的更改"
                git status --short
                
                read -p "是否提交 $SUBMODULE 的更改？(y/n): " -n 1 -r
                echo
                if [[ $REPLY =~ ^[Yy]$ ]]; then
                    read -p "请输入提交信息（留空使用默认）: " COMMIT_MSG
                    if [ -z "$COMMIT_MSG" ]; then
                        COMMIT_MSG="chore: update $SUBMODULE"
                    fi
                    
                    git add .
                    git commit -m "$COMMIT_MSG"
                    git push
                    print_success "$SUBMODULE 已提交并推送"
                fi
            else
                print_info "$SUBMODULE 无更改"
            fi
            
            cd - > /dev/null
        fi
    done
    
    # 更新主仓库的 Submodule 引用
    if [ -n "$(git status --porcelain .gitmodules enterprise_impl 2>/dev/null)" ]; then
        print_info "更新主仓库的 Submodule 引用..."
        git add .gitmodules enterprise_impl
    fi
}

# 提交主仓库更改
commit_main_repo() {
    print_info "检查主仓库状态..."
    
    # 检查是否有未提交的更改
    if [ -z "$(git status --porcelain)" ]; then
        print_warning "没有未提交的更改"
        return 0
    fi
    
    # 显示更改
    print_info "当前更改："
    git status --short
    
    # 询问是否提交
    read -p "是否提交这些更改？(y/n): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_warning "已取消提交"
        return 1
    fi
    
    # 输入提交信息
    read -p "请输入提交信息: " COMMIT_MSG
    if [ -z "$COMMIT_MSG" ]; then
        print_error "提交信息不能为空"
        return 1
    fi
    
    # 提交
    print_info "正在提交..."
    git add .
    git commit -m "$COMMIT_MSG"
    print_success "提交成功: $COMMIT_MSG"
}

# 推送到远程仓库
push_to_remotes() {
    print_info "推送到远程仓库..."
    
    CURRENT_BRANCH=$(git branch --show-current)
    print_info "当前分支: $CURRENT_BRANCH"
    
    # 推送到 GitHub (origin)
    print_info "推送到 GitHub (origin)..."
    if git push origin "$CURRENT_BRANCH"; then
        print_success "GitHub 推送成功"
    else
        print_error "GitHub 推送失败"
        read -p "是否继续推送到 Gitee？(y/n): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
    
    # 推送到 Gitee
    print_info "推送到 Gitee..."
    if git push gitee "$CURRENT_BRANCH"; then
        print_success "Gitee 推送成功"
    else
        print_error "Gitee 推送失败"
        print_warning "GitHub 已推送，但 Gitee 推送失败"
        exit 1
    fi
    
    print_success "所有代码已成功推送到 GitHub 和 Gitee"
}

# 主函数
main() {
    print_info "========================================="
    print_info "Git 一键提交脚本"
    print_info "========================================="
    echo
    
    # 检查 Git 仓库
    check_git_repo
    
    # 检查远程仓库
    check_remotes
    
    # 检查 Submodule
    check_submodules
    
    # 提交主仓库
    commit_main_repo
    
    # 推送到远程仓库
    push_to_remotes
    
    print_success "========================================="
    print_success "完成！所有代码已提交并推送"
    print_success "========================================="
}

# 运行主函数
main
