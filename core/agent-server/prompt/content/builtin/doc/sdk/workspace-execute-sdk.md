# 工作台执行能力说明（执行模式 SDK）

本文档说明在**执行模式**下，对工作区已生成应用做「查数据、提交表单、查图表」等操作时，应使用哪些工具、如何传参。执行前请先用 `read_dir` 确认当前工作区下有哪些 tables、plugins、charts 路径，必要时用 `read_go_file` 查看对应 .go 里的 Request 结构以确定可传参数。

---

## 一、常见操作与工具对应

| 操作           | 工具               | 说明 |
|----------------|--------------------|------|
| 查询列表数据   | run_table_search   | 调用 Table 查询接口，支持分页、排序、筛选（eq/like/in 等） |
| 新增表格记录   | run_table_create   | 调用 Table 新增接口，传单条记录 JSON body，触发 OnTableAddRow |
| 提交表单       | run_form_submit    | 调用 Form 提交接口，传 JSON body |
| 查询图表数据   | run_chart_query    | 调用 Chart 查询接口，参数由该 Chart 的 Req 决定 |
| 编辑/删除列表行 | 见下文「列表增删改」 | 若后续提供 run_table_update / run_table_delete 则按工具文档调用；当前可通过对应 Form 或前端表格操作完成 |

---

## 二、查询列表数据（run_table_search）

### 2.1 用途

请求工作区 Table 函数的查询接口，返回分页列表（items、total_count、current_page 等）。例如：问卷列表、工单列表、简历列表。

### 2.2 工具参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| full_code_path | string | 是 | 表格函数的**完整路径（必须包含函数名）**，如 `/luobei/myapp/nps/nps_questionnaire_list`；**不能只填包路径**如 `/luobei/myapp/nps`，否则接口无法匹配到具体表格，会返回空数据。 |
| url_query | string | 否 | 完整 URL 查询串，与 pkg/gormx/query 约定一致（见下）；不传则默认 page=1&page_size=20 |
| page | number | 否 | 页码，默认 1；若已传 url_query 则优先用 url_query 内参数 |
| page_size | number | 否 | 每页条数，默认 20 |
| sorts | string | 否 | 排序，如 `id:desc` 或 `-updated_at` |

**重要：full_code_path 必须到「具体表格函数」**  
后端按 full_code_path 精确匹配一个 Table 函数（每个函数在 init 里注册，如 `GET("nps_questionnaire_list", ...)`）。若只传包路径（如 `.../nps`），没有函数与之对应，会返回空。正确做法：先 `read_dir` 看包下有哪些 .go，再打开对应列表的 .go 看 `init()` 里注册的函数名（如 `nps_questionnaire_list`），把 full_code_path 拼成 `.../nps/nps_questionnaire_list`。

### 2.3 查询串约定（url_query）

与 **pkg/gormx/query** 对齐，表格接口会把 URL Query 绑定到 `SearchFilterPageReq` 及该 Table 的 Req 自定义字段。

- **分页与排序**：`page`、`page_size`、`sorts`（如 `id:desc,name:asc` 或 `-updated_at`）。
- **筛选操作符**（格式 + 含义）：
  - `eq`：精确匹配，格式 `field:value`。含义：该字段等于给定值（如 `eq=status:待处理` 表示查询状态为「待处理」的）。
  - `like`：模糊匹配，格式 `field:value`。含义：该字段包含给定字符串（如 `like=title:会议` 表示标题里包含「会议」）。
  - `in`：多选 IN，格式 `field:v1,v2`。含义：该字段在给定多个值中任选其一（如 `in=status:待处理,已完成` 表示查询状态是「待处理」或「已完成」的）。
  - `contains`：多选包含（FIND_IN_SET 语义），格式 `field:v1,v2`。含义：该字段（逗号分隔存库）包含给定值之一。
  - `gte` / `lte`：大于等于 / 小于等于，格式 `field:value`。含义：该字段 ≥ 或 ≤ 给定值（常用于时间、数字范围）。
  - `gt` / `lt`：大于 / 小于，格式 `field:value`。含义：该字段 > 或 < 给定值。
  - `not_eq` / `not_like` / `not_in`：取反。含义：该字段不等于、不包含、不在给定值中。

