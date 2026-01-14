# Git 工作流程说明

## 策略说明

### 开发阶段（当前）
- **全量代码提交到 Gitee**：包括主代码和企业代码（enterprise_impl）
- **个人账号**：使用项目级别配置（执念 <1210227080@qq.com>）
- **仓库地址**：https://gitee.com/lliubaorui/ai-agent-os.git

### 开源阶段（后续）
- **过滤企业代码后推送到 GitHub**：使用 `git-sync-to-github.sh` 脚本
- **GitHub 仓库**：git@github.com:ai-agent-os/ai-agent-os.git

## 脚本说明

### 1. git-push-all-auto.sh
**功能**：开发阶段一键提交所有代码到 Gitee

**使用方式**：
```bash
bash scripts/git-push-all-auto.sh "提交信息"
```

**特点**：
- 自动提交主代码和 Submodule
- 推送到 Gitee（全量代码，包括企业代码）
- 使用项目级别的 Git 配置（不修改全局配置）

### 2. git-sync-to-github.sh
**功能**：同步代码到 GitHub（过滤企业代码）

**使用方式**：
```bash
bash scripts/git-sync-to-github.sh
```

**过滤规则**：
- `enterprise_impl/` - 企业功能实现
- `licenses/` - License 文件目录
- `license*.json` - License 文件

**注意**：
- 会创建临时分支进行过滤
- 推送到 GitHub 后可以删除临时分支
- 不会影响 Gitee 的代码

## Git 配置

### 项目级别配置（已设置）
```bash
git config --local user.name "执念"
git config --local user.email "1210227080@qq.com"
```

### 远程仓库配置
```bash
# Gitee（开发阶段，全量代码）
git remote add gitee https://gitee.com/lliubaorui/ai-agent-os.git

# GitHub（开源版本，过滤企业代码）
git remote add github git@github.com:ai-agent-os/ai-agent-os.git
```

## 工作流程

### 日常开发
1. 编写代码
2. 执行 `bash scripts/git-push-all-auto.sh "提交信息"`
3. 代码自动提交并推送到 Gitee

### 开源同步
1. 确保代码已提交到 Gitee
2. 执行 `bash scripts/git-sync-to-github.sh`
3. 脚本会自动过滤企业代码并推送到 GitHub

## 注意事项

1. **不要修改全局 Git 配置**：项目使用项目级别配置，不会影响其他项目
2. **主提交到公司账号**：公司项目使用全局配置（wb_liubeiluo <wb_liubeiluo@kuaishou.com>）
3. **个人项目使用项目配置**：本项目使用项目级别配置（执念 <1210227080@qq.com>）
4. **Submodule 配置**：脚本会自动确保 Submodule 使用项目级别的 Git 配置

## 常见问题

### Q: 为什么使用项目级别配置？
A: 因为主提交是提交到公司账号的，这个项目是个人项目，使用项目级别配置可以避免影响全局配置。

### Q: Submodule 的提交会使用什么账号？
A: 脚本会自动确保 Submodule 使用项目级别的 Git 配置，与主项目保持一致。

### Q: 如何查看当前 Git 配置？
A: 
```bash
# 查看项目级别配置
git config user.name
git config user.email

# 查看全局配置
git config --global user.name
git config --global user.email
```
