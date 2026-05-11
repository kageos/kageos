# SDK Table CRUD 基础任务包

本文档用于简单 CRUD、后台管理列表、台账、记录库等 Table 场景。它是一个闭环任务包：说明什么时候用 Table、前端长什么样、目录/路由怎么命名、最小代码结构、字段规则、验证方式和常见错误。字段组件细节统一读取 `/system/prompt/sdk/widget-system`。

## 什么时候用 Table

用户需求包含以下特征时，优先 Table：

- 管理一批长期存在的业务记录。
- 需要列表、搜索、分页、查看详情。
- 需要新增、编辑、删除或批量导入。
- 典型词：管理系统、后台、列表、台账、CRUD、记录、客户、工单、订单、库存、商品、会员、岗位、简历。

Table 前端形态：Element Plus `el-table` 风格的数据表格。用户会看到搜索区、分页列表、列展示、工具栏，以及由 TableTemplate 回调决定的新建、编辑、删除、批量导入入口。

不要把一次性文件处理、收银结算、复杂派单、发送通知、纯统计图表写成 Table；简单状态更新或普通编辑可以放在 Table update 回调。

## 最小目录和路由

单表 CRUD 通常一个业务目录 + 一个 `.table` 函数：

```text
/用户/应用/ticket
  ticket_list.table
```

路由命名：

- 列表管理统一用 `xxx_list.table`。
- 路由最后一段必须 `.table`。
- 注册使用 `packageContext.GET("xxx_list.table", ListHandler, XxxTableTemplate)`。

## 最小结构

一个基础 Table 文件通常包含：

1. Model：落库字段、前端展示字段和校验。
2. Request：显式列表筛选条件，嵌入 `query.PageSortReq` 承载分页和排序。
3. TableTemplate：声明 Request、CreateTables、AutoCrudTable 和写操作回调。
4. List 函数：手写 Where/Joins/Preload，显式查询 `items + total`，并 `resp.Table(response.TableResult{...}).Build()`。
5. `init()`：注册路由。

示例骨架：

```go
package ticket

import (
    "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
    "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback"
    "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
    "github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
    "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
    "gorm.io/gorm"
)

type Ticket struct {
    ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
    CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
    UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
    DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

    Title    string `json:"title" gorm:"column:title" widget:"name:标题;type:input" validate:"required,min=2,max=100"`
    Priority string `json:"priority" gorm:"column:priority" widget:"name:优先级;type:select;options:低,中,高;options_colors:67C23A,E6A23C,F56C6C;render_default:中" validate:"required,oneof=低 中 高"`
    Status   string `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:待处理,处理中,已完成;options_colors:909399,E6A23C,67C23A;render_default:待处理" validate:"required,oneof=待处理 处理中 已完成"`
    Remark   string `json:"remark" gorm:"column:remark;type:text" widget:"name:备注;type:text_area"`
}

type TicketListReq struct {
    Title    string `json:"title" form:"title" widget:"name:标题;type:input"`
    Priority string `json:"priority" form:"priority" widget:"name:优先级;type:select;options:低,中,高;options_colors:67C23A,E6A23C,F56C6C"`
    Status   string `json:"status" form:"status" widget:"name:状态;type:select;options:待处理,处理中,已完成;options_colors:909399,E6A23C,67C23A"`

    query.PageSortReq `widget:"-"`
}

var TicketListTemplate = &app.TableTemplate{
    BaseConfig: app.BaseConfig{
        Name:         "工单列表",
        Request:      &TicketListReq{},
        CreateTables: []interface{}{&Ticket{}},
    },
    AutoCrudTable:     &Ticket{},
    OnTableAddRow:     CreateTicket,
    OnTableUpdateRow:  UpdateTicket,
    OnTableDeleteRows: DeleteTickets,
}

