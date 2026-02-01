# Agent-App SDK 使用说明

本文档说明**框架的用法与能力**。完整业务示例（PRD + 代码）在案例文档中，按需 read_doc 对应路径即可。

---

## 一、定位与文档分工

- **本 SDK 文档**：框架怎么用——结构体与标签、Table/Form 模式、注册方式、目录约定。
- **案例文档**（`/builtin/doc/case_catalog/xxx`）：具体业务长什么样——PRD + 完整 Go 代码。系统消息中「可读的目录」会列出各案例路径与说明；需要单表 CRUD、多表、Form、图表等时，read_doc 对应案例获取 PRD 与代码。

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
    Status   string `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:待处理,已完成;default:待处理" search:"in"`
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
    packageContext.GET("crm_ticket", CrmTicketList, CrmTicketTemplate)
}
```

单表完整示例（含所有常用组件与回调）：read_doc ` /builtin/doc/case_catalog/table/ticket`。

### Form 模式（POST，无 Table）

1. **定义请求/响应结构体**：字段加 `widget`、`validate`；请求体可含 files、input、select、table 等。
2. **写处理函数**：`ctx.ShouldBindValidate(&req)`，业务逻辑，`return resp.Form(&respStruct).Build()` 或 `resp.BizErrorf(...).Build()`。
3. **配置 FormTemplate**：`BaseConfig`（Name、Request、Response）+ 可选 `OnSelectFuzzyMap` 等。
4. **注册**：`init()` 中 `packageContext.POST("路由名", Handler, FormTemplate)`。

最小可用片段示例：

```go
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
    packageContext.POST("excel_or_csv", ExcelOrCsvHandler, &app.FormTemplate{
        BaseConfig: app.BaseConfig{Name: "Excel/CSV 解析", Request: &ExcelOrCsvReq{}, Response: &ExcelOrCsvResp{}},
    })
}
```

单 Form 完整示例：read_doc ` /builtin/doc/case_catalog/form/excelorcsv`。

### Chart 模式（GET，统计/图表）

1. **定义请求结构体**：图表筛选条件（如时间范围、状态）加 `widget` 标签，前端会渲染成图表筛选表单。
2. **写统计函数**：`ctx.ShouldBind(&req)` 绑定参数，查库/聚合得到数据，构造 `types.Chart`（ChartType、Title、XAxis、Series、Metadata），`return resp.Chart(chart).Build()`。
3. **配置 ChartTemplate**：`BaseConfig`（Name、Request、Response: `&types.Chart{}`）；无回调，只读展示。
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
    chart := &types.Chart{
        ChartType: "line",
        Title:     "销售趋势",
        XAxis:     dateLabels,
        Series:    []types.ChartSeries{{Name: "销售额", Type: "line", Data: seriesData}},
        Metadata:  map[string]interface{}{"总销售额": total},
    }
    return resp.Chart(chart).Build()
}

func init() {
    packageContext.GET("sales_trend_statistics", SalesTrendChart, &app.ChartTemplate{
        BaseConfig: app.BaseConfig{Name: "销售趋势", Request: &SalesStatisticsReq{}, Response: &types.Chart{}},
    })
}
```

完整 Chart 示例（折线图/饼图/仪表盘、时间筛选、多图表同包）：read_doc ` /builtin/doc/case_catalog/form_table_chart/cashier`。

---

## 三、结构体与标签

### 1. widget 标签

格式：`widget:"name:显示名;type:组件类型;配置项:值"`。常用配置：`default`、`options`、`options_colors`、`min`/`max`/`step`/`unit`、`format`、`precision` 等。

片段示例：

