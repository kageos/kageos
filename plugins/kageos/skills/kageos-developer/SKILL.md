---
name: kageos-developer
description: 用于规划 kageos 场景包、判断目录与数据库边界，以及在本地开发、修改和验证 kageos 工作空间应用。强调面向中小企业的低部署门槛、MVP、简单优先、案例优先、真正有价值的无人值守和 docs-first 知识闭环。当用户要求设计场景包、拆分或合并目录、设计 PRD、创建 Form/Table/Chart、编辑 namespace/{user}/{app}/code/api 下 SDK package、维护 packageContext/runbook/docs/AgentTask，或运行 gofmt/go test/go build 时使用。
---

# kageos Developer

这个 skill 只做一件事：让 Codex 按 kageos 的平台形态和 SDK 案例开发工作空间应用。

kageos 应用不是自定义 HTML/React 页面生成器。应用能力只能由三种主要模板排列组合出来：

- `FormTemplate`：一次输入，一次执行，一次返回。
- `TableTemplate`：管理一批业务记录，是大多数系统的默认主界面。
- `ChartTemplate`：基于已有数据做标准图表统计。

Docs、runbook、AgentTask、通知和定时任务是辅助能力，不是新的自定义页面形态。

## MVP 硬规则

默认先做最小可用版本，不要先做完整系统。用户不喜欢复杂设计；如果一个需求能用一个清楚的小工具解决，就不要包装成大平台。

每次设计或开发前先问：

```text
用户真正想解决的一个问题是什么？
最少几个字段、几个入口能跑通？
有没有一个更小的名字能描述它？
哪些能力应该明确不做？
```

判断规则：

| 情况 | 应该怎么做 |
|---|---|
| 用户说“合同管理”，但核心是别忘了到期 | 做“合同到期提醒”，不是合同管理系统 |
| 用户说“客户管理”，但核心是下次跟进 | 做“客户跟进提醒”，不是 CRM |
| 用户说“证书管理”，但核心是过期风险 | 做“证书到期提醒/巡检”，不是运维平台 |
| 用户说“收银系统” | 做商品表 + 收银 Form + 支付流水，不做会员/优惠券/财务对账 |
| 用户说“管理后台/系统”但没有明确复杂流程 | 先 1 张主 Table，不上多表/审批/日志/统计图 |

复杂功能必须有明确理由再加。默认不做：审批流、节点表、日志表、通用配置中心、权限模型、统计大盘、AgentTask、多个角色入口、复杂状态机、外部系统集成。某个目录为跑通自身能力所需的一条最小外部服务配置，不算“通用配置中心”，应按下面的配置入口规则实现。

如果用户反馈“太复杂、不好用、看不懂”，立刻收缩到一个主入口和 6-10 个用户能理解的字段，不要解释为什么复杂是必要的。

## 场景包与目录边界硬规则

先区分三个层级：

- **场景包**：面向同一类客户或同一段业务的发现、推荐和组合安装单元；本身不意味着共享数据库或运行时通信。
- **目录**：独立应用、独立业务库和 HTTP API 边界；不同目录不得假设能直接查表、关联记录或共享事务。
- **目录内资源**：同一目录中的 Form/Table/Chart/Docs；只有确实需要共享核心数据、保持一致或共同完成一个闭环的功能才放在一起。

默认优先“多个可独立安装的目录组成一个场景包”。满足任一条件时才合并进同一目录：操作同一批核心记录、一次动作必须更新多个功能、拆开后无法独立闭环，或数据不一致会直接破坏业务。反过来，如果两个能力只服务同一类客户，但各自能在五分钟内独立产生价值，就拆成不同目录，即使它们属于同一个场景包。

不同目录只能通过 HTTP 通信。跨目录 HTTP 默认是可选增强，不得成为目录首次安装即可使用的隐含前置条件。如果核心路径离不开同步跨目录读写，先重新评估是否应该合并目录。确需 HTTP 时，必须定义唯一数据所有者、最小请求载荷、鉴权配置、幂等键、超时与重试、可见失败状态和人工补偿；不得设计跨库事务、全量双向同步或把 token 写入代码、日志、截图、示例数据和 Hub 包。

