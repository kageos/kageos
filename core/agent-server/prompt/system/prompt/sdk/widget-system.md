# SDK Widget 组件系统

本文档只负责字段组件系统。创建或修改 Form/Table/Chart 时，若涉及字段建模、前端控件、搜索、校验、展示场景、link、files、动态下拉，必须读取本文档。业务类型选择先读 `/system/prompt/platform-function-architecture`，具体 Table/Form/Chart 写法再读对应任务包。

## 总原则

SDK 代码中的 struct tag 是前端 schema 协议。模型写字段时不是只写 Go 类型，还要同时决定：

1. 前端展示什么组件。
2. 字段值如何提交给后端。
3. 是否落库。
4. 是否可搜索。
5. 是否参与新增、编辑、列表展示。
6. 是否需要校验。

标准字段写法：

```go
Title string `json:"title" gorm:"column:title" widget:"name:标题;type:input;placeholder:请输入标题" search:"like" validate:"required,min=2,max=100"`
```

只要写 `widget` 标签，必须显式写 `type`。不要生成 SDK 不支持的组件类型。

widget tag 只能使用本文档列出的组件类型和配置 key。不要把前端习惯参数直接写进 SDK tag；如果某个 key 没在本文档或已读源码里出现，就先读取对应知识点或源码确认。

## 常用组件选择

| 业务含义 | Go 类型 | widget type | 说明 |
|---|---|---|---|
| 主键 ID | `int` | `ID` | 必须大写 `ID`，常配 `display:"scenes:list"` 和 `search:"eq"` |
| 单行文本 | `string` | `input` | 标题、名称、手机号、编号 |
| 多行文本 | `string` | `text_area` | 描述、备注、原因 |
| 富文本 | `string` | `richtext` | 公告正文、长内容 |
| 只读文本 | `string` | `text` | 计算结果、展示字段 |
| 单选下拉 | `string` / `int` | `select` | 状态、类型、关联对象；固定枚举要写 `options_colors` |
| 单选按钮 | `string` | `radio` | 2-5 个固定选项 |
| 固定多选 | `[]string` 或 `string` | `checkbox` | 少量固定复选项 |
| 下拉多选 | `string` 或 `[]int` | `multiselect` | 标签、多个关联项；候选项来自 `options` 或 OnSelectFuzzy |
| 数字 | `int` | `number` | 数量、次数 |
| 小数 | `float64` | `float` | 金额、比例；配置 `precision`、`step`、`unit` |
| 进度滑块 | `int` | `slider` | 进度、评分，配置 `min`、`max` |
| 开关 | `bool` | `switch` | 是否启用、是否匿名 |
| 日期时间 | `types.Time` | `datetime` | 使用真实时间类型，格式通常为 `YYYY-MM-DD HH:mm:ss` |
| 文件 | `string` | `files` | 文件引用，字段值为 `bucket/object_key`，多文件逗号分隔 |
| 用户 | `string` | `user` | 单个用户 |
| 多用户 | `string` | `users` | 多个用户 |
| 部门 | `string` | `department` | 单个部门 |
| 多部门 | `string` | `departments` | 多个部门 |
| 跳转链接 | `string` | `link` | 只读跳转，一般 `gorm:"-"` + `display:"scenes:list"` |
| 表单内子表 | `[]Struct` | `table` | Form 场景的明细行，如收银商品清单 |
| 嵌套对象 | `Struct` | `form` | Form 响应或复杂输入的分组对象 |

## 枚举组件

`select`、`multiselect`、`radio`、`checkbox` 的静态选项写在 `options` 中。生成静态 `select` 和静态 `multiselect` 代码时必须同时写 `options_colors`；动态 OnSelectFuzzy 下拉不写 `options`，也不要写 `options_colors`。

