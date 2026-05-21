# Namespace 应用与工具规划

> 用途：说明 namespace 下面应该沉淀什么，不应该混成什么，避免把测试应用、工具函数、官方应用和商业套件放在一个概念里讨论。

## Namespace 的真实含义

`namespace` 是运行和生成用户应用的工作空间。它承载：

- 用户应用代码。
- 应用运行数据。
- 构建产物。
- 日志和元数据。
- 本地测试和实验应用。

因此，`namespace` 不应该直接等同于“官方应用市场”。出海产品化时，要把本地工作区内容整理成可安装的 capability bundle、官方应用包或场景套件。

## 推荐分层

### 1. 系统工具库

定位：通用 Form 工具和轻量集成入口，解决文件、文本、数据处理，也可以把第三方 API 包装成可运行的工作台工具。

适合放：

- PDF 工具。
- 图片工具。
- OCR 工具。
- Excel/CSV 工具。
- 文档转换工具。
- 视频/音频工具。
- JSON/HTML/Markdown 工具。
- 数据库 inspect/query 工具。
- LLM 摘要、翻译、分类、抽取工具。
- 第三方 API 查询工具。
- Webhook 发送工具。
- 内部 API 操作工具。

特点：

- 多数是 Form。
- 输入是文件或文本。
- 输出是文件、文本或结构化结果。
- 可以调用外部 API、LLM、内部系统 API。
- 用户可以马上试用。
- 适合免费体验、SEO、demo。

不应该过度包装成：

- 完整业务系统。
- 多人协同产品。
- 企业治理能力。

### 2. 官方轻业务应用

定位：可安装、可复制、可由 AI 修改的业务小应用。

适合放：

- 工单管理。
- 投票/问卷/NPS。
- 库存台账。
- 资产台账。
- 采购申请记录。
- 会议室/资源预约。
- 候选人管理。
- 小型收银/销售记录。

特点：

- 以 Table 为主。
- 可搭配 Form 做提交入口。
- 可搭配 Chart 做基础统计。
- 需要示例数据。
- 需要英文名称、说明、截图和可修改示例。

设计原则：

- 优先做台账，不做完整行业 SaaS。
- 优先做状态字段，不做通用审批流。
- 优先做标准图表，不做复杂 BI。
- 优先做可构建、可运行、可验证的闭环。
- 第三方 API 集成先做手动触发和轻量配置，不先承诺 OAuth 连接器、后台同步和定时任务。

### 3. 场景套件

定位：多个轻应用组合成一个可售卖或可展示的 pack。

适合方向：

- File Tools Pack。
- Operations Pack。
- HR Lite Pack。
- Retail Lite Pack。
- Feedback Pack。
- Agency Ops Pack。
- API Ops Pack。
- AI Text Ops Pack。

特点：

- 套件不是新技术形态，只是多个 Form/Table/Chart 应用的组合。
- 每个应用仍要单独可运行。
- 套件要强调“同一个 workspace、同一个登录、同一套文件和数据入口”。

## 现有能力盘点

从当前仓库看，已经有这些基础：

- `system/tools/工具库.capability-bundle.json` 覆盖 archive、audio、chart、database、diagram、document、file、html、image、json、ocr、pdf、runtime、table、text、video。
- case catalog 覆盖单 Table、会议室预约、HR、投票、收银台、Excel/CSV、PDF、图片、视频、NLP、Python 输出文件。
- `namespace/liubeiluo/demos/code/api` 已有 demo，包括 PDF、图片、Excel/CSV、OCR、视频、投票、收银、工单、HR、会议室预约。

这些说明项目当前最强的不是“任意生成 SaaS”，而是：

1. 结构化台账。
2. 文件处理工具。
3. 表单提交到记录。
4. 简单统计图。
5. 多表业务逻辑和事务。
6. 第三方 API / LLM / 内部 API 的轻量请求-响应式封装。

## 当前不建议进入官方应用库的能力

以下能力如果做成官方承诺，容易超出当前平台边界：

- 销售漏斗图，当前没有 Funnel chart。
- 拖拽看板。
- 甘特图。
- 复杂仪表盘布局。
- 自定义 ECharts 配置。
- 实时聊天。
- 评论流。
- 通知中心。
- 自动定时提醒。
- 自动邮件营销。
- OAuth 连接器矩阵。
- 双向同步 Google Sheets、Airtable、Notion。
- 多组织、多 workspace 切换。
- 企业级 SSO/SCIM。
- 复杂 RBAC 和审批。

可以记录为后续平台能力，但不应作为第一阶段 namespace 应用承诺。

但这不等于不能调用第三方 API。第一阶段可以做的是“轻量 API 工具”和“手动触发式集成”：

- 用户填 API key 或 webhook URL，Form 调用一次外部接口。
- Table 行操作触发一次外部写入。
- Chart 查询时调用外部指标 API 并渲染标准图表。
- 调用后把结果写入本地记录，方便追踪。

第一阶段不要承诺的是完整连接器平台：

- OAuth 授权中心。
- token refresh 生命周期管理。
- webhook 长驻接收。
- 后台定时同步。
- 冲突解决。
- 双向同步状态机。

## 第一阶段命名建议

工具类：

- PDF Tools
- Image Tools
- CSV and Excel Tools
- OCR Tools
- Document Tools
- Video Tools

业务类：

- Ticket Tracker
- Feedback Collector
- Poll and Vote
- Inventory Tracker
- Asset Tracker
- Purchase Request Log
- Recruiting Tracker
- Resource Booking
- Retail Sales Log

集成类：

- Slack Webhook Sender
- GitHub Issue Creator
- AI Text Classifier
- Invoice Field Extractor
- Lead Enrichment Lookup
- Stripe Payment Link Generator
- Internal API Console

注意这些名字故意使用 Tracker、Collector、Log、Lite、Tools，而不是 Helpdesk、CRM、ERP、BI、Workflow 这类容易引发过高预期的词。

## 应用准入规则

一个应用进入官方库前，至少满足：

- 能归类为 Form、Table、Chart 或它们的组合。
- 不依赖当前不存在的 widget。
- 不依赖当前不存在的图表类型。
- 不依赖平台定时任务、通知、审批、评论。
- 有示例数据或示例输入。
- 有 1 到 3 个用户可理解的修改示例。
- 能 build 通过。
- Form 能提交验证。
- Table 能搜索验证；如有写操作，能新增、编辑、删除验证。
- Chart 能查询验证。

## 推荐出海主张

推荐：

> One workspace for lightweight tools and business trackers.

推荐：

> Install a working app first, then customize fields, forms, tables, and basic charts with AI.

避免：

> Replace every SaaS tool.

更准确的版本：

> Replace scattered spreadsheets and small one-off tools where a lightweight Form/Table/Chart app is enough.
