# AI 工作台作为垂直行业助手的价值分析

> 用途：给 AI、产品、开发和商业讨论读取。本文讨论 Kageos 的“工作台”是否可以作为一种独立能力售卖，而不只是卖模板、Form、Table 或文件工具。

## 一句话判断

工作台本身可以成为一个可售卖能力。

它的价值不是“聊天”，而是：

> 让 AI 在一个有数据、有工具、有文件、有表格、有代码执行能力的 workspace 里，持续帮助用户处理真实业务操作。

如果 Form / Table / Chart 是应用原语，那么工作台就是“操作层”和“智能协作层”。它可以把一个普通轻应用升级成行业助手。

## 和模板的区别

模板解决的是：

- 我有一个固定业务场景。
- 安装后直接运行。
- 用户按字段、表格、表单完成操作。

工作台解决的是：

- 用户不知道下一步该点哪里。
- 用户有一堆文件、记录、历史上下文。
- 用户希望 AI 帮他查、整理、分析、生成、修改、执行。
- 用户希望用自然语言驱动多个工具和数据。

模板是“应用”。工作台是“帮你使用和改造应用的人机界面”。

## 典型垂直场景：律师助手

律师助手不是一个单一 Form 或 Table，而是一个 workspace：

### 数据对象

- 案件 Table。
- 当事人 Table。
- 证据材料 Table。
- 时间线 Table。
- 文书模板目录。
- 合同、判决书、聊天记录、邮件、PDF、图片等文件。
- 案件笔记和会议纪要。

### 工作台能做的事

- 根据用户描述创建案件结构。
- 上传一批材料后，提取当事人、日期、金额、争议焦点。
- 把证据整理成时间线。
- 从案件记录里生成案情摘要。
- 对比两份合同差异。
- 从聊天记录里提取关键承诺和时间点。
- 根据案件事实生成待补充材料清单。
- 根据已有模板生成文书草稿。
- 查询某个案件的所有相关文件和记录。
- 写 Python 或 Go 脚本批量处理文件，例如 OCR、重命名、拆分 PDF、提取表格。
- 根据案件 Table 生成统计，例如案件阶段、客户来源、回款状态。

这不是“替代律师”，而是帮助律师和助理减少整理、检索、归档、初稿生成、材料分析的重复劳动。

## 其他垂直行业也类似

### Agency Assistant

- 管理客户、需求、交付物、周报、会议纪要。
- AI 帮忙生成客户周报、整理会议 action items、归档文件。
- 工作台可以直接跑文件转换、报告生成、CSV 清洗。

### E-commerce Ops Assistant

- 管理订单、退款、物流、库存、客户反馈。
- AI 帮忙查询异常订单、生成退款判断、整理差评原因。
- Form 调 API，Table 存结果，工作台解释和串联操作。

### Support Assistant

- 管理工单、反馈、FAQ、客户对话。
- AI 分类反馈，生成回复草稿，提取 bug 和 feature request。
- 工作台可以按自然语言查询历史问题。

### Finance/Admin Assistant

- 管理发票、报销、供应商、合同、付款记录。
- AI 抽取发票字段，检查异常金额，生成报销摘要。
- Python 批量处理 PDF 和 CSV。

### Recruiting Assistant

- 管理岗位、简历、候选人阶段、面试反馈。
- AI 提取简历要点，生成面试问题，汇总候选人对比。

## 工作台的独特价值

### 1. 它有上下文

普通 ChatGPT 不知道用户的业务表、文件目录、历史操作和应用结构。

工作台知道：

- 当前 app。
- 当前服务树目录。
- 当前函数。
- 当前表格 schema。
- 当前文件。
- 当前用户和团队。
- 历史 PRD、代码和运行结果。

这让它能做更具体的事情。

### 2. 它能执行操作

工作台不是只回答：

- 可以创建 Form。
- 可以创建 Table。
- 可以修改字段。
- 可以运行 Form。
- 可以查询 Table。
- 可以生成文件。
- 可以写代码处理数据。
- 可以 build、修复、验证。

这比“问答助手”更接近业务操作员。

### 3. 它能把一次性需求沉淀成应用

用户第一次说：

> 帮我整理这些案件材料。

工作台可以先用 Python 一次性处理。

如果这个需求重复出现，可以沉淀成：

- Evidence Extractor Form。
- Case Timeline Table。
- Case Summary Generator。

这就是从“临时操作”到“可复用轻应用”的路径。

### 4. 它能持续改造模板

模板不可能完全适配每个行业和团队。工作台可以让用户继续说：

- 给案件增加“争议金额”字段。
- 给证据增加“证明目的”字段。
- 增加一个“生成时间线”的 Form。
- 增加一个“案件阶段统计”的 Chart。

