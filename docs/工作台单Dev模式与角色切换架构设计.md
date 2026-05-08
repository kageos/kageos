# 工作台单 Dev 模式与角色切换架构设计

## 一、背景

当前工作台的提示词、SOP 文档包、SDK 文档、案例文档和执行工具之间存在重复链路：

- 多个工作台模式同时存在：`dev`、`agent`、`modify`、`execute`、`qa`。
- `dev / agent / modify` 还会默认拼接很长的 all-in-one 系统提示词。
- 创建应用时既有 `sop.create-project`，又有 `sdk.combo-table-form-chart` 等任务包，文档容易重复注入。
- SDK 写法散落在多个文档里，模型容易只读片段，不读完整 SDK 主文档。
- 长会话里混合开发、测试、build 排错、修改和操作记录，旧上下文持续污染新任务。

新架构目标是把工作台从“多模式 + 多碎片文档 + 大提示词”收敛为：

```text
单 dev 模式 + 角色切换 + 完整 SDK 主文档 + help 辅助
```

## 二、核心结论

1. 工作台只保留一个 `dev` 模式。
2. `dev/system_prompt.md` 只负责环境说明、意图列表、角色路由和少量硬约束。
3. 不再让模型手动寻找流程入口，改为调用 `change_role`。
4. `change_role` 负责切换角色，并自动返回该角色需要的文档、案例和执行规则。
5. SDK 开发角色必须读取完整 SDK 主文档：

```text
/Users/beiluo/Documents/work/code/gitee.com/ai-agent-os/core/agent-server/prompt/system/prompt/sdk/agent-app-sdk-readme.md
```

对应工作台路径：

```text
/system/prompt/sdk/agent-app-sdk-readme
```

6. 其它 SDK 碎片文档后续要么合并进 SDK 主文档，要么降级为 FAQ/help 资料，要么删除。

## 三、新链路

```text
用户输入
  -> dev/system_prompt.md 判断意图
  -> 调用 change_role(role_id, task_summary, carry_context, reset_context)
  -> change_role 自动返回角色说明 + 必读文档 + 推荐案例 + 下一步流程
  -> 模型按角色执行任务
  -> 阶段完成后输出结果和下一步建议
  -> 需要进入新阶段时再次 change_role
  -> 卡住时调用 help(query, error, role_id)
```

例子：

```text
用户：帮我搞个 NPS 系统
  -> change_role("app_developer")
  -> 返回：开发角色 SOP + 完整 SDK 主文档 + NPS/投票案例建议
  -> 输出 PRD
  -> 用户确认
  -> 开发代码
  -> build 成功
  -> change_role("app_tester", carry_context=已生成路由和测试建议)
  -> 测试工程师执行 run_table / run_form / run_chart
  -> 测试失败则 change_role("build_doctor" 或 "app_modifier")
```

## 四、dev/system_prompt.md 的职责

`dev/system_prompt.md` 不承载 SDK 细节，只保留这些内容：

1. 当前工作台是什么。
2. 用户输入后必须先判断意图。
3. 意图和角色映射表。
4. 进入具体任务前调用 `change_role`。
5. 如果卡住，调用 `help`。
6. 不要写独立前端页面。
7. 开发完成后建议切换到测试角色。

不应该放进 system prompt 的内容：

- Form/Table/Chart 详细代码。
- widget 组件大全。
- 大量 build 错误说明。
- 大量 import 规则。
- 大量案例代码。

这些都应该放到 SDK 主文档、角色文档、案例和 help。

## 五、角色设计

### 1. `app_developer`：应用开发工程师

使用场景：

- 创建新系统。
- 新建目录。
- 新建 Form / Table / Chart。
- 从零开发管理后台。

自动返回文档：

- `/system/prompt/roles/app-developer`
- `/system/prompt/sdk/agent-app-sdk-readme`
- 1 个或多个匹配案例：
  - NPS/投票/评价：`/system/prompt/case_catalog/formandtable/vote`
  - 收银/库存/流水：`/system/prompt/case_catalog/form_table_chart/cashier`
  - 单表 CRUD：`/system/prompt/case_catalog/table/ticket`
  - 多表管理：`/system/prompt/case_catalog/tables/meeting`

核心流程：

```text
读当前目录
-> 选择 Form/Table/Chart 拆分
-> 输出 PRD
-> 用户确认
-> 完整落盘所有本轮文件
-> 轻量诊断
-> 统一 build_workspace
-> build 成功后 change_role("app_tester")
```

