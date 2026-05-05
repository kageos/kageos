# SDK 与工作台代码生成逻辑说明

这份文档用于让大模型、后端开发、前端开发或新加入项目的人快速理解：

- `sdk/agent-app` 到底提供了什么能力；
- 工作台为什么能基于 SDK、skills、docs 和 tools 生成可运行代码；
- `namespace` 里的代码是如何被平台识别、渲染、调用和调试的；
- 大模型在生成代码时必须遵守哪些边界。

一句话概括：

> AI Agent OS 的 SDK 不是普通 Go 工具库，而是工作台应用协议层。开发者或大模型写 Go 代码，SDK 把 Go 结构体、模板、路由、回调、DB、文件、用户上下文转换成工作台可以渲染、调用、调试和管理的函数资产。

## 1. 整体心智模型

一个工作台应用通常长这样：

```text
namespace/{user}/{app}/code/api/{package}/
  init_.go
  xxx_table.go
  xxx_form.go
  xxx_chart.go
```

开发者或大模型写的是 Go 文件，但平台最终关心的是一棵函数树：

```text
应用
└── 业务包 package
    ├── 表格函数 xxx.table
    ├── 表单函数 xxx.form
    └── 图表函数 xxx.chart
```

每个函数由两部分组成：

- **Template**：声明这个函数是什么、需要什么输入、返回什么输出、前端应该如何渲染；
- **Handler**：真正执行业务逻辑，读请求、查数据库、处理文件、返回结果。

因此，最重要的理解方式是：

> Go struct tag 生成 schema，schema 驱动工作台 UI，handler 执行业务逻辑，tools 按 full_code_path 调用函数。

## 2. 核心链路

从大模型生成代码到用户在工作台里使用，大致流程如下：

```text
用户需求
  ↓
Agent 按 Skills 目录直接读取匹配 SOP
  ↓
Agent 按 skill.required_docs 读取平台规则、SDK 文档、案例和当前函数树
  ↓
Agent 生成 Go 代码
  ↓
write_go_file 写入 namespace
  ↓
build_workspace 编译和部署
  ↓
SDK 启动 app，发现所有 Template 和路由
  ↓
平台保存函数 schema 和 full_code_path
  ↓
前端根据 schema 渲染表单、表格、图表
  ↓
用户或 Agent 通过 run_form_submit / run_table_search / run_chart_query 调用
  ↓
SDK handler 执行业务逻辑并返回结构化结果
```

其中最关键的对象是 `full_code_path`。它是平台定位函数的唯一稳定路径，通常类似：

```text
/namespace/liubeiluo/work/code/api/ticket_system/ticket_list.table
```

## 3. SDK 的能力分层

### 3.1 应用运行层

主要位置：

- `sdk/agent-app/app/app.go`
- `sdk/agent-app/app/register.go`
- `sdk/agent-app/app/on_app_update.go`

这一层负责把一个 Go 应用接入平台运行时：

- 连接 NATS；
- 注册 app；
- 监听函数调用消息；
- 处理 app 启动、关闭、更新、API 发现；
- 把当前代码中的函数 schema 上报给平台；
- 支持优雅关闭，等待正在执行的函数完成。

对业务代码来说，一般不直接关心 NATS 或服务发现，只需要通过 `PackageContext` 注册函数。

### 3.2 Package 和路由层

每个业务目录都会有一个 `init_.go`，里面通常会声明：

```go
var packageContext = app.PackageContext{
    RouterGroup: "/ticket_system",
    Name:        "工单管理系统",
    Desc:        "工单管理相关功能",
}
```

业务函数通过 `packageContext.GET`、`POST`、`PUT`、`DELETE` 注册：

```go
packageContext.GET("ticket_list.table", TicketListTemplate, TicketList)
packageContext.POST("create_ticket.form", CreateTicketTemplate, CreateTicket)
packageContext.GET("ticket_statistics.chart", TicketStatisticsTemplate, TicketStatistics)
```

