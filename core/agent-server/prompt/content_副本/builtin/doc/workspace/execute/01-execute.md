# 操作项目

当用户要**查列表数据、提交表单、查图表、新增/更新表格记录**等执行类操作时，按本文档执行。不写代码、不落盘，只调用执行类工具。

**工具用法与传参**见本文档（操作 SOP、易错点、何时用什么工具等）。

---

## 操作 SOP（执行时按此执行）

1. **确认路径与参数**：环境中的「当前目录下的可执行函数」会列出 table/form/chart 的 full_code_path，可直接用。若需确认可搜字段、Req 自定义参数或 Chart 的 Request 结构，用 read_dir / read_go_file 看对应 .go。
2. **选对工具**：查列表 → run_table_search；新增表格记录 → run_table_create；更新表格记录 → run_table_update；提交表单 → run_form_submit；查图表 → run_chart_query。
3. **传参**：full_code_path 须到**具体函数名**（如 `/luobei/myapp/nps/nps_questionnaire_list.table`），**不能只填包路径**（如 `.../nps`），否则接口无法匹配、会返回空。**路由名一定会带类型后缀**：`.table` = 表格列表，`.form` = 表单，`.chart` = 图表；看到后缀即可知该函数类型。url_query、body 等约定见本文档下方易错点与表格。
4. **调用**：按文档传参后调用对应工具即可。

---

## 测试场景：用户与组织数据（抄送人、关联组织等）

部分执行或测试会依赖**用户/组织相关数据**（抄送人、关联组织、审批人、可见范围、指定处理人等）。**测试前先分析属于下面哪一种场景**，再决定是直接开干、用兜底、还是出测试计划。

---

### 测试前先分析：三种场景

| 场景 | 说明 | 做法 |
|------|------|------|
| **最好（最简单）** | 不依赖「指定用户/组织」的测试数据，或仅需当前用户身份即可完成全流程。 | **不生成测试文档**，自主测试即可；**测试完毕给出测试报告**即可，无需问用户要数据。 |
| **其次** | 需要「填谁/填哪个组织」这类数据（如抄送人、抄送部门、可见范围、关联组织），但**不要求被填的人/部门亲自来操作**——只是数据里填一下。 | **优先用兜底**：`test_user` + `/org/virtual/test`，**不必让用户确认**，直接开干。**除非**用户自己提供了测试用户/组织，才用用户提供的。 |
| **最复杂（多人协同）** | 业务上**指定了某个用户必须亲自操作**，且该用户是**另一人**（如指定处理人是 sina，必须由 sina 把工单流转到下一状态）。你无法以 sina 的身份代劳，只能由同事（sina）登录后操作。 | **只有这种才需要用户确认和生成文档**。应生成一份**测试计划/测试文档**，让用户确认后再 write_doc 写入，按角色分步写清「谁、做什么」，交给同事按文档协助执行。 |

**判断要点**：  
- 只是「数据里填抄送人、抄送部门、可见范围」→ 属于**其次**，可用兜底。  
- 是「必须由某用户登录并执行某操作」（如审批、流转、处理），且**该用户就是当前用户自己** → **能用自己测试**：直接用当前用户身份完成全流程即可，无需协作者、无需 test_user。  
- 是「必须由**另一**用户登录并执行某操作」→ 属于**最复杂**，出测试计划、分角色列项。

---

### 其次场景：优先用兜底，不必确认

当属于**其次**场景时：

- **默认**：直接用**测试用户** `test_user` + **测试组织** `/org/virtual/test` 开干，**不必让用户确认**，无需先问。
- **例外**：若用户**自己提供了**测试用户或组织（如「抄送人填张三」「关联到销售部」），则用用户提供的，不再用兜底。

---

### 多人协同时：生成测试计划文档（最复杂场景）

