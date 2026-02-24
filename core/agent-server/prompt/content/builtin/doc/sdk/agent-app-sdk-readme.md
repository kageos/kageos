# Agent-App SDK 使用说明

本文档说明**框架的用法与能力**。完整业务示例（PRD + 代码）在案例文档中，按需 read_doc 对应路径即可。

---

## 一、定位与文档分工

- **本 SDK 文档**：框架怎么用——结构体与标签、Table/Form 模式、注册方式、目录约定。
- **案例文档**（`/builtin/doc/case_catalog/xxx`）：具体业务长什么样——PRD + 完整 Go 代码。系统消息中「可读的目录」会列出各案例路径与说明；需要单表 CRUD、多表、Form、图表等时，read_doc 对应案例获取 PRD 与代码。
- **平台横切能力（禁止自己实现）**：权限管理、流程审批、评论/点赞/收藏、定时任务、操作记录——这些由平台统一提供，**禁止**在 PRD 中添加「审批状态/审批人/审批时间」等字段，**禁止**在代码中自己实现审批表/审批流程/权限判断/评论功能。业务代码只管业务数据本身。

---

## 二、快速开始

### Table 模式（单表 CRUD，GET）

1. **定义结构体**：业务字段加 `gorm`、`widget`、`search`、`validate` 等标签；主键、CreatedAt、DeletedAt 等系统字段按约定写。
2. **配置 TableTemplate**：`BaseConfig`（Name、Request、Response、CreateTables）+ `AutoCrudTable` + 可选 `OnTableAddRow` / `OnTableUpdateRow` / `OnTableDeleteRows`。
3. **写 List 函数**：请求体嵌 `*query.SearchFilterPageReq`，`resp.Table(&lists).AutoSearchFilterPaged(db, &Model{}, req.SearchFilterPageReq).Build()`。
4. **注册**：`init()` 中 `packageContext.GET("路由名", ListFunc, TableTemplate)`。

最小可用片段示例：

```go
// 结构体（系统字段 + 业务字段，此处省略系统字段）
type CrmTicket struct {
    Title    string `json:"title" gorm:"column:title" widget:"name:标题;type:input" search:"like" validate:"required,min=2,max=200"`
    Status   string `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:待处理,已完成;options_colors:warning,success;default:待处理" search:"in"`
    // ... ID, CreatedAt, DeletedAt 等见案例
}

var CrmTicketTemplate = &app.TableTemplate{
    BaseConfig:    app.BaseConfig{Name: "工单管理", CreateTables: []interface{}{&CrmTicket{}}},
    AutoCrudTable: &CrmTicket{},
}

func CrmTicketList(ctx *app.Context, resp response.Response) error {
    var req struct{ *query.SearchFilterPageReq }
    ctx.ShouldBind(&req)
    var lists []*CrmTicket
    return resp.Table(&lists).AutoSearchFilterPaged(ctx.GetGormDB(), &CrmTicket{}, req.SearchFilterPageReq).Build()
}

func init() {
    packageContext.GET("crm_ticket.table", CrmTicketList, CrmTicketTemplate)
}
```

单表完整示例（含所有常用组件与回调）：read_doc ` /builtin/doc/case_catalog/table/ticket`。

### Form 模式（POST，无 Table）

1. **定义请求/响应结构体**：字段加 `widget`、`validate`；请求体可含 files、input、select、table 等。
2. **写处理函数**：`ctx.ShouldBindValidate(&req)`，业务逻辑，`return resp.Form(&respStruct).Build()`；系统错误需加 `[系统错误]` 前缀并带详细参数（见第六节「系统错误」）。
3. **配置 FormTemplate**：`BaseConfig`（Name、Request、Response）+ 可选 `OnSelectFuzzyMap` 等。
4. **注册**：`init()` 中 `packageContext.POST("路由名", Handler, FormTemplate)`。

最小可用片段示例（使用 files 时需 import `github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types`）：

```go
import "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"

type ExcelOrCsvReq struct {
    File *types.Files `json:"file" widget:"name:上传文件;type:files" validate:"required"`
}
type ExcelOrCsvResp struct {
    Rows int `json:"rows" widget:"name:解析行数;type:number"`
}

func ExcelOrCsvHandler(ctx *app.Context, resp response.Response) error {
    var req ExcelOrCsvReq
    if err := ctx.ShouldBindValidate(&req); err != nil {
        return err
    }
    // 业务逻辑：解析文件，得到行数
    return resp.Form(&ExcelOrCsvResp{Rows: 100}).Build()
}

func init() {
    packageContext.POST("excel_or_csv.form", ExcelOrCsvHandler, &app.FormTemplate{
        BaseConfig: app.BaseConfig{Name: "Excel/CSV 解析", Request: &ExcelOrCsvReq{}, Response: &ExcelOrCsvResp{}},
    })
}
```

单 Form 完整示例：read_doc ` /builtin/doc/case_catalog/form/excelorcsv`。

### Chart 模式（GET，统计/图表）

**⚠️ 一个 GET 路由只能返回一张图表**，多张图时每张单独一个路由。图表只支持 4 种类型（`LineChart`/`BarChart`/`PieChart`/`GaugeChart`），详见第七节「图表类型说明」。

1. **定义请求结构体**：筛选条件加 `widget` 标签。
2. **写统计函数**：`ctx.ShouldBind(&req)` → 查库聚合 → 构造具体图表类型（只填 Title、XAxis、Series、Metadata，**无需填 ChartType 或 Series[].Type**）→ `return resp.Chart(chart).Build()`。
3. **配置 ChartTemplate**：`BaseConfig`（Name、Request、Response 填**与返回值一致的具体类型**，如 `Response: &types.LineChart{}`）。
4. **注册**：`init()` 中 `packageContext.GET("路由名", ChartHandler, ChartTemplate)`。

最小可用片段示例：

```go
type SalesStatisticsReq struct {
    StartTime int64  `json:"start_time" form:"start_time" widget:"name:开始时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss"`
    EndTime   int64  `json:"end_time" form:"end_time" widget:"name:结束时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss"`
}

func SalesTrendChart(ctx *app.Context, resp response.Response) error {
    var req SalesStatisticsReq
    if err := ctx.ShouldBind(&req); err != nil { return err }
    db := ctx.GetGormDB()
    // 聚合查询得到 dateLabels、seriesData...
    chart := &types.LineChart{
        Title:  "销售趋势",
        XAxis:  dateLabels,
        Series: []types.ChartSeries{{Name: "销售额", Data: seriesData}},
        Metadata: map[string]interface{}{"总销售额": total},
    }
    return resp.Chart(chart).Build()
}

func init() {
    packageContext.GET("sales_trend_statistics.chart", SalesTrendChart, &app.ChartTemplate{
        BaseConfig: app.BaseConfig{Name: "销售趋势", Request: &SalesStatisticsReq{}, Response: &types.LineChart{}},
    })
}
```

完整 Chart 示例（折线图/饼图/仪表盘、时间筛选、多图表同包）：read_doc ` /builtin/doc/case_catalog/form_table_chart/cashier`。

---

## 三、结构体与标签

### 1. widget 标签

格式：`widget:"name:显示名;type:组件类型;配置项:值"`。常用配置：`default`、`options`、`options_colors`、`min`/`max`/`step`/`unit`、`format`、`precision` 等。

**select / multiselect 与 options_colors（必填）**：**使用 select 或 multiselect 时，务必同时配置 `options_colors`**，与 `options` 一一对应（逗号分隔，顺序一致），前端会用颜色标签区分选项，不填则难以区分。支持**预设**：`default`、`primary`、`success`、`warning`、`danger`、`info`；也支持**自定义十六进制颜色**，如 `#FF9800` 橙色、`#9C27B0` 紫色、`#4CAF50` 绿色，同一颜色可重复使用。示例：`options:待处理,进行中,已完成` 对应 `options_colors:warning,primary,success`；自定义颜色示例：`options:VIP,普通,体验` 对应 `options_colors:#E91E63,#9E9E9E,#4CAF50`。