设计新目录或场景包时，先完整读取 `references/scenario-pack-directory-boundaries.md`，并输出：

1. 场景包服务谁、解决什么共同问题。
2. 包含哪些目录，每个目录的核心业务对象和独立闭环是什么。
3. 每个目录能否独立安装；不能时为什么不合并。
4. 是否共享数据；如需共享，是否应放回同一目录。
5. 是否需要跨目录 HTTP；如果需要，它是核心依赖还是可选增强。
6. 现有 democase 可复用项、新增项和明确不做项。

没有完成这次边界判断，不得开始写 PRD 或代码。

## 外部服务配置入口硬规则

调用模型、短信、邮件、支付或其他第三方 API 时，安装后的用户必须能在 kageos 界面内完成最小配置。不要凭空设计 kageos 当前并不存在的环境变量管理页、部署密钥挂载流程、`/run/secrets/...` 路径或容器运维步骤。只有目标项目已经存在并验证了对应配置机制，而且用户明确要按该机制部署时，才可以使用它。

先按配置数量和管理需求选择一个入口：

| 场景 | 默认设计 | 读取规则 |
|---|---|---|
| 不需要用户配置 | 不建配置入口；官方固定 endpoint、公开 model ID 等写入代码常量 | 直接使用经过核验的常量 |
| 永远只有一套凭证，用户只需保存或更换 | 一个 `config.form` / `settings.form`；提交后在应用自己的表中 singleton upsert | 按固定 singleton key 或唯一记录读取；API Key 留空表示沿用旧值 |
| 用户需要看当前配置、启停、诊断状态，或 Form 无法清楚表达维护过程 | 一个最小 `config.table` | 必须保证“最多一条启用”；读取唯一启用记录 |
| 多账号、多区域、多供应商或需要显式切换 | `config.table`，带名称、启用状态和必要范围字段 | 只读取唯一启用记录；启用一条时事务内停用其他记录 |

禁止把“查询第一条记录”当配置协议。数据库的“第一条”没有稳定业务含义；即使 MVP 只允许一条，也应使用 singleton key、唯一约束，或显式的 `enabled=true` + 确定性排序，并在出现多条启用记录时修复或报错。

简单优先规则：

- 单配置默认优先一个 Form，不要为了保存一条 Key 就让用户理解多行 Table。
- 如果用户已经明确希望通过 Table 管理，或确实需要看启用状态、最近调用状态、错误说明，直接使用一张最小配置 Table，不再叠加第二个配置 Form。
- 配置入口和业务入口最多各一个：例如 `config.form + generate.form`，或 `config.table + generate.form`。
- endpoint 能由官方固定地址确定时不要让用户填写；只有业务空间专属 Host、区域差异或供应商真实要求时才暴露。
- 配置保存成功只表示“已保存”，不能伪装成“调用成功”。连通性测试若会计费或产生外部副作用，必须是用户明确触发的独立动作。

凭证安全规则：

- API Key、token、密码字段不得出现在列表、搜索条件、普通响应、日志、通知、文档、截图、示例数据或 Hub 包中；编辑时不得回填明文，空值表示保留旧值。
- 明文字段应只用于本次提交绑定，例如 `gorm:"-"`、`hide:"list"` 和适用的敏感标记；持久化字段不得通过 JSON 或 widget 暴露。
- SDK 的普通业务字段不自动获得数据库加密。只有存在真实、不可由公开源码或 workspace 元数据推导的密钥来源时，才可以声称“加密保存”。不得使用硬编码常量、用户名、应用名或其他可预测信息派生密钥后宣称安全加密。
- 如果当前平台没有可信的加密密钥来源，而用户仍选择业务表保存，必须如实说明它是受 UI/日志约束的业务库字段，数据库管理员仍可能读取；不要用虚假的加密承诺掩盖边界。
- Hub 导出前必须验证只包含代码、schema 和空表结构，不包含运行态配置行、密文、明文或测试凭证。

运行手册必须只写用户实际可操作的入口。例如真实实现是 `<./config.table>`，runbook 就只说明打开配置表、新增/编辑、启用和测试；不得再写不存在的环境变量、Secret 路径或部署后台。更新已有目录时还要检查线上持久化 `runbook.docs`，因为 manifest 中的文档通常只是首次安装 seed，不会覆盖用户已经维护的运行态文档。