**时间范围**：大模型无需手写时间戳，可用**时间函数**（与 sdk/agent-app/widget/timestamp 约定一致），run_table_search 调用时会在工具内部转为毫秒时间戳再请求接口。支持：
- `Now()`：当前时间（毫秒）
- `Today()`：今天 00:00:00（毫秒）
- `Yesterday()`：昨天 00:00:00（毫秒）
- `Now(-7d)`、`Now(+1h)`：相对当前时间的偏移（单位 s/h/d/w/m/y，如 -7d 表示七天前）
- `Now(2026-02-01 13:05:05)`、`Now(2026-02-01)`：指定日期时间（解析为本地时间毫秒）

**可搜字段**由该表格** model 结构体的 search 标签**决定（如 `search:"eq"`、`search:"like"`、`search:"in"`）。若该 Table 的 Req 还有**自定义 form 字段**（如 `status`），也一并拼进 url_query。

### 2.4 示例（按 model 与 URL 对照，更直观）

**示例一：仅分页 / 分页 + 排序**

- 仅分页：`url_query` = `page=1&page_size=20`
- 分页 + 排序：`url_query` = `page=1&page_size=20&sorts=id:desc` 或 `page=1&page_size=20&sorts=-updated_at`

**示例二：GORM model 的 search 标签 → 对应 URL 参数**

假设某表格的 **model** 定义如下（只列与搜索相关的字段和标签）：

```go
type Questionnaire struct {
    ID          int    `json:"id" search:"eq"`                    // 精确匹配
    Title       string `json:"title" search:"like"`              // 模糊
    Status      string `json:"status" search:"in"`                // 多选 IN
    TargetGroup string `json:"target_group" search:"in"`          // 多选 IN
    CreateBy    string `json:"create_by" search:"in"`             // 多选 IN
    CreatedAt   int64  `json:"created_at" search:"gte,lte"`       // 范围
    // ...
}
```

则 **url_query** 可按下面方式拼（与 pkg/gormx/query 约定一致）。每行都写了「含义」方便理解：

| 需求           | url_query 片段 | 含义 |
|----------------|----------------|------|
| 精确查某 ID    | `eq=id:123` | 查询 id 等于 123 的记录 |
| 标题模糊       | `like=title:会议` | 查询标题中包含「会议」的记录 |
| 状态多选       | `in=status:待处理,已完成` | 查询状态为「待处理」或「已完成」的记录 |
| 多字段 IN      | `in=target_group:全部用户,create_by:beiluo` | 查询目标用户组为「全部用户」且创建人为 beiluo 的记录（多字段 IN 时格式为 field1:v1,v2,field2:v3,v4） |
| 时间范围（推荐用时间函数） | `gte=created_at:Now(-7d)&lte=created_at:Now()` | 查询创建时间在「七天前至今」的记录；工具内部会把 Now(-7d)、Now() 转为毫秒时间戳 |
| 时间范围（指定日期） | `gte=created_at:Now(2026-02-01 00:00:00)&lte=created_at:Now(2026-02-01 23:59:59)` | 查询 2026-02-01 当天的记录；无需手写时间戳 |
| 组合           | `eq=id:1&like=title:问卷&in=status:进行中,已结束` | 同时满足：id=1、标题含「问卷」、状态为「进行中」或「已结束」 |

**示例三：List Req 带自定义 form 字段 → 也要拼进 URL**

假设该表格的 **List 请求体**除了内嵌 `SearchFilterPageReq`，还有自定义筛选：

```go
type QuestionnaireListReq struct {
    Status string `json:"status" form:"status"`  // 前端筛「未开始/进行中/已结束」
    query.SearchFilterPageReq `widget:"-"`
}
```

则 **url_query** 里既要包含 model 的 search 约定，也要包含 Req 的自定义 form 字段：