片段示例：

```go
Title    string `widget:"name:标题;type:input" search:"like" validate:"required,min=2,max=200"`
Status   string `widget:"name:状态;type:select;options:待处理,已完成;options_colors:warning,success;default:待处理" search:"in" validate:"oneof=待处理 已完成"`
Priority string `widget:"name:优先级;type:select;options:低,中,高;options_colors:success,warning,danger;default:中"`
Handler  string `widget:"name:处理人;type:user;default:Me()" search:"in"`
CreateBy string `widget:"name:创建用户;type:user" permission:"read"`  // 只读，在 OnTableAddRow 里 ctx.GetRequestUser() 赋值
Tags     string `widget:"name:标签;type:multiselect;options:紧急,重要;options_colors:danger,warning" search:"contains"`  // 多选用 string 逗号分隔，须配 options_colors
Level    string `widget:"name:级别;type:select;options:VIP,普通,体验;options_colors:#E91E63,#9E9E9E,#4CAF50;default:普通"`  // 自定义颜色示例：十六进制
Progress int    `widget:"name:进度;type:slider;min:0;max:100;unit:%" search:"gte,lte"`
Deadline int64  `widget:"name:截止时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte"`
Attachment *types.Files `gorm:"type:json" widget:"name:附件;type:files"`
```

| 组件类型 | 说明 | 典型用法 |
|----------|------|----------|
| input | 单行文本 | 标题、电话、邮箱 |
| text_area | 多行文本 | 描述、备注 |
| richtext | 富文本 | 详细内容 |
| select | 下拉单选 | 状态、优先级；**须配 options_colors**（与 options 顺序一致），前端用颜色区分选项 |
| radio | 单选 | 来源、性别（2–5 个选项） |
| multiselect | 多选 | 标签（string，逗号分隔）；**须配 options_colors**（与 options 顺序一致） |
| number | 整数 | 数量、工时 |
| float | 小数 | 价格、金额 |
| slider | 滑块 | 进度、评分（min/max/step/unit） |
| rate | 星级 | 评价（max、allow_half、texts） |
| switch | 开关 | 是否启用 |
| timestamp | 日期时间 | 创建时间、截止时间；**严格要求毫秒时间戳**（见下「timestamp 组件约定」） |
| color | 颜色 | format:hex，default:#xxx |
| files | 文件上传/文件下载 | 字段类型 `*types.Files`，gorm `type:json`；**需 import** `github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types` |
| user / users | 用户选择 | default:Me()、Me(),MyLeader() 等 |
| department / departments | 部门选择 | default:MyDepartment()，max_count 等 |
| table | 子表（Form 请求） | 数组结构体，可配 OnSelectFuzzy |
| form | 子表单（Form 响应） | 嵌套结构体展示 |
| link | 跳转链接 | 列表/表单中跳转到另一 GET 或 Form、或外链；不落库，后端 BuildFunctionUrlWithText 赋值 |

**timestamp 组件约定（必读）**：**timestamp 组件严格要求使用毫秒时间戳（毫秒级 Unix 时间戳），禁止使用秒级时间戳。** 后端无需在代码里做日期格式化，直接使用 **int64** 类型存、传**毫秒时间戳**即可，前端会根据 widget 的 `format` 自动格式化展示；若误用秒级时间戳，前端展示、筛选、排序会错误。

- **正确**：`BidTime int64 \`json:"bid_time" widget:"name:出价时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss"\`` —— 字段类型为 int64，值为毫秒时间戳，后端只读写时间戳，不转字符串。
- **错误**：`BidTime string \`json:"bid_time" widget:"name:出价时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss"\`` —— 不要用 string 类型，也不要后端格式化成 "YYYY-MM-DD HH:mm:ss" 等字符串；timestamp 的 format 仅用于前端展示，后端只返回时间戳。
- **错误**：使用秒级时间戳（如 `time.Now().Unix()`）—— 必须用毫秒级（如 `time.Now().UnixMilli()` 或 gorm 的 `autoCreateTime:milli` / `autoUpdateTime:milli`）。

**files 类型约定**：使用 `type:files` 时字段类型必须为 `*types.Files`，需在文件顶部 **import** 包：`import "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"`。否则会编译报错「undefined: types」。完整上传、下载与存储流程见第六节「文件上传、下载与存储」。

#### link 组件（跳转链接）

用于在**列表**或**表单**中展示可点击链接，点击后跳转到另一个 GET（Table/Chart）、Form，或打开外链。字段通常**不落库**（`gorm:"-"`）、**只读**（`permission:"read"`），值由后端在 **List 函数 Build 之后**或 **Form 响应**里用 `ctx.BuildFunctionUrlWithText(target, params, linkText)` 赋值。

- **widget 配置**：`type:link`；可选 `target:_blank`（新窗口）或 `_self`（当前窗口）；可选 `text`、`type`（样式 primary/success 等）、`icon`。
- **赋值 API**：`ctx.BuildFunctionUrlWithText(target string, params interface{}, linkText string) (string, error)`  
  - **target**：函数路径（如 `"meeting_room_list.table"`、`"vote_submit.form"`、`"vote_result.form"`），或带查询（如 `"hr_resume_list.table?_tab=OnTableAddRow"` 表示打开该列表并切到「新增」Tab），或**外链**（如 `"https://example.com"`）。路由名需带类型后缀（.table / .form / .chart），与注册约定一致。  
  - **params**：见下「params 类型约定」；外链时传 `nil`。  
  - **linkText**：链接展示文本（如「查看会议室详情」「点击参与投票」「查看投票结果」）。

- **params 类型约定（必读，不可混用）**：  
  - **跳转到 Table（GET 列表）**：params 必须是**目标 Table 对应的列表 Model**，即该 GET 路由的 `AutoCrudTable` 指向的结构体。前端打开列表时会用 params 的字段（如 ID）做筛选/定位。  
    - 例：target 为 `"meeting_room_list.table"` 时，params 用 `MeetingRoom{ID: roomID}`，其中 **MeetingRoom** 是会议室表（meeting_room_list.table）的 **Model**，不能写成别的结构体。  
    - 例：target 为 `"hr_resume_list.table?_tab=OnTableAddRow"` 时，params 用 `HrJob{ID: jobID}`（职位表的 Model），打开简历列表并预填职位 ID。  
  - **跳转到 Form（POST 表单）**：params 必须是**目标 Form 的请求结构体**，即该 POST 路由的 `Request` 结构体。前端打开表单时会预填 params 的字段。  
    - 例：target 为 `"vote_result.form"` 时，params 用 `VoteResultReq{TopicID: topicID}`，其中 **VoteResultReq** 是查看结果 Form 的 **请求结构体**，不能写成 VoteTopic（Model）。  
    - 例：target 为 `"vote_submit.form"` 时，params 用 `VoteSubmitReq{TopicID: topicID}`（提交投票 Form 的请求结构体）。  
  - **外链**：params 传 `nil`。  
  - 总结：**跳 Table 用该表的 Model，跳 Form 用该 Form 的 Request 结构体**，二者不要搞混。

- **典型场景**：  
  1. **Table 列表「查看详情」列**：当前行关联另一张表，链接跳转到该表并带上当前行 ID（如预约列表的「会议室详情」→ 跳会议室列表，params 用 **MeetingRoom{ID: RoomID}**，MeetingRoom 是目标表的 Model）。  
  2. **Table 列表「操作」列**：根据状态动态生成链接（如投票主题列表「投票操作」→ 跳 `vote_submit` 用 **VoteSubmitReq**，跳 `vote_result` 用 **VoteResultReq**；职位列表「投递简历」→ 跳简历列表用 **HrJob{ID: JobID}** 并 `_tab=OnTableAddRow`）。  
  3. **Form 响应**：提交后返回一个「查看结果」链接（如投票提交后返回「查看投票结果」，params 用 **VoteResultReq{TopicID: req.TopicID}**）。