当分析出属于**最复杂**场景（必须某用户亲自操作、你无法代劳）时，**不要卡住**，改为生成一份**测试计划/测试文档**，用 **write_doc** 写入当前目录；便于你联系另一用户时，只需告诉他「有个测试事项需要协助」，对方打开文档即可按角色执行。

**文档表述方式**（按角色写「给谁看、要做什么」，而不是「我作为谁」）：

- **如果你作为 beiluo 的用户身份，你需要做：**
  - 1）创建 x 个待处理工单，指定处理人为 sina；
  - 2）……（其他仅当前用户可执行的操作）
- **如果你作为 sina 的用户身份，你需要做：**
  - 1）将 beiluo 创建的上述待处理工单流转为「xxx」状态；
  - 2）……（其他必须由 sina 登录后执行的操作）
- 如还有第三、第四角色，依次列项：「如果你作为 xxx 的用户身份，你需要做：……」

这样只需把文档或链接发给对方，对方以自己身份打开即可按「如果你作为 xxx」对照执行。

**流程要求**：

1. **先出文档内容、再落盘**：先生成完整测试计划内容并展示给用户，**让用户确认**（如「以上测试计划是否 OK，确认后我写入当前目录」）；用户确认 OK 后，再用 **write_doc** 将文档写入当前目录（如测试计划-xxx.md），不要未经确认就写。
2. **依次执行当前用户能做的部分**：文档写入后，按测试计划**依次执行**当前用户（如 beiluo）能完成的步骤（创建工单、指定处理人等），执行完毕再进入下一步。
3. **需要另一用户协助时**：当步骤走到「必须由另一用户（如 sina）登录操作」时，向当前用户说明「这部分需要另一用户帮忙测试」，并生成**一句话 + 背景**，供当前用户转发给协作者（如 sina）：
   - **话术示例**：「需要 sina 用户进入当前目录，打开工作台，然后输入：**背景：当前用户 beiluo，需要协助完成工单流转等测试，可读取当前目录下的《测试计划-xxx》文档了解详细步骤。**」
   - 协作者（sina）进入当前目录、打开工作台后，按这句话输入，即可让助手读取同目录下的测试文档，理解「如果你作为 sina 的用户身份，你需要做」的内容并协助执行。

**若不存在需要协同的操作**（例如只有抄送人、抄送部门这类只需填数据的），则不必走多人协同，直接用兜底用户和组织即可；若完全不需要用户/组织数据，自主测试、测完给测试报告即可，不生成测试文档。

**总结**：测试前先判断是「最好 / 其次 / 最复杂」三种之一。**最好**：不生成测试文档，自主测试，测完给出测试报告。**其次**：优先用 test_user + 测试组织，不必让用户确认，直接开干；除非用户自己提供了用户/组织才用用户的。**最复杂**：才需要用户确认和生成测试计划文档（write_doc、确认后再写）、分角色写清「如果你作为 xxx 需要做」、依次执行当前用户步骤，需要协作者时给出「进入当前目录 + 打开工作台 + 输入背景与文档名」的一句话指引。**能用当前用户自己完成全流程的**（如指定处理人/审批人填自己），直接用自己测试即可，无需协作者、无需 test_user。

---

## 易错点（必看，含完整示例）

以下示例与后端约定对齐，可多不可少。执行前用 read_dir / read_go_file 确认路径与参数结构。

---

### 1. full_code_path 必须到具体函数名

后端按 full_code_path 精确匹配一个 Table/Form/Chart 函数（每个函数在 init 里注册，如 `GET("nps_questionnaire_list.table", ...)`）。若只传包路径，没有函数与之对应，会返回空。

**错误示例**（只填包路径，查不到数据）：
- `full_code_path: "/luobei/testfunctioncall/testwork/testcfxprj/nps"` → 查不到问卷列表或响应列表，返回 `{}` 或 `items: []`。

