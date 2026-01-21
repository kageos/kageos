#!/bin/bash

# Git 一键提交脚本（自动模式 - 适合大模型/AI助手使用）
# 功能：自动提交所有代码并推送到 Gitee
# 使用方式：./scripts/git-push-all-auto.sh "提交信息"
# 
# 策略说明：
# - 开发阶段：全量代码（包括企业代码）都提交到 Gitee
# - 开源同步：使用 scripts/git-sync-to-github.sh 单独同步到 GitHub（过滤企业代码）
# - enterprise_impl 已作为普通目录包含在主仓库中，不再使用 Submodule
# 
# 特点：
# - 完全自动化，无需交互
# - 适合大模型/AI助手调用
# - 自动处理所有 Git 操作

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
    
    # 检查 Gitee（主仓库推送到 Gitee）
    if ! git remote get-url gitee > /dev/null 2>&1; then
        print_error "未找到 gitee 远程仓库"
        exit 1
    fi
    
    GITEE_URL=$(git remote get-url gitee)
    print_info "Gitee: $GITEE_URL"
    print_info "策略：开发阶段全量代码提交到 Gitee，开源时使用 git-sync-to-github.sh 同步到 GitHub"
}

# 提交 Submodule 更改（自动模式）
# 注意：现在 enterprise_impl 已作为普通目录，不再需要 Submodule 处理
commit_submodules_auto() {
    # 检查是否还有 Submodule（向后兼容）
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
            
            # 确保 Submodule 使用项目级别的 Git 配置（不修改全局配置）
            if [ -f "../.git/config" ]; then
                PARENT_USER_NAME=$(git -C .. config user.name 2>/dev/null || echo "")
                PARENT_USER_EMAIL=$(git -C .. config user.email 2>/dev/null || echo "")
                if [ -n "$PARENT_USER_NAME" ] && [ -n "$PARENT_USER_EMAIL" ]; then
                    git config --local user.name "$PARENT_USER_NAME" 2>/dev/null || true
                    git config --local user.email "$PARENT_USER_EMAIL" 2>/dev/null || true
                fi
            fi
            
            # 检查是否有未提交的更改
            if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
                print_warning "$SUBMODULE 有未提交的更改，自动提交..."
                git add . 2>/dev/null || true
                git commit -m "chore: auto update $SUBMODULE $(date +%Y-%m-%d\ %H:%M:%S)" 2>/dev/null || {
                    print_warning "$SUBMODULE 提交失败（可能没有更改）"
                    cd - > /dev/null
                    continue
                }
                
                # 推送到 Submodule 的远程仓库（Gitee）
                print_info "$SUBMODULE 推送到 Gitee..."
                if git push origin 2>&1; then
                    print_success "$SUBMODULE 已提交并推送到 Gitee"
                else
                    PUSH_ERROR=$?
                    # 检查是否是因为已经是最新版本
                    if git push origin 2>&1 | grep -q "Everything up-to-date"; then
                        print_info "$SUBMODULE 已是最新版本，无需推送"
                    else
                        print_warning "$SUBMODULE 推送到 Gitee 失败（错误码: $PUSH_ERROR），但继续执行"
                    fi
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
# 注意：主仓库和 Submodule 都推送到 Gitee
push_to_remotes_auto() {
    print_info "推送到远程仓库..."
    
    CURRENT_BRANCH=$(git branch --show-current 2>/dev/null || echo "main")
    print_info "当前分支: $CURRENT_BRANCH"
    
    # 检查是否有需要推送的提交
    LOCAL=$(git rev-parse @ 2>/dev/null || echo "")
    REMOTE_GITEE=$(git rev-parse gitee/$CURRENT_BRANCH 2>/dev/null || echo "")
    
    if [ "$LOCAL" = "$REMOTE_GITEE" ] && [ -n "$LOCAL" ]; then
        print_warning "本地和远程已同步，无需推送"
    fi
    
    # 主仓库推送到 Gitee
    print_info "推送到 Gitee - 主代码（全量，包括企业代码）..."
    if git push gitee "$CURRENT_BRANCH" 2>&1; then
        print_success "Gitee 推送成功"
    else
        PUSH_ERROR=$?
        # 检查是否是因为已经是最新版本
        if git push gitee "$CURRENT_BRANCH" 2>&1 | grep -q "Everything up-to-date"; then
            print_info "Gitee 已是最新版本，无需推送"
        else
            print_error "Gitee 推送失败（错误码: $PUSH_ERROR）"
            exit 1
        fi
    fi
    
    print_success "推送操作完成"
    print_info "提示：如需同步到 GitHub（开源），请使用 scripts/git-sync-to-github.sh"
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