```go
// Table 列表：不落库、只读，List Build 之后对每条记录赋值
RoomLink string `json:"room_link" gorm:"-" widget:"name:会议室详情;type:link;target:_blank" permission:"read"`

// List 函数内，Build 之后：跳转到 Table 必须用目标表的 Model
// meeting_room_list.table 的 AutoCrudTable 是 MeetingRoom，故 params 用 MeetingRoom{ID: ...}
for i := range bookings {
    params := MeetingRoom{ID: bookings[i].RoomID}  // MeetingRoom 是会议室表的 Model
    bookings[i].RoomLink, _ = ctx.BuildFunctionUrlWithText("meeting_room_list.table", params, "查看会议室详情")
}

// 带 _tab 参数：打开列表并切到「新增」Tab（如投递简历），params 用目标表 Model
params := HrJob{ID: jobs[i].ID}  // HrJob 是职位表的 Model
jobs[i].ApplyLink, _ = ctx.BuildFunctionUrlWithText("hr_resume_list.table?_tab=OnTableAddRow", params, "投递简历")

// Form 响应：跳转到 Form 必须用该 Form 的请求结构体
// vote_result.form 的 Request 是 VoteResultReq，故 params 用 VoteResultReq{TopicID: ...}
params := VoteResultReq{TopicID: req.TopicID}
functionLink, _ := ctx.BuildFunctionUrlWithText("vote_result.form", params, "查看投票结果")
return resp.Form(&VoteSubmitResp{..., FunctionLink: functionLink}).Build()
```

完整示例：read_doc `/builtin/doc/case_catalog/tables/meeting`（预约列表会议室详情 link）、`/builtin/doc/case_catalog/tables/hr`（职位/简历列表 link、_tab=OnTableAddRow）、`/builtin/doc/case_catalog/formandtable/vote`（投票操作/选项列表/提交结果 link）。

- **隐藏字段**：`widget:"-"` 表示该字段**被前端直接忽略**，不参与列表/表单的渲染，也不会被提交；常用于系统字段（如 DeletedAt、DeletedBy）或内部关联（如 `json:"-"` 的关联表）。
- **只读/仅创建等**：用 `permission` 控制，见下节。

### 2. validate 标签

遵循 `github.com/go-playground/validator/v10`。常用：`required`、`min`/`max`、`oneof=值1 值2`（空格分隔，值含空格用单引号）、`email` 等。

```go
Title string `validate:"required,min=2,max=200"`
Status string `validate:"required,oneof=待处理 处理中 已完成"`
Email string `validate:"required,email"`
```

### 3. search 标签

**有搜索需求的字段必须加上 `search` 标签，并配上适合的搜索方式。** 只有配了 `search` 标签的字段才支持 Table 列表的搜索/筛选；不配 `search` 的字段不支持搜索，前端不会出现该字段的搜索条件。

| 值 | 含义 | 适用 |
|----|------|------|
| like | 模糊 | input、text_area |
| in | 精确 IN | select、radio、user、department |
| contains | FIND_IN_SET | multiselect、users、departments |
| eq | 精确 = | ID、switch |
| gte,lte | 范围 | timestamp、number、float、slider |

示例：需要支持搜索的字段都配上 `search`，未配的字段列表里不可搜。系统字段（ID、创建时间、更新时间）若有搜索需求也要配；参考工单等 Table 结构体。