## 面向中小企业的安装价值门禁

默认客户没有专职研发、运维和数据团队。设计 Demo 或 Hub 应用前先证明“安装后马上有用”：

- 优先复用 kageos 内置 Table/Form/Chart/Docs、附件、通知、调度和系统工具，少量业务配置后即可跑通。
- 主路径要清楚展示“有效输入 → 后台持续处理 → 结果回写 → 异常有人接手”，用户必须知道从哪里开始、去哪里看结果。
- 优先做高频、可见、低风险、能直接节省人工盯单和重复查询的内部工作流。
- 不把自建服务、复杂部署、登录态网页抓取、脆弱页面解析或长周期定制集成作为 MVP 成立前提；这些只能是明确标注的增强项。
- 不因为“技术上能做”就做。删除 AI 或后台任务后，如果用户并不需要持续盯、等、查或协调，就不要增加它。

Demo 的五分钟价值路径应尽量完整：安装 → 打开主入口 → 提交或产生一条有效业务记录 → 无人值守给出可见处理结果或转人工 → 人工处理后形成下一次可复用的组织经验。

## 无人值守价值门禁

不要把“能在后台跑”当成“值得无人值守”。设计定时函数、AgentTask、巡检、提醒或后台工作流前，先判断信息在什么时候才可知：

| 信息何时可知 | 正确处理位置 | 例子 |
|---|---|---|
| 用户提交时已经可以确定 | 同步校验并当场反馈；不合格就不进入成功状态 | 必填、格式、附件类型/大小、权限、关联记录是否存在、状态是否合法、确定性重复、金额计算 |
| 校验较慢，但通过校验是业务受理前提 | 明确保存为“草稿/校验中”，通过后才变成“已受理” | 大文件解析、外部实名核验、耗时 OCR；不能先显示成功再事后报错 |
| 时间流逝或外部状态变化后才出现 | 定时函数或事件驱动后台任务 | SLA 临近、无人接单、承诺到期、支付回执、设备恢复/失联 |
| 必须跨记录、跨资源或做语义判断才发现 | AgentTask，结果回写业务工作台或形成明确待办 | 相似工单聚类、跨客户事故识别、历史方案匹配、异常归因 |

无人值守必须产生前台当时无法完成的新增价值，例如持续观察、等待外部变化、跨记录归纳、自动调度或减少人工反复查询。它不能用来补偿糟糕的提交校验、延迟暴露错误，或故意增加“提交—后台检查—退回—重新提交”的返工链路。

设计前必须回答：

1. 为什么这件事不能在用户当前操作里直接完成？
2. 删除后台任务后，是否真的需要某个人持续盯、等、查或协调？如果不需要，不做自动化。
3. 自动执行后，结果回写到哪张主表、哪个状态或哪个待办，而不是只生成一篇没人处理的报告？
4. 谁接手建议或异常，什么状态表示完成，何时停止提醒，如何保证幂等和避免重复打扰？

推荐闭环：

```text
同步校验通过并产生有效记录
→ 时间/外部状态/跨记录条件出现
→ 程序规则或 Agent 无人值守处理
→ 回写状态、建议、证据或内部通知
→ 人在高风险决策点确认
→ 状态更新后自动停止或进入下一周期
```

红线：

- 能当场发现的问题不得留给定时任务或 AgentTask 事后退回。
- 不能为了展示 AI 能力，增加额外的错误表、催收表、复核入口或往返流程。
- Agent 的语义分诊可以在有效记录创建后补充摘要、分类和建议，但不要把有效性校验偷换成后台分诊。
- 异步深度校验必须让用户看到真实状态，不能把“已收到/校验中”伪装成“已成功/已完成”。
- 自动化必须有可见结果、失败处理、幂等、停止条件和人工接管边界。

## 文档优先的能力成长闭环

业务需要让 Agent 随人工经验持续变得更有用时，默认使用 docs-first，不要先造一套重复知识系统：

