# Git 一键提交脚本使用说明

## 📋 概述

提供了两个 Git 提交脚本，用于一键提交代码到 GitHub 和 Gitee：

1. **`git-push-all.sh`** - 交互式脚本（推荐日常使用）
2. **`git-push-all-auto.sh`** - 自动脚本（适合 CI/CD 或快速提交）

---

## 🚀 使用方法

### 方式 1：交互式脚本（推荐）

```bash
# 在项目根目录执行
./scripts/git-push-all.sh
```

**特点**：
- ✅ 交互式提示，每一步都会询问确认
- ✅ 自动检查 Submodule 状态
- ✅ 自动处理 Submodule 提交
- ✅ 自动添加 Gitee 远程仓库（如果不存在）
- ✅ 显示详细的执行过程

**使用场景**：
- 日常开发提交
- 需要确认提交内容的场景
- 首次使用（会自动配置 Gitee 远程仓库）

---

### 方式 2：自动脚本（快速提交）

```bash
# 使用默认提交信息
./scripts/git-push-all-auto.sh

# 或指定提交信息
./scripts/git-push-all-auto.sh "feat: 添加新功能"
```

**特点**：
- ✅ 完全自动，无需交互
- ✅ 自动提交所有更改
- ✅ 自动处理 Submodule
- ✅ 自动推送到 GitHub 和 Gitee

**使用场景**：
- 快速提交小改动
- CI/CD 自动化
- 批量提交

---

## 📝 功能说明

### 1. 自动检查 Git 状态

- 检查是否在 Git 仓库中
- 检查是否有未提交的更改
- 检查 Submodule 状态

### 2. 自动处理 Submodule

- 检测所有 Submodule
- 检查 Submodule 是否有未提交的更改
- 自动提交并推送 Submodule 更改
- 更新主仓库的 Submodule 引用

### 3. 远程仓库管理

- 自动检测 GitHub (origin) 和 Gitee 远程仓库
- 如果 Gitee 不存在，自动添加（交互式脚本）
- 支持从 GitHub URL 自动推断 Gitee URL

### 4. 提交和推送

- 提交主仓库的所有更改
- 推送到 GitHub (origin)
- 推送到 Gitee
- 如果推送失败，会显示错误信息

---

## ⚙️ 配置说明

### 首次使用

1. **确保已配置 Git 用户信息**：
   ```bash
   git config --global user.name "你的名字"
   git config --global user.email "你的邮箱"
   ```

2. **确保已添加 Gitee 远程仓库**：
   ```bash
   git remote add gitee https://gitee.com/your-org/your-repo.git
   ```

   或者使用脚本自动添加（交互式脚本会提示）

### 远程仓库配置

**查看当前远程仓库**：
```bash
git remote -v
```

**手动添加 Gitee 远程仓库**：
```bash
git remote add gitee https://gitee.com/your-org/your-repo.git
```

**修改远程仓库 URL**：
```bash
git remote set-url gitee https://gitee.com/your-org/your-repo.git
```

---

## 🔍 使用示例

### 示例 1：日常提交

```bash
# 1. 修改代码
vim src/main.go

# 2. 运行交互式脚本
./scripts/git-push-all.sh

# 3. 按提示操作：
#    - 输入提交信息
#    - 确认提交
#    - 自动推送到 GitHub 和 Gitee
```

### 示例 2：快速提交

```bash
# 快速提交并推送
./scripts/git-push-all-auto.sh "fix: 修复 bug"
```

### 示例 3：处理 Submodule

```bash
# 如果 Submodule 有更改，脚本会自动处理：
# 1. 进入 Submodule 目录
# 2. 提交更改
# 3. 推送 Submodule
# 4. 更新主仓库的 Submodule 引用
# 5. 提交主仓库
# 6. 推送主仓库
```

---

## ⚠️ 注意事项

### 1. Submodule 处理

- 脚本会自动检测并处理 Submodule
- 如果 Submodule 有未提交的更改，会先提交 Submodule
- 然后更新主仓库的 Submodule 引用

### 2. 分支管理

- 脚本会推送到当前分支
- 确保当前分支已设置上游分支：
  ```bash
  git branch --set-upstream-to=origin/main main
  ```

### 3. 权限问题

- 确保有 GitHub 和 Gitee 的推送权限
- 如果使用 SSH，确保已配置 SSH 密钥
- 如果使用 HTTPS，可能需要输入用户名和密码

### 4. 冲突处理

- 如果推送时遇到冲突，需要先解决冲突
- 脚本不会自动处理冲突，需要手动解决

---

## 🐛 常见问题

### Q1: 提示 "未找到 gitee 远程仓库"

**解决**：
```bash
# 手动添加 Gitee 远程仓库
git remote add gitee https://gitee.com/your-org/your-repo.git
```

### Q2: 推送失败，提示权限错误

**解决**：
- 检查 SSH 密钥配置
- 或使用 HTTPS 并配置访问令牌

### Q3: Submodule 推送失败

**解决**：
- 检查 Submodule 的远程仓库配置
- 确保有 Submodule 仓库的推送权限

### Q4: 脚本执行失败

**解决**：
- 检查脚本是否有执行权限：`chmod +x scripts/git-push-all.sh`
- 检查是否在项目根目录执行
- 查看错误信息，根据提示解决

---

## 📊 脚本对比

| 特性 | 交互式脚本 | 自动脚本 |
|------|-----------|---------|
| 交互提示 | ✅ | ❌ |
| 自动提交 | ❌ | ✅ |
| 自动推送 | ✅ | ✅ |
| 适合场景 | 日常开发 | 快速提交/CI |
| 首次使用 | ✅ 推荐 | ⚠️ 需要配置 |

---

## 🔗 相关文档

- [Git Submodule 使用说明](../../note/企业代码Submodule使用说明.md)
- [企业代码版本控制方案](../../note/企业代码版本控制方案.md)
