# Kageos 项目能力与边界说明

> 用途：在讨论出海 SaaS、官方应用库、namespace 应用规划时，先校准项目真实能力，避免把普通 SaaS 想象直接套到当前平台上。

## 一句话结论

当前项目最适合做：

> 基于 Form / Table / Chart 的轻量业务应用工作台。

它不是完整低代码平台、BI 平台、流程引擎、协同前端框架或通用 SaaS 复制机。应用开发建议从“平台已经稳定支持什么”出发，而不是从市场上常见 SaaS 功能反推。

## 当前稳定产品形态

### 1. Form

Form 适合“一次输入，一次业务动作，一次输出”的场景。它不只是文件处理工具，也可以作为内部业务动作入口、第三方 API 调用入口、AI/LLM 调用入口和多系统编排入口。

典型能力：

- 文件上传和处理。
- 表单字段输入。
- 文本、JSON、CSV、Markdown、HTML 等内容生成或转换。
- 返回结构化结果。
- 返回输出文件。
- 调用本地 CLI、Go 库或 Python 子进程完成处理。
- 调用第三方 HTTP API，例如 OpenAI-compatible API、Slack webhook、GitHub API、Stripe API、Google API、内部业务系统 API 等。
- 调用平台 API，但必须通过 `ctx.APICall` 或 `/system/openapi` 包装。
- 调用外部 API 后把结果写入 Table，或返回给用户。
- 组合“查库 + 调 API + 写库 + 生成文件”的一次性业务流程。

适合应用：

- PDF 合并、拆分、压缩、OCR。
- 图片压缩、格式转换、缩略图。
- Excel/CSV 转换、清洗、分析。
- 文档转换。
- 视频转码、抽帧、缩略图总览。
- 文本摘要、关键词提取、分类。
- AI 内容生成、翻译、改写、信息抽取。
- 第三方数据查询，例如域名信息、汇率、物流状态、GitHub issue、CRM 线索 enrichment。
- 外部系统动作，例如发送 Slack webhook、创建 GitHub issue、生成 Stripe payment link。
- 内部系统操作台，例如输入参数后调用公司内部 API，返回结果并记录日志。

不适合承诺：

- 长时间后台任务队列。
- 实时协同编辑。
- 持续监听外部事件。
- 定时自动执行。
- 复杂多步骤前端向导，除非退化成多个独立 Form。
- 需要 webhook 长驻接收器的集成，除非平台后续明确支持公开回调入口。
- 需要 OAuth 授权管理、token 刷新、连接器市场的完整集成，除非先做成“用户手填 API key 的轻量版本”。

### 2. Table

Table 适合“管理一批结构化记录”的场景。

典型能力：

- 单表 CRUD。
- 多表关联查询。
- 列表筛选、分页、排序。
- 新增、编辑、删除回调。
- 只读事实记录表。
- 状态字段和简单状态流转。
- 当前用户、当前部门等上下文字段。
- 平台记录 Table 更新日志。
- Table 列表数据源可以来自本地 GORM，也可以来自第三方 API；但如果来自第三方 API，要明确分页、排序、筛选哪些由外部 API 支持，哪些只能本地处理。

适合应用：

- 工单台账。
- 客户跟进记录。
- 库存台账。
- 资产管理。
- 采购申请记录。
- 候选人管理。
- 会议室或资源预约。
- 投票主题、选项、投票记录。

不适合承诺：

- 拖拽看板。
- 复杂权限矩阵。
- 多级审批流。
- 评论、点赞、收藏、通知中心。
- 类 Jira 的完整项目管理。
- 类 Salesforce 的完整 CRM。

### 3. Chart

Chart 适合“基于已有数据做标准统计图”。

当前 SDK 明确支持的图表类型只有：

- `LineChart`
- `BarChart`
- `PieChart`
- `GaugeChart`

重要限制：

- 一个 Chart 路由只返回一张图。
- 多张图必须拆成多个 `.chart` 路由。
- Form 不能直接返回 Chart 结构体。
- 统计指标可以放在 `Metadata`。
- 不支持自定义 ECharts 全量配置。
- 不支持 Funnel、Sankey、Radar、TreeMap、Heatmap 等图表类型。

因此，“销售漏斗图”当前不要作为标准承诺。可以降级为：

- 用 `BarChart` 展示各销售阶段数量。
- 用 `PieChart` 展示阶段占比。
- 用 Table 展示线索和阶段字段。

但不要叫“漏斗图”，也不要承诺漏斗图视觉。

Chart 的数据源也不必只来自本地表。可以从第三方 API 拉取数据后聚合成折线图、柱状图、饼图或仪表盘，但仍受当前图表类型和“一路由一图”的限制。

## 当前应用侧可以做的业务逻辑

应用侧 Go 代码可以实现：

- 表单校验。
- 数据库读写。
- 多表事务。
- 库存扣减、余额变更、票数累计等一致性逻辑。
- 业务状态变更。
- 只读流水表。
- 文件下载、处理和输出。
- 统计聚合查询。
- 基于当前用户和部门的默认值或筛选。
- 调用平台 Web API，但必须通过 `ctx.APICall`。
- 调用第三方 HTTP API。
- 调用 OpenAI-compatible LLM API。
- 调用外部 webhook。
- 调用公司内部业务 API。
- 做 API 返回值解析、字段映射、错误归一化和结果落库。

