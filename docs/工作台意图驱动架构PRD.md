# 工作台意图驱动架构 PRD

## 一、背景

当前工作台在创建、修改、测试、排错、临时工具、定时任务等任务之间缺少明确边界。模型经常在一个长会话中同时携带 PRD、旧错误、旧代码、测试记录和新的用户目标，导致上下文噪音越来越大。

典型问题包括：

- 创建应用时没有读取完整 SDK 文档，只靠片段案例和经验生成代码。
- 应用开发、应用测试、构建排错、应用修改混在同一个任务上下文里。
- build 成本高，但模型容易在半成品阶段提前 build。
- 修改任务没有细分，字段改名、新增字段、新增选项、新增回调、新增图表都走同一套粗流程。
- 任务完成后缺少下一步意图引导，用户需要反复重新描述目标。

因此需要把工作台从“长提示词 + 长会话执行”调整为“意图识别 + 专业文档 + 意图上下文切换”的架构。

## 二、目标

建立一个意图驱动的工作台主链路：

```text
用户输入
-> 意图识别
-> 选择一个主意图
-> 读取该意图对应的 SOP / SDK / 平台规则 / 案例文档包
-> 执行任务
-> 输出结果和下一步建议
-> 必要时切换到新意图并压缩上下文
```

目标不是让首屏 system prompt 覆盖所有知识，而是让首屏 prompt 只负责路由和约束。具体知识由意图文档承载，并按任务需要读取。

## 三、设计原则

### 1. 每轮必须有明确意图

模型收到用户输入后，必须先判断当前任务属于哪个意图。没有明确意图时，不能直接写代码、执行函数或调用构建。

### 2. 高风险任务读取完整文档

应用开发类任务必须读取完整 SDK 文档、完整 widget 白名单、构建校验规则和匹配案例，不允许只读取碎片文档后直接生成代码。

### 3. 低风险修改读取专项文档

字段改名、新增字段、修改选项、新增消息通知等低风险修改不需要每次读取全量 SDK，但必须读取对应专项修改文档，明确注意事项和常见错误。

### 4. 完整落盘后统一 build

build 是昂贵动作。创建或较大修改时，应先完整写完本轮涉及的 Go 文件，再通过轻量检查工具排除低级错误，最后统一调用 `build_workspace`。

禁止为了半成品提前 build。

### 5. 意图可切换，上下文可压缩

应用开发完成后可以切换到应用测试意图；测试失败后可以切换到应用修改意图；修改后再切回测试意图。

切换意图时不应携带旧会话全文，只保留结构化 handoff 摘要。

## 四、总体架构

```text
Intent Router
  -> Intent Document Registry
  -> Intent Session Manager
  -> Executor
  -> Cheap Check Tools
  -> Build / Run Verify
  -> Next Intent Planner
```

### 1. Intent Router

负责识别用户输入属于哪个一级意图，并输出结构化判断结果。

示例输出：

```json
{
  "intent": "app.create",
  "confidence": 0.92,
  "reason": "用户要求创建一个 NPS 系统",
  "required_docs": [
    "/system/prompt/intents/app-create",
    "/system/prompt/sdk/agent-app-sdk-readme",
    "/system/prompt/sdk/widget-reference",
    "/system/prompt/sdk/build-validation-reference"
  ],
  "next_action": "read_intent_docs"
}
```

### 2. Intent Document Registry

维护意图和文档包的映射关系。模型不再自行猜测“该读哪个文档”，而是由 registry 返回必读文档、可选文档、案例文档和允许工具。

### 3. Intent Session Manager

每个意图拥有独立任务上下文。切换意图时生成 handoff packet，只保留必要摘要。

示例：

```json
{
  "from_intent": "app.create",
  "to_intent": "app.operate_test",
  "directory": "/liubeiluo/test121212/a/nps",
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
  "recommended_next_steps": [
    "创建测试问卷",
    "提交一次 NPS 评分",
    "查询统计图表"
  ]
}
```

### 4. Executor

根据当前意图执行对应流程。Executor 不直接决定业务知识，业务知识来自意图文档和系统注入的文档包。

### 5. Cheap Check Tools

build 前执行轻量检查，尽量在源码层面发现低级错误。

建议检查项：

- Go 文件是否缺 import 或存在未使用 import。
- widget type 是否在全部 widget 白名单内。
- `options_colors` 是否为不带 `#` 的 6 位十六进制 `RRGGBB`。
- `options_colors` 数量是否和 `options` 一致。
- Table Request 是否和 Model 字段 `json` code 冲突。
- `OnSelectFuzzyMap` 的 key 是否对应 `select` 或 `multiselect` 字段。
- 路由后缀是否和 Template 类型匹配。
- Go 文件名是否错误写成 `.table.go`、`.form.go` 或 `.chart.go`。
- 是否使用未在完整 SDK 文档、案例或源码中确认过的 SDK API。

