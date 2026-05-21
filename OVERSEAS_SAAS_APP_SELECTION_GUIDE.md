# 出海 SaaS 应用选型指南

> 用途：给产品、开发和 AI 协作时使用。先判断哪些应用适合当前 Kageos，哪些应该降级，哪些暂时不要做。

## 选型总原则

第一阶段不要追求“覆盖所有小 SaaS”。应该选择最贴合当前平台原语的应用：

- Form：提交、处理、转换、生成。
- Table：记录、查询、编辑、状态流转。
- Chart：折线图、柱状图、饼图、仪表盘。

这里的 Form 要按“业务动作入口”理解，不只是文件工具。Form 可以调用第三方 API、LLM、内部业务系统 API、本地 CLI、Python 子进程，也可以把结果写入 Table 或返回文件。

最好的应用不是功能最多的，而是：

- 用户 5 分钟内能理解。
- 能用示例数据跑起来。
- 能通过 AI 修改字段和状态。
- 不依赖复杂前端。
- 不依赖企业治理能力。
- 如果依赖第三方 API，也能用手动触发、API key 配置、请求-响应式调用先跑通。

## 适合第一阶段做的应用

### 1. Ticket Tracker

适合度：高。

可做能力：

- 工单标题、描述、优先级、状态、负责人、提交人。
- 新增、编辑、删除。
- 按状态、优先级、负责人筛选。
- 状态分布饼图。
- 每日新增/关闭趋势折线图。
- 按优先级柱状图。

不要承诺：

- 邮件自动收件。
- SLA 自动提醒。
- 评论流。
- 客户门户。
- 自动通知。

可降级表达：

- SLA 字段可以做。
- 逾期筛选可以做。
- 自动提醒暂不做。

### 2. Feedback Collector / NPS

适合度：高。

可做能力：

- 反馈提交 Form。
- NPS 分数、标签、建议文本。
- 反馈记录 Table。
- 分数分布柱状图或饼图。
- NPS 指标放在 Chart Metadata。
- 趋势折线图。

不要承诺：

- 自动邮件发送。
- 多渠道自动采集。
- 复杂文本情感分析，除非明确有 LLM 或 NLP 能力配置。

### 3. Poll and Vote

适合度：高。

可做能力：

- 投票主题 Table。
- 选项 Table。
- 投票提交 Form。
- 投票记录 Table。
- 结果饼图或柱状图。

不要承诺：

- 实时刷新。
- 防刷风控体系。
- 匿名投票的强安全证明。

### 4. Inventory Tracker

适合度：高。

可做能力：

- 商品/SKU Table。
- 当前库存字段。
- 入库/出库 Form。
- 库存流水只读 Table。
- 低库存筛选。
- 分类库存柱状图。

不要承诺：

- 条码枪实时接入。
- 多仓复杂调拨。
- 自动采购。
- 电商平台同步。

### 5. Asset Tracker

适合度：高。

可做能力：

- 资产台账。
- 领用人、部门、状态、购买日期、保修期。
- 领用/归还记录。
- 按状态和部门统计。

不要承诺：

- 资产盘点移动端 App。
- 扫码实时盘点。
- 复杂折旧会计。

### 6. Resource Booking

适合度：中高。

可做能力：

- 资源 Table。
- 预约 Table。
- 新增预约时检查时间冲突。
- 按资源、时间、状态筛选。

不要承诺：

- 日历视图。
- 拖拽改期。
- Google Calendar 双向同步。
- 自动提醒。

### 7. Recruiting Tracker

适合度：中高。

可做能力：

- 职位 Table。
- 候选人 Table。
- 面试阶段状态。
- 评分、备注、简历附件。
- 阶段分布柱状图。

不要承诺：

- ATS 全流程。
- 邮件模板发送。
- 面试日历同步。
- Offer 审批。

### 8. Retail Sales Log

适合度：中高。

可做能力：

- 商品 Table。
- 会员 Table。
- 收银 Form。
- 支付记录只读 Table。
- 销售趋势折线图。
- 分类销售柱状图。
- 支付状态饼图。

不要承诺：

- 真正 POS 硬件接入。
- 支付网关集成。
- 发票税务系统。
- 多门店复杂经营分析。

### 9. API Action Forms

适合度：高。

这是 Form 的重要出海方向：把第三方 API 或内部 API 包装成团队可用的小工具。

可做能力：

- 输入参数。
- 调用外部 API。
- 解析响应。
- 返回结构化结果。
- 可选写入本地 Table。
- 可选返回文件。

适合例子：

- GitHub Issue Creator。
- Slack Webhook Sender。
- Stripe Payment Link Generator。
- Lead Enrichment Lookup。
- Domain / WHOIS Lookup。
- Shipping Status Lookup。
- Internal API Console。

不要承诺：

- OAuth 授权中心。
- 后台定时同步。
- webhook 长驻接收。
- 双向同步。
- 连接器市场。

### 10. AI Processing Forms

适合度：高。

