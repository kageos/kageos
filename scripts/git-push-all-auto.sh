#!/bin/bash

# Git 一键提交脚本（自动模式 - 适合大模型/AI助手使用）
# 功能：自动提交所有代码（包括 Submodule）并推送到 GitHub 和 Gitee
# 使用方式：./scripts/git-push-all-auto.sh "提交信息"
# 
# 特点：
# - 完全自动化，无需交互
# - 适合大模型/AI助手调用
# - 自动处理所有 Git 操作
# - 支持 Submodule 自动提交

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

# 获取提交信息
COMMIT_MSG="${1:-chore: auto commit $(date +%Y-%m-%d\ %H:%M:%S)}"

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
    if ! git remote get-url gitee > /dev/null 2>&1; then
        print_warning "未找到 gitee 远程仓库，尝试自动添加..."
        
        # 尝试从 origin 推断 Gitee URL
        if [[ "$ORIGIN_URL" == *"github.com"* ]]; then
            REPO_NAME=$(echo "$ORIGIN_URL" | sed -E 's|.*github.com[:/]([^/]+/[^/]+)(\.git)?$|\1|' | sed 's/\.git$//')
            GITEE_URL="https://gitee.com/$REPO_NAME.git"
            print_info "自动添加 Gitee 远程仓库: $GITEE_URL"
            git remote add gitee "$GITEE_URL" 2>/dev/null || {
                # 如果添加失败，尝试使用已知的 Gitee 仓库
                if [[ "$REPO_NAME" == *"ai-agent-os"* ]]; then
                    GITEE_URL="https://gitee.com/lliubaorui/ai-agent-os.git"
                    git remote add gitee "$GITEE_URL" 2>/dev/null || print_warning "Gitee 远程仓库已存在或添加失败"
                fi
            }
        else
            print_error "无法自动推断 Gitee URL"
            exit 1
        fi
    fi
    
    GITEE_URL=$(git remote get-url gitee 2>/dev/null || echo "未配置")
    print_info "Gitee: $GITEE_URL"
}

# 提交 Submodule 更改（自动模式）
commit_submodules_auto() {
    if [ ! -f .gitmodules ]; then
        return 0
    fi
    
    print_info "检查 Submodule 状态..."
    
    # 获取所有 Submodule 路径
    SUBMODULES=$(git config --file .gitmodules --get-regexp path 2>/dev/null | awk '{print $2}' || echo "")
    
    if [ -z "$SUBMODULES" ]; then
        print_info "未找到 Submodule"
        return 0
    fi
    
    for SUBMODULE in $SUBMODULES; do
        if [ -d "$SUBMODULE" ] && [ -d "$SUBMODULE/.git" ]; then
            print_info "检查 Submodule: $SUBMODULE"
            
            cd "$SUBMODULE"
            
            # 检查是否有未提交的更改
            if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
                print_warning "$SUBMODULE 有未提交的更改，自动提交..."
                git add . 2>/dev/null || true
                git commit -m "chore: auto update $SUBMODULE $(date +%Y-%m-%d\ %H:%M:%S)" 2>/dev/null || {
                    print_warning "$SUBMODULE 提交失败（可能没有更改）"
                    cd - > /dev/null
                    continue
                }
                
                # 尝试推送到 Submodule 的远程仓库
                if git push 2>/dev/null; then
                    print_success "$SUBMODULE 已提交并推送"
                else
                    print_warning "$SUBMODULE 推送失败（可能没有远程仓库或权限），但继续执行"
                fi
            else
                print_info "$SUBMODULE 无更改"
            fi
            
            cd - > /dev/null
        fi
    done
    
    # 更新主仓库的 Submodule 引用
    if [ -f .gitmodules ]; then
        SUBMODULE_PATHS=$(git config --file .gitmodules --get-regexp path 2>/dev/null | awk '{print $2}' || echo "")
        for SUBMODULE in $SUBMODULE_PATHS; do
            if [ -d "$SUBMODULE" ]; then
                git add "$SUBMODULE" 2>/dev/null || true
            fi
        done
        git add .gitmodules 2>/dev/null || true
    fi
}