```go
type CrmTicket struct {
    // 系统字段：仅列表展示，新增/修改表单不展示；配 search 后列表可搜索
    ID        int   `json:"id" gorm:"primaryKey;column:id" widget:"name:ID;type:ID" permission:"read" search:"eq"`           // 仅列表展示、不可编辑，列表支持按 ID 精确搜索
    CreatedAt int64 `json:"created_at" gorm:"autoCreateTime:milli;column:created_at" widget:"name:创建时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" permission:"read"` // 仅列表展示、不可编辑，列表支持按创建时间范围搜索
    UpdatedAt int64 `json:"updated_at" gorm:"autoUpdateTime:milli;column:updated_at" widget:"name:更新时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" permission:"read"` // 仅列表展示、不可编辑，列表支持按更新时间范围搜索
    // 软删除：gorm.DeletedAt + widget:"-" 不在前端展示，GORM 查询时自动过滤已删除记录
    DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"` // 不做展示
    DeletedBy string         `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`       // 删除操作人，不做展示（可选）

    // 业务字段：配 search 的列表可搜索，未配则不可搜（展示/可编辑由 permission 或不设置决定，见下方 permission 标签）
    Title       string `json:"title" gorm:"column:title" widget:"name:工单标题;type:input" search:"like"`           // 列表支持模糊搜索
    Description string `json:"description" gorm:"column:description" widget:"name:问题描述;type:text_area" search:"like"` // 列表支持模糊搜索
    Priority    string `json:"priority" gorm:"column:priority" widget:"name:优先级;type:select;options:低,中,高;options_colors:success,warning,danger" search:"in"`   // 列表支持精确筛选
    Status      string `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:待处理,处理中,已完成;options_colors:info,warning,success" search:"in"` // 列表支持精确筛选
    IsUrgent    bool   `json:"is_urgent" gorm:"column:is_urgent" widget:"name:是否紧急;type:switch" search:"eq"`   // 列表支持精确筛选
    Progress    int    `json:"progress" gorm:"column:progress" widget:"name:完成进度;type:slider;min:0;max:100;unit:%" search:"gte,lte"` // 列表支持范围搜索
    Handler     string `json:"handler" gorm:"column:handler" widget:"name:处理人;type:user" search:"in"`           // 列表支持精确筛选
    CcUsers     string `json:"cc_users" gorm:"column:cc_users" widget:"name:抄送人;type:users" search:"contains"`  // 列表支持 FIND_IN_SET 搜索
    Deadline    int64  `json:"deadline" gorm:"column:deadline" widget:"name:截止时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte"` // 列表支持范围搜索
    Remark      string `json:"remark" gorm:"column:remark" widget:"name:备注;type:text_area"`                     // 未配 search，列表不可搜索
}
```

### 4. permission 标签

用于控制字段在**新增表单、更新表单、列表**中的展示与编辑权限，务必按业务需要设置，避免用户改不该改的字段。

| 权限值 | 新增表单 | 更新表单 | 列表展示 | 适用场景 |
|--------|----------|----------|----------|----------|
| `read` | ❌ 不展示可编辑 | ❌ 不展示可编辑 | ✅ 展示 | 仅展示、后端赋值或计算字段 |
| `create` | ✅ 可编辑 | ❌ 不展示 | ❌ 不展示 | 仅创建时填写，编辑/列表不可改 |
| `update` | ❌ 不展示 | ✅ 可编辑 | ❌ 不展示 | 仅编辑时可改 |
| `create,update` | ✅ 可编辑 | ✅ 可编辑 | ❌ 不展示 | 敏感信息，列表不展示 |
| 不设置 | ✅ 可编辑 | ✅ 可编辑 | ✅ 展示 | 普通业务字段 |

#### permission 最佳实践场景

**场景 1：仅列表展示、不落库的计算字段（permission: read + gorm:"-"）**

列表需要展示「剩余时间」等由后端根据其它字段计算出的值，不落库、表单不需要编辑。用 `gorm:"-"` 不落库，`permission:"read"` 仅列表展示。

```go
RemainingTime string `json:"remaining_time" gorm:"-" widget:"name:剩余时间;type:text" permission:"read"`
```

在 List 函数 Build 之后遍历 `lists`，根据截止时间等计算并赋值给 `RemainingTime`。

**场景 2：仅展示、由后端在回调中赋值的字段（permission: read）**

创建人、提单部门等只读，不允许用户在前端选择，必须在 OnTableAddRow 里用 `ctx.GetRequestUser()`、`ctx.GetRequestUserDept()` 等赋值。

```go
Department string `json:"department" gorm:"column:department" widget:"name:提单部门;type:department" search:"in" permission:"read"`
CreateBy   string `json:"create_by" gorm:"column:create_by" widget:"name:创建用户;type:user" search:"in" permission:"read"`
```

在 OnTableAddRow 中：`row.Department = ctx.GetRequestUserDept()`；`row.CreateBy = ctx.GetRequestUser()`。

**场景 3：仅新增时可编辑、编辑/列表不展示（permission: create）**

某些字段只在「新增」时填写，编辑时不允许改（如投票主题的选项列表，创建后不可改）。用 `permission:"create"`，新增表单可编辑，更新表单和列表不展示该字段。

```go
Options []VoteOptionItem `json:"options" gorm:"-" widget:"name:投票选项;type:table" permission:"create" validate:"required,min=2"`
```

**场景 4：仅编辑时可改、新增/列表不展示（permission: update）**

某些字段只在「更新」时填写，新增时没有或不需要填。例如：实际完成时间（创建时未知）、关闭原因/处理备注（仅在结单或更新时填）、审核意见（仅审核人在更新时填）。用 `permission:"update"`，更新表单可编辑，新增表单和列表不展示该字段。

```go
ClosedReason  string `json:"closed_reason" gorm:"column:closed_reason" widget:"name:关闭原因;type:text_area" permission:"update"`
FinishedAt   int64  `json:"finished_at" gorm:"column:finished_at" widget:"name:实际完成时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" permission:"update"`
```

**场景 5：新增和更新都可编辑、但列表不展示（permission: create,update）**

敏感或内部信息需要在新/改表单里编辑，但不在列表里展示，避免列表信息过载或泄露。例如：内部备注、成本价、二次确认密码等。用 `permission:"create,update"`，新增和更新表单都可编辑，列表不展示。

```go
InternalNote string `json:"internal_note" gorm:"column:internal_note" widget:"name:内部备注;type:text_area" permission:"create,update"`
CostPrice    float64 `json:"cost_price" gorm:"column:cost_price" widget:"name:成本价;type:float;precision:2;unit:元" permission:"create,update"`
```

**小结**：`read` = 只展示不编辑（含计算字段与后端赋值字段）；`create` = 仅新增时填；`update` = 仅编辑时填；`create,update` = 新/改可编辑、列表不展示；不设 = 正常可编辑且列表展示。配合 `widget:"-"` 可完全隐藏字段。

---

## 四、Table 模式要点

- **TableTemplate**：`BaseConfig` 含 Name、Request、Response、CreateTables；`AutoCrudTable` 指向列表结构体；可选 `OnTableAddRow`、`OnTableUpdateRow`、`OnTableDeleteRows`；若新增/编辑表单中有 select 需后端动态选项，配 `OnSelectFuzzyMap`（用法见「六、Form 模式要点 → OnSelectFuzzy」）。
- **AutoCrudTable 的 model 可落库字段类型**：model 里凡是有 **gorm 列**（会被 GORM 写入数据库）的字段，**只能是**以下可落库类型：**基础类型**（int、string、bool、int64、float64 等）、**files.Files**（`gorm:"type:json"`）、**gorm.DeletedAt**（软删除，GORM 特例）。除此以外，**其他 struct、slice（如 type:table / type:form）不能作为一列写入数据库**；若在 model 里出现这类 struct/slice，须为：**外键关联**（如 `Room *MeetingRoom` 配 `gorm:"foreignKey:RoomID;references:ID"`，实际存的是 RoomID，不占一列）或 **gorm:"-"**（不落库，仅展示/表单用，如 RoomName、Status、Options、link 等）。否则 GORM 无法把该列写进数据库。
- **List 函数**：请求体包含 `*query.SearchFilterPageReq`，使用 `resp.Table(&lists).AutoSearchFilterPaged(db, &Model{}, req.SearchFilterPageReq).Build()`；Build 后可在内存中给计算字段赋值（如剩余时间、**link 跳转 URL**，见「三、结构体与标签 → link 组件」）。
- 主键、CreatedAt、UpdatedAt、DeletedAt、DeletedBy 等系统字段约定见案例；init_.go 由脚手架生成，不要手写。

完整 Table 示例（单表/多表/回调/OnSelectFuzzy/link）：read_doc ` /builtin/doc/case_catalog/table/ticket`、`/builtin/doc/case_catalog/tables/meeting`、`/builtin/doc/case_catalog/tables/hr`。

---

## 五、Table 回调函数

Table 的三个回调用于新增、更新、删除时的业务逻辑；可选配置，不配则走框架默认行为。

### 1. OnTableAddRow（新增行）

- **作用**：绑定并校验请求体，填充只读字段（如创建人、部门），落库，可选调用第三方。
- **关键 API**：`ctx.ShouldBindValidate(&row)`、`ctx.GetRequestUser()`、`ctx.GetRequestUserDept()`、`ctx.GetGormDB()`。
- **返回**：`&callback.OnTableAddRowResp{Data: &row}`。

```go
OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
    db := ctx.GetGormDB()
    var row CrmTicket
    if err := ctx.ShouldBindValidate(&row); err != nil {
        return nil, err
    }
    row.CreateBy = ctx.GetRequestUser()
    row.Department = ctx.GetRequestUserDept()
    if err := db.Create(&row).Error; err != nil {
        return nil, err
    }
    return &callback.OnTableAddRowResp{Data: &row}, nil
}
```

### 2. OnTableUpdateRow（更新行）

- **作用**：只更新变更字段，支持零值（空字符串、0）；可用 `req.IsFieldUpdated("字段名")` 做状态流转、自动计算等。
- **关键 API**：`req.BindUpdates(&updateFields)`、`req.GetUpdates()`、`req.GetId()`、`req.IsFieldUpdated("fieldName")`。
- **注意**：必须用 `db.Model(&Model{}).Where("id = ?", req.GetId()).Updates(updates)` 更新，以便支持零值；本回调不校验 validate（仅部分字段更新）。

```go
OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
    db := ctx.GetGormDB()
    var updateFields CrmTicket
    if err := req.BindUpdates(&updateFields); err != nil {
        return nil, err
    }
    updates := req.GetUpdates()
    if req.IsFieldUpdated("status") && updateFields.Status == "已关闭" {
        var current CrmTicket
        if err := db.First(&current, req.GetId()).Error; err != nil {
            return nil, err
        }
        durationMinutes := float64(time.Now().UnixMilli()-current.CreatedAt) / 1000 / 60
        updates["handle_duration"] = float64(int(durationMinutes*100+0.5)) / 100
    }
    err := db.Model(&CrmTicket{}).Where("id = ?", req.GetId()).Updates(updates).Error
    if err != nil {
        return nil, err
    }
    return &callback.OnTableUpdateRowResp{}, nil
}
```

### 3. OnTableDeleteRows（删除行）

- **作用**：批量删除；推荐软删除并记录 `deleted_by`、`deleted_at`，便于恢复与审计。
- **关键 API**：`req.GetIds()`、`ctx.GetRequestUser()`。

```go
OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
    db := ctx.GetGormDB()
    err := db.Model(&CrmTicket{}).Where("id in (?)", req.GetIds()).Updates(map[string]interface{}{
        "deleted_by": ctx.GetRequestUser(),
        "deleted_at": time.Now(),
    }).Error
    if err != nil {
        return nil, err
    }
    return &callback.OnTableDeleteRowsResp{}, nil
}
```

### 4. List 函数：Build 前处理与 Build 后处理

List 函数可在 **Build 之前** 和 **Build 之后** 两处做自定义处理：

- **Build 之前**：在调用 `AutoSearchFilterPaged` 之前，对 `queryDB` 做 Where（外表筛选、计算字段的筛选条件）、Preload 等，再传入 `AutoSearchFilterPaged(queryDB, ...)`。
- **Build 之后**：对返回的 `lists` 逐条做计算、填充不落库字段（如剩余时间、状态、关联表名称、link URL）等。

下面先给一个**仅后处理**的最小示例（剩余时间），再给一个**前处理 + 后处理**的示例（会议室预约：外表/状态筛选 + 填充会议室名称/状态/link）。

```go
// 结构体：ID、标题、截止时间（落库），剩余时间（不落库，仅展示）
type Task struct {
    ID             int    `json:"id" gorm:"primaryKey;autoIncrement" widget:"name:ID;type:ID" permission:"read"`
    Title          string `json:"title" gorm:"column:title" widget:"name:标题;type:input" search:"like"`
    Deadline       int64  `json:"deadline" gorm:"column:deadline" widget:"name:截止时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte"`
    RemainingTime  string `json:"remaining_time" gorm:"-" widget:"name:剩余时间;type:input" permission:"read"` // gorm:"-" 不落库
}