func TicketList(ctx *app.Context, resp response.Response) error {
    var req TicketListReq
    if err := ctx.ShouldBind(&req); err != nil {
        return err
    }
    var lists []*Ticket
    queryDB := ctx.GetGormDB().Model(&Ticket{})
    if req.Title != "" {
        queryDB = queryDB.Where("title LIKE ?", "%"+req.Title+"%")
    }
    if req.Priority != "" {
        queryDB = queryDB.Where("priority = ?", req.Priority)
    }
    if req.Status != "" {
        queryDB = queryDB.Where("status = ?", req.Status)
    }
    return resp.Table(response.TableResult{Items: lists, TotalCount: total, PageInfo: &req.PageSortReq}).Build()
}

func CreateTicket(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
    var row Ticket
    if err := ctx.ShouldBindValidate(&row); err != nil {
        return nil, err
    }
    if err := ctx.GetGormDB().Create(&row).Error; err != nil {
        return nil, err
    }
    return &callback.OnTableAddRowResp{Data: &row}, nil
}

func UpdateTicket(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
    updates := req.ChangedFields()
    if err := ctx.GetGormDB().Model(&Ticket{}).Where("id = ?", req.GetId()).Updates(updates).Error; err != nil {
        return nil, err
    }
    return &callback.OnTableUpdateRowResp{}, nil
}

func DeleteTickets(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
    if err := ctx.GetGormDB().Where("id IN ?", req.GetIds()).Delete(&Ticket{}).Error; err != nil {
        return nil, err
    }
    return &callback.OnTableDeleteRowsResp{}, nil
}

func init() {
    packageContext.GET("ticket_list.table", TicketList, TicketListTemplate)
}
```

实际项目以仓库现有 SDK API 为准；写代码前优先读匹配案例，例如 `/system/prompt/case_catalog/table/ticket`。

## TableTemplate 回调

Table 前端按钮由回调决定：

- 配置 `OnTableAddRow`：前端显示新增按钮，并自动支持批量导入能力。
- 配置 `OnTableUpdateRow`：前端显示编辑入口。
- 配置 `OnTableDeleteRows`：前端显示删除入口。
- 不配置某个回调：前端不显示对应写操作。
- 三个回调都不配置：前端就不会出现新增、编辑、删除入口，只保留列表、搜索、分页、查看和只读展示。

只读表：

- 支付流水、收银记录、消费记录、投票记录、评价记录、操作日志、审计日志、导入历史这类事实记录表默认只读。
- 只读表也建议显式配置 `AutoCrudTable`，它决定前端列表按哪个 Model 渲染字段、筛选、分页和 schema。
- 只读表通常只配置查询，不配置 `OnTableAddRow`、`OnTableUpdateRow`、`OnTableDeleteRows`。
- 除非用户明确要求人工补录或修正，否则不要给事实记录表开放新增、编辑、删除。

只读表 Template 示例：

```go
var PaymentRecordListTemplate = &app.TableTemplate{
    BaseConfig: app.BaseConfig{
        Name:         "支付流水",
        Request:      &PaymentRecordListReq{},
        CreateTables: []interface{}{&PaymentRecord{}},
    },
    AutoCrudTable: &PaymentRecord{},
}
```

注意：`AutoCrudTable` 建议保留，用于明确列表 Model。只读由“不配置 `OnTableAddRow`、`OnTableUpdateRow`、`OnTableDeleteRows`”控制，因此前端不会显示新增、编辑、删除按钮。支付流水、收银记录这类数据应由收银 Form 或业务事务写入，而不是由用户在表格里手工新增。

## List 函数规则

基础列表：

1. `Request` 显式声明筛选字段，并嵌入 `query.PageSortReq`，用 `widget:"-"` 隐藏分页字段。
2. `queryDB := ctx.GetGormDB().Model(&Model{})`。
3. Build 前做手写 `Where`、`Joins`、`Preload` 等。
4. 调 `resp.Table(response.TableResult{Items: lists, TotalCount: total, PageInfo: &req.PageSortReq}).Build()`。
5. Build 后填充不落库字段，例如 link、计算状态、关联展示名。

外表筛选和计算字段筛选：

- 所有筛选条件默认都放在 Request 结构体中显式声明。
- 在 Build 前手写查询逻辑。
- Table 筛选字段写在 Request 中，Model 不承担筛选协议。

## Table 必备 SDK 能力

Table 场景必须把搜索、关联展示、link 和 display 一次性设计好。否则前端只会出现最基础的表格，用户看不到关联名、跳转入口或复杂筛选。

### Request 自定义搜索参数

`query.PageSortReq` 只接收分页和排序。业务筛选字段放在 Request 中，并在 List 函数 Build 前手写查询条件。

```go
type BookingListReq struct {
    RoomName string `json:"room_name" form:"room_name" widget:"name:会议室名称;type:input"`
    Status   string `json:"status" form:"status" widget:"name:预约状态;type:select;options:待开始,进行中,已结束"`
    query.PageSortReq `widget:"-"`
}