```go
Status string `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:待处理,处理中,已完成;options_colors:909399,E6A23C,67C23A;render_default:待处理" search:"in" validate:"required,oneof=待处理 处理中 已完成"`
```

规则：

- `options` 的文本就是前端实际提交值。
- `options_colors` 必须和 `options` 一一对应，数量完全一致。
- `options_colors` 只允许 6 位十六进制 `RRGGBB`，不带 `#`，例如 `67C23A`、`E6A23C`、`F56C6C`；不要生成 `success`、`warning`、`danger`、`primary`、`info`、`default`、`secondary`、`#67C23A`、`rgb(...)`。
- `validate:"oneof=..."`、`required_if`、`excluded_unless` 等条件值必须和实际提交值一致。
- 不要用中文展示值，但校验里写英文 code。
- `creatable:true` 只表示允许用户新增选项，不能替代选项来源。
- 动态选项必须加 `callback:"OnSelectFuzzy"`，并在模板的 `OnSelectFuzzyMap` 注册同名 key。

## 搜索标签

Table 的 Request/列表字段可以使用 `search`，前端会按 schema 渲染搜索条件。

常用值：

- `eq`：精确匹配，适合 ID、开关、单值。
- `like`：模糊搜索，适合标题、名称、描述。
- `in`：多值筛选，适合状态、用户、部门、枚举。
- `contains`：字符串集合包含，适合多选值存储。
- `gte,lte`：范围筛选，适合时间、金额、进度。

不要给所有字段都加搜索。搜索字段应服务用户真实查询路径。

## 展示场景

`display:"scenes:..."` 控制字段在哪些界面出现：

| scenes | 含义 |
|---|---|
| `list` | 只在 Table 列表展示，不进入新增/编辑表单 |
| `create` | 只在新增表单展示 |
| `update` | 只在编辑表单展示 |
| `create,update` | 新增/编辑展示，列表不展示 |

规则：

- 系统字段如 `ID`、创建时间、更新时间、创建人通常 `display:"scenes:list"`。
- 只读计算字段、link 字段、关联展示字段通常 `gorm:"-"` + `display:"scenes:list"`。
- 不要写未确认的 `readonly` 之类 widget 参数来控制只读。Table 字段是否进入新增/编辑表单由 `display` 控制；内部字段用 `widget:"-"` 隐藏。
- `widget:"-"` 表示完全不进入前端 schema，适合 `DeletedAt`、内部关联对象、敏感中间字段。

## 系统字段约定

Table Model 常见系统字段：

