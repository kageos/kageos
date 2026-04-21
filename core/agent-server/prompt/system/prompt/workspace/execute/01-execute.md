# 操作项目

当用户要**查列表数据、提交表单、查图表、新增/更新表格记录**等执行类操作时，按本文档执行。不写代码、不落盘，只调用执行类工具。

---

## 操作 SOP

1. **确认路径**：环境信息中的「可执行函数」已列出 table/form/chart 的 full_code_path，可直接用。
2. **选对工具**：查列表 → `run_table_search`；新增 → `run_table_create`；更新 → `run_table_update`；提交表单 → `run_form_submit`；查图表 → `run_chart_query`；**测试下拉模糊搜索/回调查询** → `run_on_select_fuzzy`（仅支持按关键词或空关键词：传 full_code_path、code、可选 keyword；不支持 by_value/by_values，用于验证 OnSelectFuzzy 回调是否正常）。
3. **传参**：full_code_path 须到**具体函数名**（如 `.../nps_questionnaire_list.table`），不能只填包路径。路由名带类型后缀：`.table` / `.form` / `.chart`。
4. **确认参数结构**：执行前必须已有该函数的字段摘要或源码定义。环境列表只提供路径，不提供完整参数；若上下文没有字段名、必填项、枚举值、文件字段和默认值行为，先用 `search_tools` 获取字段摘要，或 `read_go_file` 查看 Request/model 定义。
5. 调用执行工具。

**禁止猜参**：不要根据函数名、路由名、相似工具或底层命令行工具习惯拼 body/url_query。遇到 `参数校验失败`、`required`、`oneof`、字段不存在、url_query 格式错误时，先读取字段摘要/源码/本文档，再按定义重试。

---

## 测试场景：用户与组织数据

| 场景 | 做法 |
|------|------|
| **不依赖指定用户/组织** | 自主测试，测完给报告 |
| **需要填用户/组织但不需要对方操作**（如抄送人） | 用兜底 `test_user` + `/org/virtual/test`，不必问用户，直接开干 |
| **必须由另一用户亲自操作**（如审批流转） | 生成测试计划文档（按角色写「如果你作为 xxx 需要做」），用户确认后 write_doc |

---

## 工具传参规范与易错点

### 1. run_table_search：url_query 用操作符格式

筛选操作符：`eq`（精确）、`like`（模糊）、`in`（多选）、`contains`（多选包含）、`gte`/`lte`（范围）、`gt`/`lt`、`not_eq`/`not_like`/`not_in`（取反）。

格式为 `操作符=字段:值`，**不要**写成 `字段=值`。

| 需求 | url_query | 说明 |
|------|-----------|------|
| 分页 | `page=1&page_size=20` | |
| 分页+排序 | `page=1&page_size=20&sorts=id:desc` | |
| 精确查 ID | `eq=id:123` | |
| 模糊搜标题 | `like=title:会议` | model 需有 search:"like" |
| 状态多选 | `in=status:待处理,已完成` | |
| 时间范围 | `gte=created_at:Now(-7d)&lte=created_at:Now()` | 时间函数：Now()、Today()、Now(-7d)、Now(2026-02-01) |
| 组合 | `eq=id:1&like=title:问卷&in=status:进行中,已结束` | |

**Req 自定义 form 字段**也拼进 url_query（如 `status=未开始`），与 model 的 search 操作符并存。

### 2. run_table_create：body 必须是 JSON 数组

即使只新增一条也必须是数组：
```json
[{"title":"问卷A","description":"描述","target_group":"全部用户"}]
```
键名与 model 的 json 标签一致。create_by、created_at、updated_at 由系统自动填充。业务时间字段须为**毫秒时间戳**。若 model 有 files 类型字段（如 attachment、resume_file），该字段须传文件引用字符串，见下方「带上传文件时」。

### 3. run_form_submit：body 为 JSON 对象

```json
{"questionnaire_id":1,"score":8,"comment":"满意"}
```
无额外字段时传 `{}`。

提交前先确认该 Form 的 Request 字段：看字段的 `json` 名、`validate:"required"`、`oneof`/`widget` options。字段摘要里有【必填】就必须显式传入；前端默认值只是界面初始值，不会自动进入 body。

**带上传文件时**：**表单（run_form_submit）和表格（run_table_create、run_table_update）** 里若有 `widget.type === "files"` 的字段（如 input_files、attachment、resume_file），该字段传**字符串**，值为 `bucket/object_key` 文件引用；多文件用英文逗号分隔。示例：`{"input_files":"ai-agent-os/workspace/chat/2026/04/20/xxx.png"}`，多文件：`{"attachment":"ai-agent-os/a.pdf,ai-agent-os/b.xlsx"}`。

### 4. run_chart_query：url_query 由该 Chart 的 Request 决定

一个 Chart 路由一次只返回一张图。full_code_path 须到具体图表函数名，多张图需分别调用。参数看对应 .go 的 Request 结构。

**多系列图表（重要）**：
- `chart` 路由虽然一次只返回一张图，但这张图的 `series` 可以有多组。
- 更推荐的模式是：时间范围作为主筛选，`status/department/store` 这类业务维度直接展开成多组 `series` 做对比。
- 例如“工单趋势统计”可以直接同时展示 `待处理/处理中/已完成` 三条折线，而不是先让用户选一个状态。

### 5. run_table_update：body 为 JSON 数组

每项为 `{ "id": 行ID, "updates": { "字段名": 新值 } }`：
```json
[{"id":1,"updates":{"status":"已处理","title":"新标题"}}]
```
若 updates 中含 files 类型字段，该字段须传文件引用字符串，见上方「带上传文件时」。

---

## 如何获知路径与参数

1. **路径**：环境信息的「可执行函数」或 `read_dir` 查看。
2. **Table 可搜字段**：`read_go_file` 看 model 的 search 标签和 Req 的 form 标签。
3. **Chart 参数**：`read_go_file` 看 Chart 的 Request 结构。
4. **Form 字段**：`read_go_file` 看 Request 的 json 标签。

若通过 `search_tools` 找到函数，并且返回里已有字段摘要，可直接按摘要传参；若只有名称、路径、描述，必须继续读源码或请求更完整的 request 输出后再执行。批量测试多个函数时，先把每个函数的参数结构确认完，再进入提交阶段。