func TaskList(ctx *app.Context, resp response.Response) error {
    var req struct{ *query.SearchFilterPageReq }
    ctx.ShouldBind(&req)
    db := ctx.GetGormDB()
    var lists []*Task
    if err := resp.Table(&lists).AutoSearchFilterPaged(db, &Task{}, req.SearchFilterPageReq).Build(); err != nil {
        return err
    }
    // Build 之后：按截止时间计算「剩余时间」展示
    now := time.Now().UnixMilli()
    for _, item := range lists {
        if item.Deadline <= 0 {
            item.RemainingTime = "-"
            continue
        }
        if now >= item.Deadline {
            item.RemainingTime = "已过期"
            continue
        }
        diffMs := item.Deadline - now
        d, h := diffMs/86400/1000, (diffMs/3600/1000)%24
        if d > 0 {
            item.RemainingTime = fmt.Sprintf("%d天%d小时", d, h)
        } else {
            item.RemainingTime = fmt.Sprintf("%d小时", h)
        }
    }
    return nil
}
```

要点：计算字段用 `gorm:"-"`，不写库；`permission:"read"` 表示仅列表展示、表单不编辑。

**示例二：Build 前处理 + 后处理（会议室预约）**

请求里包含**外表筛选**（会议室名称）和**计算字段筛选**（预约状态：待开始/进行中/已结束，由开始/结束时间与当前时间算出）。需在 Build 前对 `queryDB` 做 Where；Build 后填充不落库字段（会议室名称、状态、详情 link）。参考：read_doc `/builtin/doc/case_catalog/tables/meeting`（见 meeting_room_booking.go）。

```go
// 列表结构体：RoomName、Status、RoomLink 为不落库展示字段（gorm:"-"）
type MeetingRoomBooking struct {
    ID        int    `json:"id" gorm:"primaryKey;column:id" widget:"name:预约ID;type:ID" permission:"read" search:"eq"`
    RoomID    int    `json:"room_id" gorm:"column:room_id" widget:"name:会议室;type:select" callback:"OnSelectFuzzy"`
    Room      *MeetingRoom `json:"-" gorm:"foreignKey:RoomID"`
    RoomName  string `json:"room_name" gorm:"-" widget:"name:会议室名称;type:text" permission:"read"`   // 后处理从 Room 取
    RoomLink  string `json:"room_link" gorm:"-" widget:"name:会议室详情;type:link" permission:"read"`  // 后处理 BuildFunctionUrlWithText
    StartTime int64  `json:"start_time" gorm:"column:start_time" widget:"name:开始时间;type:timestamp" search:"gte,lte"`
    EndTime   int64  `json:"end_time" gorm:"column:end_time" widget:"name:结束时间;type:timestamp" search:"gte,lte"`
    Status    string `json:"status" gorm:"-" widget:"name:预约状态;type:select;options:待开始,进行中,已结束;options_colors:info,primary,success" permission:"read"` // 后处理按时间计算
}

// 列表请求：RoomName、Status 为筛选条件，非表字段，需在 List 内手写 Where
type MeetingRoomBookingListReq struct {
    RoomName string `json:"room_name" form:"room_name"` // 按会议室名称模糊查
    Status   string `json:"status" form:"status"`       // 按预约状态筛选（待开始/进行中/已结束）
    query.SearchFilterPageReq
}

func MeetingRoomBookingList(ctx *app.Context, resp response.Response) error {
    db := ctx.GetGormDB()
    var req MeetingRoomBookingListReq
    if err := ctx.ShouldBind(&req); err != nil { return err }

    queryDB := db.Model(&MeetingRoomBooking{})

    // Build 前处理 1：按会议室名称筛选（查外表得 roomIDs，再 Where room_id IN ?）
    if req.RoomName != "" {
        var roomIDs []int
        if err := db.Model(&MeetingRoom{}).Where("name LIKE ?", "%"+req.RoomName+"%").
            Pluck("id", &roomIDs).Error; err == nil && len(roomIDs) > 0 {
            queryDB = queryDB.Where("room_id IN ?", roomIDs)
        } else {
            return resp.Table(&[]MeetingRoomBooking{}).Build()
        }
    }

    // Build 前处理 2：按预约状态筛选（计算字段，用 start_time/end_time 与当前时间比较）
    if req.Status != "" {
        now := time.Now().UnixMilli()
        switch req.Status {
        case "待开始": queryDB = queryDB.Where("start_time > ?", now)
        case "进行中": queryDB = queryDB.Where("start_time <= ? AND end_time > ?", now, now)
        case "已结束": queryDB = queryDB.Where("end_time <= ?", now)
        }
    }

    queryDB = queryDB.Preload("Room")
    var bookings []MeetingRoomBooking
    if err := resp.Table(&bookings).AutoSearchFilterPaged(queryDB, &MeetingRoomBooking{}, &req.SearchFilterPageReq).Build(); err != nil {
        return err
    }

    // Build 后处理：填充不落库字段（会议室名称、状态、详情 link）
    for i := range bookings {
        if bookings[i].Room != nil {
            bookings[i].RoomName = bookings[i].Room.Name
        }
        bookings[i].Status = calculateBookingStatus(bookings[i].StartTime, bookings[i].EndTime)
        bookings[i].RoomLink, _ = ctx.BuildFunctionUrlWithText("meeting_room_list", MeetingRoom{ID: bookings[i].RoomID}, "查看会议室详情")
    }
    return nil
}