```go
ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" search:"eq" display:"scenes:list"`
CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" display:"scenes:list"`
UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" display:"scenes:list"`
DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
```

创建人、更新人如果由业务维护，使用 `type:user`，并通常只在列表展示：

```go
CreateBy string `json:"create_by" gorm:"column:create_by" widget:"name:创建人;type:user" search:"in" display:"scenes:list"`
```

## files 字段

文件上传统一用 `type:files`，Go 字段类型用 `string`。

```go
Attachment string `json:"attachment" gorm:"column:attachment;type:text" widget:"name:附件;type:files;max_count:5"`
```

规则：

- 字段值是稳定文件引用：`bucket/object_key`。
- 多文件用英文逗号分隔。
- 图片、PDF、视频、Excel 都先用 `files`，不要编造 `file`、`image`、`pdf`、`video` widget。
- 多文件能力通过 `max_count` 表达，不要写未确认的 `multiple` 参数。
- Form 文件处理工具通常用 `files` 输入，Response 用普通字段、link 或 files 字段返回结果。

## link 字段

link 用于在 Table 列表或 Form 响应中跳转到另一个函数。

```go
DetailLink string `json:"detail_link" gorm:"-" widget:"name:查看详情;type:link;target:_blank" display:"scenes:list"`
```

后端用 SDK 构建链接：

```go
items[i].DetailLink, _ = ctx.BuildFunctionUrlWithText("ticket_list.table", Ticket{ID: items[i].ID}, "查看详情")
```

规则：

- 跳 Table：params 使用目标 Table 的 Model 或 Request，可用于筛选。
- 跳 Form：params 使用目标 Form 的 Request，可用于预填。
- 跳 Chart：params 使用目标 Chart 的 Request，可用于图表筛选。
- link 字段通常不落库，列表 Build 后填充。
- 不要手拼 URL JSON。

## OnSelectFuzzy

当 select/multiselect 的选项来自数据库、需要关键字搜索、需要按其他字段联动时，使用 OnSelectFuzzy。

字段：

```go
TopicID int `json:"topic_id" widget:"name:投票主题;type:select" validate:"required" callback:"OnSelectFuzzy"`
OptionIDs []int `json:"option_ids" widget:"name:投票选项;type:multiselect;depend_on:topic_id" validate:"required,min=1" callback:"OnSelectFuzzy"`
```

外键搜索也优先用 OnSelectFuzzy。字段 code 保持数据库语义，例如 `object_id`、`customer_id`、`product_id`；前端名称写业务名，不要写“评价对象ID”“客户ID”。

Table 新增/编辑表单里的外键字段必须写在 Table Model 上，并在同一个 `TableTemplate.BaseConfig.OnSelectFuzzyMap` 注册同名 key。不要只把外键字段放到 Table Request；Request 是搜索入参，不会给 `AutoCrudTable` 的新增/编辑字段提供回调。例：拍卖品管理的场次选择，应在 `AuctionItem.SessionID` 上写 `widget:"name:场次;type:select" callback:"OnSelectFuzzy"`，并注册 `"session_id": onSelectFuzzySession`。

```go
ObjectID int `json:"object_id" gorm:"column:object_id" widget:"name:评价对象;type:select" search:"in" callback:"OnSelectFuzzy"`
ObjectName string `json:"object_name" gorm:"-" widget:"name:评价对象;type:text" display:"scenes:list"`
```

模板：

```go
OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
    "topic_id": voteOnSelectFuzzyTopic,
    "option_ids": voteOnSelectFuzzyOption,
}
```

规则：

- `OnSelectFuzzyMap` 的 key 必须等于字段 `json` 名。
- 依赖其他字段时使用 `depend_on`，并保证依赖字段在表单上方。
- 回显已选值时也会调用回调，回调需要支持 by_value / by_values。
- Form 子表里的 select 也可以用 OnSelectFuzzy，例如收银台商品明细。
- Table 新增/编辑表单里的 select 也可以用 OnSelectFuzzy，例如会议室预约选择可用会议室。
- Table 搜索里的外键 select 也可以用 OnSelectFuzzy，例如评价记录按评价对象筛选；用户按名称搜索，前端提交 ID。
- 历史记录查询的外键回调不要只返回当前可用对象；已关闭对象也可能有历史记录。提交类 Form 才按业务限制只返回可提交对象。

## 常见错误

- 写了 `widget:"name:状态"` 但没写 `type`。
- 把 ID 写成 `type:id`，正确是 `type:ID`。
- 编造 `file`、`date`、`time`、`range`、`image`、`tag`、`tree`、`cascader` 等未支持类型。
- 按前端习惯编造 `readonly`、`multiple` 等未支持 widget 参数；只读用 `display` 或 `widget:"-"` 控制。
- `float64` 字段用了 `type:number`；小数、均值、金额、比例要用 `type:float`。
- `select` 有 `options` 但没有 `options_colors`。
- `options_colors` 数量和 `options` 不一致，或使用了语义色、带 `#` 的 hex、`rgb(...)` 等非 `RRGGBB` 值。
- `select` 没有 `options`，也没有 `callback:"OnSelectFuzzy"`。
- 多选枚举该用 `multiselect` 却用了 `list`。
- `files` 字段写成 `[]string` 或复杂结构，正确是 `string`。
- link 字段落库，或进入新增/编辑表单。
- 关联对象、子表结构直接作为 GORM 列落库。