- `runbook.docs` 用普通业务语言保存稳定规则：怎么开始、系统能做到哪一步、何时转人工、谁接手、怎样通知和结束。
- `docs/readme.docs` 用普通人的语言说明解决方案目录怎么用、怎么新增方案、谁审核和何时启用。
- `docs/*.docs` 保存具体场景的业务事实：什么时候使用、需要哪些信息、怎么处理、系统可以做到哪一步、怎么回复和失败找谁。
- 只有人工审核且正文明确“已启用”的场景文档，才能作为正式回复或自动执行依据；“待确认”和“已停用”必须忽略。
- 场景文档引用 Form/Table/Chart 时遵循“相对路径优先”：当前目录用 `<./xxx.form>`，同一可复制能力包的兄弟目录用 `<../other/xxx.table>`。只有用户明确要求绑定能力包外的其他工作空间，而且无法用稳定相对关系表达时，才允许 `</user/app/...>` 绝对资源标记；必须同时说明该依赖不可移植，复制或安装到其他实例后需要重新绑定。不要为了省事把本可相对引用的资源写成绝对路径。业务人员只需说明该功能的业务用途；Agent 必须自行搜索真实 schema，确认参数来源、权限、风险、幂等和执行后验证，资源标记本身不授权执行。

推荐闭环：

```text
Agent 搜索已启用方案
→ 命中且匹配：回复、查询或执行低风险可验证动作
→ 无可靠方案：记录证据并转人工
→ 人工填写原因、解决方案和验证结果
→ 处理人确认无需沉淀或待沉淀
→ 有维护权限的人创建/更新 docs 并人工启用
→ 后续相似问题复用
```

运行态没有 `write_doc` 时，把“待确认/待沉淀”设计成明确人工接管点，不得声称已经自动写入。只有知识条目确实需要独立生命周期、负责人、结构化筛选、统计或批量治理时才增加 knowledge Table；不要让 Table 和 docs 保存同一份权威内容。

## 业务文档可维护性门禁

默认 `runbook.docs`、`docs/readme.docs` 和 `docs/*.docs` 由不懂 kageos、数据库或 Agent 工具的业务人员长期维护。生成或修改这些文档时，先做“普通员工能否看懂并改对”的检查；不能要求维护者兼职做 Agent 工程师。

业务文档必须写清模型不能替公司决定的内容：

- 什么情况下使用这条规则或解决方案。
- 处理前需要哪些业务信息；缺少时向谁补充或转交。
- 公司认可的处理步骤、回复口径和完成标准。
- 系统可以做到哪一步：只答复、可以查询并答复、可以完成明确的低风险操作，还是必须人工处理。
- 哪些情况必须停止、转人工，以及谁来接手。

业务文档不要写模型可以从真实资源确认的实现细节：schema、JSON 字段名、参数映射表、工具名、分页参数、认领锁、超时时间、重试次数、幂等键、接口返回和数据库写法。这些规则不能删除，应放进 `AgentTask.Message`、代码或测试，由 Agent 执行前自行确认；无法确认时转人工。

推荐 Runbook 结构：

```text
这个工具能帮你做什么
怎么开始
提交后会发生什么
系统可以做到哪一步
什么情况会转人工
人工处理后如何留下解决方案
通知和结束
```

推荐场景文档结构：

```text
状态：待确认 / 已启用 / 已停用
什么时候使用
处理前需要什么信息
怎么处理
系统可以做到哪一步
可以使用的功能（通过 / 插入；没有则写无）
怎么回复
处理失败怎么办
```

技术能力需要体现它造成的业务后果，例如“可以查询但不能修改”“付款和审批必须人工确认”；不解释它底层如何调用工具。若 Form/Table 字段改名后业务文档也必须跟着改，通常说明文档写得太技术化。

同一 package 的子目录文档继续使用原 `packageContext`：

```go
packageContext.AddDocs(app.DocManifest{
    Code: "./docs/readme.docs",
    Name: "文档/目录说明",
    Content: docsReadme,
    Format: "markdown",
})
```

不要为 docs 子目录创建 Go 子包、blank import 或第二个 `PackageContext`。SDK 会在 build/update 时自动补齐中间目录并随 Hub package 分发。

## 任务路由

先判断用户到底要什么，不要所有请求都直接写代码：