**正确做法**：先 read_dir 看包 `nps` 下有哪些 .go；再 read_go_file 看 `nps_questionnaire.go` 的 `init()`，得到函数名（如 `nps_questionnaire_list.table`），则 full_code_path 填 `/luobei/.../nps/nps_questionnaire_list.table`；查响应列表则用 `nps_response_list.table`，即 `/luobei/.../nps/nps_response_list.table`。**路由名一定会带类型后缀**：`.table` = 表格列表，`.form` = 表单，`.chart` = 图表；看到后缀即可知该函数类型。

**正确示例**（必须包含 init 里注册的函数名；路由名一定带类型后缀，看到后缀即知类型）：
- 查问卷列表：`full_code_path: "/luobei/myapp/nps/nps_questionnaire_list.table"`
- 查问卷响应列表：`full_code_path: "/luobei/myapp/nps/nps_response_list.table"`
- 查工单列表：`full_code_path: "/luobei/myapp/crm/ticket/crm_ticket_list.table"`
- 表单：`full_code_path: "/luobei/myapp/plugins/cashier_desk.form"`
- 图表：`full_code_path: "/luobei/myapp/nps/nps_current_score_statistics.chart"`（不能到包路径，多张图每张图有各自路由，需分别调用）

---

### 2. 查列表（run_table_search）：url_query 必须用操作符格式，不要用「字段=值」

**筛选操作符**（与 pkg/gormx/query 对齐）：`eq`（精确）、`like`（模糊）、`in`（多选 IN）、`contains`（多选包含，逗号分隔存库）、`gte`/`lte`（范围）、`gt`/`lt`（大于/小于）、`not_eq`/`not_like`/`not_in`（取反）。格式为 `操作符=字段:值`，**不要**写成 `字段=值`。

**错误示例**（不会按筛选生效）：
- `url_query: "name=tencent"`、`url_query: "title=会议"`、`url_query: "status=待处理"`  
- 后端不会按模糊/精确筛选处理，可能被忽略或当其它参数，导致查不到预期数据。

**正确示例（按 model 的 search 标签用对应操作符）**

假设某表格的 model 定义如下（与 pkg/gormx/query 约定一致）：

```go
type Questionnaire struct {
    ID          int    `json:"id" search:"eq"`                    // 精确匹配
    Title       string `json:"title" search:"like"`              // 模糊
    Status      string `json:"status" search:"in"`               // 多选 IN
    TargetGroup string `json:"target_group" search:"in"`         // 多选 IN
    CreateBy    string `json:"create_by" search:"in"`             // 多选 IN
    CreatedAt   int64  `json:"created_at" search:"gte,lte"`      // 范围
    // ...
}
```

则 url_query 可按下面方式拼（每行含义见说明）：

| 需求 | url_query 片段 | 含义 |
|------|----------------|------|
| ⚠️ 错误写法（勿用） | `name=tencent`、`title=会议` | 字段=值 不会按筛选生效；必须用下面操作符格式 |
| 仅分页 | `page=1&page_size=20` | 第 1 页、每页 20 条 |
| 分页 + 排序 | `page=1&page_size=20&sorts=id:desc` 或 `sorts=-updated_at` | 按 id 降序 / 按更新时间降序 |
| 精确查某 ID | `eq=id:123` | 查询 id 等于 123 的记录 |
| 名称/标题模糊 | `like=name:tencent`、`like=title:会议` | 该字段包含给定字符串（model 需有 search:"like"） |
| 状态多选 | `in=status:待处理,已完成` | 查询状态为「待处理」或「已完成」的记录 |
| 多字段 IN | `in=target_group:全部用户,create_by:beiluo` | 目标用户组为「全部用户」且创建人为 beiluo（多字段 IN 格式为 field1:v1,v2,field2:v3,v4） |
| 时间范围（时间函数） | `gte=created_at:Now(-7d)&lte=created_at:Now()` | 创建时间在「七天前至今」；工具内部把 Now(-7d)、Now() 转为毫秒时间戳 |
| 时间范围（指定日期） | `gte=created_at:Now(2026-02-01 00:00:00)&lte=created_at:Now(2026-02-01 23:59:59)` | 查询 2026-02-01 当天的记录；无需手写时间戳 |
| 组合 | `eq=id:1&like=title:问卷&in=status:进行中,已结束` | 同时满足：id=1、标题含「问卷」、状态为「进行中」或「已结束」 |

