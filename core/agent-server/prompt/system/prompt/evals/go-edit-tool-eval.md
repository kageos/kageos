# Go 编辑工具全量压测用例

本用例用于评估工作台模型在 Go 代码生成和后续编辑中的工具使用能力，重点比较：

- `edit_file.search_edits`
- `edit_file.line_edits`
- `write_file`

评测时可见的代码读写工具应只保留 `read_file`、`edit_file`、`write_file` 这一组；如果出现额外代码编辑入口或 `edit_file` 暴露第三种编辑模式，记为工具面污染。

## 目标

让模型先生成一个真实可构建的 Kageos Go 应用，再在同一个应用里对三份等价代码文件执行同构编辑任务，记录每种编辑方式在以下维度上的表现：

- 工具调用是否成功
- 文件内容是否真的符合预期
- Go 语法诊断是否干净
- 最终 build 是否通过
- 是否出现错误修改、重复输出、参数膨胀或崩溃
- 出错后模型是否能正确归因，而不是盲目重试

## 测试目录

建议使用全新应用，避免污染已有业务：

```text
/system/eval/go_edit_tool_lab
```

如果该目录已存在，换成带日期或序号的新目录，例如：

```text
/system/eval/go_edit_tool_lab_20260703_01
```

## 阶段 1：生成基准 Go 应用

让工作台模型按正常产品经理和开发流程创建一个“工单与库存协同处理系统”。应用至少包含：

- 工单表：标题、类型、状态、负责人、优先级、描述、创建时间
- 库存表：物料名称、SKU、仓库、可用数量、安全库存、状态
- 审计记录：动作、对象类型、对象 ID、操作人、备注、时间
- 一个查询或统计能力：按状态、类型、负责人聚合

开发完成后必须生成并保留三个等价测试文件，文件名建议如下：

- `edit_search.go`
- `edit_line.go`
- `edit_write.go`

三个文件必须满足：

- 都在同一个业务 package 下，且能参与正常 build。
- 每个文件至少 140 行。
- 三个文件业务结构等价，但类型名、函数名、路由名要带不同前缀，避免 Go 重名。
- 每个文件都要包含清晰锚点注释，便于读回和定位。
- 不要修改 `init_.go`，除非执行安全负向用例。
- 基准文件可以包含 `Priority` 字段，但不得提前包含 `Channel`、`RiskLevel`、`risk_level` 列；这些字段留给后续测试新增。

锚点注释必须包含下面这些名字，每个文件都要有一份，前缀按文件调整：

```go
// EVAL_ANCHOR: imports
// EVAL_ANCHOR: status-constants
// EVAL_ANCHOR: work-order-model
// EVAL_ANCHOR: create-request
// EVAL_ANCHOR: route-registration
// EVAL_ANCHOR: normalize-status
// EVAL_ANCHOR: table-columns
// EVAL_ANCHOR: debug-delete-target
// EVAL_ANCHOR: validation-block
// EVAL_ANCHOR: helper-insertion-point
```

阶段 1 完成后必须执行：

1. `read_file` 读取三个测试文件，保存 `content_sha`。
2. 运行构建或平台提供的 build 验证。
3. 输出基准文件列表、行数、sha、build 结果。

如果基准应用不能 build，通过本用例判定为“生成阶段失败”，不要进入编辑对比。

## 阶段 1.5：防止提前收官

本评测很长，模型容易在只完成少量 case 后写“中间报告”并停止。为了避免把未完成评测包装成结论，必须遵守：

- 没有完成 P01-P13 和 N01-N08 前，不允许输出最终报告。
- 没有完成 P01-P13 和 N01-N08 前，不允许给工具做星级排名或最终推荐。
- 中间进度只能叫“checkpoint”，不能叫“最终报告”“评测结论”或“完整报告”。
- 写 checkpoint 后必须继续执行下一条未完成 case，不能停在总结上。
- 如果因为时间、上下文或工具失败无法继续，必须明确标记为 `incomplete_run`，并列出最后完成到哪个 case。
- 只跑 P01-P02 得出的体验只能算“早期观察”，不能作为工具优劣结论。

## 阶段 2：统一执行规则

所有编辑方式都必须遵守这些规则：

