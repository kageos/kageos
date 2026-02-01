# pkg/gormx/query 用法说明

本包提供**分页、排序、搜索过滤**的统一能力：后端用 GORM 构建 WHERE/ORDER/LIMIT，前端用同一套参数约定传参。表格接口 `GET /workspace/api/v1/table/search/{full-code-path}` 的查询串与这里完全对齐。

---

## URL 查询参数示例

表格列表接口基础路径：`GET /workspace/api/v1/table/search/{full-code-path}`，下面所有示例的 query 都拼在该路径后。

### 仅分页

```
?page=1&page_size=20
```

### 分页 + 单列排序

```
?page=1&page_size=20&sorts=id:desc
```

或使用减号表示降序：

```
?page=1&page_size=20&sorts=-updated_at
```

### 分页 + 多列排序

```
?page=1&page_size=20&sorts=status:asc,created_at:desc
```

### 精确匹配（eq）

单个字段：

```
?page=1&page_size=20&eq=status:待处理
```

多个字段（同一 eq 内用逗号分隔多组 `field:value`）：

```
?page=1&page_size=20&eq=status:待处理,dept_id:5
```

### 模糊匹配（like）

```
?page=1&page_size=20&like=name:张三
```

多字段模糊：

```
?page=1&page_size=20&like=name:张,remark:测试
```

### 多选 IN（in）

单个字段多个值：

```
?page=1&page_size=20&in=status:待处理,处理中,已完成
```

多字段（格式 `field1:v1,v2,field2:v3,v4`）：

```
?page=1&page_size=20&in=status:待处理,处理中,level:1,2
```

### 多选包含（contains，FIND_IN_SET 语义）

```
?page=1&page_size=20&contains=tags:高,中
```

### 范围（gte / lte）

```
?page=1&page_size=20&gte=created_at:1704067200&lte=created_at:1704153600
```

数字范围示例：

```
?page=1&page_size=20&gte=score:60&lte=score:100
```

### 大于 / 小于（gt / lt）

```
?page=1&page_size=20&gt=amount:100&lt=amount:1000
```

### 否定条件（not_eq / not_in / not_like）

```
?page=1&page_size=20&not_eq=status:已删除
?page=1&page_size=20&not_in=status:草稿,已废弃
?page=1&page_size=20&not_like=name:测试
```

### 组合示例（分页 + 排序 + 多条件）

```
?page=2&page_size=10&sorts=-created_at,id:asc&eq=status:已发布&like=title:会议&gte=created_at:1704067200&lte=created_at:1704153600
```

对应含义：第 2 页、每页 10 条；按 `created_at` 降序、`id` 升序；`status` 精确为「已发布」；`title` 模糊包含「会议」；`created_at` 在给定时间戳范围内。

### 完整 URL 示例

```
GET /workspace/api/v1/table/search/luobei/myapp/tables/hr_resume_list?page=1&page_size=20&sorts=-updated_at&eq=job_department:技术&like=job_title:工程师&in=status:待筛选,已通过
```

---

## 一、后端核心类型

### 1. SearchFilterPageReq（请求参数）

前端通过 URL Query 传过来的分页 + 排序 + 筛选，在后端用该结构体承接（需带 `form` 标签，便于 GET 时从 Query 解码）：

| 字段 | 类型 | 说明 |
|------|------|------|
| `Page` | int | 页码，从 1 开始 |
| `PageSize` | int | 每页条数 |
| `Sorts` | string | 排序，格式：`field1:asc,field2:desc`，也支持 `-field` 表示 desc |
| `Keyword` | string | 关键词（若业务用） |
| `Eq` | []string | 精确匹配，格式：`field:value`，可多条 |
| `Like` | []string | 模糊匹配，格式：`field:value` |
| `In` | []string | IN 查询，格式：`field:value1,value2` 或 多字段 `field1:v1,v2,field2:v3,v4` |
| `Contains` | []string | 多选语义，存的是逗号分隔串时用 FIND_IN_SET 语义，格式：`field:value1,value2` |
| `Gt`, `Gte`, `Lt`, `Lte` | []string | 大于/大于等于/小于/小于等于，格式：`field:value` |
| `NotEq`, `NotLike`, `NotIn` | []string | 不等于 / 模糊取反 / NOT IN |

- 多条件同一类型：如多条 `eq`，可在前端用逗号拼成一条，或后端用 `[]string` 多元素，最终都会解析成多组 `field:value` 施加到 WHERE。

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

### 3. QueryConfig（可选：字段白名单/黑名单）

需要对“可查字段”做限制时使用：

- `AllowField(field, operators...)`：允许某字段使用哪些操作符（如 `eq`, `like`, `in`）。
- `DenyField(field)`：禁止某字段参与查询。

不传 `QueryConfig` 时，仅做列名安全校验（防注入），不限制字段。

---

## 二、后端常用用法

### 1. 在 Table 接口里：请求体嵌入 SearchFilterPageReq

GET 请求的 Query 会被 SDK 的 `ShouldBind` 解析到结构体。表格列表的请求体通常**内嵌** `query.SearchFilterPageReq`，再在 handler 里把“当前请求”的这份分页/排序/筛选传给框架：

```go
import (
    "github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
    "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
)

type MyListReq struct {
    Status string `json:"status" form:"status" widget:"name:状态;type:select;options:待处理,已完成"`
    query.SearchFilterPageReq `widget:"-"`
}

func MyList(ctx *app.Context, resp response.Response) error {
    var req MyListReq
    if err := ctx.ShouldBind(&req); err != nil {
        return err
    }
    db := ctx.GetGormDB()
    var list []MyModel
    return resp.Table(&list).AutoSearchFilterPaged(db, &MyModel{}, &req.SearchFilterPageReq).Build()
}
```