| 用户意图 | 应该做什么 |
|---|---|
| “帮我设计/分析/出 PRD” | 不写代码；先读边界、SDK、匹配案例，输出模板组合、路由设计、字段和暂不做 |
| “在 /user/app/... 做/实现/新增” | 定位工作台路径，读取目标模块和匹配案例，再写代码并验证 |
| “这个已有目录怎么用/帮我查/帮我提交” | 先看已有 Form/Table/Chart 是否能完成；能完成就调用或说明使用方式，不要改代码 |
| “改已有功能/修 bug” | 读取现有代码和相邻案例，小范围修改，不重写整个应用 |
| “要漂亮页面/自定义看板/复杂前端” | 说明平台边界：长期应用只能 Form/Table/Chart 组合，不能写自定义 HTML 页面 |

## 案例优先工作流

写代码前必须按这个顺序读材料：

1. `references/boundaries.md`：平台形态、代码落点、不能做什么。
2. `references/sdk-prompt.md`：从工作台搬来的 SDK 主提示词，查 Form/Table/Chart 真实写法。
3. `references/examples/index.md`：选择最接近当前需求的案例。
4. 涉及业务通知、提醒、附件推送时读 `references/notifications.md`。
5. 涉及场景包、目录拆分或跨目录通信时，必须读 `references/scenario-pack-directory-boundaries.md`；涉及场景目录、复杂工作流、AI 后台任务或多表协同时，按需读 `references/solution-design-principles.md`、`references/ai-native-workflow-modeling.md`、`references/workflow-product-quality.md`。
6. 至少读 1 个 `references/case_catalog/` 或 `references/examples/` 下的完整最佳实践案例，再设计方案；组合型需求读多个案例。

不要只读概念就开始写。模型更应该照着完整案例做。

## 开发前案例对齐 SOP

每次开发前都必须先做一次案例对齐，确认“这个需求的结构”和某个最佳实践案例对应上了，再开始写代码。

步骤：

1. 判断需求结构：是单 Table、Form、Form + Table、多 Table、Form + Table + Chart、定时函数，还是 AgentTask。
2. 从 `references/examples/index.md` 选 1 个结构最接近的案例；组合型需求可选 2 个。
3. 读取案例的 `readme.md` 和完整 `prd.md`；需要字段/schema 时再读 `prd.json`。
4. 对照案例提炼写法：它用了哪些 Table/Form/Chart、数据怎么流、哪些表只读、哪些回调负责写入、路由怎么命名。
5. 再输出方案或开始编码。

开始写代码前，必须能写出这段对齐说明：

```text
参考案例：
需求结构：
案例对应关系：
照搬的写法：
需要简化/不照搬：
准备创建的路由：
```

如果说不清“案例对应关系”和“照搬的写法”，说明还没看懂案例，继续读案例，不要开始写。

## 本地 PRD 输出格式

本地 Codex 里给用户看的 PRD 默认用 Markdown 表格，不使用工作台内部的 PRD v2 JSON 契约。PRD v2 的 `project/tables/forms/charts/rules` 只用于 kageos 工作台 `write_prd` artifact 或明确要求结构化 PRD 的场景。

本地输出 PRD 时，必须让用户一眼看懂系统怎么用：

1. 先给“参考案例”和“模板组合”。
2. 再给路由表，列出每个 `.table` / `.form` / `.chart` 的用途。
3. 每个 Table/Form/Chart 都用 Markdown 表格说明字段。
4. Table 必须给 1-3 行示例数据。
5. Form 必须给示例请求和示例返回。
6. Chart 必须给统计口径和示例数据点。
7. 最后列“暂不做”，避免模型把增强项塞进 MVP。

推荐结构：

```text
参考案例
模板组合
路由设计
Table 字段与示例数据
Form 入参与返回示例
Chart 口径与示例数据
自动化/通知策略
暂不做
```

不要在本地 PRD 里默认输出大段 JSON。用户要的是可读方案，不是工作台内部 artifact。

## 需求分析流程

用户说“做一个系统”时，不要直接建很多表。先把需求翻译成 kageos 的三种模板组合：