关键规则：

- 开发角色必须读取完整 SDK 主文档，不读碎片 SDK 文档代替。
- SDK 写法以 `agent-app-sdk-readme.md` 为准。
- 业务最佳实践以匹配案例为准。

### 2. `app_modifier`：应用修改工程师

使用场景：

- 字段改名。
- 新增字段。
- 删除字段。
- 修改 widget。
- 新增 select 选项。
- 新增 OnSelectFuzzy。
- 新增 Table 回调逻辑。
- 新增消息通知。
- 新增 link。
- 新增 Form/Table/Chart。
- 修改 Chart 指标。
- 修业务 bug。

自动返回文档：

- `/system/prompt/roles/app-modifier`
- `/system/prompt/sdk/agent-app-sdk-readme`
- 目标修改类型 FAQ。
- 必要时返回匹配案例。

核心流程：

```text
读目标文件
-> 判断修改类型
-> 输出修改方案和影响范围
-> 小改 search_replace_file，大改 write_go_file
-> 轻量诊断
-> build_workspace
-> change_role("app_tester")
```

修改类型建议：

| 修改类型 | 说明 |
| --- | --- |
| `field_rename` | 字段名、中文名、json code、数据库列名调整 |
| `field_add` | 新增业务字段、系统字段、计算字段 |
| `field_remove` | 删除字段，处理数据库兼容和引用 |
| `select_options` | 新增/修改静态下拉选项和颜色 |
| `onselect_fuzzy` | 新增动态下拉选择 |
| `table_callback` | 新增新增/编辑/删除回调逻辑 |
| `send_message` | 在业务节点发消息 |
| `function_link` | 表格 link 跳转其他函数 |
| `add_function` | 新增 Form/Table/Chart |
| `chart_metric` | 修改统计口径或图表类型 |
| `bugfix` | 修业务逻辑、schema、运行错误 |

### 3. `app_tester`：应用测试工程师

使用场景：

- 开发完成后验证。
- 用户要求“试一下”“跑一下”“提交一下”“查一下”。
- build 成功后的核心路径测试。

自动返回文档：

- `/system/prompt/roles/app-tester`
- 运行工具说明。
- 当前目录 schema / 函数列表。

核心流程：

```text
读取目录和函数
-> 确认 schema
-> 准备测试数据
-> run_table_search / run_table_create / run_form_submit / run_chart_query
-> 区分业务错误和代码错误
-> 输出测试结论
```

业务错误处理原则：

- “问卷状态为待发布，无法提交评价”不是工具问题，也不是必须重新读执行文档。
- 测试工程师应创建或更新一条满足业务规则的数据，再继续测。

### 4. `build_doctor`：构建排错工程师

使用场景：

- `build_workspace` 失败。
- Go 编译失败。
- schema compile failed。
- widget 校验失败。
- callback / OnSelectFuzzy 配置失败。

自动返回文档：

- `/system/prompt/roles/build-doctor`
- `/system/prompt/sdk/agent-app-sdk-readme`
- build FAQ。
- 必要时返回多个案例，让模型对照最佳实践。

核心流程：

```text
读取完整 build 错误
-> 按错误类型归类
-> 批量修同类错误
-> 不要盲目重写整文件
-> build_workspace
-> 成功后 change_role("app_tester")
```

重要原则：

- build 卡住时，最佳方案不是猜 API，而是读取更多匹配案例对照写法。
- import 错误优先按当前文件实际符号修，不要复制常用 import。

### 5. `temp_worker`：临时任务工程师

使用场景：

- 视频转换。
- 图片处理。
- CSV / Excel 转换。
- 临时数据图表。
- PDF / OCR / 压缩。
- 一次性脚本。

自动返回文档：

- `/system/prompt/roles/temp-worker`
- system tools 说明。
- 必要时返回官方 Python runtime 案例。

原则：

- 临时处理优先复用 `/system/tools`。
- 不要把一次性任务误判成需要新建长期应用。

### 6. `scheduler`：定时任务工程师

使用场景：

- 每天 10 点执行任务。
- 定时抓新闻并写入应用。
- 周期生成分析。
- 查询/取消定时任务。

自动返回文档：

- `/system/prompt/roles/scheduler`
- 定时任务 OpenAPI 文档。
- 单次函数设计说明。

原则：

