# AI-Agent-OS

AI-Agent-OS 是一个面向个人用户和小团队的 AI 轻应用工作台。

当前 MVP 聚焦一条主链路：

1. 用户在工作台用自然语言描述需求。
2. Agent 生成可确认的轻量 PRD。
3. Agent 按 PRD 创建或修改 `Form` / `Table` / `Chart`。
4. 系统构建、运行，并让用户直接使用生成出的轻应用。

项目暂时不追求“大而全”的企业平台形态。默认产品形态是 **Focused Mode / 产品聚焦模式**，优先保证一个人或一个小企业可以把常见管理系统、工具表单、数据表和统计看板跑起来。

## 当前核心能力

- AI 工作台：围绕需求澄清、PRD、代码生成、构建修复和运行验证展开。
- Service Tree：用目录组织工作空间里的应用能力。
- 标准函数形态：业务能力收敛为 `Form`、`Table`、`Chart`。
- 动态 UI：前端根据后端元数据渲染表单、表格、详情和图表。
- 运行时：生成后的应用通过 app-runtime 运行。
- Docs / LLM 管理：作为当前主链路的辅助能力保留。

## 产品聚焦模式

前端默认开启聚焦模式，只展示主链路入口。Table 更新日志、能力包、Docs、LLM 管理由 `web/src/architecture/shared/config/features.ts` 统一控制；组织管理、消息中心、定时任务、权限治理、License 和 Board 入口已从 MVP 中删除。

常用开关：

| 环境变量 | 默认行为 |
|---|---|
| `VITE_AOS_FOCUSED_MODE` | 默认 `true`，测试环境默认 `false` |
| `VITE_AOS_FEATURE_OPERATE_LOGS` | 默认开启，只保留 Table 更新日志 |
| `VITE_AOS_FEATURE_CAPABILITY_BUNDLE` | 默认开启 |
| `VITE_AOS_FEATURE_DOCS` | 默认开启 |
| `VITE_AOS_FEATURE_LLM_MANAGEMENT` | 默认开启 |

## MVP 边界

本仓库面向开源 MVP，不再保留企业版 License、权限治理、消息中心、控制面和备份控制面实现。Table 更新日志作为个人用户和小团队也能直接用到的审计能力保留，Form 操作日志和企业升级入口已移除。

## 代码结构

| 路径 | 说明 |
|---|---|
| `core/agent-server` | Agent 工作台、工具调用、PRD 与代码生成链路 |
| `core/app-server` | 工作空间 API、元数据、运行时调度入口 |
| `core/app-runtime` | 用户轻应用运行时 |
| `core/app-storage` | 上传文件和对象存储访问 |
| `core/api-gateway` | API 网关、静态资源入口和鉴权透传 |
| `core/hr-server` | 用户、登录、组织基础能力 |
| `web` | Vue 3 前端 |
| `docs` | 产品聚焦、部署等项目文档 |
| `deploy` | 开发和生产部署配置 |

## 开发入口

- 开发环境：[deploy/dev/README.md](deploy/dev/README.md)
- 生产部署：[deploy/prod/README.md](deploy/prod/README.md)
- 产品减法计划：[docs/产品减法与主链路打深计划.md](docs/产品减法与主链路打深计划.md)
- 前端架构：[web/README.md](web/README.md)

## 当前非目标

为了让开源 MVP 更清爽，以下方向暂不作为默认产品承诺：

- 跨函数工作流编排引擎
- 完整组织治理中心
- 配置管理中心
- 快链、回收站、变更日志等历史占位功能
- 企业 License / 权限审批 / 消息通知 / 备份控制面
- 复杂企业审批平台

这些能力可以在主链路稳定后逐个恢复，但不应该阻塞当前个人用户和小企业的轻应用创建体验。