应用侧不要自造：

- 通用权限系统。
- 通用审批系统。
- 通用评论系统。
- 通用通知中心。
- 定时任务平台。
- 后台调度平台。
- 全局消息中心。
- 备份控制面。

## 平台侧能力边界

平台侧负责：

- 工作空间。
- 服务树。
- Form / Table / Chart 渲染。
- Widget 渲染和校验。
- 文件上传和输出展示。
- 构建、运行、注册。
- Table 更新日志。
- 当前用户、部门、trace 等运行上下文。

业务应用不要绕过平台侧能力：

- 不裸写 HTTP client 调平台内部接口。
- 不硬编码 token。
- 不伪造 request_user。
- 不直连平台数据库。
- 不调用已删除的定时任务、全局消息、备份、License 等接口。

注意：这里禁止的是“裸写 HTTP client 调平台内部接口”。调用第三方 API 或用户自己的外部业务系统，属于应用侧能力，可以使用 Go 标准库或合适的 SDK，但要遵守下面的集成边界。

## 第三方 API 集成边界

第三方 API 是当前项目非常重要的应用扩展面。Form、Table、Chart 都可以使用外部 API：

- Form：提交参数后调用外部 API，返回结果、文件或写入记录。
- Table：从外部 API 拉取列表，或把本地记录同步到外部系统。
- Chart：从外部 API 拉取指标后渲染标准图表。

适合做：

- AI 总结、翻译、分类、信息抽取。
- 发票、收据、名片、合同字段抽取。
- GitHub issue 创建和查询。
- Slack / Discord / Teams webhook 发送。
- Stripe payment link 创建。
- Google Maps / Places 数据查询。
- Shopify / WooCommerce 订单查询。
- Postmark / SendGrid 邮件发送。
- HubSpot / Pipedrive 线索查询或写入。
- 内部 ERP / CRM / OA API 操作台。

需要明确的限制：

- API key / token 如果落业务库，默认不会自动加密；字段可用 `sensitive:"true"` 避免进入平台操作日志，但这不等于加密存储。
- 长期 token 管理、OAuth 授权跳转、refresh token 轮换、连接器市场，不属于当前应用侧默认能力。
- 外部 API 调用必须设置超时，不能让请求无限等待。
- 失败要返回可读错误，并在日志中保留必要上下文；不要把密钥、完整隐私正文、大文件内容打进日志。
- 外部 API 和本地数据库无法处在同一个事务里；需要强一致时要设计状态字段和补偿逻辑。
- Webhook 接收、后台轮询、定时同步不属于当前 MVP 平台能力；可以先做手动触发的 Form 或 Table 操作。
- 高风险付费动作、删除动作、发送消息动作要在 PRD 里明确输入、确认字段和审计需求。

## Widget 真实边界

当前适合使用的组件包括：

- `ID`
- `input`
- `text`
- `text_area`
- `richtext`
- `select`
- `radio`
- `checkbox`
- `multiselect`
- `list`
- `number`
- `float`
- `slider`
- `rate`
- `switch`
- `datetime`
- `color`
- `files`
- `user`
- `users`
- `department`
- `departments`
- `progress`
- `link`
- `table`
- `form`

不要承诺或生成：

- `date`
- `time`
- `range`
- `image`
- `tag`
- `tree`
- `cascader`
- `code`
- `password:true`
- 未确认的 widget 参数。

图片、视频、PDF 等都优先通过 `files` 处理。日期时间统一使用 `datetime`。

## 可验证闭环

一个应用需求进入开发前，必须能回答：

1. 它主要是 Form、Table 还是 Chart？
2. 是否只需要请求-响应式执行？
3. 是否能用已有 widget 表达输入和输出？
4. 是否能用 Go、CLI、Python 子进程或已有库完成处理？
5. 是否不依赖通用审批、通知、定时任务、复杂前端？
6. 写完后是否能通过 build 和对应运行工具验证？

如果任一项不满足，应该先降级设计，而不是硬承诺。

## 对外表达建议

推荐说法：

> Build and run lightweight internal apps with Form, Table, and Chart primitives.

不推荐说法：

> Generate any SaaS app.

推荐说法：

> Install a practical app, then customize its fields, tables, forms, and basic charts with AI.

不推荐说法：

> Recreate CRM, Jira, Airtable, Retool, BI, workflow automation, and approval systems in one product.

## Public Form 边界

外部用户提交问卷、反馈、报名、外部工单这类场景，需要 Public Form 能力。当前建议只做单点能力：

> 某个 Form 节点开启公开提交后，未登录用户通过公开链接访问这一个 Form 并提交。

不把匿名用户扩展成完整 guest 工作台权限，不允许匿名用户访问服务树、Table、Chart 或其他函数。

详细设计见根目录 `PUBLIC_FORM_ANONYMOUS_SUBMIT.md`。