- 例：`page=1&page_size=20&sorts=id:desc&in=target_group:全部用户,create_by:beiluo&status=未开始`
  - **含义**：第 1 页、每页 20 条、按 id 降序；筛选目标用户组为「全部用户」且创建人为 beiluo；并且只查「未开始」的问卷（status 为 Req 自定义筛选项）。
  - `in=target_group:...,create_by:...` 来自 **model** 的 `search:"in"`
  - `status=未开始` 来自 **Req** 的 `form:"status"`

**示例四：完整 URL 示例（与真实调用一致）**

- full_code_path：`/luobei/myapp/nps/nps_questionnaire_list`
- url_query：`in=target_group:全部用户,create_by:beiluo&status=未开始&page=1&page_size=20&sorts=id:desc`
- **含义**：查询「NPS 问卷列表」— 目标用户组为「全部用户」、创建人为 beiluo、问卷状态为「未开始」、第 1 页每页 20 条、按 id 降序。

则实际请求为：  
`GET /workspace/api/v1/table/search/luobei/myapp/nps/nps_questionnaire_list?in=target_group:全部用户,create_by:beiluo&status=未开始&page=1&page_size=20&sorts=id:desc`

调用 run_table_search 时：`full_code_path` 填表格路径，`url_query` 填上述查询串即可。

### 2.5 常见错误：只传包路径导致查不到数据

若 `full_code_path` 只填到**包路径**（如 `/luobei/.../nps`），没有包含**具体表格函数名**（如 `nps_questionnaire_list`），后端无法匹配到任何一个 Table 函数，会返回空数据（如 `{}` 或 `items: []`）。

- **错误示例**：`full_code_path: "/luobei/testfunctioncall/testwork/testcfxprj/nps"` → 查不到问卷列表或响应列表。
- **正确做法**：先 `read_dir` 看包 `nps` 下有哪些 .go；再 `read_go_file` 看 `nps_questionnaire.go` 的 `init()`，得到函数名 `nps_questionnaire_list`，则 full_code_path 填 `/luobei/.../nps/nps_questionnaire_list`；查响应列表则用 `nps_response_list`，即 `/luobei/.../nps/nps_response_list`。

---

## 三、新增表格记录（run_table_create）

### 3.1 用途

请求工作区 Table 函数的**新增接口**，新增一条表格记录。会触发该表格的 OnTableAddRow 回调。例如：新增问卷、新增工单、新增简历。

### 3.2 工具参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| full_code_path | string | 是 | 表格函数的**完整路径（必须包含函数名）**，与 run_table_search 一致，如 `/luobei/myapp/nps/nps_questionnaire_list` |
| body | string | 是 | 单条记录的**字段 JSON 字符串**，键为表格 model 的 json 标签，必填字段需包含，可选项可省略 |

### 3.3 传参说明

- `full_code_path` 与 run_table_search 相同，必须是**具体表格函数**的完整路径（不能只填包路径）。环境中的「当前目录下的可执行函数」会列出 table 类型及 full_code_path，可直接使用。
- `body` 为 JSON 对象字符串，键名与表格 **model 结构体的 json 标签**一致（与 OnTableAddRow 绑定的结构体一致）。必填字段需包含，否则后端校验可能失败；可选项可省略。
- 时间字段若为毫秒时间戳，需传数字；若表格支持字符串格式，按该表格约定传。

### 3.4 示例

- 新增 NPS 问卷：`full_code_path` = `/luobei/myapp/nps/nps_questionnaire_list`，`body` = `{"title":"2025Q1 满意度","description":"第一季度用户满意度调研","target_group":"全部用户","start_time":1738339200000,"end_time":1738944000000,"send_reminder":true}`（字段名与 NPSQuestionnaire 的 json 标签一致）。
- 新增 NPS 响应：`full_code_path` = `/luobei/myapp/nps/nps_response_list`（表格列表函数路径），`body` = 单条 NPSResponse 字段 JSON，如 `{"questionnaire_id":1,"score":9,"category":"推荐者","feedback":"很好用","user_id":"u1","user_group":"付费用户","channel":"Web网页"}`。

---

## 四、提交表单（run_form_submit）

### 4.1 用途