# 提交主仓库更改（自动模式）
commit_main_repo_auto() {
    print_info "检查主仓库状态..."
    
    # 检查是否有未提交的更改
    if [ -z "$(git status --porcelain 2>/dev/null)" ]; then
        print_warning "没有未提交的更改"
        return 0
    fi
    
    # 显示更改（限制输出行数，避免输出过多）
    print_info "当前更改："
    git status --short | head -20
    
    # 自动提交
    print_info "自动提交: $COMMIT_MSG"
    git add . 2>/dev/null || true
    
    # 再次检查是否有需要提交的内容
    if [ -z "$(git diff --cached --name-only 2>/dev/null)" ] && [ -z "$(git status --porcelain 2>/dev/null)" ]; then
        print_warning "没有需要提交的内容（可能已全部提交）"
        return 0
    fi
    
    git commit -m "$COMMIT_MSG" || {
        print_error "提交失败"
        exit 1
    }
    print_success "提交成功: $COMMIT_MSG"
}

# 推送到远程仓库（自动模式）
push_to_remotes_auto() {
    print_info "推送到远程仓库..."
    
    CURRENT_BRANCH=$(git branch --show-current 2>/dev/null || echo "main")
    print_info "当前分支: $CURRENT_BRANCH"
    
    # 检查是否有需要推送的提交
    LOCAL=$(git rev-parse @ 2>/dev/null || echo "")
    REMOTE_ORIGIN=$(git rev-parse @{u} 2>/dev/null || echo "")
    
    if [ "$LOCAL" = "$REMOTE_ORIGIN" ] && [ -n "$LOCAL" ]; then
        print_warning "本地和远程已同步，无需推送"
    fi
    
    # 推送到 GitHub (origin)
    print_info "推送到 GitHub (origin)..."
    if git push origin "$CURRENT_BRANCH" 2>&1; then
        print_success "GitHub 推送成功"
    else
        PUSH_ERROR=$?
        # 检查是否是因为已经是最新版本
        if git push origin "$CURRENT_BRANCH" 2>&1 | grep -q "Everything up-to-date"; then
            print_info "GitHub 已是最新版本，无需推送"
        else
            print_error "GitHub 推送失败（错误码: $PUSH_ERROR）"
            # 不立即退出，尝试推送到 Gitee
        fi
    fi
    
    # 推送到 Gitee
    if git remote get-url gitee > /dev/null 2>&1; then
        print_info "推送到 Gitee..."
        if git push gitee "$CURRENT_BRANCH" 2>&1; then
            print_success "Gitee 推送成功"
        else
            PUSH_ERROR=$?
            # 检查是否是因为已经是最新版本
            if git push gitee "$CURRENT_BRANCH" 2>&1 | grep -q "Everything up-to-date"; then
                print_info "Gitee 已是最新版本，无需推送"
            else
                print_warning "Gitee 推送失败（错误码: $PUSH_ERROR），但继续执行"
            fi
        fi
    else
        print_warning "Gitee 远程仓库未配置，跳过 Gitee 推送"
    fi
    
    print_success "推送操作完成"
}

# 主函数
main() {
    print_info "========================================="
    print_info "Git 一键提交脚本（自动模式）"
    print_info "提交信息: $COMMIT_MSG"
    print_info "========================================="
    echo
    
    # 检查 Git 仓库
    check_git_repo
    
    # 检查远程仓库
    check_remotes
    
    # 提交 Submodule（自动）
    commit_submodules_auto
    
    # 提交主仓库（自动）
    commit_main_repo_auto
    
    # 推送到远程仓库（自动）
    push_to_remotes_auto
    
    print_success "========================================="
    print_success "完成！所有代码已提交并推送"
    print_success "========================================="
}

# 运行主函数
main
