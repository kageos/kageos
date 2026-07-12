# Kageos 代码结构重构路线图

本文记录面向外部贡献者的代码结构整理方向。目标是在不改变 API、业务规则、数据结构和运行行为的前提下，降低单文件认知负担，让第一次参与项目的人能更快找到修改入口。

## 约束

- 优先在原 package、原组件目录内按职责拆文件，不先引入新接口、框架或通用抽象层。
- 不在结构重构中同时修改路由、DTO、JSON 字段、数据库查询、错误文案或业务分支。
- 生成文件和数据文件不按普通源码治理，例如 Swagger `docs.go` 不做人工拆分。
- 一个重构 PR 聚焦一个模块，并保留原有测试；前端和后端重构不要混在同一 PR。
- 只有多个调用方已经出现相同稳定概念时才抽公共包，避免产生 `utils`、`helpers` 之类无边界目录。

## 已完成

### Capability Bundle 服务第一轮拆分

原 `core/app-server/service/service_tree_capability_bundle.go` 同时包含目录包导出、安装、AgentTask 转换与安装、文档安装、校验和路径安全逻辑。

第一轮只做同 package 函数移动：

| 文件 | 职责 |
| --- | --- |
| `service_tree_capability_bundle.go` | Capability Bundle 主流程、导出与安装编排 |
| `service_tree_capability_bundle_agent_tasks.go` | AgentTask 导出转换、安装请求与幂等键 |
| `service_tree_capability_bundle_paths.go` | package、文件和目标目录路径校验与拼接 |

本轮不修改任何函数签名、分支、错误信息或调用顺序。主文件由 1,859 行降至 1,437 行，AgentTask 和路径安全逻辑分别形成 296 行与 147 行的独立文件。

## 后续拆分顺序

| 优先级 | 当前文件 | 建议边界 | 风险与验证 |
| --- | --- | --- | --- |
| P1 | `core/app-server/service/service_tree_capability_bundle.go` | 继续拆成 `export`、`install`、`validation`、`docs` 四个职责文件 | 同 package 机械移动；运行 capability bundle 与 app-server service 测试 |
| P1 | `core/app-server/api/v1/standard_api.go` | 按 `table`、`form`、`chart/runtime`、`callback` 拆 handler；示例数据生成器单独成文件 | 保持 Gin 路由注册和响应完全不变；运行 app-server API 测试 |
| P1 | `core/app-runtime/service/app_manage_service.go` | 按应用生命周期、版本容器启动、清理任务、日志读取与 Git 提交拆文件 | receiver 保持不变；运行 app-runtime service 测试 |
| P1 | `core/app-runtime/service/container_service.go` | 按运行时安装准备、Podman JSON 解析、容器操作、secret 管理拆文件 | 保持 `ContainerOperator` 不变；运行容器解析和 runtime 测试 |
| P1 | `web/src/architecture/presentation/features/access/pages/TeamAccessPage.vue` | 页面只保留编排；拆当前权限、继承权限、角色编辑区和数据 composable | 保留请求时机与权限判断；运行 type-check、unit、build |
| P1 | `web/src/architecture/presentation/components/StructuredPromptComposer.vue` | 拆编辑器、上下文选择、附件区和 draft composable | 保留 `props`、`emits` 与提交 payload；运行组件测试和 build |
| P2 | `web/src/architecture/presentation/components/MiniWorkstation.vue` | 延续现有 composable/component 边界，拆消息区、输入区、会话操作与状态视图 | 当前分支已有相关改动，应独立收口后再继续，避免交叉冲突 |
| P2 | `web/src/architecture/presentation/components/WorkspaceInbox.vue` | 拆会话列表、线程详情、筛选与选择状态 | 先补关键交互测试，再移动模板与状态 |
| P2 | `core/app-server/service/app_service.go` | 按调用编排、操作日志、API 元数据同步、应用查询拆文件 | 先确认内部方法调用图，再做同 package 移动 |
| P3 | `web/src/architecture/shared/i18n/locales/*.ts` | 按业务域拆词条模块，入口文件只负责合并 | 容易产生 key 漏失，必须增加 key 对称性检查后再做 |

## 不建议现在做

- 不把每个 Go 函数都变成独立 package，也不为同 package 文件拆分创建接口。
- 不统一重命名大量文件、路由或导出符号；这会扩大外部贡献者难以审核的 diff。
- 不在没有复用场景时创建全局 `common`、`shared`、`utils` 目录。
- 不手工拆 Swagger、锁文件、生成声明或嵌入式 prompt 等生成物。
- 不在结构重构里顺便升级依赖、调整错误文案或格式化整个仓库。

## 每一批的完成标准

1. 拆分前后公开类型、函数签名、路由和序列化字段保持不变。
2. `git diff --check` 通过，Go 文件执行 `gofmt`。
3. 后端至少运行目标 package 测试；合并前运行 `scripts/test-core-go.sh`。
4. 前端至少运行 `npm run type-check`、`npm run test:unit -- --run` 和 `npm run build`。
5. PR 描述明确列出“只移动了什么”和“刻意没有改什么”。

## 开源发布说明

仓库已有贡献指南、行为准则、安全策略、CI、敏感文件检查和仓库体积守卫。当前许可证是 Business Source License 1.1，因此准确表述是“source-available and self-hostable”，不是 OSI 定义的开源软件。如果后续要改为真正的 OSI 开源发布，应把许可证选择和对外文案作为单独的治理决策处理，不与代码结构重构混合。