func BookingList(ctx *app.Context, resp response.Response) error {
    var req BookingListReq
    if err := ctx.ShouldBind(&req); err != nil {
        return err
    }

    queryDB := ctx.GetGormDB().Model(&Booking{}).Preload("Room")
    if req.RoomName != "" {
        queryDB = queryDB.Joins("LEFT JOIN meeting_room ON meeting_room.id = booking.room_id").
            Where("meeting_room.name LIKE ?", "%"+req.RoomName+"%")
    }
    if req.Status != "" {
        now := time.Now()
        switch req.Status {
        case "待开始":
            queryDB = queryDB.Where("start_time > ?", now)
        case "进行中":
            queryDB = queryDB.Where("start_time <= ? AND end_time > ?", now, now)
        case "已结束":
            queryDB = queryDB.Where("end_time <= ?", now)
        }
    }

    var lists []*Booking
    if err := resp.Table(response.TableResult{Items: lists, TotalCount: total, PageInfo: &req.PageSortReq}).Build(); err != nil {
        return err
    }
    return nil
}
```

规则：

- `query.PageSortReq` 要匿名嵌入并加 `widget:"-"`，不要把分页字段暴露成普通表单字段。
- Request 筛选字段要有 `widget`，否则前端不会渲染筛选控件。
- 主表字段、关联字段、计算字段、特殊权限过滤、跨表筛选，都在 Build 前手写 `Where` / `Joins`。
- `Table` 只做 Count、排序、Offset、Limit、Find 和分页信息；业务筛选在 Build 前写到 `queryDB`。
- 默认使用 `resp.Table(response.TableResult{Items: lists, TotalCount: total, PageInfo: &req.PageSortReq}).Build()`。
- Table Template 通过 `AutoCrudTable` 指定 schema 来源。

### GORM Preload 和后置关联填充

关联展示有两种常用写法：

1. Build 前 `Preload`：适合 GORM 关联对象，例如预约表加载会议室、投票记录加载主题和选项。
2. Build 后后置关联填充：适合 link、计算字段、统计值、跨表聚合字段。

```go
queryDB := ctx.GetGormDB().Model(&VoteRecord{}).Preload("Topic").Preload("Option")
var records []VoteRecord
if err := resp.Table(response.TableResult{Items: records, TotalCount: total, PageInfo: &req.PageSortReq}).Build(); err != nil {
    return err
}