命名约束非常重要：

- 表格函数必须以 `.table` 结尾；
- 表单函数必须以 `.form` 结尾；
- 图表函数必须以 `.chart` 结尾。

工作台和 tools 会根据后缀判断函数类型。

### 3.3 Template 层

SDK 目前主要提供三类模板：

- `TableTemplate`：表格、列表、CRUD、搜索、分页；
- `FormTemplate`：表单提交、工具执行、文件处理、文本处理；
- `ChartTemplate`：图表查询和可视化。

Template 负责描述函数，而不是执行业务。

常见字段包括：

- `Code`：函数编码；
- `Name`：展示名称；
- `Desc`：描述；
- `Request`：请求结构体；
- `Response`：响应结构体；
- `CreateTables`：需要自动迁移的数据表；
- `AutoCrudTable`：是否开启自动 CRUD；
- `OnApiCreate`：函数首次创建或更新时执行的初始化逻辑；
- `OnSelectFuzzyMap`：下拉、用户选择、关联选择等模糊搜索回调。

### 3.4 Schema 和 Widget 层

工作台前端不直接读取 Go 代码，而是读取 SDK 生成的 schema。

Schema 的来源是 Go struct tag，例如：

```go
type CreateTicketReq struct {
    Title string `json:"title" widget:"name:标题;type:input" validate:"required"`
    Desc  string `json:"desc" widget:"name:描述;type:text_area"`
}
```

常见 tag：

- `json`：字段编码；
- `widget`：前端控件类型、名称、描述等；
- `validate`：后端校验规则；
- `search`：表格搜索条件；
- `display`：字段在哪些场景显示；
- `data`：枚举、选项、格式等数据；
- `callback`：字段级回调。

常见 widget 类型包括：

```text
input, text, text_area, select, switch, datetime, user, users,
department, departments, ID, number, float, files, checkbox,
radio, multiselect, slider, rate, color, richtext, table, form,
link, progress
```

注意：

- 不应该使用 `type:date`，日期也应该用 `datetime` 并通过格式约束；
- 不应该使用 SDK 不支持的 `type:array`；
- 嵌套结构应该使用 `type:table` 或 `type:form`。

### 3.5 Context 层

handler 的标准签名通常是：

```go
func Xxx(ctx *app.Context, resp response.Response) error
```

`ctx` 提供运行时上下文能力：

- `ShouldBind`：绑定请求参数；
- `ShouldBindValidate`：绑定并校验请求参数；
- `GetGormDB`：获取当前 package 的数据库；
- `GetFS`：获取文件系统能力；
- `GetRequestUser`：获取当前用户；
- `GetRequestUserDept`：获取当前用户部门；
- `GetTraceId`：获取本次调用 trace；
- `SendMessage`：发送工作台消息。

`resp` 提供结构化响应：

- `resp.Form(data)`；
- `resp.Table(tableData)`；
- `resp.Chart(chartData)`；
- `resp.BizErrorf(...)`。

### 3.6 DB 层

SDK 使用 Gorm + SQLite。

默认情况下，一个 package 会对应一个 DB 文件，数据目录在运行时工作空间中。

常见模式：

```go
db := ctx.GetGormDB()
if err := db.Create(&model).Error; err != nil {
    return fmt.Errorf("[系统错误]创建失败: %w", err)
}
```

表格函数常配合：

```go
resp.Table(&lists).AutoSearchFilterPaged(db, &Model{}, &req.SearchFilterPageReq)
```

这会自动处理：

- 分页；
- 搜索；
- 排序；
- 总数；
- 表格数据包装。

### 3.7 文件层

文件字段通常使用 `widget:"type:files"`。

前端和平台传递的是文件引用字符串，SDK 负责下载和上传：