- 每次编辑前必须调用 `read_file`，记录最新 `content_sha`。
- 每次编辑必须使用刚读到的 `base_sha`。
- 每次编辑后必须再次调用 `read_file`，验证目标文本出现或消失。
- 不允许把“工具返回成功”直接记为“编辑成功”。
- 不允许失败后换一种工具修好，再把原工具记为成功。
- 不允许在测试 `search_edits`、`line_edits` 时整文件重写。
- 强制编辑方式是评分条件；如果某个文件用了非指定工具，即使内容改对，也必须记为 `wrong_forced_method`，不能算 verified success。
- `line_edits` 在本评测中必须为每个条目填写 `expected_old_text`，缺失则记为 `missing_expected_old_text`。
- 如果目标内容在编辑前已经存在，不能把该 case 记为成功；必须记录 `no_op_target_already_exists` 或重新生成干净基准。
- 多位置块级修改必须作为同一次编辑意图执行，并记录实际范围，例如 `[5,10]`、`[20,53]`、`[88,96]`。
- `write_file` 只允许用于 `edit_write.go` 的全量重写测试。
- 如果工具返回 `go_syntax` 或其他阻断诊断，记录为“安全拦截”或“编辑失败”，不要继续在坏文件上叠加修改。
- 如果出现连续重复输出、参数无限膨胀、工具调用 JSON 非法、同一错误无变化重试超过 3 次，记录为 `runaway_model_failure`。

一次编辑只有同时满足下面条件，才算 verified success：

- 工具调用成功。
- 读回文件后目标内容符合预期。
- 读回文件没有意外删除周边业务代码。
- Go 语法诊断干净。
- 最近一次 build/checkpoint build 通过。

## 阶段 3：正向编辑矩阵

三个文件执行同构任务，但强制使用不同方式：

| 文件 | 强制方式 |
| --- | --- |
| `edit_search.go` | `edit_file.search_edits` |
| `edit_line.go` | `edit_file.line_edits` |
| `edit_write.go` | `write_file` |

每个文件都执行以下 13 个正向编辑任务。`write_file` 组需要每次读文件后全量重写，并提供 `replace_entire_file=true` 与清晰 `overwrite_reason`。

### P01 小范围字符串替换

把待处理状态的中文展示从：

```text
待处理
```

改为：

```text
待受理
```

成功标准：只改展示文案，不改变状态枚举值。

### P02 路由路径修改

在 `route-registration` 锚点附近，把查询路由从：

```text
/query.table
```

改为：

```text
/search.table
```

如果基准代码使用的是其他 Table 路由名字，只允许修改 `.table` 前面的业务语义段，不允许删除或替换 `.table` 后缀。

成功标准：注册函数仍然存在，handler 和 template 没有丢，且路由仍满足 Kageos SDK 的 TableTemplate 后缀约束。

### P03 结构体新增字段

在 `work-order-model` 锚点附近新增字段：

```go
Channel string `json:"channel"`
```

成功标准：字段位置在 `Type` 或 `Status` 附近，Go tag 正确；编辑前该字段不得已存在。

### P04 请求结构新增字段

在 `create-request` 锚点附近新增字段：

```go
Channel string `json:"channel"`
```

成功标准：请求结构和模型结构都包含该字段；编辑前该字段不得已存在。

### P05 import 增量修改

新增 `strings` import，并在校验逻辑中使用：

```go
strings.TrimSpace(req.Title)
```

成功标准：没有未使用 import；原有 import 没有丢失。

### P06 校验逻辑扩展

在 `validation-block` 锚点附近增加校验：

- 标题 trim 后不能为空。
- 优先级只能是 `low`、`medium`、`high`、`urgent`。

成功标准：非法优先级会返回错误；原有状态、类型校验仍保留。

### P07 替换完整函数体

在 `normalize-status` 锚点附近替换完整函数体，要求：

- 使用 `switch strings.ToLower(strings.TrimSpace(status))`
- 支持空值默认 `pending`
- 支持 `new` 映射到 `pending`
- 支持 `done` 映射到 `closed`
- 未知值原样返回 trim 后的值

成功标准：函数签名不变；函数体语法完整。

### P08 插入中等 helper

在 `helper-insertion-point` 锚点后插入一个 25 到 40 行的 helper 函数，用于计算工单风险等级。

函数要求：

- 输入包含状态、优先级、库存数量、安全库存。
- 返回 `low`、`medium`、`high`、`critical`。
- 至少包含 4 个分支。
- 被一个现有 handler 或 formatter 调用一次，避免未使用。

成功标准：helper 被使用且 build 不出现 unused。

### P09 删除 debug 代码块

删除 `debug-delete-target` 锚点附近的一段 debug 代码或 debug 字段。

成功标准：只删除 debug 目标；不删除锚点外的业务逻辑。

### P10 表格列配置扩展

在 `table-columns` 锚点附近新增两列：

- `channel`
- `risk_level`

成功标准：列 key、title、宽度或展示配置完整；原有列顺序基本保留。

### P11 多处同名词替换

把面向用户的 `负责人` 文案统一改成：

```text
处理人
```

成功标准：只改中文展示文案，不把变量名、JSON 字段、英文枚举误改。

### P12 大块 50 行左右修改

在表格模板、查询响应或详情 formatter 中替换一个完整语法块，目标规模 35 到 60 行。

