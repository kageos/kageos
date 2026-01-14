#!/bin/bash

# Git 同步到 GitHub 脚本（开源版本）
# 功能：将代码同步到 GitHub，自动过滤企业代码
# 使用方式：./scripts/git-sync-to-github.sh
# 
# 策略说明：
# - 开发阶段：全量代码提交到 Gitee
# - 开源同步：过滤企业代码后推送到 GitHub
# 
# 过滤规则：
# - enterprise_impl/ - 企业功能实现
# - licenses/ - License 文件
# - license*.json - License 文件

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
    if ! git remote get-url github > /dev/null 2>&1; then
        # 尝试从 origin 获取 GitHub URL
        ORIGIN_URL=$(git remote get-url origin 2>/dev/null || echo "")
        if [[ "$ORIGIN_URL" == *"github.com"* ]]; then
            print_info "使用 origin 作为 GitHub 远程仓库"
            git remote add github "$ORIGIN_URL" 2>/dev/null || {
                print_warning "GitHub 远程仓库已存在或添加失败，使用 origin"
            }
        else
            print_error "未找到 GitHub 远程仓库，请手动添加："
            print_error "  git remote add github git@github.com:ai-agent-os/ai-agent-os.git"
            exit 1
        fi
    fi
    
    GITHUB_URL=$(git remote get-url github 2>/dev/null || git remote get-url origin 2>/dev/null || echo "")
    print_info "GitHub: $GITHUB_URL"
}

# 创建临时分支用于同步到 GitHub（过滤企业代码）
sync_to_github() {
    print_info "开始同步到 GitHub（过滤企业代码）..."
    
    CURRENT_BRANCH=$(git branch --show-current 2>/dev/null || echo "main")
    SYNC_BRANCH="github-sync-${CURRENT_BRANCH}"
    
    print_info "当前分支: $CURRENT_BRANCH"
    print_info "同步分支: $SYNC_BRANCH"
    
    # 检查是否有未提交的更改
    if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
        print_warning "检测到未提交的更改，请先提交或暂存"
        print_warning "建议先执行: bash scripts/git-push-all-auto.sh \"提交信息\""
        read -p "是否继续同步？(y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "已取消同步"
            exit 0
        fi
    fi
    
    # 保存当前分支
    ORIGINAL_BRANCH=$CURRENT_BRANCH
    
    # 创建或切换到同步分支
    if git show-ref --verify --quiet refs/heads/$SYNC_BRANCH; then
        print_info "切换到已存在的同步分支: $SYNC_BRANCH"
        git checkout $SYNC_BRANCH
        git reset --hard $ORIGINAL_BRANCH
    else
        print_info "创建新的同步分支: $SYNC_BRANCH"
        git checkout -b $SYNC_BRANCH $ORIGINAL_BRANCH
    fi
    
    # 过滤企业代码
    print_info "过滤企业代码..."
    
    # 删除企业代码目录和文件
    if [ -d "enterprise_impl" ]; then
        print_info "删除 enterprise_impl/ 目录"
        git rm -rf enterprise_impl/ 2>/dev/null || rm -rf enterprise_impl/
    fi
    
    if [ -d "licenses" ]; then
        print_info "删除 licenses/ 目录"
        git rm -rf licenses/ 2>/dev/null || rm -rf licenses/
    fi
    
    # 删除 License 文件
    for license_file in license*.json; do
        if [ -f "$license_file" ]; then
            print_info "删除 $license_file"
            git rm -f "$license_file" 2>/dev/null || rm -f "$license_file"
        fi
    done
    
    # 删除 .gitmodules（如果存在且只包含 enterprise_impl）
    if [ -f ".gitmodules" ]; then
        # 检查是否只包含 enterprise_impl
        if grep -q "enterprise_impl" .gitmodules && ! grep -v "enterprise_impl" .gitmodules | grep -q "\[submodule"; then
            print_info "删除 .gitmodules（只包含 enterprise_impl）"
            git rm -f .gitmodules 2>/dev/null || rm -f .gitmodules
        fi
    fi
    
    # 提交过滤后的更改
    if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
        print_info "提交过滤后的更改..."
        git add -A
        git commit -m "chore: remove enterprise code for open source" || {
            print_warning "没有需要提交的更改（可能已经过滤过）"
        }
    else
        print_info "没有需要过滤的内容（可能已经过滤过）"
    fi
    
    # 推送到 GitHub
    print_info "推送到 GitHub..."
    if git push github $SYNC_BRANCH:$CURRENT_BRANCH --force 2>&1; then
        print_success "GitHub 同步成功"
    else
        print_error "GitHub 推送失败"
        git checkout $ORIGINAL_BRANCH
        exit 1
    fi
    
    # 切换回原分支
    git checkout $ORIGINAL_BRANCH
    
    # 询问是否删除临时分支
    read -p "是否删除临时同步分支 $SYNC_BRANCH？(y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        git branch -D $SYNC_BRANCH 2>/dev/null || true
        print_info "已删除临时分支"
    else
        print_info "保留临时分支: $SYNC_BRANCH"
    fi
    
    print_success "同步完成！"
}

# 主函数
main() {
    print_info "========================================="
    print_info "Git 同步到 GitHub 脚本（开源版本）"
    print_info "功能：过滤企业代码后推送到 GitHub"
    print_info "========================================="
    echo
    
    # 检查 Git 仓库
    check_git_repo
    
    # 检查远程仓库
    check_remotes
    
    # 同步到 GitHub
    sync_to_github
    
    print_success "========================================="
    print_success "完成！代码已同步到 GitHub（已过滤企业代码）"
    print_success "========================================="
}

# 运行主函数
main
