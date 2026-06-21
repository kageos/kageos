# pkg/gormx/query 用法说明

本包提供 Table 列表的**分页、排序**能力。业务筛选字段显式写在 Request 结构体中，并在 Handler 里手写 `Where` / `Joins` / `Preload`。

---

## URL 查询参数示例

表格列表接口基础路径：`GET /workspace/api/v1/table/search/{full-code-path}`，下面所有示例的 query 都拼在该路径后。分页排序统一使用 `page`、`page_size`、`sorts`。业务筛选参数使用 Request 字段自己的 `form` 名，例如 `status=启用&name=张三`。

### 仅分页

```
?page=1&page_size=20
```

### 分页 + 单列排序

```
?page=1&page_size=20&sorts=[{"field":"updated_at","order":"desc"}]
```

### 分页 + 多列排序

```
?page=1&page_size=20&sorts=[{"field":"status","order":"asc"},{"field":"created_at","order":"desc"}]
```

### 组合示例（分页 + 排序 + 业务筛选）

```
?page=2&page_size=10&sorts=[{"field":"created_at","order":"desc"},{"field":"id","order":"asc"}]&status=已发布&title=会议&start_time=2026-04-21 00:00:00&end_time=2026-04-21 23:59:59
```

对应含义：第 2 页、每页 10 条；按 `created_at` 降序、`id` 升序；`status`、`title`、`start_time`、`end_time` 由业务 Request 接收，并在 Handler 中显式转成查询条件。datetime 字段直接传 `YYYY-MM-DD HH:mm:ss`，工作台也支持 `CURRENT_TIMESTAMP`、`CURRENT_DATE`、`DATE_SUB(CURRENT_TIMESTAMP, INTERVAL 7 DAY)` 等白名单表达式。

### 完整 URL 示例

```
GET /workspace/api/v1/table/search/luobei/myapp/tables/hr_resume_list?page=1&page_size=20&sorts=[{"field":"updated_at","order":"desc"}]&job_department=技术&job_title=工程师&status=待筛选
```

---

## 一、后端核心类型

### 1. PageSortReq（分页和排序）

`PageSortReq` 只承接分页和排序。业务筛选字段应显式写在业务 Request 结构体中，并在 Handler 里手写 `Where` / `Joins` / `Preload`。

| 字段 | 类型 | 说明 |
|------|------|------|
| `Page` | int | 页码，从 1 开始 |
| `PageSize` | int | 每页条数 |
| `Sorts` | string | 结构化排序 JSON 数组，例如 `[{"field":"created_at","order":"desc"}]` |

### 2. PaginatedTable（分页结果）

泛型分页结果，和前端表格期望的字段一致：

```go
type PaginatedTable[T any] struct {
    Items       T     `json:"items"`        // 当前页数据
    CurrentPage int   `json:"current_page"`   // 当前页
    TotalCount  int64 `json:"total_count"`   // 总条数
    TotalPages  int   `json:"total_pages"`   // 总页数
    PageSize    int   `json:"page_size"`     // 每页条数
}
```

## 二、后端常用用法

### 1. 在 Table 接口里：请求体嵌入 PageSortReq

GET 请求的 Query 会被 SDK 的 `ShouldBind` 解析到结构体。表格列表的请求体通常显式声明业务筛选字段，并**内嵌** `query.PageSortReq`：

```go
import (
    "github.com/kageos/kageos-sdk/pkg/gormx/query"
    "github.com/kageos/kageos-sdk/agent-app/response"
)

type MyListReq struct {
    Status string `json:"status" form:"status" widget:"name:状态;type:select;options:待处理,已完成"`
    query.PageSortReq `widget:"-"`
}

func MyList(ctx *app.Context, resp response.Response) error {
    var req MyListReq
    if err := ctx.ShouldBind(&req); err != nil {
        return err
    }
    queryDB := ctx.GetGormDB().Model(&MyModel{})
    if req.Status != "" {
        queryDB = queryDB.Where("status = ?", req.Status)
    }

    if order := req.PageSortReq.GetOrder(); order != "" {
        queryDB = queryDB.Order(order)
    }

    var total int64
    if err := queryDB.Count(&total).Error; err != nil {
        return err
    }

    var list []MyModel
    if err := queryDB.
        Offset(req.PageSortReq.GetOffset()).
        Limit(req.PageSortReq.GetLimit()).
        Find(&list).Error; err != nil {
        return err
    }

    return resp.Table(response.TableResult{
        Items:      list,
        TotalCount: total,
        PageInfo:   &req.PageSortReq,
    }).Build()
}
```

- `PageSortReq` 只负责分页和排序参数；`GetOrder()` 返回可传给 GORM `Order()` 的安全排序 SQL。
- Handler 自己从任意数据源拿当前页数据和总数；DB 场景通常显式写 `Count`、`Order`、`Offset`、`Limit`、`Find`。
- `resp.Table(response.TableResult{...})` 只负责渲染 `items + paginated`，不查询数据库。

### 2. 排序格式说明

- 前端请求默认把结构化排序 JSON 放入 `sorts`，例如 `sorts=[{"field":"updated_at","order":"desc"},{"field":"name","order":"asc"}]`。
- 后端 `PageSortReq.GetOrder()` 解析后得到 ORDER BY 子句（列名会做安全校验并加反引号）。

### 3. 分页辅助方法

- `pageInfo.GetLimit(defaultSize...)`：每页条数，未传时默认 20。
- `pageInfo.GetOffset()`：当前页的 offset。
- `pageInfo.GetOrder()`：得到可传给 GORM `Order()` 的排序字符串。

---

## 三、前端与 URL 约定

Table 请求统一使用 `page`、`page_size`、`sorts` 做分页排序。筛选字段按 Request 字段名直接出现在 Query 里，并由后端 Handler 手写 `Where`。

### 1. 分页与排序

| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | number | 页码，从 1 开始 |
| `page_size` | number | 每页条数 |
| `sorts` | string | 结构化排序 JSON 数组，例如 `[{"field":"created_at","order":"desc"}]` |

### 2. 前端类型定义（types）

```ts
// web/src/types/index.ts
export interface SearchParams {
  sorts?: string
  page?: number
  page_size?: number
  [key: string]: unknown
}
```

- 多列排序：前端请求把 `sorts` 传成结构化 JSON，统一由 `GetOrder()` 解析。
- 业务筛选：前端按 Table Request 字段 code 直接拼普通 query 参数，例如 `status=启用&name=张三`。

---

## 四、前后端对接小结

1. **表格列表接口**：`GET /workspace/api/v1/table/search/{full-code-path}`，Query 里带 `page`、`page_size`、`sorts` 和业务 Request 字段，如 `status=启用&name=张三`。
2. **后端**：Req 嵌入 `query.PageSortReq`，筛选字段显式声明并经 `ShouldBind` 绑定；Handler 里手写 `Where` / `Joins` / `Preload` / `Count` / `Find`，最后用 `resp.Table(response.TableResult{Items: rows, TotalCount: total, PageInfo: &req.PageSortReq}).Build()` 返回。

这样业务筛选收敛在 Request 和 Handler 中，分页排序继续复用 `pkg/gormx/query` 的稳定约定。
