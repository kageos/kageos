# workspace-execute-sdk 与 01-execute 对比：建议迁入内容

## 一、01-execute 已有且足够的内容

- 操作 SOP、选对工具、传参要点
- 测试场景（三种场景、兜底、多人协同、测试计划）
- 易错点 1：full_code_path 必须到具体函数名
- 易错点 2：run_table_search 操作符格式、model search 标签、时间函数、List Req form 字段、完整示例
- 易错点 3：run_table_create body 必须数组、传参说明、返回
- 易错点 4：run_form_submit body 为对象、示例
- 易错点 5：run_chart_query 约定、Request 决定、示例
- 「何时用什么工具」表格

以上与 workspace-execute-sdk 对应部分已基本覆盖，无需再搬。

---

## 二、建议从 workspace-execute-sdk 迁入 01-execute 的内容

### 1. **run_table_update 专用小节**（建议必加）

- **现状**：01 里只在「何时用什么工具」表有一行，没有易错点或用法说明。
- **workspace-execute-sdk**：有完整一节（用途、工具参数、示例、返回）。
- **建议**：在 01 的易错点中增加 **「### 6. 更新表格记录（run_table_update）」**，包含：
  - body 为 **JSON 数组**，每项 `{ "id": 行ID, "updates": { "字段": 新值 } }`；单条也写一项数组。
  - 单条示例：`[{"id":1,"updates":{"status":"已处理"}}]`
  - 批量示例：`[{"id":1,"updates":{"status":"已处理"}},{"id":2,"updates":{"status":"已关闭"}}]`
  - 返回：updated_count、failed_count、data_list、errors。
  - 不传 old_values，由 app-server 自动查表填充。

### 2. **run_table_search 操作符补充**（可选）

- **现状**：01 易错点 2 已列 eq/like/in/gte/lte/contains/gt/lt/not_*，时间函数也有。
- **workspace-execute-sdk**：对 **contains**（多选包含、FIND_IN_SET）、**not_eq/not_like/not_in**（取反）有单独一句含义说明。
- **建议**：在 01 易错点 2 的操作符列表后补一句：**contains** 表示多选包含（逗号分隔存库）；**not_eq/not_like/not_in** 表示取反（不等于/不包含/不在列表中）。若当前篇幅已够可略。

### 3. **「如何获知路径与参数」**（建议加）

- **现状**：01 只在操作 SOP 第 1 条提到 read_dir/read_go_file，没有集中说明「去哪查、查什么」。
- **workspace-execute-sdk**：「八、如何获知路径与参数」4 条：列表/图表/表单路径 → read_dir；Table 可搜字段 → model search + Req form；Chart 参数 → Chart Request；Form 字段 → Request json。
- **建议**：在 01 末尾（或「何时用什么工具」之后）加一小节 **「如何获知路径与参数」**，3～4 条即可，便于不熟悉时一次查清。

### 4. **删除列表行说明**（建议加一句）

- **现状**：01 未提「删除」。
- **workspace-execute-sdk**：删除列表行当前通过 Form 或前端表格完成；若后续提供 run_table_delete 则按工具文档调用。
- **建议**：在「何时用什么工具」表下方或小结中补一句：**删除列表行**：当前通过对应 Form 或前端表格操作完成；若后续提供 run_table_delete 则按工具文档调用。

### 5. **run_form_submit 无字段时 body**（可选）

- **现状**：01 已有「无额外字段：body: {}」。
- **workspace-execute-sdk**：明确写 body 否、无字段时传 `{}`。
- **建议**：在 01 易错点 4 中补一句「无字段时 body 传 `{}`」即可，若已有类似表述可不再加。

### 6. **工具参数表**（可选）

- workspace-execute-sdk 对各工具有「参数 | 类型 | 必填 | 说明」表。
- 01 以易错点+示例为主，不强制要完整参数表；若希望更规整，可为 run_table_search / run_table_create / run_table_update 各加 2～3 行参数说明，不必照抄整表。

---

## 三、结论

- **建议必迁**：**run_table_update 专用小节**、**「如何获知路径与参数」**、**删除列表行一句说明**。
- **可选**：run_table_search 的 contains/not_* 一句、run_form_submit 无字段时 `{}`、工具参数表。

迁移后，01-execute 即可作为执行类操作的唯一入口，无需再依赖 workspace-execute-sdk。
