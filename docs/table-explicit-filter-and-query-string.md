# Table 显式筛选与前端查询串逻辑

本文记录 Table 列表筛选、分页、排序在前端 URL query 和后端 Request 之间的约定。Table 列表统一使用 `query.PageSortReq`；业务筛选字段显式写在 Request 结构体中，并在 Handler 里手写查询条件。

对应代码主要在：

- `sdk/agent-app/widget/decode.go`
- `web/src/utils/functionSchemaSelectors.ts`
- `web/src/architecture/domain/services/TableDomainService.ts`
- `web/src/architecture/presentation/views/utils/tableViewURLRuntime.ts`
- `pkg/gormx/query/query1.go`

## 1. Go 字段如何进入前端 schema

字段 code 默认来自 `json` tag，例如 `json:"title"` 会生成 `field.code = "title"`。`json:"-"` 或 `widget:"-"` 会跳过该字段。

Table Request 中的业务字段会进入筛选区；嵌入的 `query.PageSortReq` 使用 `widget:"-"` 隐藏，只负责接收分页和排序参数。

```go
type TicketListReq struct {
    Title  string `json:"title" form:"title" widget:"name:工单标题;type:input"`
    Status string `json:"status" form:"status" widget:"name:状态;type:select;options:待处理,处理中,已完成;options_colors:909399,E6A23C,67C23A"`

    query.PageSortReq `widget:"-"`
}
```

## 2. URL 查询串格式

分页和排序参数固定为：

```text
page=1
page_size=20
sorts=-created_at,title
```

排序也支持数组 query 形态：

```text
sort[]=-created_at&sort[]=title
```

业务筛选字段按 Request 字段 code 直接进入 query：

```text
title=会议&status=处理中&handler=zhangsan
```

完整示例：

```text
/workspace/api/v1/table/search/demo/ticket/list?page=1&page_size=20&sorts=-id&title=会议&status=处理中
```

空值不会进入查询串。这里的空值包括 `null`、`undefined`、空字符串和空数组。浏览器地址栏里中文、空格、冒号等字符会被 URL encode；前端状态和 axios 参数对象里仍然按上面的逻辑组装。

## 3. URL 同步与恢复

Table 筛选状态保存在前端的 `searchForm` 中。筛选控件变更时会更新 `searchForm`、同步 URL，并重新加载表格数据。

URL 中 Table 相关参数包括：

```text
page
page_size
sorts
sort[]
request field code
```

页面初始化或浏览器 query 变化时，前端会从 URL 恢复筛选状态。URL 同步时还会保留平台内部 `_` 开头的状态参数；生成态或展示态参数不会作为 Table 筛选参数继续写回。

## 4. 后端处理方式

后端先 `ShouldBind` 到 Request，再显式构建 `queryDB`：

```go
func TicketList(ctx *app.Context, resp response.Response) error {
    var req TicketListReq
    if err := ctx.ShouldBind(&req); err != nil {
        return err
    }

    queryDB := ctx.GetGormDB().Model(&Ticket{})
    if req.Title != "" {
        queryDB = queryDB.Where("title LIKE ?", "%"+req.Title+"%")
    }
    if req.Status != "" {
        queryDB = queryDB.Where("status = ?", req.Status)
    }

    var rows []Ticket
    return resp.Table(&rows, queryDB, &Ticket{}, &req.PageSortReq).Build()
}
```

`Table` 只处理 Count、排序、Offset、Limit、Find 和分页信息写回；所有业务筛选、关联查询、预加载都在 `Build()` 前显式完成。
