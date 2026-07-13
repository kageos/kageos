# Kageos 代码结构重构路线图

本文记录面向外部贡献者的代码结构整理方向。目标是在不改变 API、业务规则、数据结构和运行行为的前提下，降低单文件认知负担，让第一次参与项目的人能更快找到修改入口。

## 约束

- 优先在原 package、原组件目录内按职责拆文件，不先引入新接口、框架或通用抽象层。
- 不在结构重构中同时修改路由、DTO、JSON 字段、数据库查询、错误文案或业务分支。
- 生成文件和数据文件不按普通源码治理，例如 Swagger `docs.go` 不做人工拆分。
- 一个重构 PR 聚焦一个模块，并保留原有测试；前端和后端重构不要混在同一 PR。
- 只有多个调用方已经出现相同稳定概念时才抽公共包，避免产生 `utils`、`helpers` 之类无边界目录。

## 已完成

### Capability Bundle 服务拆分

原 `core/app-server/service/service_tree_capability_bundle.go` 同时包含目录包导出、安装、AgentTask 转换与安装、文档安装、校验和路径安全逻辑。

前两轮只做同 package 函数移动：

| 文件 | 职责 |
| --- | --- |
| `service_tree_capability_bundle_agent_tasks.go` | AgentTask 导出转换、安装请求与幂等键 |
| `service_tree_capability_bundle_docs.go` | Docs 内容导出、节点创建更新与内容写入 |
| `service_tree_capability_bundle_export.go` | Bundle 导出编排、源码收集、TreeNode 与 package 元数据组装 |
| `service_tree_capability_bundle_install.go` | Bundle 安装编排、子目录过滤、安装计划与文件冲突检查 |
| `service_tree_capability_bundle_paths.go` | package、文件和目标目录路径校验与拼接 |
| `service_tree_capability_bundle_validation.go` | Bundle、TreeNode、Docs、AgentTask 和文件声明校验 |

拆分不修改任何函数签名、分支、错误信息或调用顺序。原 1,859 行混合职责文件已被 6 个职责文件替代；除导出和安装编排外，其余文件保持在 147-296 行之间。

### Standard API 入口拆分

原 `core/app-server/api/v1/standard_api.go` 的请求构造、Table、Form、Chart、Runtime 和 Callback handler 已按职责拆分：

| 文件 | 职责 |
| --- | --- |
| `standard_api.go` | 处理器类型、路径解析、请求与 callback envelope 构造 |
| `standard_api_table.go` | Table 查询、新增、模板下载、更新和删除 |
| `standard_api_table_examples.go` | Table 模板示例值与示例行生成 |
| `standard_api_execution.go` | Form 提交、Python Runtime 与 Chart 查询 |
| `standard_api_callback.go` | OnSelectFuzzy callback 入口 |

拆分保留原 Gin handler、Swagger 注释、权限检查、请求和响应顺序，不修改路由注册。

### App Manage Service 拆分

原 `core/app-runtime/service/app_manage_service.go` 已按应用管理职责拆分：

| 文件 | 职责 |
| --- | --- |
| `app_manage_service.go` | 服务类型、构造、应用创建/构建/更新/删除与状态查询 |
| `app_manage_service_runtime.go` | 版本容器规格、NATS secret、启动、停止与旧版本关闭 |
| `app_manage_service_cleanup.go` | 定时清理、容器级清理与非当前版本回收 |
| `app_manage_service_logs.go` | 应用日志读取、行范围合并与 Git 提交 |

拆分保留 `AppManageService` receiver、`ContainerOperator` 调用、通知等待和清理顺序，不修改应用生命周期行为。

### Container Service 拆分

原 `core/app-runtime/service/container_service.go` 已按 Podman 运行时职责拆分：

| 文件 | 职责 |
| --- | --- |
| `container_service.go` | 数据类型、`ContainerOperator`、服务构造、启停与 LSM 检测 |
| `container_service_install.go` | Podman 安装、操作系统适配、运行环境准备与连接 |
| `container_service_operations.go` | 命令执行、容器生命周期、挂载、环境变量与 Secret |
| `container_service_json.go` | Podman CLI JSON 解码与容器、镜像、运行状态解析 |

