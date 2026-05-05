# search 标签与前端查询串逻辑

本文记录 Table 搜索栏里 `search` 标签、前端 URL query、后端查询参数之间的约定。对应代码主要在：

- `sdk/agent-app/widget/decode.go`
- `web/src/utils/functionSchemaSelectors.ts`
- `web/src/utils/searchParams.ts`
- `web/src/architecture/domain/services/TableDomainService.ts`
- `web/src/architecture/presentation/views/utils/tableViewURLRuntime.ts`
- `pkg/gormx/query/query1.go`

## 1. Go 标签如何进入前端 schema

Go struct 字段上的 `search` tag 会被 widget decoder 读取，并写入前端字段配置的 `field.search`。

```go
Title string `json:"title" widget:"name:标题;type:input" search:"like"`
Status string `json:"status" widget:"name:状态;type:select;options:启用,禁用;options_colors:success,default" search:"in"`
```

字段 code 默认来自 `json` tag，例如 `json:"title"` 会生成 `field.code = "title"`。`json:"-"` 或 `widget:"-"` 会跳过该字段。

`search:"-"` 表示该字段不进入 Table 搜索栏。

## 2. Table 搜索字段分两类

前端会把 Table 的 `request` 字段和 `fields` 字段合并成一个搜索栏，但二者生成 query string 的方式不同。

### request 字段

`table.request` 是 sdk-app 的函数入参。它进入搜索栏的规则是：只要 `search` 不是 `-`，就可以作为搜索条件。

生成查询串时，request 字段使用原始字段名，不包在后端搜索操作符里：

```text
genre=诗
author=李白
```

也就是说，对于 request 字段，`search` 主要用于控制是否展示搜索控件，并影响搜索控件形态；最终传给 sdk-app 时仍然是 `field.code=value`。

### response 字段

`table.fields` 是表格结果集字段。它进入搜索栏的规则更严格：必须显式配置 `search`，且不能是 `-`。

生成查询串时，response 字段会走后端搜索操作符：

```text
like=title:李白
in=status:open,closed
gte=created_at:2026-05-01 00:00:00&lte=created_at:2026-05-02 23:59:59
```

request 字段和 response/table model 字段 code 不能相同。旧逻辑里前端会优先保留 request 字段，容易让同名表字段的搜索条件被覆盖；现在 SDK 启动期会直接拦截这类重复 code。

## 3. 前端当前支持的搜索操作符

前端 Table URL 当前会生成这些搜索参数：

| search tag | URL 参数 | 语义 |
| --- | --- | --- |
| `eq` | `eq=field:value` | 精确匹配 |
| `like` | `like=field:value` | 模糊匹配 |
| `in` | `in=field:v1,v2` | 字段值在列表内 |
| `contains` | `contains=field:v1,v2` | CSV/多值字段包含某些值 |
| `gte,lte` | `gte=field:min&lte=field:max` | 范围查询 |

SDK schema 只允许这些前端 Table URL builder 会生成的操作符。不要在 `search` tag 里写 `gt`、`lt`、`not_eq`、`not_like`、`not_in`；这些底层后端参数即使存在，也不会作为当前 SDK Table 搜索栏能力暴露，启动期会直接失败。

`search` tag 可以是逗号组合，例如 `search:"gte,lte"`。但除了范围查询必须同时配置 `gte,lte` 外，前端构造查询串时是按优先级命中第一个操作符，不会为一个字段同时生成多种搜索条件。

SDK 启动期会同步校验这条约束：`search:"eq,like"`、`search:"like,in"` 这类组合会失败；合法组合只有单个操作符或 `search:"gte,lte"`。

当前优先级是：

```text
eq -> like -> contains -> in -> gte,lte
```

注意：前端判断搜索类型时使用字符串包含判断，`contains` 字符串里包含 `in`。因此构造查询串时必须先判断 `contains` 再判断 `in`。

## 4. 查询串格式

response 字段使用统一格式：

```text
operator=field:value
```

多个字段使用同一个 operator 时，用逗号追加：

```text
like=title:李白,summary:唐诗
eq=status:published,kind:poem
```

数组值也使用逗号拼接：

```text
in=status:open,closed
contains=tags:唐诗,律诗
```

范围值拆成 `gte` 和 `lte`：

```text
gte=created_at:2026-05-01 00:00:00&lte=created_at:2026-05-02 23:59:59
gte=score:60&lte=score:100
```

空值不会进入查询串。这里的空值包括 `null`、`undefined`、空字符串和空数组。

浏览器地址栏里中文、空格、冒号等字符会被 URL encode；前端状态和 axios 参数对象里仍然按上面的逻辑组装。

## 5. URL 同步与恢复