func calculateBookingStatus(startTime, endTime int64) string {
    now := time.Now().UnixMilli()
    if now < startTime { return "待开始" }
    if now < endTime { return "进行中" }
    return "已结束"
}
```

要点：**前处理**用自定义 `queryDB`（外表 Where、计算字段 Where、Preload）再传 `AutoSearchFilterPaged(queryDB, ...)`；**后处理**在 Build 之后遍历 `lists` 填 `RoomName`、`Status`、`RoomLink` 等不落库字段。

---

## 六、Form 模式要点

- **请求/响应结构体**：字段加 `widget`、`validate`；请求可含 `type:table`（子表）、`type:select` + `callback:"OnSelectFuzzy"` 等。
- **处理函数**：`ctx.ShouldBindValidate(&req)`；业务逻辑；成功 `return resp.Form(&respStruct).Build()`。涉及文件读写时见下「文件上传、下载与存储」。
- **FormTemplate**：`BaseConfig` 含 Name、Request、Response；若请求中有下拉需联动后端数据，配 `OnSelectFuzzyMap`。
- **注册**：`packageContext.POST("路由名", Handler, FormTemplate)`。

#### 系统错误（必读）

系统错误（数据库异常、网络超时、Python/外部调用失败、未预期的 panic 等）需要**统一加上 `[系统错误]` 前缀**，并带上**报错信息和详细参数**（如请求体 `req`），方便大模型定位和排查问题。

- **规范写法**：`return nil, fmt.Errorf("[系统错误]-[函数名] 简短描述, req: %+v, err: %w", req, err)`；打日志时同样加上 `[系统错误]-[函数名]` 并输出 req、err。
- **参考实现**：read_doc `/builtin/doc/case_catalog/form/nlp`（见 jieba_segment.go 中 `DoJiebaSegment` 的 Python 执行失败分支）。

```go
// 系统错误：必须带 [系统错误]、函数名、req 与 err，方便大模型排查
if err := executor.ExecuteJSON(ctx, &result); err != nil {
    logger.Errorf(ctx, "[系统错误]-[DoJiebaSegment] Python 执行失败, req: %+v, err: %v", req, err)
    return nil, fmt.Errorf("[系统错误]-[DoJiebaSegment] 执行中文分词失败, req: %+v, err: %w", req, err)
}
return resp.Form(&respStruct).Build()
```

#### 文件上传、下载与存储

- **上传**：请求或 Table 新增/编辑里用 `*types.Files` 字段，widget `type:files`；可选 `accept:.csv`、`max_size:50MB`、`max_count:10` 等。**Table 模式**下该字段落库用 `gorm:"column:xxx;type:json"`，Create/Update 时直接写入 model 即可，框架负责存储与列表/详情展示、下载。
- **读上传的文件（Form 内）**：需要访问文件内容时（如解析 CSV、转 Excel），用 `fs := ctx.GetFS()`，`inputFiles := fs.DownloadFiles(req.xxx)` 得到带本地路径的 `*types.Files`；遍历 `inputFiles.GetFiles()`，用 `file.LocalPath`（如 `os.Open(file.LocalPath)`）读内容；用完后 **必须** `defer fs.RemoveFiles(inputFiles)` 清理临时文件。
- **响应里返回文件（供下载）**：业务生成文件到本地路径后，用 `outputFiles := fs.ResponseFiles([]string{outputPath})` 得到 `*types.Files` 填到响应结构体，前端即可下载；用完后可 `defer fs.RemoveFiles(outputFiles)`。若无上传、仅生成文件给用户（如 CSV 文本转 Excel），可先用 `ctx.GetFS().GetTraceOutputDir()` 得到当前 Trace 输出目录，在该目录下生成文件再 `ResponseFiles`。
- **参考实现**：Table 存储文件字段：read_doc `/builtin/doc/case_catalog/tables/hr`（见 hr_resume_list.go 的 `ResumeFile`）；Form 上传读文件 + 响应返回文件：read_doc `/builtin/doc/case_catalog/form/excelorcsv`（见 `DoCsvToExcel`、`DoCsvTextToExcel`）。

```go
// Form 内：读上传文件 → 处理 → 返回生成的文件
func DoCsvToExcel(ctx *app.Context, req *CsvToExcelReq) (*CsvToExcelResp, error) {
    fs := ctx.GetFS()
    inputFiles := fs.DownloadFiles(req.InputFiles)
    defer fs.RemoveFiles(inputFiles)

    var outputFilePaths []string
    for _, file := range inputFiles.GetFiles() {
        if file.LocalPath == "" { continue }
        outPath, err := csvToExcel(ctx, file.LocalPath)
        if err != nil { /* 记录错误 */ continue }
        outputFilePaths = append(outputFilePaths, outPath)
    }

    var outputFiles *types.Files
    if len(outputFilePaths) > 0 {
        outputFiles = fs.ResponseFiles(outputFilePaths)
        defer fs.RemoveFiles(outputFiles)
    }
    return &CsvToExcelResp{OutputFiles: outputFiles, ...}, nil
}
```

```go
// Table 模式：文件字段落库，无需 GetFS/Download/Remove
type HrResume struct {
    ResumeFile *types.Files `json:"resume_file" gorm:"column:resume_file;type:json" widget:"name:简历附件;type:files"`
}
// OnTableAddRow/OnTableUpdateRow 里直接 db.Create(&row) / db.Updates(updates)，ResumeFile 会按 json 存储
```

#### OnSelectFuzzy（下拉联动后端数据）

当 **select** 或 **multiselect** 的选项需要从后端查库、按关键字模糊搜索或按业务条件过滤（如只显示「可用」会议室、只显示「上架」商品）时，使用 **OnSelectFuzzy**。前端在下拉里输入关键字或回显已选值时，会调用该回调，由后端返回选项列表。

- **适用**：Form 请求中的 select、Form 请求里 **table 子表**中的 select、**Table 模式**新增/编辑表单中的 select（如预约选会议室）。只要该 select 需要「后端动态选项」，就配 OnSelectFuzzy。
- **字段配置**：在字段上加 `callback:"OnSelectFuzzy"`；字段的 **json 名**（如 `product_id`、`member_id`、`room_id`）作为模板里 `OnSelectFuzzyMap` 的 key。
- **模板配置**：在 **FormTemplate** 或 **TableTemplate** 的 `BaseConfig` 里设置 `OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{"字段json名": handler}`，key 与请求/列表结构体里该字段的 json 名一致。
- **回调签名**：`func(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error)`。

**OnSelectFuzzyReq**：前端会传 `Code`（字段标识）、`Type`（`by_keyword` 用户输入关键字 / `by_value` 回显单个值 / `by_values` 回显多选值）、`Value`（关键字或已选值）。常用方法：`req.IsByKeyword()`、`req.IsByValue()`、`req.IsByValues()`；`req.Keyword()` 取关键字；`req.GetValue()`、`req.GetValues()` 取已选值（用于回显时查库）。

**OnSelectFuzzyResp**：返回 `Items []*SelectFuzzyItem`（每项含 `Value`、`Label`、可选 `DisplayInfo` 供详情展示）；`MaxSelections`（0 表示不限制，1 表示单选）；可选 `Statistics map[string]interface{}`，用于在表单旁**聚合展示**（见下「Statistics 与聚合计算」）。

**SelectFuzzyItem**：`Value`（提交给后端的值，如 ID）、`Label`（下拉展示文本）、`Icon`、`DisplayInfo`（额外展示信息，如单价、库存；字段名会参与聚合表达式）。

示例：Form 收银台请求中「商品清单」table 的 `product_id`、顶层「会员卡」`member_id` 均用 OnSelectFuzzy；Table 预约表的「会议室」`room_id` 用 OnSelectFuzzy 且只查 `status='可用'`。

```go
// 1. 请求/列表结构体里字段加 callback:"OnSelectFuzzy"
type CashierDeskReq struct {
    ProductQuantities []struct {
        ProductID int `json:"product_id" widget:"name:商品;type:select" validate:"required" callback:"OnSelectFuzzy"`
        Quantity  int `json:"quantity" widget:"name:数量;type:number" validate:"required,min=1"`
    } `json:"product_quantities" widget:"name:商品清单;type:table"`
    MemberID int `json:"member_id" widget:"name:会员卡;type:select" validate:"required" callback:"OnSelectFuzzy"`
}

// 2. 模板里配置 OnSelectFuzzyMap，key 为字段 json 名
var CashierDeskTemplate = &app.FormTemplate{
    BaseConfig: app.BaseConfig{
        Name: "收银台", Request: &CashierDeskReq{}, Response: &CashierDeskResp{},
        OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
            "product_id": onSelectFuzzyProduct,
            "member_id":  onSelectFuzzyMember,
        },
    },
}