```go
Title    string `widget:"name:标题;type:input" search:"like" validate:"required,min=2,max=200"`
Status   string `widget:"name:状态;type:select;options:待处理,已完成;default:待处理" search:"in" validate:"oneof=待处理 已完成"`
Priority string `widget:"name:优先级;type:select;options:低,中,高;options_colors:success,warning,danger;default:中"`
Handler  string `widget:"name:处理人;type:user;default:Me()" search:"in"`
CreateBy string `widget:"name:创建用户;type:user" permission:"read"`  // 只读，在 OnTableAddRow 里 ctx.GetRequestUser() 赋值
Tags     string `widget:"name:标签;type:multiselect;options:紧急,重要" search:"contains"`  // 多选用 string 逗号分隔
Progress int    `widget:"name:进度;type:slider;min:0;max:100;unit:%" search:"gte,lte"`
Deadline int64  `widget:"name:截止时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte"`
Attachment *types.Files `gorm:"type:json" widget:"name:附件;type:files"`
```

| 组件类型 | 说明 | 典型用法 |
|----------|------|----------|
| input | 单行文本 | 标题、电话、邮箱 |
| text_area | 多行文本 | 描述、备注 |
| richtext | 富文本 | 详细内容 |
| select | 下拉单选 | 状态、优先级（options 多时） |
| radio | 单选 | 来源、性别（2–5 个选项） |
| multiselect | 多选 | 标签（string，逗号分隔） |
| number | 整数 | 数量、工时 |
| float | 小数 | 价格、金额 |
| slider | 滑块 | 进度、评分（min/max/step/unit） |
| rate | 星级 | 评价（max、allow_half、texts） |
| switch | 开关 | 是否启用 |
| timestamp | 日期时间 | 创建时间、截止时间（毫秒时间戳；autoCreateTime/autoUpdateTime 自动填充） |
| color | 颜色 | format:hex，default:#xxx |
| files | 文件上传 | 字段类型 `*types.Files`，gorm `type:json` |
| user / users | 用户选择 | default:Me()、Me(),MyLeader() 等 |
| department / departments | 部门选择 | default:MyDepartment()，max_count 等 |
| table | 子表（Form 请求） | 数组结构体，可配 OnSelectFuzzy |
| form | 子表单（Form 响应） | 嵌套结构体展示 |
| link | 跳转链接 | 列表/表单中跳转到另一 GET 或 Form、或外链；不落库，后端 BuildFunctionUrlWithText 赋值 |

#### link 组件（跳转链接）

用于在**列表**或**表单**中展示可点击链接，点击后跳转到另一个 GET（Table/Chart）、Form，或打开外链。字段通常**不落库**（`gorm:"-"`）、**只读**（`permission:"read"`），值由后端在 **List 函数 Build 之后**或 **Form 响应**里用 `ctx.BuildFunctionUrlWithText(target, params, linkText)` 赋值。

- **widget 配置**：`type:link`；可选 `target:_blank`（新窗口）或 `_self`（当前窗口）；可选 `text`、`type`（样式 primary/success 等）、`icon`。
- **赋值 API**：`ctx.BuildFunctionUrlWithText(target string, params interface{}, linkText string) (string, error)`  
  - **target**：函数路径（如 `"meeting_room_list"`、`"vote_submit"`、`"vote_result"`），或带查询（如 `"hr_resume_list?_tab=OnTableAddRow"` 表示打开该列表并切到「新增」Tab），或**外链**（如 `"https://example.com"`）。  
  - **params**：结构体参数，用于拼成 URL 查询（如 `MeetingRoom{ID: roomID}`）；外链时传 `nil`。  
  - **linkText**：链接展示文本（如「查看会议室详情」「点击参与投票」「查看投票结果」）。
- **典型场景**：  
  1. **Table 列表「查看详情」列**：当前行关联另一张表，链接跳转到该表并带上当前行 ID（如预约列表的「会议室详情」→ 跳会议室列表并带 `RoomID`）。  
  2. **Table 列表「操作」列**：根据状态动态生成链接（如投票主题列表「投票操作」→ 进行中且未投票时「点击参与投票」跳 `vote_submit`，否则「查看投票结果」跳 `vote_result`；职位列表「投递简历」→ 跳简历列表并打开新增 Tab 且带职位 ID）。  
  3. **Form 响应**：提交后返回一个「查看结果」链接（如投票提交后返回「查看投票结果」跳 `vote_result`）。