1. 先改小名字：把“XX 管理系统”改成一个更具体的小工具名，例如“合同到期提醒”“客户跟进提醒”“证书到期巡检”。
2. 找主入口：用户每天最常打开的是哪张表？如果说不清，先不要加多入口。
3. 找最少字段：新增一条记录时，6-10 个字段能不能跑通？能跑通就不要继续加。
4. 找一次性动作：哪些动作是“填一次参数，执行一次，返回结果”？这些才是 Form。
5. 找统计视角：只有用户明确要趋势、分布、排行时才加 Chart。

只问必要问题。目标目录、核心字段、外部系统凭证这类缺了会写错的，才问用户；能按常识给最小版本的，就先给最小版本。

输出方案或写代码前，先说清“暂不做”。暂不做不是缺陷，是 MVP 的一部分。

## 模板路由

按需求选择模板：

| 用户说法 | 优先设计 | 推荐案例 |
|---|---|---|
| “做一个管理系统/台账/后台” | 先把名字收小，再做 1 个主 `TableTemplate` | `references/case_catalog/table/ticket` |
| “合同管理/合同系统” | 默认先做“合同到期提醒”：合同清单 + 到期扫描 Form | `references/case_catalog/table/ticket`、`references/examples/scheduled-function-case.md` |
| “收银系统/库存销售/经营统计” | 商品 Table + 收银 Form + 支付记录 Table，统计 Chart 可选 | `references/case_catalog/form_table_chart/cashier` |
| “预约/会议室/资源占用” | 资源 Table + 预约 Table + 查询空闲 Form + 提醒 Form | `references/case_catalog/tables/meeting` |
| “投票/问卷/报名” | 提交 Form + 结果 Table | `references/case_catalog/formandtable/vote` |
| “上传文件处理” | 文件处理 Form | `references/case_catalog/form/excelorcsv`、`form/pdf`、`form/images` |
| “接一个模型/API，只配置一个 Key” | `config.form` singleton upsert + 业务 Form；需要查看状态时可改为最小配置 Table | 本 Skill“外部服务配置入口硬规则” |
| “管理多个模型账号/区域/供应商” | 配置 Table + 业务 Form；只允许一条启用配置 | 本 Skill“外部服务配置入口硬规则” |
| “看趋势/排行/分布” | 一个统计视角一个 `.chart` | `references/examples/chart-case.md` |
| “每天自动提醒/巡检” | 先判断用定时函数还是 AgentTask，默认优先定时函数 | `references/examples/scheduled-function-case.md`、`references/examples/agent-session-runbook-case.md` |
| “漂亮页面/自定义页面/看板布局” | 解释平台不能自定义 HTML 页面，只能 Form/Table/Chart 组合 | `references/boundaries.md` |

## 定时函数 vs AgentTask

遇到“每天、每小时、自动、定时、巡检、提醒、日报”时，先判断任务是不是确定性函数。

| 选择 | 适合场景 | 不适合场景 |
|---|---|---|
| 定时函数：`FormTemplate.Schedules` | 固定参数、固定逻辑、可由 Go 函数确定执行；例如扫描到期合同、证书续期检查、会议开始前提醒、每天汇总表内数据 | 需要临场判断、搜索资料、读多个目录、写长报告、根据情况选择工具 |
| AgentTask：`packageContext.AddAgentTask` | 需要工作台 Agent 无人值守地阅读上下文、跨资源查询、调用多个工具、分析总结、生成 Markdown 报告或长期维护知识；例如每日行业情报、跨表异常归因、竞品监控日报 | 简单扫描、固定通知、固定表单提交、能用一个 Form 函数稳定完成的任务 |

默认选定时函数。只有满足以下任一条件，才升级 AgentTask：

- 任务需要模型判断“查什么、选哪些、怎么总结”。
- 任务需要跨多个 Form/Table/Chart、外部网页或连接器组合执行。
- 输出是自然语言报告、分析结论、行动建议，而不是固定结构响应。
- 需要读取 `runbook.docs`、维护长期上下文或按复杂 SOP 无人值守运行。

反例：合同到期提醒、证书到期扫描、库存低于阈值通知，默认都是定时函数，不要上 AgentTask。

AgentTask 成本更高、可控性更弱，安装后默认应谨慎开启；固定业务逻辑优先写 Go Form，再用 schedule 跑。