// 3. 回调：按关键字查库或按 value(s) 回显，返回 Items；table 子表可返回 Statistics 做聚合展示
func onSelectFuzzyProduct(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
    db := ctx.GetGormDB()
    var products []Product
    if req.IsByValue() {
        db.Model(&Product{}).Where("id = ?", req.GetValue()).Limit(1).Find(&products)
    } else if req.IsByValues() {
        db.Model(&Product{}).Where("id in ?", req.GetValues()).Find(&products)
    } else {
        db.Model(&Product{}).Where("name LIKE ?", "%"+req.Keyword()+"%").Limit(20).Find(&products)
    }
    items := make([]*callback.SelectFuzzyItem, 0, len(products))
    for _, p := range products {
        items = append(items, &callback.SelectFuzzyItem{
            Value: p.ID, Label: p.Name,
            DisplayInfo: map[string]interface{}{"价格": p.Price, "库存": p.Stock},
        })
    }
    return &callback.OnSelectFuzzyResp{
        Items: items,
        Statistics: map[string]interface{}{
            "商品原价总额(元)":  statistics.Sum("价格 * quantity"),
            "会员折扣后价格(元)": statistics.Sum("价格 * quantity * 折扣率"),
            "商品种类数":      statistics.Count("价格"),
            "商品总数量(件)":   statistics.Sum("quantity"),
        },
    }, nil
}
```

#### Statistics 与聚合计算（OnSelectFuzzyResp.Statistics）

`OnSelectFuzzyResp.Statistics` 的键值对会在前端表单旁展示（如收银台「商品原价总额」「会员折扣后价格」「当前余额」等）。值可以是**静态字符串**，也可以是 **`statistics` 包**返回的表达式，由前端根据当前 **table 行数据**或**选中项**动态计算。需导入：`import "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/statistics"`。

**1. table 子表场景（对当前 table 多行聚合）**

当 OnSelectFuzzy 用于 **table 子表**中的 select（如收银台商品清单里的「商品」）时，Statistics 里可用：

- **`statistics.Sum(expression)`**：对当前 table 所有行按表达式求和。表达式与 **MySQL/SQL 一致**：空格分隔、`*` 表示乘，字段名来自 `SelectFuzzyItem.DisplayInfo` 的 key 或行内字段名（如 `quantity`）。条件用 **MySQL IF(cond, thenExpr, elseExpr)**。  
  例：`statistics.Sum("价格")`、`statistics.Sum("价格 * quantity")`、`statistics.Sum("IF(price > 0, price * quantity, 销售价 * quantity)")`（有输入价用输入价×数量，否则用默认销售价×数量）、`statistics.Sum("价格 * quantity * (1 - 折扣率)")`（优惠金额）。
- **`statistics.Count(field)`**：对当前行按某字段非空计数，如 `statistics.Count("价格")` 表示「选了几种商品」。
- **`statistics.Avg(field, ...)`**、**`statistics.Min(field)`**、**`statistics.Max(field)`**：平均值、最小值、最大值。

DisplayInfo 里需提供聚合用到的 key（如「价格」「折扣率」），行内字段如 `quantity` 直接写字段名。

```go
// 商品 Items 的 DisplayInfo 含：价格、库存、折扣率 等
DisplayInfo: map[string]interface{}{"价格": p.Price, "库存": p.Stock, "折扣率": p.DiscountRate}

Statistics: map[string]interface{}{
    "商品原价总额(元)":  statistics.Sum("价格 * quantity"),
    "会员折扣后价格(元)": statistics.Sum("价格 * quantity * 折扣率"),
    "优惠金额(元)":    statistics.Sum("价格 * quantity * (1 - 折扣率)"),
    "商品种类数":      statistics.Count("价格"),
    "商品总数量(件)":   statistics.Sum("quantity"),
    "折扣说明":       "每个商品可设置不同折扣率（如0.9表示9折）",  // 静态字符串
}
```

**2. 单选 select 场景（取当前选中项展示）**

当 OnSelectFuzzy 用于**单选**（如收银台「会员卡」）时，Statistics 里可用：

- **`statistics.Value(field)`**：取**当前选中项**的 `DisplayInfo` 里该 key 的值展示，如 `statistics.Value("余额")`、`statistics.Value("卡号")`、`statistics.Value("状态")`。选中项变化时，前端会更新展示。

```go
// 会员卡 Items 的 DisplayInfo 含：余额、卡号、客户姓名、状态 等
DisplayInfo: map[string]interface{}{"余额": m.Balance, "卡号": m.CardNo, "客户姓名": m.Name, "状态": m.Status}

Statistics: map[string]interface{}{
    "当前余额": statistics.Value("余额"),
    "会员卡号": statistics.Value("卡号"),
    "客户姓名": statistics.Value("客户姓名"),
    "会员状态": statistics.Value("状态"),
}
```

**3. 静态说明**

键值对可直接写字符串，如 `"配送说明": "满99元包邮，不满99元运费10元"`，无需 statistics 包。

完整收银台示例（商品清单 Sum/Count、会员卡 Value、表达式格式）：read_doc `/builtin/doc/case_catalog/form_table_chart/cashier`。

**Table 模式**下同样可用 OnSelectFuzzy：在**列表结构体**（AutoCrudTable 指向的模型）里给需要后端动态选项的 select 字段加 `callback:"OnSelectFuzzy"`，在 **TableTemplate** 的 `BaseConfig.OnSelectFuzzyMap` 里按「字段 json 名」注册回调即可。例如会议室预约表：新增/编辑时「会议室」下拉从后端查库且只显示「可用」的会议室。

```go
// Table 模式：列表结构体（预约表）里会议室字段加 callback:"OnSelectFuzzy"
type MeetingRoomBooking struct {
    // ...
    RoomID   int    `json:"room_id" gorm:"column:room_id" widget:"name:会议室;type:select" validate:"required" callback:"OnSelectFuzzy"`
    RoomName string `json:"room_name" gorm:"-" widget:"name:会议室名称;type:text" permission:"read"`
    // ...
}

// TableTemplate 的 BaseConfig 里配置 OnSelectFuzzyMap，key 为 room_id
var MeetingRoomBookingListTemplate = &app.TableTemplate{
    BaseConfig: app.BaseConfig{
        Name: "会议室预约管理", Request: &MeetingRoomBookingListReq{}, Response: query.PaginatedTable[[]MeetingRoomBooking]{},
        CreateTables: []interface{}{&MeetingRoomBooking{}},
        OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
            "room_id": onSelectFuzzyMeetingRoom,
        },
    },
    AutoCrudTable: &MeetingRoomBooking{},
    OnTableAddRow: func(...) { ... },
    OnTableUpdateRow: func(...) { ... },
}