拆分保留 `ContainerOperator`、Podman 命令参数、Secret stdin 传递、网络和时区默认值，不修改容器运行行为。

### App Service 拆分

原 `core/app-server/service/app_service.go` 同时包含应用生命周期、运行时调用、Table 审计、API 元数据同步和应用查询。

本轮按同一个 `AppService` receiver 原样移动函数：

| 文件 | 职责 |
| --- | --- |
| `app_service.go` | 服务类型、runtime client 契约、依赖集合和构造函数 |
| `app_service_lifecycle.go` | 应用创建、更新、删除、工作区更新和发布后元数据收口 |
| `app_service_request.go` | 工作区应用调用、来源上下文、连接器前置检查和运行次数 |
| `app_service_table_log.go` | Table 操作日志、敏感字段过滤和摘要生成 |
| `app_service_metadata_sync.go` | Function/Package/ServiceTree 元数据增删改同步 |
| `app_service_queries.go` | 应用列表、可见性合并、详情和按用户名查询 |

原入口文件由 1,273 行缩到 55 行；46 个顶层声明的格式化 AST 与拆分前逐一比对一致。

### Storage API 拆分

原 `core/app-storage/api/v1/storage.go` 混合公开 Logo、公开分享、登录用户上传和文件管理入口。

| 文件 | 职责 |
| --- | --- |
| `storage.go` | 常量、Handler 类型和构造函数 |
| `storage_public_company_logo.go` | 企业 Logo 上传令牌、完成通知和校验 |
| `storage_public_share.go` | 匿名分享身份、公开上传和文件引用解析 |
| `storage_upload.go` | 登录用户单个/批量上传令牌和完成通知 |
| `storage_files.go` | 文件引用、描述、下载、删除、详情、统计和列表 |
| `storage_format.go` | IP、路径和文件大小展示格式化 |

原入口文件由 1,233 行缩到 23 行；路由注册、DTO、响应和函数体均未调整。

### Service Tree API 拆分

原 `core/app-server/api/v1/service_tree.go` 同时承载资源 CRUD、查询搜索、Bundle 和工作区文件桥接入口。

| 文件 | 职责 |
| --- | --- |
| `service_tree.go` | Handler 类型和构造函数 |
| `service_tree_create.go` | Package、Function、Docs 创建入口 |
| `service_tree_query.go` | 详情、批量查询、目录概览和资源搜索 |
| `service_tree_mutation.go` | Package、Function、Docs 更新删除和目录复制 |
| `service_tree_bundle.go` | 批量函数、Capability Bundle 导出和安装 |
| `service_tree_workspace.go` | 工作区上下文、文件写入/替换/删除和日志读取 |

原入口文件由 868 行缩到 18 行；Swagger 注释随对应 Handler 一起移动。

## 后续拆分顺序

| 优先级 | 当前文件 | 建议边界 | 风险与验证 |
| --- | --- | --- | --- |
| P1 | `web/src/architecture/presentation/features/access/pages/TeamAccessPage.vue` | 页面只保留编排；拆当前权限、继承权限、角色编辑区和数据 composable | 保留请求时机与权限判断；运行 type-check、unit、build |
| P1 | `web/src/architecture/presentation/components/StructuredPromptComposer.vue` | 拆编辑器、上下文选择、附件区和 draft composable | 保留 `props`、`emits` 与提交 payload；运行组件测试和 build |
| P2 | `web/src/architecture/presentation/components/MiniWorkstation.vue` | 延续现有 composable/component 边界，拆消息区、输入区、会话操作与状态视图 | 当前分支已有相关改动，应独立收口后再继续，避免交叉冲突 |
| P2 | `web/src/architecture/presentation/components/WorkspaceInbox.vue` | 拆会话列表、线程详情、筛选与选择状态 | 先补关键交互测试，再移动模板与状态 |
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