## 路由和项目组织

一个 kageos 目录就是一个可组合能力包。先定包，再定资源：

```text
工作台路径：/<user>/<app>/<package>
代码目录：namespace/<user>/<app>/code/api/<package>
RouterGroup："/<package>"
资源路由：products.table / checkout.form / payments.table / sales_trend.chart
```

路由命名规则：

| 类型 | 命名 | 示例 |
|---|---|---|
| Table | 名词或名词复数 + `.table` | `products.table`、`payments.table`、`contracts.table` |
| Form | 动词或动作名 + `.form` | `checkout.form`、`query_available.form`、`extract_text.form` |
| Chart | 指标或视角 + `.chart` | `sales_trend.chart`、`status_distribution.chart` |

文件组织建议：

| 文件 | 内容 |
|---|---|
| `init_.go` | `packageContext` 和路由注册 |
| `models.go` | GORM Model、常量、TableName |
| `tables.go` | Table 请求、列表查询、Table 回调 |
| `forms.go` | Form 请求、响应、处理函数 |
| `charts.go` | Chart 请求、聚合查询、图表响应 |
| `callbacks.go` | `OnSelectFuzzy`、联动选择 |
| `kageos_manifest.go` | runbook/docs/AgentTask 种子，只有需要时加 |

不要把所有东西写进一个超大文件；也不要为一个很小的单表需求拆太多文件。

## 示例：收银系统怎么拆

用户说“我要一个收银系统”时，先读 `references/case_catalog/form_table_chart/cashier`，然后按最小可用版本拆：

| 能力 | 模板 | 路由 | 说明 |
|---|---|---|---|
| 商品管理 | Table | `products.table` | 商品名称、分类、单价、库存、上架状态 |
| 收银结账 | Form | `checkout.form` | 选择商品和数量，选择支付方式，提交后生成支付记录并扣库存 |
| 支付记录 | Table | `payments.table` | 只读流水，按订单号、支付状态、时间筛选 |
| 销售趋势 | Chart | `sales_trend.chart` | 可选增强，基于支付记录统计销售额和订单数 |

不要一上来做会员体系、优惠券、退货、班次、权限、财务对账、复杂库存盘点。除非用户明确要，先把收银闭环做通。

输出方案时先写：

```text
参考案例：
需求拆解：
最小版本：
模板组合：
主工作台：
路由设计：
字段草案：
示例数据：
暂不做：
```

## 简单优先

任何系统都有简单设计。默认先做用户愿意每天打开的一张主 Table：

- 能用 1 个 Table 解决的，不要拆成多个入口。
- 能用 Table 的新增/编辑/状态字段解决的，不要新建 Form。
- 能用表格筛选解决的，不要新建“今日待办”“队列”“日志”。
- Chart、AgentTask、runbook、自动化、复杂审批是增强，不是默认 MVP。
- 只有明确出现独立生命周期、独立历史、多次明细、跨对象统计、外部动作参数或多角色协同时，才升级多表/多入口。
- 字段默认控制在 6-10 个用户能理解的字段；系统字段、运行标记、通知次数、冷却时间这类内部信息要隐藏，除非用户确实需要操作。
- 名字要小：能叫“到期提醒”就不要叫“管理系统”；能叫“跟进清单”就不要叫“CRM”。

如果用户说“太复杂/不好用/怎么用”，立刻降级为一张主 Table 和少量字段。

## 路径和代码落点

用户说的 `/<user>/<app>/<package...>` 是 kageos 工作台路径，不是真实磁盘路径：

```text
/<user>/<app>/<package...>
=> <kageos_repo_root>/namespace/<user>/<app>/code/api/<package...>
```

规则：

- 如果用户只给 `/<user>/<app>`，先查 `namespace/<user>/<app>/code/api/`，复用最贴近的 package；没有再新建清晰的业务 package。
- 每个 package 至少有 `init_.go` 声明 `packageContext`。
- 新增 package 后必须在 `namespace/<user>/<app>/code/cmd/app/main.go` 添加 blank import。
- 业务代码只写 `namespace/<user>/<app>/code/api/<package...>/*.go`，必要时更新该应用模块的 `main.go`、`go.mod`、`go.sum`。
- 不要把业务代码写到真实磁盘 `/system/...`、kageos 主仓库根目录、`core/...`、`kageos-sdk/...`、skill 目录或无关 workspace。