**List Req 带自定义 form 字段时，也要拼进 url_query**

假设 List 请求体除内嵌 SearchFilterPageReq 外还有自定义筛选：

```go
type QuestionnaireListReq struct {
    Status string `json:"status" form:"status"`  // 前端筛「未开始/进行中/已结束」
    query.SearchFilterPageReq `widget:"-"`
}
```

则 url_query 既要包含 model 的 search 约定，也要包含 Req 的 form 字段：

- 例：`page=1&page_size=20&sorts=id:desc&in=target_group:全部用户,create_by:beiluo&status=未开始`
  - **含义**：第 1 页、每页 20 条、按 id 降序；筛选目标用户组为「全部用户」且创建人为 beiluo；并且只查「未开始」的问卷（status 为 Req 自定义筛选项）。
  - `in=target_group:...,create_by:...` 来自 **model** 的 `search:"in"`
  - `status=未开始` 来自 **Req** 的 `form:"status"`

**完整调用示例（与真实请求一致）**

- full_code_path：`/luobei/myapp/nps/nps_questionnaire_list.table`
- url_query：`in=target_group:全部用户,create_by:beiluo&status=未开始&page=1&page_size=20&sorts=id:desc`
- **含义**：查询 NPS 问卷列表 — 目标用户组为「全部用户」、创建人为 beiluo、问卷状态为「未开始」、第 1 页每页 20 条、按 id 降序。

实际请求为：  
`GET /workspace/api/v1/table/search/luobei/myapp/nps/nps_questionnaire_list.table?in=target_group:全部用户,create_by:beiluo&status=未开始&page=1&page_size=20&sorts=id:desc`

调用 run_table_search 时：full_code_path 填表格路径，url_query 填上述查询串即可。

**时间函数**（工具内部转为毫秒时间戳）：`Now()`、`Today()`、`Yesterday()`、`Now(-7d)`、`Now(+1h)`（单位 s/h/d/w/m/y）、`Now(2026-02-01 13:05:05)`、`Now(2026-02-01)`。可搜字段由该 Table **model 的 search 标签**决定。

---

### 3. 新增表格记录（run_table_create）：body 必须是 JSON 数组

**错误示例**（传成单个对象，会报错或无效）：
- `body: "{\"title\":\"问卷A\",\"description\":\"描述\"}"` → 应为数组，不是单对象。

**正确示例**（即使只新增一条也必须是数组）

- **单条新增**：  
  `full_code_path`: `/luobei/myapp/nps/nps_questionnaire_list.table`  
  `body`: `[{"title":"问卷A","description":"描述","target_group":"全部用户","start_time":1738339200000,"end_time":1738944000000}]`

- **批量新增**：  
  `full_code_path`: `/luobei/myapp/nps/nps_questionnaire_list.table`  
  `body`: `[{"title":"2025Q1 满意度","description":"第一季度调研","target_group":"全部用户","start_time":1738339200000,"end_time":1738944000000},{"title":"2025Q2 满意度","description":"第二季度调研","target_group":"全部用户","start_time":1741017600000,"end_time":1741622400000}]`

**传参说明**：键名与表格 **model 的 json 标签**一致；必填字段需包含，可选项可省略。**create_by、created_at、updated_at 由系统自动填充，无需在 body 中填写。** 业务时间字段（如 start_time、end_time）**必须为毫秒时间戳**（int64），禁止秒级。

**返回**：data_list（成功插入的每条记录）、created_count、failed_count、errors。

---

### 4. 提交表单（run_form_submit）：body 为 JSON 对象（不是数组）