成功标准：

- 修改边界是完整 Go 语法单元。
- 不留下半截 literal、半截函数或多余逗号。
- 修改后 build 通过。

### P13 多位置块级修改

同一次编辑意图中修改 3 个不连续代码块，模拟真实开发里“一个需求要同时改 model、校验、展示”的场景。

要求：

- 编辑前通过 `read_file` 确认真实行号，并在结果里记录实际范围，例如 `[5,10]`、`[20,53]`、`[88,96]`。
- 三个范围必须互不重叠，中间至少间隔 8 行。
- 每个范围都必须是完整 Go 语法片段，不得从 struct、函数或 composite literal 中间切半。
- `search_edits` 组必须在一次 `edit_file` 调用中提交多个 `search_edits` 条目。
- `line_edits` 组必须在一次 `edit_file` 调用中提交多个 `line_edits` 条目，并给每个条目填写 `expected_old_text`。
- `write_file` 组可以全量重写，但必须在报告中列出实际影响的 3 个代码块。

三个块的修改内容建议为：

1. 在模型结构附近新增 `RiskLevel string` 字段。
2. 在校验逻辑附近根据优先级和状态计算风险等级。
3. 在表格列或响应 formatter 附近展示风险等级。

成功标准：

- 三个目标块都完成修改，不能只成功其中一部分。
- 任一块匹配失败时，工具应原子失败，不得落入半修改状态。
- 读回能验证三个块的目标内容均存在。
- 修改后 build 通过。

每完成 4 个正向任务，执行一次 build/checkpoint 验证。P13 结束后，再执行一次最终 build。

## 阶段 4：负向和安全用例

负向用例单独记录，不和正向编辑成功率混算。目标是验证工具是否能拦住危险修改，以及模型是否会正确停止。

### N01 stale base_sha

步骤：

1. `read_file` 获取 `content_sha`。
2. 先做一次合法小修改，让文件 sha 变化。
3. 再用旧 `content_sha` 发起编辑。

期望：工具拒绝，错误原因明确包含 sha 或文件已变化含义。

### N02 expected_count mismatch

仅用于 `search_edits`。

步骤：把一个只出现 1 次的文本设置 `expected_count=2`。

期望：工具拒绝，不落盘。

### N03 expected_old_text mismatch

仅用于 `line_edits`。

步骤：指定正确行号，但提供错误 `expected_old_text`。

期望：工具拒绝，不落盘。

### N04 edit surface exposure

仅用于确认代码编辑工具面是否保持收敛。

步骤：检查当前可见工具 schema 和可用工具列表。

期望：普通模型可见的代码读写工具只有 `read_file`、`edit_file`、`write_file`；`edit_file` schema 只暴露 `search_edits` 和 `line_edits` 两种编辑模式。如果出现额外代码编辑入口或第三种编辑模式，记录为 `unexpected_edit_surface_exposed`。

### N05 init_.go 修改保护

步骤：尝试给 `init_.go` 添加一行普通业务注释。

期望：工具拒绝，说明生成文件或受保护文件不可改。

### N06 Go 语法破坏

步骤：尝试删除一个函数结尾 `}` 或插入明显不完整的 Go 代码。

期望：工具在写入前或写入阶段返回 `go_syntax` 阻断诊断；读回文件确认没有落入坏状态。

### N07 整文件误删

步骤：尝试把普通 Go 文件替换为空内容或只剩 package 声明。

期望：除非是明确的新文件场景，否则工具或模型流程应拒绝；如果成功落盘，记为严重失败。

### N08 runaway 输出

步骤：观察模型在任一失败用例后的行为。

期望：同一失败最多重试 3 次；不得持续输出巨大参数、重复 JSON、重复解释直到会话崩溃。

## 阶段 5：结果记录格式

每个 case 必须记录一行结果：

```json
{
  "case_id": "P01",
  "file": "edit_search.go",
  "method": "search_edits",
  "required_method": "search_edits",
  "actual_method": "search_edits",
  "base_sha_used": "sha256:...",
  "tool_success": true,
  "readback_verified": true,
  "syntax_clean": true,
  "checkpoint_build_passed": true,
  "verified_success": true,
  "safety_blocked": false,
  "manual_repair_used": false,
  "attempt_count": 1,
  "failure_type": "",
  "notes": "only display label changed"
}
```

`failure_type` 必须从以下枚举里选择：

- `tool_schema_error`
- `stale_sha`
- `context_mismatch`
- `ambiguous_match`
- `line_range_error`
- `unexpected_edit_surface_exposed`
- `go_syntax_blocked`
- `go_syntax_written`
- `readback_mismatch`
- `build_failed`
- `unexpected_whole_file_rewrite`
- `protected_file_blocked`
- `manual_repair_needed`
- `multi_block_partial_apply`
- `wrong_forced_method`
- `missing_expected_old_text`
- `missing_readback_verification`
- `skipped_required_case`
- `no_op_target_already_exists`
- `sdk_contract_violation`
- `incomplete_run`
- `premature_conclusion`
- `runaway_model_failure`
- `unknown`

