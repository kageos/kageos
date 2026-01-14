# Hub - 应用市场

> **注意**：Hub 是独立项目，不提交到开源仓库。

## 项目说明

Hub 是 AI-Agent-OS 的应用市场，提供应用浏览、发布、试用、克隆等功能。

## 核心定位

**Hub + OS 一体化生态系统**：
- ✅ **Hub**：应用市场，卖软件、介绍软件
- ✅ **OS**：运行平台，可以试用、克隆、运行软件
- ✅ **互通**：Hub 和 OS 用户系统完全互通

## 商业模式

**服务费模式**：
- ✅ **从"克隆费"到"服务费"**：强调后续服务，而不是软件本身
- ✅ **代码完全开源**：代码完全开源，增强信任，促进付费
- ✅ **构造记录完全开源**：构造记录完全开源，增强信任，促进付费
- ✅ **开发者收益分成**：开发者 80%，Hub 平台 10%，OS 平台 10%

## 核心价值

**软件可以模仿，但服务无法模仿**：
- ✅ **技术支持**：有问题可以随时找开发者
- ✅ **需求调整**：可以帮你调整，例如新增字段（免费 3 个需求）
- ✅ **文档知识库**：免费文档知识库共享
- ✅ **版本升级**：可以同步升级新的版本，无缝升级
- ✅ **生态支持**：无法复用别人的生态
- ✅ **社区支持**：社区的活跃度

## 目录结构

```
hub/
├── backend/          # Hub 后端服务
│   ├── api/         # API 接口
│   ├── service/     # 业务逻辑
│   ├── model/       # 数据模型
│   └── client/      # OS API 客户端
├── frontend/        # Hub 前端页面
│   ├── src/
│   └── public/
├── DESIGN.md        # 设计方案
├── DEVELOPMENT_PLAN.md  # 开发计划
└── README.md        # 本文件
```

## 与主系统的集成

Hub 通过 REST API 与主系统（AI-Agent-OS）通信：

- **Hub → OS**：用户认证、获取源代码、获取构造记录、克隆应用
- **OS → Hub**：发布应用、同步应用元数据

## 用户系统互通

**统一用户系统**：
- ✅ **共享用户数据库**：Hub 和 OS 共享同一个用户数据库
- ✅ **统一认证**：使用统一的 JWT Token 认证
- ✅ **单点登录**：Hub 和 OS 之间单点登录（SSO）

**实现方式**：
- ✅ **Hub 调用 OS API**：Hub 通过 API 调用 app-server 的用户服务
- ✅ **OS 使用本地服务**：OS 直接使用 app-server 的用户服务

## 开发说明

1. Hub 是闭源项目，代码不提交到开源仓库
2. 本地开发时，Hub 和主系统可以一起开发测试
3. 生产环境建议 Hub 独立部署

## 代码提交

Hub 使用独立的 Git 仓库进行版本控制：

**远程仓库**：git@github.com:ai-agent-os-hub/hub.git（私有）

**提交方式**：
```bash
# 在 hub 目录下操作
cd hub

# 正常使用 git 命令
git add .
git commit -m "feat: add new feature"
git push
```

**注意事项**：
- Hub 目录在主仓库的 `.gitignore` 中，不会被主仓库跟踪
- 所有 Hub 代码都在 `hub/` 目录下的独立 Git 仓库中管理
- 主仓库的提交不会包含 Hub 代码

## 技术栈

- **后端**：Go + Gin
- **前端**：Vue 3 + Element Plus
- **数据库**：PostgreSQL
- **缓存**：Redis（可选）

## 设计方案

详细的设计方案请查看 [DESIGN.md](./DESIGN.md)，包含：
- 应用发布方案（SaaS用户 + 私有化部署）
- 应用试用和克隆方案（Hub → OS 跳转）
- 服务费模式设计
- 架构设计
- 安全设计
- 实现步骤

## 开发计划

详细的开发计划请查看 [DEVELOPMENT_PLAN.md](./DEVELOPMENT_PLAN.md)，包含：
- 数据库设计
- API 设计
- 开发任务清单
- 实施步骤

## 快速开始

### 1. 数据库初始化

```bash
# 创建数据库
createdb hub_db

# 运行迁移脚本
psql hub_db < migrations/001_initial.sql
```

### 2. 后端启动

```bash
cd backend
go mod download
go run main.go
```

### 3. 前端启动

```bash
cd frontend
npm install
npm run dev
```

## 开发任务

详细的任务清单请查看 [DEVELOPMENT_PLAN.md](./DEVELOPMENT_PLAN.md#五开发任务清单)