如果用户配置了 OpenAI-compatible API 或其他可调用 LLM，Form 可以做很多业务工具。

可做能力：

- 文本摘要。
- 翻译。
- 改写。
- 分类。
- 信息抽取。
- CSV 字段标准化。
- 客服反馈归类。
- 简历要点提取。
- 合同条款摘要。

不要承诺：

- 没有模型配置时仍能稳定完成专业 AI 能力。
- 长文档无限上下文处理。
- 强合规行业的自动决策。
- 需要人工复核的场景不要包装成完全自动。

## 暂时谨慎的应用

### CRM Lite

可以做，但要非常克制。

可做：

- 客户台账。
- 联系人。
- 跟进记录。
- 阶段字段。
- 下次跟进时间。
- 按阶段统计柱状图或饼图。
- 手动触发线索 enrichment API。
- 手动创建外部 CRM 记录，如果用户提供 API key 且外部 API 简单。

不要说：

- Sales pipeline funnel chart。
- Marketing automation。
- Email sequences。
- Forecasting。
- Salesforce alternative。
- 双向同步 HubSpot / Salesforce。

推荐命名：

- Client Tracker
- Lead Tracker
- CRM Lite

但页面说明必须写清楚是轻量客户跟进台账。

### Project Task Tracker

可以做 Table 版，不要做项目管理平台。

可做：

- 任务列表。
- 状态、负责人、优先级、截止时间。
- 简单统计。

不要承诺：

- Kanban 拖拽。
- 甘特图。
- Sprint。
- 复杂依赖关系。

### Purchase Request Log

可以做记录管理，不要做审批系统。

可做：

- 采购申请记录。
- 状态字段。
- 申请人、金额、用途、附件。
- 人工修改状态。

不要承诺：

- 多级审批。
- 审批权限。
- 审批通知。
- 审批后自动执行采购。

## 不建议第一阶段做的应用

- BI Dashboard Builder。
- Workflow Automation。
- Full CRM。
- Full Helpdesk。
- Full HRIS。
- Accounting。
- Payroll。
- Contract lifecycle management。
- Marketing automation。
- Customer portal。
- Real-time chat。
- Calendar product。
- Enterprise approval system。

这些不是永远不能做，而是当前会迫使平台补复杂前端、连接器、权限、通知、定时和企业治理，容易拖垮第一阶段。

## 应用需求降级模板

遇到超边界需求时，按下面方式降级：

### 图表超边界

原需求：

> 做销售漏斗图。

降级：

> 当前 Chart 支持折线图、柱状图、饼图、仪表盘。可以用柱状图展示各销售阶段线索数量，用饼图展示阶段占比；暂不承诺漏斗图视觉。

### 审批超边界

原需求：

> 新增采购申请后自动走三级审批。

降级：

> 当前可以做采购申请 Table 和状态字段，由负责人手动更新状态。通用审批流、审批权限和通知暂不属于 MVP 应用侧能力。

### 定时任务超边界

原需求：

> 每天自动生成报表并发送邮件。

降级：

> 当前可以做手动运行的报表 Form 或统计 Chart。定时自动执行和邮件发送暂不作为第一阶段承诺。

### 第三方 API 集成超边界

原需求：

> 连接 HubSpot，实时双向同步客户数据。

降级：

> 当前可以先做手动触发的 HubSpot 查询或写入 Form：用户输入 API key 和客户信息，提交后调用一次 HubSpot API，并把结果写入本地 Table。OAuth、后台同步、冲突解决和 webhook 接收暂不作为第一阶段承诺。

### 看板超边界

原需求：

> 做拖拽式任务看板。

降级：

> 当前可以做任务 Table，支持状态筛选、编辑状态和统计图。拖拽看板是平台前端能力，暂不承诺。

## 第一批推荐组合

第一批不要一次做太多，建议先做：

1. Ticket Tracker
2. Feedback Collector / NPS
3. Poll and Vote
4. Inventory Tracker
5. Asset Tracker
6. Resource Booking
7. Recruiting Tracker
8. Retail Sales Log
9. PDF Tools
10. Image Tools
11. CSV and Excel Tools
12. OCR Tools
13. GitHub Issue Creator
14. Slack Webhook Sender
15. AI Text Classifier
16. Lead Enrichment Lookup

这个组合覆盖：

- 业务 Table。
- Form 提交。
- 文件处理。
- 多表关系。
- 基础 Chart。
- 用户能立刻试用的工具。
- 第三方 API 轻量集成。
- AI 请求-响应式处理。

## 对 AI 协作的要求

在给出应用建议前，必须先问自己：

- 当前项目是否已有对应案例？
- 是否能在 SDK 文档中找到真实 API？
- 是否需要当前不存在的图表、widget 或前端交互？
- 是否会偷偷引入通用审批、通知、定时任务？
- 是否能降级成 Form/Table/Chart？

如果不能确定，就先说不确定并回到代码和 prompt 查证。不要根据市场常识直接承诺。