```go
// Table 列表：不落库、只读，List Build 之后对每条记录赋值
RoomLink string `json:"room_link" gorm:"-" widget:"name:会议室详情;type:link;target:_blank" permission:"read"`

// List 函数内，Build 之后：
for i := range bookings {
    params := MeetingRoom{ID: bookings[i].RoomID}
    bookings[i].RoomLink, _ = ctx.BuildFunctionUrlWithText("meeting_room_list", params, "查看会议室详情")
}

// 带 _tab 参数：打开列表并切到「新增」Tab（如投递简历）
jobs[i].ApplyLink, _ = ctx.BuildFunctionUrlWithText("hr_resume_list?_tab=OnTableAddRow", params, "投递简历")

// Form 响应：提交后返回链接
functionLink, _ := ctx.BuildFunctionUrlWithText("vote_result", params, "查看投票结果")
return resp.Form(&VoteSubmitResp{..., FunctionLink: functionLink}).Build()
```

完整示例：read_doc `/builtin/doc/case_catalog/tables/meeting`（预约列表会议室详情 link）、`/builtin/doc/case_catalog/tables/hr`（职位/简历列表 link、_tab=OnTableAddRow）、`/builtin/doc/case_catalog/formandtable/vote`（投票操作/选项列表/提交结果 link）。

- **隐藏字段**：`widget:"-"` 表示该字段**被前端直接忽略**，不参与列表/表单的渲染，也不会被提交；常用于系统字段（如 DeletedAt、DeletedBy）或内部关联（如 `json:"-"` 的关联表）。
- **只读/仅创建等**：用 `permission` 控制，见下节。

### 2. validate 标签

遵循 `validator/v10`。常用：`required`、`min`/`max`、`oneof=值1 值2`（空格分隔，值含空格用单引号）、`email` 等。

```go
Title string `validate:"required,min=2,max=200"`
Status string `validate:"required,oneof=待处理 处理中 已完成"`
Email string `validate:"required,email"`
```

### 3. search 标签

| 值 | 含义 | 适用 |
|----|------|------|
| like | 模糊 | input、text_area |
| in | 精确 IN | select、radio、user、department |
| contains | FIND_IN_SET | multiselect、users、departments |
| eq | 精确 = | ID、switch |
| gte,lte | 范围 | timestamp、number、float、slider |

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

### 4. List 函数内对列表数据的后处理

Build 之后可对 `lists` 做计算、脱敏等。下面是一个**完整最小示例**：3～5 个字段，列表里增加一个「仅展示、不落库」的剩余时间。

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

---

## 六、Form 模式要点

- **请求/响应结构体**：字段加 `widget`、`validate`；请求可含 `type:table`（子表）、`type:select` + `callback:"OnSelectFuzzy"` 等。
- **处理函数**：`ctx.ShouldBindValidate(&req)`；业务逻辑；成功 `return resp.Form(&respStruct).Build()`，失败 `return resp.BizErrorf("错误信息").Build()`。如需读文件：`ctx.GetFS()`，DownloadFiles/RemoveFiles。
- **FormTemplate**：`BaseConfig` 含 Name、Request、Response；若请求中有下拉需联动后端数据，配 `OnSelectFuzzyMap`。
- **注册**：`packageContext.POST("路由名", Handler, FormTemplate)`。

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

- **`statistics.Sum(expression)`**：对当前 table 所有行按表达式求和。表达式格式：**空格分隔**，`*` 表示乘，字段名来自 `SelectFuzzyItem.DisplayInfo` 的 key 或行内字段名（如 `quantity`）。  
  例：`statistics.Sum("价格")`、`statistics.Sum("价格 * quantity")`（价格×数量）、`statistics.Sum("价格 * quantity * 折扣率")`（折扣后金额）、`statistics.Sum("价格 * quantity * (1 - 折扣率)")`（优惠金额）。
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

完整收银台示例（商品清单 Sum/Count、会员卡 Value、表达式格式）：read_doc ` /builtin/doc/case_catalog/form_table_chart/cashier`；statistics 包说明见 `sdk/agent-app/statistics/README.md`。

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