```go
fs := ctx.GetFS()
files, err := fs.DownloadFiles(req.InputFiles)
defer fs.RemoveFiles(files)

outputDir := fs.GetTraceOutputDir()
outputFiles, err := fs.ResponseFiles([]string{outputPath})
```

典型场景：

- PDF 转文本；
- 图片处理；
- 视频处理；
- Python 代码生成文件；
- 报表导出。

### 3.8 回调层

SDK 支持多个工作台回调：

- `OnTableAddRow`；
- `OnTableUpdateRow`；
- `OnTableDeleteRows`；
- `OnTableCreateInBatches`；
- `OnPageLoad`；
- `OnSelectFuzzy`。

其中 `OnSelectFuzzy` 常用于：

- 下拉选项动态搜索；
- 根据关键词查用户、商品、项目；
- 根据一个字段联动另一个字段；
- 返回选项的展示信息和统计信息。

### 3.9 Chart 层

图表函数通过 `ChartTemplate` 声明。

SDK 提供：

- `chart.LineChart`；
- `chart.BarChart`；
- `chart.PieChart`；
- `chart.GaugeChart`。

handler 中通常构造 chart 对象，然后：

```go
resp.Chart(chartData)
```

图表函数的职责是返回结构化图表数据，而不是让业务代码直接操作前端图表库。

### 3.10 Python Runtime 和系统工具层

SDK 也支持把 Python 或系统 CLI 包装成工作台函数。

Python runtime 约定入口：

```python
def agentos_entry(args, output_dir):
    return {"data": "..."}
```

Go handler 负责：

- 接收用户参数；
- 下载输入文件；
- 调用 Python executor；
- 收集 stdout/stderr；
- 上传输出文件；
- 返回结构化结果。

这让工作台可以承载很多“工具型函数”，例如：

- PDF 解析；
- 图片处理；
- 数据可视化；
- 文档转换；
- 脚本执行；
- 批处理任务。

## 4. 三种函数的标准写法

### 4.1 Table：管理一类数据

适合：

- 客户列表；
- 工单列表；
- 订单列表；
- 员工列表；
- 任务列表。

标准结构：

```text
Model struct
  ↓
TableTemplate
  ↓
GET xxx.table
  ↓
AutoSearchFilterPaged
  ↓
OnTableAddRow / OnTableUpdateRow / OnTableDeleteRows
```

Table 的重点是 schema、搜索、分页、CRUD 和列表展示。

### 4.2 Form：执行一次动作

适合：

- 创建记录；
- 提交审批；
- 文件转换；
- 文本生成；
- 调用 Python；
- 发送通知。

标准结构：

```text
Request struct
Response struct
  ↓
FormTemplate
  ↓
POST xxx.form
  ↓
ShouldBindValidate
  ↓
业务处理
  ↓
resp.Form(response)
```

Form 的重点是输入、校验、执行和结果展示。

### 4.3 Chart：返回可视化数据

适合：

- 趋势图；
- 饼图；
- 柱状图；
- 仪表盘；
- 统计看板。

标准结构：

```text
Request struct
  ↓
ChartTemplate
  ↓
GET xxx.chart
  ↓
查询和聚合
  ↓
resp.Chart(chartData)
```

Chart 的重点是数据聚合和图表结构。

## 5. 工作台 tools 的作用

工作台 tools 是大模型和代码之间的安全操作层。

常见工具：

- `read_go_file`：读取已有 Go 文件；
- `read_dir`：查看目录；
- `write_go_file`：写 Go 文件；
- `search_replace_file`：局部修改文件；
- `build_workspace`：编译和部署当前工作区；
- `run_table_search`：调用 `.table` 查询；
- `run_form_submit`：调用 `.form` 提交；
- `run_chart_query`：调用 `.chart` 查询；
- `run_on_select_fuzzy`：调试模糊选择回调。

这些 tools 的价值是：

- 限制大模型只能在工作区内做合理操作；
- 要求大模型先看 schema 再调用；
- 避免把 `.form` 当 `.table` 调；
- 避免跳过 build；
- 让生成代码可以被立即验证。