- 定时任务本身只负责调度。
- 具体业务动作应先做成一个可单次执行的 Form/Table/Chart 或平台函数。

### 7. `platform_operator`：平台能力工程师

使用场景：

- 发消息。
- Hub 搜索/发布/复制。
- 权限查询。
- 审计/操作日志。
- 平台 OpenAPI。

自动返回文档：

- `/system/prompt/roles/platform-operator`
- 平台 OpenAPI 主文档。
- 对应接口 FAQ。

### 8. `explainer`：解释和 Review 工程师

使用场景：

- 解释项目。
- 解释某个函数。
- 代码 review。
- 说明系统能力。

自动返回文档：

- `/system/prompt/roles/explainer`
- 代码阅读和 review 规范。

## 六、change_role 工具设计

### 入参

```json
{
  "role_id": "app_developer",
  "task_summary": "开发 NPS 系统，包含问卷管理、评价提交、记录查询和统计图表",
  "target_directory": "/liubeiluo/test11/nps",
  "carry_context": "用户已确认要开发完整系统",
  "reset_context": false
}
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `role_id` | 目标角色 ID |
| `task_summary` | 当前任务摘要 |
| `target_directory` | 当前目录或目标目录 |
| `carry_context` | 从上一阶段携带的必要上下文 |
| `reset_context` | 是否建议丢弃旧上下文，仅保留摘要 |

### 出参

```json
{
  "role_id": "app_developer",
  "role_name": "应用开发工程师",
  "instructions": "...角色 SOP...",
  "docs": [
    {
      "path": "/system/prompt/sdk/agent-app-sdk-readme",
      "content": "...完整 SDK 主文档..."
    }
  ],
  "recommended_cases": [
    {
      "path": "/system/prompt/case_catalog/formandtable/vote",
      "reason": "NPS/评价/投票类系统参考"
    }
  ],
  "next_steps": [
    "读取当前目录",
    "输出 PRD",
    "用户确认后完整落盘代码",
    "统一 build_workspace",
    "切换到 app_tester"
  ]
}
```

### 设计要点

- `change_role` 替代主链路里的手动流程入口。
- 角色文档和 required docs 由工具自动返回，不靠模型记忆。
- 角色切换是阶段边界，不是每一步都切。
- 默认 `reset_context=false`，但工具要支持 `reset_context=true`，用于开发完成转测试、测试失败转修改等场景。

## 七、help 工具设计

`help` 是卡点查询工具，不是主流程工具。

### 入参

```json
{
  "role_id": "build_doctor",
  "query": "OnSelectFuzzyMap 报错怎么修",
  "error": "widget select requires options or OnSelectFuzzyMap entry"
}
```

### 出参

```json
{
  "answer": "select 字段必须有静态 options，或 callback:\"OnSelectFuzzy\" + Template.BaseConfig.OnSelectFuzzyMap。",
  "examples": [
    "字段 json:\"questionnaire_id\" 时，OnSelectFuzzyMap key 也必须是 questionnaire_id"
  ],
  "recommended_docs": [
    "/system/prompt/sdk/agent-app-sdk-readme",
    "/system/prompt/case_catalog/tables/meeting"
  ]
}
```

### FAQ 初始清单

| 问题 | 返回内容 |
| --- | --- |
| 不知道怎么发消息 | 指向 SDK 主文档 SendMessage 章节 |
| 编译失败不知道怎么修 | 建议读取完整错误、匹配案例、按同类错误批量修 |
| import 乱了 | 按当前文件真实符号修，不复制常用 import |
| OnSelectFuzzy 报错 | 解释 select/callback/OnSelectFuzzyMap key 的对应关系 |
| Table 不显示新增按钮 | 解释不配置 `OnTableAddRow` 就没有新增 |
| Table 不显示编辑按钮 | 解释不配置 `OnTableUpdateRow` 就没有编辑 |
| 记录表该不该新增/编辑 | 事实记录、流水、提交记录默认只读 |
| `types.Time` 怎么用 | 字段用 `types.Time`，比较/格式化用 `t.Time()` |
| `Build()` 前后处理 | Build 前拼 queryDB，Build 后填计算字段/link |
| 提交表单业务错误 | 区分业务规则错误和代码错误 |

## 八、SDK 文档策略

### 1. SDK 主文档唯一化

应用开发和修改必须以这个文件为主：

```text
core/agent-server/prompt/system/prompt/sdk/agent-app-sdk-readme.md
```

它应该成为完整 SDK 文档，包含：

- Agent-App SDK 是什么。
- Form / Table / Chart 心智模型。
- struct tag 和 widget 规则。
- Form 标准骨架。
- Table 标准骨架。
- Chart 标准骨架。
- OnSelectFuzzy。
- Table 回调。
- `ChangedFields` / `BindChangedFields`。
- `ctx.GetRequestUser()` / `ctx.GetRequestUserDept()` / `ctx.SendMessage()`。
- `resp.Table(...).Build()` 前后处理。
- 常见 build/schema 错误。
- 代表性代码示例。

### 2. 其它 SDK 文档的处理

后续可逐步做减法：

| 当前文档 | 建议 |
| --- | --- |
| `widget-reference.md` | 合并进 SDK 主文档，或作为附录 |
| `widget-system.md` | 合并进 SDK 主文档后删除 |
| `form-submit-basic.md` | 合并核心内容进 SDK 主文档，保留案例即可 |
| `table-crud-basic.md` | 合并核心内容进 SDK 主文档 |
| `combo-table-form.md` | 合并成 SDK 主文档里的组合模式章节 |
| `combo-table-form-chart.md` | 合并成 SDK 主文档里的组合模式章节 |
| `build-validation-reference.md` | 合并核心内容进 SDK 主文档和 help FAQ |
| `form-table-chart-reference.md` | 如果 SDK 主文档足够完整，可以删除 |
| `workbench-all-in-one-system-prompt.md` | 不再默认注入；仅作为历史备份或删除 |

### 3. 案例仍然保留

案例不是碎片文档，案例是最佳实践代码，应保留。

推荐保留：

- `case_catalog/table/ticket`
- `case_catalog/tables/meeting`
- `case_catalog/formandtable/vote`
- `case_catalog/form_table_chart/cashier`
- `case_catalog/form/excelorcsv`

## 九、上下文切换策略

### 默认不丢弃上下文

`change_role` 默认 `reset_context=false`，避免模型丢失用户刚说的核心需求。

### 阶段边界建议丢弃旧上下文

这些场景建议 `reset_context=true`：

- 开发完成，进入测试。
- 测试失败，进入构建排错或修改。
- 修改完成，重新进入测试。
- 临时任务完成，转定时任务。

丢弃上下文时只保留 handoff 摘要：

```json
{
  "directory": "/liubeiluo/test11/nps",
  "routes": [
    "nps_questionnaire_list.table",
    "nps_submit.form",
    "nps_record_list.table",
    "nps_statistics.chart"
  ],
  "files": [
    "nps_questionnaire_list.go",
    "nps_submit.go",
    "nps_record_list.go",
    "nps_statistics.go"
  ],
  "last_build": "success",
  "known_issues": [],
  "next_role": "app_tester"
}
```

## 十、落地步骤

### 第一阶段：文档与角色设计

- 完成本设计文档。
- 优化 `agent-app-sdk-readme.md`，让它成为完整 SDK 主文档。
- 设计 role 文档结构。
- 设计 help FAQ 数据结构。

### 第二阶段：工具实现

- 新增 `change_role` 工具。
- 新增 `help` 工具。
- 只保留 `dev` 模式入口。
- `change_role` 自动返回角色文档和 SDK 主文档。

### 第三阶段：旧链路下线

- 停止默认拼接 `workbench-all-in-one-system-prompt`。
- 删除旧的手动流程入口。
- 删除或合并碎片 SDK 文档。
- 移除 `classify_intent` 主流程入口。

### 第四阶段：验证

用同一类任务反复压测：

- “搞个 NPS 系统”
- “给 NPS 系统加一个满意度标签字段”
- “测试一下 NPS 系统”
- “build 失败了帮我修”
- “每天 10 点生成 NPS 汇总”

验收标准：

- 模型不会跳过角色切换。
- 开发角色必读完整 SDK 主文档。
- 不再复制常用 import。
- 不再半成品 build。
- 开发完成后能自然切换到测试角色。
- 测试失败能自然切换到修改或排错角色。

## 十一、最终目标

工作台的长期形态应该是：

```text
一个 dev 模式
少量清晰角色
一个完整 SDK 主文档
少量高质量案例
一个 help 工具
一个 change_role 工具
```

系统提示词越短越好，SDK 文档越完整越好，案例越代表性越好。