Chart 用于**只读的统计/图表**（BI），GET 请求，前端根据请求体渲染筛选表单，根据返回的 `types.Chart` 渲染图表。

- **ChartTemplate**：`BaseConfig` 含 Name、Request、Response（`&types.Chart{}`）；无 OnTableAddRow 等回调，图表只读。
- **请求结构体**：筛选条件（如开始时间、结束时间、状态）加 `widget` 标签，前端会渲染成图表上方的筛选表单；`ctx.ShouldBind(&req)` 绑定。
- **处理函数**：绑定参数 → 查库/聚合（如按日期 GROUP BY、SUM/COUNT）→ 构造 `types.Chart` → `return resp.Chart(chart).Build()`。
- **types.Chart 结构**：
  - `ChartType`：图表类型，如 `line`（折线）、`bar`（柱状）、`pie`（饼图）、`gauge`（仪表盘）、`area`、`scatter`。
  - `Title`：图表标题。
  - `XAxis`：可选，X 轴刻度（如日期列表），用于 line/bar/area。
  - `Series`：数据系列，`[]types.ChartSeries`，每项含 `Name`、`Type`（可与 ChartType 一致或混合如 line+bar）、`Data`、可选 `Config`（如 gauge 的 min/max/detail 格式化）。
  - `Metadata`：可选，键值对，用于图表旁展示汇总信息（如总销售额、总订单数、数据更新时间）。
- **Series 的 Data 格式**：
  - **line/bar/area**：`[]interface{}`，如 `[]interface{}{100, 200, 150}`，与 XAxis 一一对应。
  - **pie**：`[]interface{}`，元素为 `map[string]interface{}{"name": "分类名", "value": 数值}`。
  - **gauge**：`[]interface{}` 单值如 `[]interface{}{75}`，可选 `Config` 中 `min`、`max`、`detail.formatter` 等。
- **注册**：`packageContext.GET("路由名", ChartHandler, ChartTemplate)`；同一包内可注册多个 GET 图表路由。

#### 图表类型说明

| ChartType | 说明 | 典型场景 | XAxis | Series Data 格式 |
|-----------|------|----------|-------|------------------|
| `line` | 折线图 | 时间趋势、多指标随时间的走势 | 必需，如日期列表 | `[]interface{}{y1, y2, ...}`，与 XAxis 一一对应 |
| `bar` | 柱状图 | 分类对比、各维度数量/金额 | 必需，如分类名 | `[]interface{}{v1, v2, ...}`，与 XAxis 一一对应 |
| `pie` | 饼图 | 占比分布、构成比例 | 不需要 | `[]interface{}{ map[string]interface{}{"name":"分类","value":数值}, ... }` |
| `gauge` | 仪表盘 | 单指标（完成率、平均值、达标值） | 不需要 | `[]interface{}{ 单值 }`，可选 Series.Config 的 min/max/detail |
| `area` | 面积图 | 同折线，带填充，强调累计/趋势 | 同 line | 同 line |
| `scatter` | 散点图 | 两维分布、相关性 | 可选 | `[]interface{}{ [x,y], [x,y], ... }` 坐标点 |

**line（折线图）**：按时间或类别展示趋势，可多系列（如销售额 + 订单数）。XAxis 为刻度（如日期），每个 Series 的 Data 与 XAxis 长度一致。

```go
chart := &types.Chart{
    ChartType: "line",
    Title:     "工单趋势统计",
    XAxis:     dateLabels,  // []string{"2025-01-01", "2025-01-02", ...}
    Series: []types.ChartSeries{
        {Name: "工单数量", Type: "line", Data: []interface{}{10, 25, 18, ...}},
        {Name: "已完成数", Type: "line", Data: []interface{}{5, 12, 10, ...}},  // 可多系列
    },
    Metadata: map[string]interface{}{"总工单数": totalCount, "数据更新时间": time.Now().Format("2006-01-02 15:04:05")},
}
```

**bar（柱状图）**：分类对比，如优先级/状态/部门下的数量或金额。XAxis 为分类名，Data 为对应数值。