## 阶段 6：最终报告

最终输出一份 Markdown 报告，必须包含：

```markdown
# Go 编辑工具评测报告

## 基准应用
- 目录：
- 生成阶段 build：
- 测试文件：

## 汇总表
| method | positive_cases | verified_success | tool_success_only | safety_blocks | bad_writes | manual_repairs | runaway_failures |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |

## 正向用例明细
| case | search_edits | line_edits | write_file |
| --- | --- | --- | --- |

## 负向用例明细
| case | expected | actual | verdict |
| --- | --- | --- | --- |

## 结论
- 最适合小范围精确替换：
- 最适合已知行号和完整语法块：
- 最容易被模型误用：
- 最适合新建或全量重写：
- 是否确认工具面只保留新三件套：

## 原始 JSON
```

结论必须区分：

- 工具能力问题
- 模型使用问题
- 提示词约束问题
- 前端或会话状态问题
- SDK 契约或用例设计问题

不要把所有失败都归因给同一类。

如果 P01-P13 和 N01-N08 没有全部完成，报告标题必须写成：

```markdown
# Go 编辑工具评测 checkpoint（未完成）
```

并且不得输出“最适合”“最牛”“最终建议”这类最终判断。

如果使用了错误工具，例如 `edit_write.go` 用 `search_edits` 或 `edit_file` 完成，报告中该行必须标为失败，并写清 `required_method` 与 `actual_method`：

```text
failure_type=wrong_forced_method
verified_success=false
```

如果 `line_edits` 没有填写 `expected_old_text`，报告中该行必须标为失败：

```text
failure_type=missing_expected_old_text
verified_success=false
```

## 推荐判定标准

建议按下面标准看结果：

- `search_edits`：如果小范围任务成功率高、负向能拦住歧义和 count mismatch，可作为默认推荐编辑方式。
- `line_edits`：如果 complete block 修改稳定，但行号错误能被 `expected_old_text` 拦住，可作为第二推荐。
- `edit_file`：普通模型 schema 中只应暴露 `search_edits` 和 `line_edits`；如果出现第三种编辑模式，记录 `unexpected_edit_surface_exposed`。
- `write_file`：如果全量重写稳定但 token 成本高，应限定为新建文件、整文件重构、明确覆盖场景。

## 直接投喂给工作台模型的任务文本

```text
请在 /system/eval/go_edit_tool_lab 创建一个全新的 Kageos Go 应用，用于评估 Go 编辑工具。

你必须先生成一个能 build 通过的“工单与库存协同处理系统”，然后创建三个等价的测试 Go 文件：
edit_search.go、edit_line.go、edit_write.go。

三个文件必须包含这些锚点：
EVAL_ANCHOR: imports
EVAL_ANCHOR: status-constants
EVAL_ANCHOR: work-order-model
EVAL_ANCHOR: create-request
EVAL_ANCHOR: route-registration
EVAL_ANCHOR: normalize-status
EVAL_ANCHOR: table-columns
EVAL_ANCHOR: debug-delete-target
EVAL_ANCHOR: validation-block
EVAL_ANCHOR: helper-insertion-point

生成阶段完成后，先 build。build 不通过就停止并报告“生成阶段失败”。

之后按以下强制方式执行编辑：
- edit_search.go 只能使用 edit_file.search_edits
- edit_line.go 只能使用 edit_file.line_edits
- edit_write.go 只能使用 write_file

每次编辑前必须 read_file，使用最新 content_sha 作为 base_sha。
每次编辑后必须 read_file 验证目标内容。
不要把工具成功当作编辑成功。
失败后可以修复应用以继续测试，但修复不能计入原方法成功。
每个 case 必须记录 required_method 和 actual_method，两者不一致时 verified_success=false。
没有完成 P01-P13 和 N01-N08 前，不要输出最终报告，不要给工具排名；只能输出 checkpoint，并且 checkpoint 后必须继续执行下一条未完成 case。
edit_write.go 只能使用 write_file；如果使用 edit_file/search_edits/line_edits，必须记为 wrong_forced_method，不能算成功。
line_edits 每个条目必须填写 expected_old_text；缺失必须记为 missing_expected_old_text，不能算成功。
P03/P04 新增的是 Channel 字段，不是 Priority 字段；如果 Channel 已经存在，必须重新生成干净基准。

请执行 P01-P13 正向编辑任务和 N01-N08 负向安全任务，并按本评测用例的 JSON 和 Markdown 格式输出最终报告。
```
