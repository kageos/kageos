# 操作项目

当用户要**查列表数据、提交表单、查图表、新增/更新表格记录**等执行类操作时，按本文档执行。不写代码、不落盘，只调用执行类工具。

---

## 操作 SOP

1. **确认路径**：环境信息中的「可执行函数」已列出 table/form/chart 的 full_code_path，可直接用。
2. **选对工具**：查列表 → `run_table_search`；新增 → `run_table_create`；更新 → `run_table_update`；提交表单 → `run_form_submit`；查图表 → `run_chart_query`；**测试下拉模糊搜索/回调查询** → `run_on_select_fuzzy`（仅支持按关键词或空关键词：传 full_code_path、code、可选 keyword；不支持 by_value/by_values，用于验证 OnSelectFuzzy 回调是否正常）。
3. **传参**：full_code_path 须到**具体函数名**（如 `.../nps_questionnaire_list.table`），不能只填包路径。路由名带类型后缀：`.table` / `.form` / `.chart`。
4. 调用即可。

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
键名与 model 的 json 标签一致。create_by、created_at、updated_at 由系统自动填充。业务时间字段须为**毫秒时间戳**。若 model 有 files 类型字段（如 attachment、resume_file），该字段须传对象 `{ "files": [...] }`，见下方「带上传文件时」。

### 3. run_form_submit：body 为 JSON 对象

```json
{"questionnaire_id":1,"score":8,"comment":"满意"}
```
无额外字段时传 `{}`。

**带上传文件时（易错）**：**表单（run_form_submit）和表格（run_table_create、run_table_update）** 里若有 `widget.type === "files"` 的字段（如 input_files、attachment、resume_file），该字段须传**对象**，不能传数组。正确结构：`{ "files": [ { "name": "xxx", "source_name": "原始文件名", "storage": "minio", "url": "...", "server_url": "...", "size": 12345, "is_uploaded": true } ], "widget_type": "files", "data_type": "struct" }`。用户消息附件中的文件放入该对象的 `files` 数组。run_table_create 的 body 数组里每条记录的 files 字段、run_table_update 的 updates 里对 files 字段的赋值，均按此对象结构。错误写法：`"input_files": [ {...} ]` 会报 unmarshal 错误。详见 read_doc("/system/prompt/workspace/misc-tasks") 中「files 组件传参」。

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
若 updates 中含 files 类型字段，该字段须传对象 `{ "files": [...] }`，见上方「带上传文件时」。

---

## 如何获知路径与参数

1. **路径**：环境信息的「可执行函数」或 `read_dir` 查看。
2. **Table 可搜字段**：`read_go_file` 看 model 的 search 标签和 Req 的 form 标签。
3. **Chart 参数**：`read_go_file` 看 Chart 的 Request 结构。
4. **Form 字段**：`read_go_file` 看 Request 的 json 标签。