```go
chart := &types.Chart{
    ChartType: "bar",
    Title:     "工单优先级分布统计",
    XAxis:     []string{"低", "中", "高"},
    Series: []types.ChartSeries{
        {Name: "工单数量", Type: "bar", Data: []interface{}{8, 20, 5}},
    },
    Metadata: map[string]interface{}{"总工单数": totalCount, "完成率": "66.67%", ...},
}
```

**pie（饼图）**：展示占比，不需要 XAxis。Data 中每个元素为 `{"name": "分类名", "value": 数值}`。

```go
pieData := make([]interface{}, 0)
for _, stat := range statusStats {
    pieData = append(pieData, map[string]interface{}{"name": stat.Status, "value": stat.Count})
}
chart := &types.Chart{
    ChartType: "pie",
    Title:     "工单状态分布",
    Series:    []types.ChartSeries{{Name: "工单状态", Type: "pie", Data: pieData}},
    Metadata:  map[string]interface{}{"总工单数": totalCount, "待处理数": statusMap["待处理"], ...},
}
```

**gauge（仪表盘）**：单指标，如完成率 0～100、平均订单金额。Data 为单元素数组；可选 Config 指定 min、max、detail.formatter（如 `"¥{value}"`）。

```go
chart := &types.Chart{
    ChartType: "gauge",
    Title:     "工单完成率",
    Series: []types.ChartSeries{
        {
            Name:   "完成率",
            Type:   "gauge",
            Data:   []interface{}{completionRate},  // 如 66.5
            Config: map[string]interface{}{
                "min": 0, "max": 100,
                "detail": map[string]interface{}{"formatter": "{value}%", "fontSize": 20},
            },
        },
    },
    Metadata: map[string]interface{}{"总工单数": totalCount, "已完成数": completedCount, "完成率": "66.50%", ...},
}
```

**area（面积图）**：与 line 相同数据结构，Type 设为 `area`，前端会渲染为带填充的面积图，适合强调趋势或累计。

**scatter（散点图）**：每点为 `[x, y]`，用于两维分布或相关性；XAxis 可选。

完整示例：工单统计（bar/饼图/gauge/折线图）可参考项目内 `namespace/luobei/operations/code/api/crm/ticket/crm_ticket.go`；收银台统计（折线/饼图/仪表盘）read_doc `/builtin/doc/case_catalog/form_table_chart/cashier`。

---

## 八、注册与目录约定

- **init()**：在业务 .go 中写；`packageContext.GET("路由名", ListFunc, TableTemplate)` 或 `packageContext.POST("路由名", Handler, FormTemplate)` 或 `packageContext.GET("路由名", ChartHandler, ChartTemplate)`。`packageContext` 由脚手架生成，不要重复声明。
- **init_.go**：由系统生成，不要用 write_go_file 创建或修改。
- **目录**：一个包一个目录，路由名与业务含义对应；多表/多 Form 可在同包多文件，各自 GET/POST 注册。参考「可读的目录」中案例路径与 system_prompt 中「参考项目目录结构」。

---

## 九、完整案例（read_doc 路径）

以下路径均在系统消息「可读的目录」中；按需 read_doc 获取该案例的 PRD 与完整代码。

- **单 Table**：`/builtin/doc/case_catalog/table/ticket`
- **单 Form**：`/builtin/doc/case_catalog/form/excelorcsv`、`/builtin/doc/case_catalog/form/images`、`/builtin/doc/case_catalog/form/pdf`、`/builtin/doc/case_catalog/form/nlp`、`/builtin/doc/case_catalog/form/videos`
- **多 Table**：`/builtin/doc/case_catalog/tables/meeting`、`/builtin/doc/case_catalog/tables/hr`
- **Table + Form**：`/builtin/doc/case_catalog/formandtable/vote`
- **Table + Form + Chart**：`/builtin/doc/case_catalog/form_table_chart/cashier`

生成新应用时：先 read_doc 本 SDK，再按需求 read_doc 对应类型案例，再出 PRD 与代码。