## 6. Skills、文档和提示词的作用

当前工作台不再依赖超长 prompt 直接承接所有 SOP，而是以 skills 为主链路：

```text
模式 prompt：身份、模式边界、安全约束、路由提示
Skills：具体任务 SOP、required_docs、recommended_demos、allowed_tools、completion
Docs：SDK 长文档、平台能力说明、案例和细节
Tools：受控读写、构建、执行和搜索动作
```

模型默认应该：

- 先判断用户意图；
- 按 Skills 目录直接 `read_skill`，例如 `sop.create-project`、`sop.modify-project`、`sdk.widget-selection`；
- 不确定才用 `search_skills` 兜底；
- 按 skill 的 `required_docs` 读取 SDK 文档、平台能力边界和案例；
- 必须看当前目录和函数树；
- 新函数后缀必须是 `.form`、`.table`、`.chart`；
- 修改项目优先用 `search_replace_file`；
- 写完必须 `build_workspace`；
- 调试必须使用对应 runtime tool；
- 不要重写平台已有的用户、文件、消息、权限等横切能力。

因此，平台的生成质量来自三部分共同约束：

```text
SDK 类型和运行时约束
  +
tools 操作约束
  +
skills/docs/prompt 行为约束
```

只靠其中任何一个都不够。

## 7. namespace 代码说明

`namespace` 是用户、系统和官方工具生成代码的落地点。

典型路径：

```text
namespace/system/tools/code/api/pdf/pdf_to_text.go
namespace/system/tools/code/api/runtime/python_execute.go
namespace/system/openapi/code/api/message/send.go
namespace/liubeiluo/work/code/api/ticket_system/ticket.go
```

这些代码说明 SDK 已经覆盖了几类能力：

- 业务系统：工单、投票、收银、计划、统计；
- 表格 CRUD：模型、搜索、分页、增删改；
- 文件工具：PDF、图片、视频、文档；
- Python 工具：执行 Python、收集输出文件；
- 图表工具：柱状图、折线图、饼图、仪表盘；
- 表单工具：文本处理、文件转换、业务动作；
- 联动组件：模糊选择、字段依赖、动态选项。
- 平台接口封装：Hub、消息、定时任务、权限、审计等 `/system/openapi` 函数。

换句话说，`namespace` 不是临时代码目录，而是平台函数资产的源码仓库。

## 8. 大模型生成代码时必须记住的规则

大模型看到这个项目时，应该按下面规则工作。

### 8.1 先判断函数类型

用户要管理一批数据，用 `TableTemplate`。

用户要执行一次动作，用 `FormTemplate`。

用户要看统计图，用 `ChartTemplate`。

不要把所有需求都写成 form，也不要把图表写成普通表格。

### 8.2 后缀必须匹配类型

```text
TableTemplate -> xxx.table
FormTemplate  -> xxx.form
ChartTemplate -> xxx.chart
```

后缀错误会导致工作台和 tools 判断错误。

### 8.3 struct tag 是 UI 协议

字段不是随便加的。

每个字段应该有：

- `json`；
- 合理的 `widget`；
- 必要的 `validate`；
- 表格字段需要合理的 `search`；
- 列表、创建、编辑场景需要合理的 `display`。

### 8.4 业务错误和系统错误要区分

用户输入错误、业务规则不满足，应该返回业务错误：

```go
return resp.BizErrorf("请选择有效的文件")
```

数据库失败、文件系统失败、执行器失败，应该返回系统错误：

```go
return fmt.Errorf("[系统错误]查询失败: %w", err)
```

### 8.5 文件必须走 SDK 文件 API

不要假设前端上传的文件已经在本地。

应该通过：

- `ctx.GetFS()`
- `DownloadFiles`
- `GetTraceOutputDir`
- `ResponseFiles`
- `ResponseDirFiles`