Cheap check 不替代 build，只用于减少明显低级错误。

### 6. Build / Run Verify

完整落盘和 cheap check 后，统一调用 `build_workspace`。

build 成功后，根据函数类型验证：

- Table：`run_table_search`，必要时 `run_table_create` / `run_table_update`。
- Form：`run_form_submit`。
- Chart：`run_chart_query`。
- OnSelectFuzzy：`run_on_select_fuzzy`。

### 7. Next Intent Planner

每次任务完成后，模型必须给出下一步建议。

示例：

- 应用开发完成：建议进入 `app.operate_test`。
- 测试失败：建议进入 `app.modify` 或 `app.build_fix`。
- 测试通过：建议进入 `schedule.task`、`publish.hub` 或结束任务。

## 五、一级意图清单

| 意图 ID | 名称 | 典型用户输入 | 必读文档 |
| --- | --- | --- | --- |
| `app.create` | 应用开发 | 搞个 NPS 系统、做个拍卖会系统 | 创建 SOP、完整 SDK、全部 widget、build 规则、匹配案例 |
| `app.modify` | 应用修改 | 给工单加字段、改状态选项、新增图表 | 修改 SOP、修改类型专项文档 |
| `app.operate_test` | 应用操作/测试 | 帮我试一下、查表、提交表单 | 操作 SOP、run 工具文档、目标 schema |
| `app.build_fix` | 构建排错 | build 报错了、schema compile failed | build diagnostics、错误分类文档 |
| `temp.task` | 临时杂活 | 转视频、CSV 转 Excel、画临时图 | 工具检索 SOP、system tools 文档 |
| `schedule.task` | 定时任务 | 每天 10 点抓新闻并写入表 | 定时任务 SOP、单次执行函数设计 |
| `platform.openapi` | 平台接口 | 发消息、查权限、Hub、审计 | OpenAPI SOP、对应接口文档 |
| `app.explain_review` | 解释/审查 | 解释这个项目、review 一下 | 解释 SOP、review 规范 |
| `publish.hub` | 发布/复用 | 推到 Hub、复用 Hub 应用 | Hub SOP |

## 六、应用修改二级分类

`app.modify` 必须继续分类，避免所有修改都走同一套大流程。

| 修改类型 | 文档 |
| --- | --- |
| 字段改名 | `/system/prompt/intents/modify/field-rename` |
| 新增字段 | `/system/prompt/intents/modify/add-field` |
| 删除字段 | `/system/prompt/intents/modify/remove-field` |
| 修改 widget | `/system/prompt/intents/modify/widget-change` |
| 新增 select 选项 | `/system/prompt/intents/modify/select-options` |
| 修改搜索条件 | `/system/prompt/intents/modify/search-filter` |
| 新增 OnSelectFuzzy | `/system/prompt/intents/modify/onselect-fuzzy` |
| 新增 Table 回调逻辑 | `/system/prompt/intents/modify/table-callback` |
| 新增消息通知 | `/system/prompt/intents/modify/send-message` |
| 新增 link 跳转 | `/system/prompt/intents/modify/function-link` |
| 新增 Form/Table/Chart | `/system/prompt/intents/modify/add-function` |
| 修改 Chart 指标 | `/system/prompt/intents/modify/chart-metric` |
| 修业务 bug | `/system/prompt/intents/modify/bugfix` |

## 七、核心流程

### 1. 应用开发流程

```text
识别 app.create
-> 读取 create SOP
-> 读取完整 SDK readme
-> 读取全部 widget 文档
-> 读取 build validation 文档
-> 读取匹配案例
-> 输出 PRD
-> 用户确认
-> 完整写完本轮 Go 文件
-> cheap check
-> build_workspace
-> run_* 验证
-> 输出下一步建议：进入 app.operate_test
```

### 2. 应用测试流程

```text
识别 app.operate_test
-> 读取操作 SOP
-> 读取目录 schema / 函数列表
-> 设计测试用例
-> run_table_create / run_form_submit / run_chart_query
-> 发现 bug 切换 app.modify
-> 无 bug 输出验收结果
```

### 3. 构建排错流程

```text
识别 app.build_fix
-> 读取 build diagnostics
-> 解析错误类型
-> 按同类错误批量修复
-> cheap check
-> build_workspace
-> 成功后回到上一意图
```

### 4. 定时任务流程

```text
识别 schedule.task
-> 判断是否已有可执行单次函数
-> 没有则先进入 app.create 或 app.modify 创建单次执行能力
-> 读取定时任务 SOP
-> 创建平台定时任务
-> 验证一次执行记录
```