这才是 Kageos 和普通模板市场的差异。

## 这算不算可售卖能力

算，而且可以有几种卖法。

### 1. Vertical Workspace Pack

卖一套行业 workspace：

- 数据模型。
- Form。
- Table。
- Chart。
- 文档模板。
- 示例数据。
- 工作台提示词和操作手册。

例如：

- Legal Case Workspace。
- Agency Client Workspace。
- E-commerce Ops Workspace。
- Support Ops Workspace。

### 2. AI Workspace Seat

Hosted SaaS 按 seat 或 usage 收费：

- 普通用户只能运行应用。
- AI workspace 用户可以让 AI 分析、修改、生成、运行工具。

### 3. Usage-based Compute

文件处理、Python 执行、LLM 调用、OCR、视频处理都消耗资源，可以按额度收费：

- 每月任务次数。
- 每月文件处理 GB。
- 每月 Python runtime 分钟数。
- 每月 LLM token。
- 每月 OCR 页数。

### 4. Self-host Pro

私有化用户可以免费运行核心平台，但高级行业 workspace、AI 操作手册、更新包和支持收费。

## 最大风险：性能和成本

工作台越强，越容易消耗服务器资源。

高消耗来源：

- 大文件上传和转换。
- PDF OCR。
- 视频转码。
- 大 CSV / Excel 处理。
- Python 脚本执行。
- LLM 长文本分析。
- 多用户同时 build。
- 用户反复让 AI 生成、运行、修复代码。

如果 hosted SaaS 不控制，会出现：

- CPU 被视频/PDF/OCR 打满。
- 内存被大文件或 Python 任务吃光。
- 存储被输出文件撑爆。
- LLM 成本不可控。
- 构建任务排队。
- 单个用户影响其他用户。

## MVP 阶段的控制策略

第一阶段不要把工作台包装成无限执行环境。要明确额度和限制。

### 1. 文件限制

- 单文件大小限制。
- 单次上传文件数量限制。
- 输出文件保留时间。
- 工作区存储上限。
- 大文件任务只在付费层开放。

### 2. 执行限制

- Python 执行超时。
- Form 执行超时。
- 每用户并发任务数。
- 每 app 并发任务数。
- 每 workspace 每日任务次数。
- 禁止长期后台任务。

### 3. Build 限制

- 每用户每小时 build 次数。
- build 队列。
- build 超时。
- 免费层限制 AI 修改次数。

### 4. LLM 限制

- 免费层使用小模型或低额度。
- 长文档分析需要付费额度。
- 显示 token/usage。
- 支持用户自带 API key，降低平台成本。

### 5. 隔离策略

- 每个 app 运行在独立进程或容器。
- 对 CPU、内存、磁盘设置限制。
- 工作区文件按租户隔离。
- 高风险工具使用 allowlist。

### 6. 审计和可追溯

- 记录谁运行了哪个 Form。
- 记录文件输入和输出引用。
- 记录 Table 修改。
- 敏感字段脱敏。
- 不把 token、cookie、密钥写进日志。

## 对外不要怎么说

不要说：

> AI 可以替你 on-call 处理一切。

这会引出通知、后台任务、实时监控、SLA、权限和责任问题。

更准确地说：

> An AI workspace that helps teams operate lightweight business apps, analyze files, run actions, and turn repeated work into reusable tools.

中文：

> 一个能帮团队操作轻应用、分析文件、运行任务，并把重复工作沉淀成工具的 AI 工作台。

## 律师助手这种场景的注意事项

法律行业有高敏感性，不能轻易承诺：

- 法律结论正确。
- 替代律师判断。
- 自动出具法律意见。
- 强合规存储。
- 满足所有司法辖区要求。

可以承诺的更稳：

- 案件资料整理。
- 文件解析。
- 时间线生成。
- 文书初稿辅助。
- 材料清单。
- 内部知识检索。
- 人工复核前的结构化草稿。

## 我建议的产品定位

工作台可以作为第二层价值来卖：

第一层：

> Installable lightweight apps.

第二层：

> AI workspace to operate and customize those apps.

第三层：

> Vertical workspaces for specific teams.

这样不会一开始把产品说成万能 Agent，也能解释为什么用户不仅买模板，还愿意为 AI 工作台付费。

## 推荐先做的 Vertical Workspace

不要第一批就做法律行业，因为敏感、合规和专业责任重。

更适合先做：

1. Agency Client Workspace。
2. E-commerce Ops Workspace。
3. Support Feedback Workspace。
4. Finance Admin Workspace。

律师助手可以作为后续高价值方向，但需要更严格的隐私、权限、审计、免责声明和部署方式。