请求工作区 Form 函数的提交接口，提交表单数据。例如：收银台结算、投票提交、问卷提交。

### 4.2 工具参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| full_code_path | string | 是 | 表单函数的完整路径，如 `/luobei/myapp/plugins/cashier_desk` |
| body | string | 否 | 表单字段的 **JSON 字符串**，如 `{"name":"张三","amount":100}`；无字段时传 `{}` |

### 4.3 传参说明

- `body` 为 JSON 对象字符串，键为表单字段的 json 名（与对应 .go 里 Request 结构体字段一致）。
- 若 Form 有必填项，需在 body 中带上；可选项可省略。
- 文件上传等复杂类型以当前执行模式支持为准（若有专用约定再补充）。

### 4.4 示例

- 无额外字段：`body` = `{}`
- 有字段：`body` = `{"questionnaire_id":1,"score":8,"comment":"满意"}`

---

## 五、查询图表数据（run_chart_query）

### 5.1 用途

请求工作区 Chart 函数的查询接口，返回图表所需数据。例如：NPS 趋势、评分分布、当前 NPS 仪表盘。

### 5.2 工具参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| full_code_path | string | 是 | 图表函数的完整路径，如 `/luobei/myapp/nps/nps_current_score_statistics` |
| url_query | string | 否 | 完整 URL 查询串，**参数由该 Chart 的 Request 结构决定，不固定**（如 questionnaire_id、group_by、start_time、end_time 等） |

### 5.3 传参说明

- 每个 Chart 的 Req 不同，需用 `read_go_file` 查看对应 .go 里 Chart 的 **Request 结构**（form 标签），以确定可传参数名与取值。
- 常见示例：`questionnaire_id=1&group_by=按天分组`。

### 5.4 示例

- 无参数：不传 `url_query` 或传空。
- 有参数：`url_query` = `questionnaire_id=1&group_by=按天分组`（中文值会在请求中按 URL 编码处理）。

---

## 五、列表增删改（当前说明）

- **新增一行**：标准接口为 `POST /workspace/api/v1/table/create/{full-code-path}`，请求体为新增记录的字段。执行模式若后续提供 `run_table_create` 等工具，则按该工具文档传参。
- **编辑/删除**：通过表格回调（OnTableUpdateRow / OnTableDeleteRows）对应接口完成。执行模式若后续提供专用工具，则按工具文档调用。
- 当前执行模式主要提供：**run_table_search**（查列表）、**run_form_submit**（提交表单）、**run_chart_query**（查图表）。其他操作可结合 read_go_file 查看接口定义后，按需扩展工具或由产品侧说明。

---

## 六、如何获知路径与参数

1. **列表/图表/表单路径**：在工作区目录下执行 `read_dir`，查看有哪些 `tables/xxx`、`plugins/xxx`、或 charts 等路径；full_code_path 一般为 `/用户/app/.../函数名`（如 `/luobei/myapp/nps/nps_questionnaire_list`）。
2. **Table 可搜字段与 Req 自定义字段**：`read_go_file` 打开该 Table 对应的 .go，看 model 的 **search 标签**（eq/like/in/gte/lte 等）和 List Req 的 **form 标签**（如 status）。
3. **Chart 参数**：`read_go_file` 打开该 Chart 对应的 .go，看 Chart 使用的 **Request 结构**（如 NPSStatisticsReq 的 questionnaire_id、group_by）。
4. **Form 字段**：`read_go_file` 打开该 Form 对应的 .go，看 **Request 结构**的 json/widget 标签，body 的键与 json 名一致。

---

## 七、小结

- **查列表**：run_table_search，url_query 遵循 pkg/gormx/query，可搜字段看 model 的 search 标签，自定义条件看 Req 的 form 字段。
- **提交表单**：run_form_submit，body 为 JSON 字符串，字段名与 Request 的 json 一致。
- **查图表**：run_chart_query，url_query 由该 Chart 的 Request 决定，不固定，需看对应 .go。
- 执行前用 read_dir / read_go_file 确认路径与参数结构，再调用对应工具并按要求传参。