### 5. 临时杂活流程

```text
识别 temp.task
-> 优先搜索 system tools
-> 找到合适工具后确认 schema
-> 执行一次性任务
-> 返回文件或结果
-> 不创建长期业务应用
```

## 八、需要新增或调整的工具

| 工具 | 作用 |
| --- | --- |
| `classify_intent` | 根据用户输入输出意图、置信度、文档包 |
| `start_intent_session` | 开启新的意图上下文 |
| `handoff_intent` | 从当前意图切换到下一个意图，只传摘要和新文档包建议 |
| 写入后自动诊断 | `write_go_file` / `search_replace_file` 成功后自动返回文件级 Go / SDK / widget 常见问题；非阻断，不作为独立手动工具 |
| `inspect_function_schema` | 查看某个函数 schema、字段、回调、是否支持写入 |
| `summarize_task_state` | 生成当前任务摘要，供切换意图使用 |

当前第一版已落地 `classify_intent`、写入后自动诊断、`summarize_task_state`、`handoff_intent`。诊断能力不作为模型可见主流程工具，而是下沉给 `write_go_file` / `search_replace_file` 自动返回文件级非阻断结果；跨文件/schema 问题以 `build_workspace` 的完整错误为准。`handoff_intent` 不直接删除大模型历史，但会生成下一意图只应携带的极简摘要、目标身份和文档包；真正的会话裁剪可以在后续接入运行时上下文压缩机制。

## 九、文档目录建议

```text
/system/prompt/intents/
  index.md
  app-create.md
  app-modify.md
  app-operate-test.md
  app-build-fix.md
  temp-task.md
  schedule-task.md
  platform-openapi.md
  app-explain-review.md
  publish-hub.md

/system/prompt/intents/modify/
  field-rename.md
  add-field.md
  remove-field.md
  select-options.md
  widget-change.md
  search-filter.md
  onselect-fuzzy.md
  table-callback.md
  send-message.md
  function-link.md
  add-function.md
  chart-metric.md
  bugfix.md

/system/prompt/sdk/
  agent-app-sdk-readme.md
  widget-reference.md
  build-validation-reference.md
  table-form-chart-reference.md
```

## 十、首屏系统提示词职责

新的首屏 system prompt 不应该承载全部 SDK 知识，只需要说明：

1. 工作台是什么。
2. 当前支持哪些一级意图。
3. 每次必须先识别意图。
4. 识别意图后必须读取对应文档包。
5. 不能用未读取文档之外的能力胡编 SDK。
6. 任务完成后必须给出下一步意图建议。

SDK 细节、widget 细节、build 细节、修改专项知识都放到对应意图文档和系统注入的文档包中按意图读取。

## 十一、验收标准

### 当前版本

- 用户说“搞个 NPS 系统”，模型必须识别为 `app.create`。
- `app.create` 必须读取完整 SDK readme，不允许只读碎片案例。
- 写代码前必须 PRD 确认。
- 写代码时不因文件数量提前 build。
- 写入/替换成功后自动返回 cheap check 诊断；build 前按诊断修复。
- build 成功后建议进入 `app.operate_test`。
- 测试失败时能切换到 `app.modify`，并只保留 handoff 摘要。

### 后续版本

- 字段改名、新增选项、新增消息通知能走不同专项文档。
- 定时任务不会和应用开发混在一起。
- 临时 CSV、视频、图片任务不会误判成创建应用。
- 多轮任务上下文明显变短，旧 PRD 和旧错误不再污染新意图。

## 十二、风险和取舍

### 风险

- 意图枚举过细会变成新的维护负担。
- 文档过多但没有 registry 强制约束时，模型仍可能跳读。
- 上下文切换如果摘要质量差，可能丢失关键业务状态。

### 取舍

- 一级意图保持有限，二级任务类型可扩展。
- 高风险任务读完整文档，低风险任务读专项文档。
- 不追求一次性覆盖所有场景，先覆盖应用开发、应用修改、应用测试、构建排错四条主链路。

## 十三、推荐实施顺序

1. 新增 `intent-router` 主文档和一级意图 registry。
2. 改造 `sop.create-project` 为 `app.create` 文档包的一部分。
3. 新增 `app.operate_test` 和 `app.build_fix` 文档。
4. 实现写入后自动诊断，不再暴露独立预检工具作为模型主流程入口。
5. 实现 `handoff_intent` / `summarize_task_state`。
6. 拆 `app.modify` 的二级专项文档。
7. 最后接入运行时上下文裁剪，让意图切换时物理丢弃旧 token，只保留 handoff 摘要。