**正确示例**：
- 无额外字段：`body`: `{}`
- 有字段（如 NPS 评分提交）：`body`: `{"questionnaire_id":1,"score":8,"comment":"满意"}`
- 收银台等（键与 Form 的 Request json 标签一致）：`body`: `{"name":"张三","amount":100}`（具体字段需 read_go_file 看对应 .go 里 Request 结构）

---

### 5. 查询图表（run_chart_query）：url_query 由该 Chart 的 Request 决定

**约定**：一个 Chart 路由一次只返回一张图；full_code_path 须到**具体图表函数名**（如 `/luobei/myapp/nps/nps_sales_trend_statistics.chart`），不能到包路径；多张图时每张图有各自的路由与 full_code_path，需分别调用 run_chart_query。

**正确做法**：用 read_go_file 查看对应 .go 里 Chart 的 **Request 结构**（form 标签），以确定可传参数名与取值。每个 Chart 的 Req 不同，没有统一格式。

**示例**：
- 无参数：不传 url_query 或传空。
- 有参数（如 Req 有 questionnaire_id、group_by）：`url_query`: `questionnaire_id=1&group_by=按天分组`（中文值会按 URL 编码处理）

---

### 6. 更新表格记录（run_table_update）：body 为 JSON 数组，每项含 id、updates

**格式**：body 必须为 **JSON 数组**，每项为 `{ "id": 行ID, "updates": { "字段名": 新值, ... } }`；单条也写一项数组。不传 old_values，由 app-server 自动查表填充。

**正确示例**：
- **单条更新**：`body`: `[{"id":1,"updates":{"status":"已处理","title":"新标题"}}]`
- **批量更新**：`body`: `[{"id":1,"updates":{"status":"已处理"}},{"id":2,"updates":{"status":"已关闭"}}]`

**返回**：updated_count（成功条数）、failed_count、data_list（每条更新接口返回结果）、errors（失败条目的 index 与 error）。

---

## 何时用什么工具（执行相关）

| 操作 | 工具 | 说明 |
|------|------|------|
| 查询列表数据 | run_table_search | 分页、排序、筛选（eq/like/in/gte/lte 等），url_query 见本文档易错点 |
| 新增表格记录 | run_table_create | body 为 JSON 数组，full_code_path 到具体 Table 函数 |
| 更新表格记录 | run_table_update | body 为 JSON 数组，每项含 id、updates，支持批量；old_values 由 app-server 自动填充 |
| 提交表单 | run_form_submit | body 为 JSON 对象，full_code_path 到具体 Form 函数 |
| 查询图表数据 | run_chart_query | url_query 由该 Chart 的 Req 决定，需看对应 .go |
| 删除列表行 | 见下 | 当前通过对应 Form 或前端表格操作完成；若后续提供 run_table_delete 则按工具文档调用 |

详细参数、示例、时间函数（Now()、Today()、Now(-7d) 等）见本文档易错点与表格。

---

## 如何获知路径与参数

1. **列表/图表/表单路径**：在工作区目录下 `read_dir`，看有哪些 `tables/xxx`、`plugins/xxx`、charts 等；full_code_path 一般为 `/用户/app/.../函数名`（如 `/luobei/myapp/nps/nps_questionnaire_list.table`）。**函数名一定带类型后缀**：`.table` = 表格列表，`.form` = 表单，`.chart` = 图表；看到后缀即可知该函数类型。
2. **Table 可搜字段与 Req 自定义字段**：`read_go_file` 打开该 Table 对应的 .go，看 **model 的 search 标签**（eq/like/in/gte/lte 等）和 List Req 的 **form 标签**（如 status）。
3. **Chart 参数**：`read_go_file` 打开该 Chart 对应的 .go，看 Chart 的 **Request 结构**（如 questionnaire_id、group_by）。
4. **Form 字段**：`read_go_file` 打开该 Form 对应的 .go，看 **Request 结构**的 json 标签，body 的键与 json 名一致。