// 回调内只查 status='可用' 的会议室；可选返回 Statistics（statistics.Value("会议室名称") 等）在表单旁展示选中项信息
func onSelectFuzzyMeetingRoom(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
    db := ctx.GetGormDB()
    var rooms []MeetingRoom
    db = db.Model(&MeetingRoom{}).Where("status = ?", "可用")
    if req.IsByValue() { db = db.Where("id = ?", req.GetValue()).Limit(1) }
    else if req.IsByValues() { db = db.Where("id in ?", req.GetValues()) }
    else { db = db.Where("name LIKE ? OR type LIKE ? OR location LIKE ?", "%"+req.Keyword()+"%", "%"+req.Keyword()+"%", "%"+req.Keyword()+"%").Limit(20) }
    db.Find(&rooms)
    items := make([]*callback.SelectFuzzyItem, 0, len(rooms))
    for _, r := range rooms {
        items = append(items, &callback.SelectFuzzyItem{
            Value: r.ID, Label: r.Name,
            DisplayInfo: map[string]interface{}{"会议室名称": r.Name, "类型": r.Type, "容纳人数": r.Capacity, "位置": r.Location},
        })
    }
    return &callback.OnSelectFuzzyResp{
        MaxSelections: 1,
        Items:         items,
        Statistics:    map[string]interface{}{"选中会议室": statistics.Value("会议室名称"), "容纳人数": statistics.Value("容纳人数"), "位置": statistics.Value("位置")},
    }, nil
}
```

完整示例：read_doc ` /builtin/doc/case_catalog/form_table_chart/cashier`（Form + table 子表 OnSelectFuzzy + 聚合计算）、`/builtin/doc/case_catalog/tables/meeting`（**Table 模式**预约选会议室 OnSelectFuzzy，见 meeting_room_booking.go）。

Form 请求中 table 子表、OnSelectFuzzy、多 POST 同目录等：read_doc ` /builtin/doc/case_catalog/form/excelorcsv`、`/builtin/doc/case_catalog/form_table_chart/cashier`（收银台）、`/builtin/doc/case_catalog/formandtable/vote`。

---

## 七、Chart 模式要点

Chart 用于**只读的统计/图表**（BI），GET 请求。ChartTemplate、请求结构体、处理函数、注册方式见第二节「快速开始 → Chart 模式」。

#### 图表开发 Badcase（务必避免）

以下为大模型常见错误，写图表代码时请勿出现：

1. **一个函数返回多张图**  
   - **错误**：写 `resp.Charts(...)`、`resp.Chart(chart1, chart2)` 等。SDK 没有 `resp.Charts`，`resp.Chart(chart).Build()` 只接受**一个**图表。  
   - **正确**：每张图一个 GET 路由。参考收银台：4 张图 = 4 个 `.chart` 路由、4 个函数。

2. **手填 ChartType 或 Series[].Type**  
   - **错误**：使用 `&types.Chart{ ChartType: "line", ... }` 或给 Series 填 `Type: "line"`。  
   - **正确**：使用具体类型 `&types.LineChart{}`、`&types.BarChart{}` 等，只填 Title、XAxis、Series（Name、Data、可选 Config），不填 ChartType 和 Series[].Type；框架会在 `resp.Chart()` 时自动注入。

3. **误用 sdk/agent-app 下的 query 包**  
   - **错误**：`import "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/query"`，导致编译报错「包找不到」。  
   - **正确**：查询/分页等应使用 `github.com/ai-agent-os/ai-agent-os/pkg/gormx/query`（或项目内实际提供的 query 包），不要使用 `sdk/agent-app/query`。

4. **不确定时先看案例**  
   - 图表个数、路由拆分、返回格式，以收银台案例为准：read_doc `/builtin/doc/case_catalog/form_table_chart/cashier`，看每个图表是如何「一个 GET 路由 + 一个具体图表类型返回值」实现的。

#### 图表类型说明（4 种，唯一参考表）

| 必须使用的类型 | 说明 | 典型场景 | XAxis | Series Data 格式 |
|------|------|----------|-------|------------------|
| `types.LineChart` | 折线图 | 时间趋势、多指标随时间的走势 | 必需，如日期列表 | `[]interface{}{y1, y2, ...}`，与 XAxis 一一对应 |
| `types.BarChart` | 柱状图 | 分类对比、各维度数量/金额 | 必需，如分类名 | `[]interface{}{v1, v2, ...}`，与 XAxis 一一对应 |
| `types.PieChart` | 饼图 | 占比分布、构成比例 | 不需要 | `[]interface{}{ map[string]interface{}{"name":"分类","value":数值}, ... }` |
| `types.GaugeChart` | 仪表盘 | 单指标（完成率、平均值、达标值） | 不需要 | `[]interface{}{ 单值 }`，可选 Series.Config 的 min/max/detail |

**LineChart（折线图）**：按时间或类别展示趋势，可多系列。XAxis 为刻度（如日期），每个 Series 的 Data 与 XAxis 长度一致。

```go
chart := &types.LineChart{
    Title:  "工单趋势统计",
    XAxis:  dateLabels,  // []string{"2025-01-01", "2025-01-02", ...}
    Series: []types.ChartSeries{
        {Name: "工单数量", Data: []interface{}{10, 25, 18, ...}},
        {Name: "已完成数", Data: []interface{}{5, 12, 10, ...}},
    },
    Metadata: map[string]interface{}{"总工单数": totalCount, "数据更新时间": time.Now().Format("2006-01-02 15:04:05")},
}
return resp.Chart(chart).Build()
```

**BarChart（柱状图）**：分类对比，XAxis 为分类名，Data 为对应数值。

```go
chart := &types.BarChart{
    Title:  "工单优先级分布统计",
    XAxis:  []string{"低", "中", "高"},
    Series: []types.ChartSeries{{Name: "工单数量", Data: []interface{}{8, 20, 5}}},
    Metadata: map[string]interface{}{"总工单数": totalCount, "完成率": "66.67%", ...},
}
return resp.Chart(chart).Build()
```

**PieChart（饼图）**：展示占比，不需要 XAxis。Data 中每个元素为 `{"name": "分类名", "value": 数值}`。

```go
pieData := make([]interface{}, 0)
for _, stat := range statusStats {
    pieData = append(pieData, map[string]interface{}{"name": stat.Status, "value": stat.Count})
}
chart := &types.PieChart{
    Title:   "工单状态分布",
    Series:  []types.ChartSeries{{Name: "工单状态", Data: pieData}},
    Metadata: map[string]interface{}{"总工单数": totalCount, "待处理数": statusMap["待处理"], ...},
}
return resp.Chart(chart).Build()
```

**GaugeChart（仪表盘）**：单指标，Data 为单元素数组；可选 Config 指定 min、max、detail.formatter（如 `"¥{value}"`）。

```go
chart := &types.GaugeChart{
    Title: "工单完成率",
    Series: []types.ChartSeries{
        {
            Name:   "完成率",
            Data:   []interface{}{completionRate},
            Config: map[string]interface{}{
                "min": 0, "max": 100,
                "detail": map[string]interface{}{"formatter": "{value}%", "fontSize": 20},
            },
        },
    },
    Metadata: map[string]interface{}{"总工单数": totalCount, "已完成数": completedCount, "完成率": "66.50%", ...},
}
return resp.Chart(chart).Build()
```

完整示例：收银台统计（LineChart/BarChart/PieChart/GaugeChart）read_doc `/builtin/doc/case_catalog/form_table_chart/cashier`。

---

## 八、注册与目录约定

- **init()**：在业务 .go 中写；`packageContext.GET("路由名", ListFunc, TableTemplate)` 或 `packageContext.POST("路由名", Handler, FormTemplate)` 或 `packageContext.GET("路由名", ChartHandler, ChartTemplate)`。`packageContext` 由脚手架生成，不要重复声明。
- **init_.go**：由系统生成，不要用 write_go_file 创建或修改。
- **目录**：一个包一个目录，路由名与业务含义对应；多表/多 Form 可在同包多文件，各自 GET/POST 注册。参考「可读的目录」中案例路径或 read_doc("/builtin/doc/workspace/create-project") 文档末尾的案例分类。

### 路由命名约定（类型后缀，必须）

**生成代码时，路由名必须带类型后缀。** 这样从 full_code_path / URL 即可看出函数类型（自解释、大模型友好）：**看到后缀即知类型**，无需再查 DB 或文档。

| 类型 | 后缀 | 含义 | 示例 |
|------|------|------|------|
| Form | `.form` | 表单（POST） | `cashier_desk.form`、`vote_submit.form` |
| Table | `.table` | 表格列表（GET） | `ticket_list.table`、`meeting_room_list.table` |
| Chart | `.chart` | 图表（GET） | `cashier_sales_trend_statistics.chart` |

- **注册时必须带后缀**：`packageContext.POST("cashier_desk.form", ...)`；`packageContext.GET("ticket_list.table", ...)`；`packageContext.GET("sales_trend_statistics.chart", ...)`。禁止注册无后缀的路由名。
- **link 的 target**：跳转到其他函数时，target 需与注册的路由名一致（含后缀），如 `"meeting_room_list.table"`、`"vote_result.form"`。
- 示例与案例（如 `/builtin/doc/case_catalog`）均按此约定命名。

---

## 九、完整案例（read_doc 路径）

以下路径均在系统消息「可读的目录」中；按需 read_doc 获取该案例的 PRD 与完整代码。

- **单 Table**：`/builtin/doc/case_catalog/table/ticket`
- **单 Form**：`/builtin/doc/case_catalog/form/excelorcsv`、`/builtin/doc/case_catalog/form/images`、`/builtin/doc/case_catalog/form/pdf`、`/builtin/doc/case_catalog/form/nlp`、`/builtin/doc/case_catalog/form/videos`
- **多 Table**：`/builtin/doc/case_catalog/tables/meeting`、`/builtin/doc/case_catalog/tables/hr`
- **Table + Form**：`/builtin/doc/case_catalog/formandtable/vote`
- **Table + Form + Chart**：`/builtin/doc/case_catalog/form_table_chart/cashier` （全部类型的图表都有在这个里面呈现）

生成新应用时：先 read_doc 本 SDK，再按需求 read_doc 对应类型案例，再出 PRD 与代码。
