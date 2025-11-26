生成api文档：[generate-swagger.sh](generate-swagger.sh) 生成后文件在
[docs](core/app-server/docs) 下面

# AI-Agent-OS

**AI 原生操作系统 - 让描述变成应用**

[![License: BSL 1.1](https://img.shields.io/badge/License-BSL%201.1-blue.svg)](LICENSE)
[![Documentation](https://img.shields.io/badge/docs-latest-brightgreen.svg)](docs/ROADMAP.md)

---

## 🌟 项目愿景

> "所述即所得，描述即生成" - 让不懂技术的业务专家也能通过描述需求来构建完整的应用系统。

AI-Agent-OS 是一个革命性的 AI 原生操作系统，它让软件开发从"写代码"变成了"说需求"。

---

## ✨ 核心特性

### 🤖 AI 代码生成
- 通过自然语言描述需求，1 分钟生成完整可用的应用
- 基于标准 SDK，生成生产级代码
- 支持增量更新，无需重写整个应用

### 🧬 函数 Fork（独创）
- 将应用当作"基因"来复制和进化
- 物理隔离的多租户架构，数据完全独立
- 一键 Fork 他人的应用，秒级拥有相同功能

### 🔄 AI 工作流（革命性）
- 基于类型化 API 的自动工作流编排
- 大模型理解函数签名，智能匹配和组合
- 从"单点工具"升级到"无限可能"

### 📤 函数分享与交易
- 类似 Docker Hub 的函数市场
- 开发者上传函数，其他用户一键使用
- 付费函数交易，开发者获得收益

### 🎨 动态渲染引擎
- 基于 API 定义自动生成前端界面
- 无需手写前端代码
- 支持复杂的嵌套结构和回调

### 🏃 容器化运行
- 每个应用独立运行在容器中
- 物理级别的多租户隔离
- 极低的资源占用（~10MB/应用）

---

## 🤖 大模型开发者指南

> **重要**：本项目高度依赖大模型生成代码。如果你是 AI 助手，请按以下顺序阅读文档，理解项目后再生成代码。

### 📋 第一步：快速理解项目（5分钟）

**目标**：理解项目是什么、解决什么问题、核心架构

1. **📖 [项目概述](blueprint/README.md#项目概述)** - 项目是什么、解决什么问题、核心价值
   - 如果 `blueprint/` 目录还没有文档，请先阅读本 README 的"项目愿景"和"核心特性"章节
   
2. **🏗️ [系统架构](blueprint/README.md#系统架构)** - 整体架构图、核心组件、技术栈
   - 参考本 README 的"系统架构"章节
   - 核心组件：Web 渲染引擎、App Server、App Runtime、LLM Coder

3. **💡 [核心概念](blueprint/README.md#核心概念)** - 应用、服务目录、函数、模板等核心概念
   - **应用（App）**：一个 Go 项目，代表一个工作空间
   - **服务目录（Service Tree）**：一个 Go package，类似菜单分组
   - **函数（Function）**：一个 Go 文件，代表一个业务系统
   - **模板（Template）**：TableTemplate、FormTemplate、ToolTemplate

### 🔥 第二步：代码生成必读（10分钟）

**目标**：掌握如何生成符合项目规范的代码

4. **📝 [代码生成规范](blueprint/05-代码生成规范.md)** - **最重要**：如何生成符合规范的代码
   - SDK 使用规范
   - 模板类型选择（TableTemplate、FormTemplate、ToolTemplate）
   - 字段定义规范（GORM 标签、widget 标签、validate 标签）
   - 回调函数编写（OnTableAddRow、OnTableUpdateRows、OnSelectFuzzy）
   - **如果该文档不存在，请参考 `note/已归档-无需再看/prds/01系统设计.md` 中的完整示例**

5. **🛠️ [SDK使用指南](blueprint/06-SDK使用指南.md)** - SDK API 详解、模板类型、回调系统
   - SDK 位置：`sdk/agent-app/`
   - 核心包：`app`、`callback`、`response`、`widget`
   - **如果该文档不存在，请查看 `sdk/agent-app/` 目录下的代码和注释**

6. **📚 代码示例** - 完整的代码生成示例
   - 参考：`note/已归档-无需再看/prds/01系统设计.md` 中的 `crm_ticket.go` 完整示例
   - 参考：`namespace/luobei/demo/code/api/` 目录下的实际代码

### 📚 第三步：深入理解（按需阅读）

**目标**：理解数据流、前端渲染、部署运行

- **[数据流](blueprint/04-数据流.md)** - 请求流程、数据流转、前后端交互
- **[前端渲染引擎](blueprint/07-前端渲染引擎.md)** - Widget 系统、FormRenderer、TableRenderer
- **[部署与运行](blueprint/08-部署与运行.md)** - 容器化、多租户、资源管理

### 📖 文档导航

**项目文档组织规范**（详见 `.cursor/rules/项目规范.mdc`）：

- **📋 [项目蓝图](blueprint/README.md)** - **推荐从这里开始**：完整的技术架构和业务能力
  - 如果 `blueprint/` 目录为空，请先阅读本 README 和 `note/已归档-无需再看/prds/01系统设计.md`
  
- **🔧 [技术文档](docs/)** - 按模块分类的技术文档
  - 后端：`core/app-server/`、`core/app-runtime/`、`core/api-gateway/`、`core/app-storage/`
  - 前端：`web/docs/`、`web/src/core/`
  - SDK：`sdk/agent-app/`
  
- **📝 [临时分析](note/临时分析/)** - 功能分析和重构方案
  - 命名规范：`01xxx设计方案.md`、`02xxx重构方案.md`
  
- **✅ [待办事项](note/todos/)** - 待办功能列表
  - 命名规范：`01-xxx.todo.md`、`02-xxx.doing.md`、`03-xxx.done.md`

### 🎯 代码生成检查清单

生成代码前，请确保：

- [ ] 理解了项目核心概念（应用、服务目录、函数、模板）
- [ ] 选择了正确的模板类型（TableTemplate、FormTemplate、ToolTemplate）
- [ ] 所有字段都有 `widget` 标签定义前端组件
- [ ] 时间字段使用毫秒级时间戳：`gorm:"autoCreateTime:milli"`
- [ ] 用户字段使用 `type:user`：`widget:"name:创建用户;type:user" permission:"read"`
- [ ] 验证规则使用 `validate` 标签：`validate:"required,min=2,max=200"`
- [ ] 回调函数正确处理错误并返回
- [ ] 代码注释清晰，说明业务逻辑

### 💡 快速参考

**核心文件位置**：
- SDK 代码：`sdk/agent-app/`
- 示例代码：`namespace/luobei/demo/code/api/`
- 系统设计：`note/已归档-无需再看/prds/01系统设计.md`
- 项目规范：`.cursor/rules/项目规范.mdc`

**关键概念速查**：
- 一个**应用** = 一个 Go 项目 = 一个工作空间
- 一个**服务目录** = 一个 Go package = 一个菜单分组
- 一个**函数** = 一个 Go 文件 = 一个业务系统
- **模板类型**：TableTemplate（CRUD）、FormTemplate（表单）、ToolTemplate（工具）

---

## 🚀 快速开始

### 环境要求

- Go 1.21+
- Node.js 20+
- Docker/Podman
- SQLite 3

### 安装

```bash
# 克隆仓库
git clone https://github.com/your-org/ai-agent-os.git
cd ai-agent-os

# 后端
go mod download
./scripts/start.sh

# 前端
cd web
npm install
npm run dev
```

### 创建第一个应用

```bash
# 1. 注册用户并登录
# 2. 创建应用（如 "my_app"）
# 3. 在对话框输入需求
"我需要一个工单管理系统"

# 4. 等待 1 分钟，系统自动生成代码并部署
# 5. 刷新页面，查看生成的功能
```

---

## 🏗️ 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                       AI-Agent-OS                             │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │   前端渲染    │◄───│    Hub       │───►│  LLM Coder   │  │
│  │   引擎       │    │   (函数市场)  │    │  (代码生成)  │  │
│  └──────┬───────┘    └──────────────┘    └──────────────┘  │
│         │                                                     │
│         │                                                     │
│  ┌──────▼────────────────────────────────────────┐          │
│  │            App Server (应用管理)              │          │
│  └──────┬────────────────────────────────────────┘          │
│         │                                                     │
│         │                                                     │
│  ┌──────▼────────────────────────────────────────┐          │
│  │        App Runtime (容器化运行时)             │          │
│  │  ┌────────┐  ┌────────┐  ┌────────┐          │          │
│  │  │ App 1  │  │ App 2  │  │ App 3  │  ...     │          │
│  │  └────────┘  └────────┘  └────────┘          │          │
│  └───────────────────────────────────────────────┘          │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

**核心组件**：

- **Web 渲染引擎**：动态渲染前端界面
- **Hub**：函数市场，支持分享和交易
- **LLM Coder**：AI 代码生成服务
- **App Server**：应用管理和函数注册
- **App Runtime**：容器化运行时，物理隔离

---

## 📚 文档

- [开发路线图](docs/ROADMAP.md)
- [商业化策略](plan/商业化策略分析.md)
- [功能评分与优先级](plan/功能评分与优先级分析.md)
- [Hub 生态战略](plan/Hub生态战略分析.md)
- [架构设计](web/docs/架构总览.md)

---

## 🤝 参与贡献

我们欢迎社区贡献！请阅读 [贡献指南](CONTRIBUTING.md)（待创建）。

### 开发者

- 贡献代码（SDK、渲染引擎、CLI）
- 提交 Bug 和建议
- 参与功能讨论

### 内容创作者

- 开发和分享函数
- 编写文档和教程
- 制作视频教程

---

## 📄 许可证

本项目采用 **Business Source License 1.1** 许可证。

**核心条款**：
- ✅ 可以免费使用、修改、分发（非商业托管服务）
- ✅ 可以用于内部开发、测试、生产环境
- ❌ 不能基于本项目搭建商业托管服务（如 SaaS）
- ⏰ 4 年后自动转为 Apache 2.0 许可证

**商业托管服务定义**：
> 允许第三方（非你的员工和承包商）通过网络远程访问本软件功能的商业服务。

**简单来说**：
- ✅ 企业内部使用：完全免费
- ✅ 为客户定制开发：完全免费
- ❌ 搭建 SaaS 平台对外提供服务：需要购买商业许可

详见 [LICENSE](LICENSE) 文件。

---

## 🎯 为什么选择 BSL 1.1？

我们希望在开源和商业化之间找到平衡：

1. **保护社区利益**：防止大公司直接抄袭并商业化
2. **保持开源精神**：4 年后自动转为 Apache 2.0
3. **允许企业使用**：企业内部使用完全免费
4. **支持项目发展**：商业化收入用于持续开发

**成功案例**：
- HashiCorp（Terraform、Vault）
- CockroachDB
- Sentry

---

## 🌍 社区

- **GitHub Issues**: [提交 Bug 和建议](https://github.com/your-org/ai-agent-os/issues)
- **Discussions**: [参与讨论](https://github.com/your-org/ai-agent-os/discussions)
- **Discord**: [加入社区](https://discord.gg/your-invite)（待创建）

---

## 🔗 相关项目

- [SDK](sdk/agent-app) - 函数开发工具包
- [CLI](tools/agent-cli)（待创建）- 命令行工具
- [Hub](https://hub.ai-agent-os.com)（待上线）- 函数市场

---

## 📊 项目统计

- 🌟 Star 数（待更新）
- 🍴 Fork 数（待更新）
- 👥 贡献者（待更新）
- 📦 函数库（待更新）

---

## 💬 联系我们

- 邮箱：hello@ai-agent-os.com（待创建）
- Twitter：@ai_agent_os（待创建）
- 微信公众号：AI-Agent-OS（待创建）

---

## 🙏 致谢

感谢所有贡献者和支持者！

---

**Built with ❤️ by AI-Agent-OS Team**