Table 搜索状态保存在前端的 `searchForm` 中。搜索控件变更时会更新 `searchForm`、同步 URL，并重新加载表格数据。

URL 中 Table 相关参数包括：

```text
page
page_size
sorts
eq
like
in
contains
gte
lte
request field code
```

`sorts` 的格式是：

```text
sorts=created_at:desc,title:asc
```

页面初始化或浏览器 query 变化时，前端会从 URL 恢复搜索状态：

- request 字段从原始 query key 恢复，例如 `genre=诗`。
- response 字段从操作符参数恢复，例如 `like=title:李白`。
- `in`、`contains` 会按逗号拆回数组或单值。
- `gte,lte` 对 datetime 字段恢复成 `[start, end]`，对非 datetime 范围恢复成 `{ min, max }`。

URL 同步时还会保留平台内部 `_` 开头的状态参数。旧的或生成态参数，比如 `s_`、`f_`、`__display`，不会作为 Table 搜索参数继续写回。

## 6. 前端发给后端的请求

Table 搜索接口是：

```text
GET /workspace/api/v1/table/search{fullCodePath}
```

前端会把分页、排序、request 字段和 response 搜索操作符一起作为 query params 传入。

示例：

```text
/workspace/api/v1/table/search/demo/poem/list?page=1&page_size=20&sorts=id:desc&genre=唐诗&like=title:李白&in=status:published,draft&gte=created_at:2026-05-01 00:00:00&lte=created_at:2026-05-02 23:59:59
```

其中：

- `genre=唐诗` 是 request 字段，直接传给 sdk-app 函数入参。
- `like=title:李白`、`in=status:published,draft`、`gte/lte=created_at:...` 是 response 字段搜索条件。

Table 的 request 字段 code 不能和 response/table model 字段 code 重名。否则 request 原始 query 参数和表字段搜索条件会在前端查询状态里产生覆盖歧义；SDK 启动期会直接报错。

## 7. 后端解析语义

后端 `query.SearchFilterPageReq` 接收这些搜索参数：

```go
Eq       []string `form:"eq" json:"eq"`
Like     []string `form:"like" json:"like"`
In       []string `form:"in" json:"in"`
Contains []string `form:"contains" json:"contains"`
Gte      []string `form:"gte" json:"gte"`
Lte      []string `form:"lte" json:"lte"`
```

主要语义：

- `eq`：等于，后端会尝试把值转成数字或 bool。
- `like`：生成 `%value%` 模糊查询。
- `in`：生成 SQL `IN`。
- `contains`：用于存储为逗号分隔的多值字段，判断字段集合中是否包含传入值。
- `gte` / `lte`：比较查询，SDK 只暴露成 `search:"gte,lte"` 范围组合。

后端会校验字段名，只允许字母、数字和下划线，避免把不安全列名拼入查询。

## 8. 使用建议

新增 Table 搜索字段时，先判断字段属于 request 还是 response：

- 如果它是函数入参，放在 request 里，query string 会是 `field=value`。
- 如果它是表格结果字段，放在 fields 里，并显式写 `search` tag，query string 会是 `operator=field:value`。

常见配置：

```go
// request 字段：控制 sdk-app 入参
Genre string `json:"genre" widget:"name:体裁;type:input" search:"like"`

// response 字段：控制表格结果集搜索
ID        int        `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" search:"eq" display:"scenes:list"`
Title     string     `json:"title" widget:"name:标题;type:input" search:"like"`
Status    string     `json:"status" widget:"name:状态;type:select;options:草稿,已发布;options_colors:default,success" search:"in"`
Tags      string     `json:"tags" widget:"name:标签;type:multiselect;options:推荐,热门;options_colors:primary,warning" search:"contains"`
CreatedAt types.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" display:"scenes:list"`
```

系统审计字段按约定写死：主键 `id` 使用 `search:"eq"`，`created_at` / `updated_at` 使用 `search:"gte,lte"` 并只在列表展示，`create_by` 等用户字段使用 `search:"in"`，`deleted_at` 使用 `widget:"-"` 或 `json:"-"` 隐藏。

如果需要前端 Table 搜索栏支持 `gt`、`lt` 或 `not_*` 操作符，需要同时扩展：

- `sdk/agent-app/widget/validator.go` 的 search 操作符白名单和类型校验
- `web/src/core/constants/search.ts` 的操作符定义
- `web/src/utils/functionSchemaSelectors.ts` 的搜索字段筛选逻辑
- `web/src/utils/searchParams.ts` 的构造与解析逻辑
- `web/src/architecture/domain/services/TableDomainService.ts` 的 URL 恢复逻辑
- `web/src/architecture/presentation/views/utils/tableViewURLRuntime.ts` 的 URL 同步逻辑
- 对应搜索控件的展示与取值逻辑