## 实现要求

- 每个业务目录用 `packageContext = &app.PackageContext{RouterGroup: "/...", Name: "...", Desc: "..."}`。
- 每个函数必须通过 `packageContext.GET/POST(..., handler, template)` 注册。
- Table 必须显式配置 `AutoCrudTable`；不需要新增/编辑/删除时，不配置对应回调。
- Form 的 Request/Response 是普通结构体，不要把 Chart 结构塞进 Form。
- Chart 一个路由只返回一张图，用 `resp.Chart(chart).Build()`。
- 列表里不要暴露裸内部 ID；用业务名称、展示字段、`OnSelectFuzzy` 和后端补全。
- 通知用 `ctx.SendNotification`，不要业务代码硬连钉钉、企微、飞书、邮件渠道；`files` 组件值可以直接传给 `SendNotificationOpts.Files` 作为附件。
- 平台已有操作日志，不要默认自造通用日志表。
- 子目录文档种子使用原 `packageContext.AddDocs` 的相对多级 `Code`；不要为文档目录声明新的 `PackageContext`。
- 场景知识默认使用 `runbook.docs + docs/*.docs`；没有独立结构化生命周期时，不要再增加保存相同内容的 knowledge Table。
- 当前不要主动使用部门能力：不要在新应用里生成 `department` / `departments` 组件、所属部门字段或 `ctx.GetRequestUserDept()` 逻辑。部门相关能力还没完善，除非用户明确要求或正在维护已有部门代码，否则即使案例里出现也不要照搬。
- 外部服务配置必须遵守“外部服务配置入口硬规则”：单配置用 singleton Form 或最小 Table，多配置只读取唯一启用项；不得无依据发明环境变量或 `/run/secrets` 配置流程。
- 配置读取必须有确定性。不得裸用无条件 `First()` 代表“默认配置”；应按 singleton key 或唯一启用条件查询，并处理不存在、重复启用、停用和更新 Key 留空等情况。
- runbook、Form/Table 描述和真实实现必须一致。涉及配置方式变更时，除修改 `kageos_manifest.go` seed 外，还要检查并按授权更新运行态持久化文档。

## 验证

在目标应用模块根目录运行：

```bash
gofmt -w code/api/<package...>/*.go
go test ./...
go build ./code/cmd/app
```

也可运行：

```bash
scripts/verify_workspace_app.sh <full_code_path> [repo_root]
```

写文件或本地修改不会让平台自动生效；要在平台生效必须执行真正的 build/update。

外部服务配置类目录还必须验证：

1. 从全新安装视角，用户能只通过真实可见的 Form/Table 完成配置；不需要猜环境变量、容器路径或后台运维入口。
2. 未配置、重复启用、Key 留空更新、禁用和错误 Host 都有明确结果。
3. Table 列表、接口响应、操作日志、runbook、Hub 包和测试夹具均不包含真实或看起来像真实的凭证。
4. 如果文档 seed 已经存在于运行态，确认页面上的持久化文档也已更新；不能只看源码常量。
5. “保存配置”和“外部调用成功”分开验证；会计费的连通性测试不得随保存自动触发。

## 参考资料

- 平台边界: `references/boundaries.md`
- SDK 主提示词: `references/sdk-prompt.md`
- 平台能力边界原文: `references/platform-capability-boundaries.md`
- 通知 SOP: `references/notifications.md`
- 完整平台案例目录: `references/case_catalog/readme.md`
- 平台案例目录原文: `references/platform-case-catalog.md`
- 案例索引: `references/examples/index.md`
- 场景解决方案设计原则: `references/solution-design-principles.md`
- AI 原生工作空间应用建模: `references/ai-native-workflow-modeling.md`
- 工作流产品质量规则: `references/workflow-product-quality.md`
- SDK 专项细节: `references/sdk-patterns.md`
- packageContext 自动补目录: `references/package-sync.md`
- 构建和验证: `references/build-and-verify.md`
- manifest/runbook/AgentTask: `references/kageos-manifest-runbook-agenttask.md`