- 前端把 `page`、`page_size`、`sorts`、`eq`、`like`、`in` 等全部放在 URL Query 里；后端只需 `ShouldBind` 一次，即可得到 `SearchFilterPageReq`，再交给 `AutoSearchFilterPaged` 即可完成条件 + 排序 + 分页 + 写回 `items/total_count/current_page` 等。

### 2. 仅应用条件、不自动分页（自定义分页逻辑时）

若你自己写 Count/Find，只希望“把前端传来的筛选 + 排序”应用到某条 GORM 链上：

```go
dbWithConditions, err := query.ApplySearchConditions(db, pageInfo)
if err != nil {
    return err
}
// 之后用 dbWithConditions 做 Count、Order、Offset、Limit、Find 等
```

### 3. 一键分页查询（泛型）

若希望直接得到 `PaginatedTable[T]`，可用：

```go
var items []MyModel
result, err := query.AutoPaginateTable(ctx, db, &MyModel{}, &items, pageInfo)
// result.Items 即 &items，result.TotalCount / CurrentPage / TotalPages / PageSize 已填好
```

### 4. 使用 QueryConfig 限制可查字段

```go
cfg := query.NewQueryConfig()
cfg.AllowField("status", "eq", "in")
cfg.AllowField("name", "like")
cfg.DenyField("internal_secret")
dbWithConditions, err := query.ApplySearchConditions(db, pageInfo, cfg)
```

### 5. 排序格式说明

- 前端常用：`sorts=id:desc,name:asc` 或 `sorts=-updated_at`（减号表示 desc）。
- 后端 `SearchFilterPageReq.Sorts` 解析后得到 ORDER BY 子句（列名会做安全校验并加反引号）。
- `WithSorts(sorts)` 可在已有排序上追加默认排序（不重复字段）。

### 6. 分页辅助方法

- `pageInfo.GetLimit(defaultSize...)`：每页条数，未传时默认 20。
- `pageInfo.GetOffset()`：当前页的 offset。
- `pageInfo.GetSorts()`：得到可传给 GORM `Order()` 的排序字符串。

---

## 三、前端与 URL 约定（和本包对齐）

前端表格请求同一套 Query 参数，便于和后端 `SearchFilterPageReq` 一一对应。

### 1. 分页与排序

| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | number | 页码，从 1 开始 |
| `page_size` | number | 每页条数 |
| `sorts` | string | 排序：`field1:asc,field2:desc` 或 `-field` 表示 desc |

### 2. 筛选参数（与后端操作符一致）

| 参数 | 格式 | 说明 |
|------|------|------|
| `eq` | `field:value` 或 `field1:v1,field2:v2` | 精确匹配 |
| `like` | `field:value` 或 多字段逗号分隔 | 模糊匹配（前后 %） |
| `in` | `field:value1,value2` | IN 查询；多字段：`field1:v1,v2,field2:v3,v4` |
| `contains` | `field:value1,value2` | 多选/逗号分隔存库时的“包含”语义（FIND_IN_SET 风格） |
| `gte` / `lte` | `field:value` | 大于等于 / 小于等于（范围） |
| `gt` / `lt` | `field:value` | 大于 / 小于 |
| `not_eq` / `not_like` / `not_in` | 同 eq/like/in | 取反 |

- 多字段同类型：同一参数名用逗号拼多个 `field:value`，例如：`eq=status:1,name:测试`。

### 3. 前端类型定义（types）

```ts
// web/src/types/index.ts
export interface SearchParams {
  eq?: string
  like?: string
  in?: string
  contains?: string
  gte?: string
  lte?: string
  sorts?: string   // 同上
  page?: number
  page_size?: number
}
```

### 4. 前端如何生成这些参数（与后端一致）

- **response 字段（表内可搜字段）**：根据字段的 `search` 配置，用 `buildSearchParamsString` / `buildURLSearchParams` 把表单值转成 `eq`/`like`/`in`/`contains`/`gte`/`lte` 等。
- **request 字段**：通常作为额外查询条件，按业务需要拼到同一请求里（或作为普通 query 键值）。

`search` 与操作符对应关系（见 `web/src/core/constants/search.ts` 和 `web/src/utils/searchParams.ts`）：

| 字段 search 配置 | 生成参数 | 说明 |
|------------------|----------|------|
| `eq` | `eq=field:value` | 精确匹配 |
| `like` | `like=field:value` | 模糊 |
| `in` | `in=field:v1,v2` | 多选 IN |
| `contains` | `contains=field:v1,v2` | 多选 FIND_IN_SET 语义 |
| `gte` + `lte` | `gte=field:v1&lte=field:v2` | 范围（日期/数字） |

- 多列排序：前端把 `sorts` 拼成 `field1:order1,field2:order2`，和 backend `ParseSortFields` / `GetSorts()` 兼容。

---

## 四、前后端对接小结

1. **表格列表接口**：`GET /workspace/api/v1/table/search/{full-code-path}`，Query 里带 `page`、`page_size`、`sorts`、`eq`、`like`、`in`、`contains`、`gte`、`lte` 等。
2. **后端**：用嵌入 `query.SearchFilterPageReq` 的 Req 做 `ShouldBind`，再把 `&req.SearchFilterPageReq` 传给 `resp.Table(...).AutoSearchFilterPaged(..., pageInfo).Build()` 即可。
3. **前端**：用 `SearchParams` + `buildSearchParamsString` / `buildURLSearchParams` 根据表单和 `field.search` 生成上述 Query，和本包约定一致，无需再为表格单独发明一套参数。

这样后端 `pkg/gormx/query` 与前端表格搜索/分页/排序共用同一套约定，对接清晰、易维护。