for i := range records {
    if records[i].Topic != nil {
        records[i].TopicTitle = records[i].Topic.Title
    }
    if records[i].Option != nil {
        records[i].OptionContent = records[i].Option.Content
    }
}
return nil
```

如果关联值不是标准 GORM 关系，Build 后收集当前页 ID，再批量查询 map 回填，避免 N+1 查询：

```go
ids := make([]int, 0, len(lists))
for _, item := range lists {
    ids = append(ids, item.ID)
}
// db.Where("object_id IN ?", ids).Group(...).Scan(&stats)，再回填到 lists[i].AvgScore
```

### hide 标签

`hide` 决定字段在哪些前端场景隐藏：

- `hide:"create,update"`：只在 Element 表格列中展示，不进入新增/编辑表单。ID、创建时间、更新时间、创建人、link、计算字段、关联名称都应这样写。
- `hide:"list,update"`：只在新增表单中出现，例如创建投票主题时输入选项子表。
- `hide:"list,create"`：只在编辑表单中出现。
- 不写 `hide`：默认进入列表和新增/编辑，只有用户真正需要看和填的业务字段才这么做。
- `widget:"-"`：完全不进入前端 schema，适合 `DeletedAt`、GORM 关联对象、内部中间字段。

典型字段：

```go
TopicTitle string `json:"topic_title" gorm:"-" widget:"name:投票主题;type:text" hide:"create,update"`
ActionLink string `json:"action_link" gorm:"-" widget:"name:操作;type:link;target:_blank" hide:"create,update"`
```

### link 跳转

Table 列表中的行操作不要手写按钮，而是用 `link` 字段。Build 后根据每行数据填充 link。

```go
lists[i].ActionLink, _ = ctx.BuildFunctionUrlWithText(
    "evaluation_submit.form",
    EvaluationSubmitReq{ObjectID: lists[i].ID},
    "提交评价",
)
```

规则：

- 跳 Table：参数使用目标 Table 的 Request 字段或列表字段 code，前端会转成筛选参数。
- 跳 Form：参数使用目标 Form Request，用于预填。
- 跳 Chart：参数使用目标 Chart Request，用于筛选图表。
- link 字段通常 `gorm:"-"` + `hide:"create,update"`。

### Table 表单和筛选里的动态选择

Table 新增/编辑表单和搜索区都可以使用 OnSelectFuzzy。例如评价记录按评价对象筛选、预约记录选择会议室、投递记录选择职位。

```go
ObjectID int `json:"object_id" form:"object_id" widget:"name:评价对象;type:select" callback:"OnSelectFuzzy"`
ObjectName string `json:"object_name" gorm:"-" widget:"name:评价对象;type:text" hide:"create,update"`
```

同时在 `TableTemplate.BaseConfig.OnSelectFuzzyMap` 中注册 `"object_id"`。回调要支持关键字搜索，也要支持 by_value / by_values 回显。Handler 中按 `req.ObjectID` 手写 `Where("object_id = ?", req.ObjectID)`。

外键字段的 code 可以叫 `object_id`，但 `widget name` 应写“评价对象”这类业务名，不要写“评价对象ID”。列表中展示 `ObjectName`，筛选时用户通过下拉搜索对象名称，前端实际提交 `object_id`。

历史记录表的外键搜索回调不要只返回当前开放/可用对象；已关闭对象也可能有历史记录。提交类 Form 才按业务规则过滤为可提交对象。

注意：`OnSelectFuzzyMap` 的 key 必须指向当前 Table schema 中真实存在的 `select` 或 `multiselect` 字段。不要把字段写成 `type:ID` / `type:input` 后再给它注册 OnSelectFuzzy；这会在启动校验时报 `must use select or multiselect widget`。如果字段只用于内部 ID 展示，就不要注册回调；如果用户需要按名称选择或搜索，就把外键字段建模成 `type:select` 并注册回调。

### Table 附件字段和文件下载

Table 中的附件、合同、图片、评价附件等长期文件字段使用 `files`：

```go
Attachment string `json:"attachment" gorm:"column:attachment;type:text" widget:"name:附件;type:files;accept:.pdf,image/*;max_size:20MB;max_count:5"`
```

规则：

- Go 类型必须是 `string`，字段值是 `bucket/object_key`；多文件用英文逗号分隔。
- Table 新增/编辑表单会由前端上传文件并把 refs 提交给后端，后端通常直接保存这个字符串，不需要 `DownloadFiles`。
- 列表展示时前端会按 files 组件提供下载/预览能力。
- 后端只有在需要解析附件、生成摘要、批量处理或转换文件时，才调用 `ctx.GetFS().DownloadFiles(row.Attachment)` 下载到本地。
- 如果处理附件后生成新文件，输出到 `fs.GetTraceOutputDir()`，用 `fs.ResponseFiles(...)` 得到 refs，再保存到新的 files 字段或通过 Form Response 返回。

## 字段建模规则

基础 CRUD 必须一起读 `/system/prompt/sdk/widget-system`。常用规则：

- ID、创建时间、更新时间、创建人等系统字段通常只在列表展示。
- `types.Time` 转成标准库时间时使用方法调用：`t.Time().Format(...)`、`t.Time().After(...)`、`t.Time().Before(...)`。不要写 `t.Format(...)` 或 `t.Time.Format(...)`。
- 当前文档没有覆盖的时间赋值、空请求、分页结构、文件处理或上下文能力，先读对应知识点文档、案例或 SDK 源码；不要按命名直觉拼 SDK API。
- 状态、优先级、类型用 `select`，写 `options_colors`，颜色只能用不带 `#` 的 6 位十六进制 `RRGGBB`。
- 附件用 `files`，字段类型 `string`。
- 详情跳转用 `link`，字段通常 `gorm:"-"`。
- 计算字段、关联名称、状态文本通常 `gorm:"-"` + `hide:"create,update"`。
- 不要让复杂 struct、slice 作为 GORM 列落库。