完成文件输入输出。

### 8.6 不要重复实现平台横切能力

不要自己实现：

- 用户系统；
- 部门系统；
- 文件存储；
- 消息通知；
- 权限体系；
- 服务发现；
- 函数调用协议；
- 前端 schema 协议。

这些都应该使用 SDK 和平台能力。

## 9. 当前值得改进的地方

SDK 和工作台逻辑已经比较完整，但还有几个工程风险需要继续收敛。

### 9.1 启动前强校验已经进入 SDK 链路

生成代码里出现过不推荐或不支持的写法，例如：

- `widget:"type:date"`；
- `widget:"type:array"`。

这类代码可能能通过 Go 编译，但 schema 语义不正确。现在应该把它们视为 SDK schema compile 错误，而不是等到前端渲染或运行调用时才暴露。

当前落地链路是：

1. `app.Run()` 启动前调用 `CompileAndValidate()`；
2. `CompileAndValidate()` 先校验 route、Template 类型和后缀，再调用 `getApis()` 编译 schema；
3. `getApis()` 内部的 `DecodeForm` / `DecodeTable` 会调用 `widget.ValidateFieldTags()`；
4. 每个 widget 组件文件注册自己的校验器，例如 `input.go` 注册 `TypeInput`，`files.go` 注册 `TypeFiles`；
5. `functionschema.Validate()` 再校验最终 schema，确保平台拿到的是可渲染协议。

失败时 App 会发布 `startup failed` 生命周期事件，并把错误消息带回 runtime。`build_workspace` 不再只看 Go 编译是否成功，而是会等待新版本启动结果；如果 SDK schema compile 失败，新版本不会被认为启动成功，旧版本也不会被切走。

这套机制要保证几类错误尽早失败：

- 未知 widget type；
- 组件和 Go 字段类型不匹配；
- 组件必填参数缺失或格式非法；
- `depend_on` 指向不存在的兄弟字段；
- `OnSelectFuzzyMap` 指向不存在的字段；
- `search` 操作符和字段类型不匹配；
- template 类型和 route 后缀不匹配；
- 最终 schema 无法被前端稳定渲染。

### 9.2 常见 CRUD 可以继续 SDK 化

很多表格代码会重复写：

- 创建；
- 更新；
- 删除；
- 批量创建；
- 软删除；
- 时间字段；
- 当前用户审计。

这些可以沉淀成更高层 helper，减少大模型手写样板代码。

### 9.3 字符串协议可以逐步 typed 化

当前很多能力依赖字符串约定：

- widget tag；
- search tag；
- display tag；
- files 引用；
- `[系统错误]` 前缀；
- callback 名称。

这些约定对大模型友好，但长期容易漂移。组件级 validator 已经能挡住一批错误；后续还可以继续增加 typed builder 和 lint，把更多字符串约定变成结构化 API。

### 9.4 namespace 需要扫描测试

建议增加一个 namespace 级别的扫描任务，检查：

- 不支持的 widget type；
- route 后缀和 Template 不一致；
- 缺少 `json` tag；
- 有文件字段但没有使用文件 API；
- form/table/chart 调用方式不一致；
- 老版本写法。

这可以把提示词中的规则变成工程上的硬约束。

## 10. 最短总结

AI Agent OS 的生成逻辑可以压缩成一句话：

> 大模型根据 prompt 和 tools 在 `namespace` 中生成 Go 代码；SDK 从 Template、路由和 struct tag 中生成函数 schema；平台根据 schema 渲染工作台 UI；runtime tools 根据 full_code_path 调用函数；handler 通过 SDK 提供的 DB、文件、用户、消息和响应能力完成业务。

如果一个人或大模型只记住三个点，就记住：

1. `Template + route suffix + handler` 是一个工作台函数；
2. `struct tag -> schema -> 前端 UI` 是渲染链路；
3. `full_code_path -> runtime tool -> SDK handler` 是调用链路。