## PRD 中怎么描述 Table

创建类 PRD 不要只写“创建一个 Table”。应写清楚：

```text
函数类型判断：
- 工单是长期业务记录，需要后台管理，因此选择 `ticket_list.table`。
- 前端会渲染为 Element 表格列表，支持状态/优先级/时间搜索、分页、列表列展示。
- 配置新增、编辑、删除回调后，前端会展示对应操作入口。
```

“列表模式”部分至少列出：

- 系统字段：ID、创建时间、更新时间、创建人。
- 业务字段：标题、状态、优先级、负责人等。
- 搜索字段：哪些字段支持模糊、枚举、范围。
- 操作入口：是否允许新增、编辑、删除、批量导入。
- 只读限制：流水/日志是否只读。

如果是收银记录、支付流水、导入历史、操作日志这类事实记录，PRD 必须明确：

```text
该表是只读记录表，只配置查询和搜索，不配置 OnTableAddRow / OnTableUpdateRow / OnTableDeleteRows；前端不会出现新增、编辑、删除入口。
```

PRD 必须额外写：

- 落地目录和函数清单：例如“确认后创建 `/用户/应用/ticket`，生成 `ticket_list.table`”，并说明前端是 Element 表格。
- 示例数据：至少给一到两条列表样例行，包含系统字段、核心业务字段、状态、搜索/操作入口，让用户能看懂表格最终展示效果。
- 确认后创建内容：在确认语前写清“确认后我将创建目录：xxx，并生成：xxx.table”。

## 验证

写完后必须：

1. `build_workspace`。
2. `run_table_search` 验证列表能返回。
3. 如果配置了新增，使用 `run_table_create` 验证。
4. 如果配置了编辑，使用 `run_table_update` 验证。
5. 如果配置了删除，使用 `run_table_delete` 验证。

验证失败时先读 build 错误或执行错误，再修复，不要直接总结完成。

## 推荐案例

- 单表 CRUD：`/system/prompt/case_catalog/table/ticket`
- 多表/关联展示：`/system/prompt/case_catalog/tables/meeting`
- 招聘主从表和 link：`/system/prompt/case_catalog/tables/hr`

## 常见错误

- 把一次性动作做成 Table，例如“上传 PDF 转文字”。
- 没有配置 `AutoCrudTable`。
- 路由不是 `.table`，或 `.table` 路由注册了 FormTemplate。
- 给流水/日志表默认开放新增编辑删除。
- 把所有字段都放进 Request 筛选，导致前端搜索区臃肿。
- `select` 没有 `options_colors`。
- `options_colors` 使用语义色、带 `#` 的 hex、`rgb(...)` 等非 `RRGGBB` 值，或数量和 `options` 不一致。
- 关联名称字段落库，正确做法通常是外键落库，名称 `gorm:"-"` 后处理填充。
- Request 中的筛选字段应服务真实查询路径；计算/展示字段筛选也通过 Request 入参进入 Handler，再转成真实表字段查询条件。
- 多文件拆分时重复定义同一个 model 类型，或让 Handler 函数名和 Model 类型名完全相同。
- Build 后没有验证 `run_table_search`。
