# Agent-App SDK 使用说明

本文档是 **Agent-App SDK 主入口** 和权威主文档，用于 `app_developer`（应用开发工程师）与 `maintenance_engineer`（应用维护工程师）角色下的代码生成、修改和校验。它说明框架稳定契约、API、组件、schema、校验规则和常见代码模式。

完整业务示例（PRD + 代码）在案例文档中，按需 `read_doc("/system/prompt/case_catalog/xxx")` 对应路径即可。不要只靠零散参考文档写代码；创建或修改业务应用时，以本主文档 + 匹配案例 + 当前身份 SOP 为主链路。

专项参考文档在 `/system/prompt/sdk/reference`；只在主文档和匹配案例仍无法确认细节时再读取。

读 SDK 只解决“框架有哪些能力”，不等于已读最佳实践。写具体业务代码前，必须再读取至少一个与当前需求匹配的案例文档；组合型需求读取多个案例。

---

## 一、定位与文档分工

- **本 SDK 文档**：框架怎么用——结构体与标签、Table/Form/Chart 模式、注册方式、目录约定。
- **案例文档**（`/system/prompt/case_catalog/xxx`）：具体业务长什么样——PRD + 完整 Go 代码。系统消息中「可读的目录」会列出各案例路径与说明；需要单表 CRUD、多表、Form、图表等时，read_doc 对应案例获取 PRD 与代码。
- **平台横切能力（禁止自己实现）**：权限管理、流程审批、评论/点赞/收藏、定时任务、操作记录、消息通知——这些由平台统一提供，**禁止**在 PRD 中添加「审批状态/审批人/审批时间」等字段，**禁止**在代码中自己实现审批表/审批流程/权限判断/评论功能。业务代码只管业务数据本身。

## 按需参考文档

主链路只读本 SDK 主文档和匹配案例。遇到专项问题时再读：

- 程序里发送消息、取当前用户/部门、事务、副作用顺序、Python 运行时、Table 回调高级能力：`read_doc("/system/prompt/sdk/reference/runtime-capabilities")`
- 构建失败、启动期 schema/widget/路由校验、未定义 SDK API 排查：`read_doc("/system/prompt/sdk/reference/build-validation")`
- SDK 代码里调用平台 Web API 或包装 `/system/openapi` 函数：`read_doc("/system/prompt/sdk/reference/platform-api")`

这些参考文档不是默认注入内容；只有当前任务确实命中时才读取。

---

## 二、快速开始

### Table 模式（单表 CRUD，GET）

1. **定义结构体**：业务字段加 `gorm`、`widget`、`validate` 等标签；主键、CreatedAt、DeletedAt 等系统字段按约定写。Table 筛选字段写在 Request 中，Model 不承担筛选协议。
2. **配置 TableTemplate**：`BaseConfig`（Name、Request、CreateTables）+ **`AutoCrudTable`**（必须显式配置，指向列表结构体，前端据此渲染列表字段、分页和表格 schema）+ 可选 `OnTableAddRow` / `OnTableUpdateRow` / `OnTableDeleteRows`。**不需要哪种操作就删掉对应回调**：不想要新增和批量导入 → 不配 `OnTableAddRow`；不想要更新 → 不配 `OnTableUpdateRow`；不允许删除 → 不配 `OnTableDeleteRows`。前端会根据是否配置回调来显示或隐藏对应按钮；`OnTableCreateInBatches` 是系统内置批量导入能力，配置 `OnTableAddRow` 时自动暴露。**支付记录、消费流水、操作日志这类审计/流水表默认应只读**，仍必须显式配置 `AutoCrudTable`，但不配置新增、编辑、删除回调。
3. **写 List 函数**：Request 显式声明筛选字段，并嵌入 `query.PageSortReq`（`widget:"-"`）只承载分页/排序；用 `queryDB := ctx.GetGormDB().Model(&Model{})` 后在 Build 前手写 Where、Joins、Preload 等，再 `resp.Table(&lists, queryDB, &Model{}, &req.PageSortReq).Build()`；Build 后可遍历 `lists` 填计算字段、关联展示字段、link 等。
4. **注册**：`init()` 中 `packageContext.GET("路由名", ListFunc, TableTemplate)`。

最小可用片段示例：

```go
// 结构体（系统字段 + 业务字段，此处省略系统字段）
type CrmTicket struct {
    Title    string `json:"title" gorm:"column:title" widget:"name:标题;type:input" validate:"required,min=2,max=200"`
    Status   string `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:待处理,已完成;options_colors:E6A23C,67C23A;render_default:待处理"`
    // ... ID, CreatedAt, DeletedAt 等见案例
}

type CrmTicketListReq struct {
    Title  string `json:"title" form:"title" widget:"name:标题;type:input"`
    Status string `json:"status" form:"status" widget:"name:状态;type:select;options:待处理,已完成;options_colors:E6A23C,67C23A"`

    query.PageSortReq `widget:"-"`
}

var CrmTicketTemplate = &app.TableTemplate{
    BaseConfig: app.BaseConfig{
        Name:         "工单管理",
        Request:      &CrmTicketListReq{},
        CreateTables: []interface{}{&CrmTicket{}},
    },
    AutoCrudTable: &CrmTicket{},
}

func CrmTicketList(ctx *app.Context, resp response.Response) error {
    var req CrmTicketListReq
    if err := ctx.ShouldBind(&req); err != nil {
        return err
    }
    queryDB := ctx.GetGormDB().Model(&CrmTicket{}) // Build 前手写 Where、Joins、Preload("关联名") 等
    if req.Title != "" {
        queryDB = queryDB.Where("title LIKE ?", "%"+req.Title+"%")
    }
    if req.Status != "" {
        queryDB = queryDB.Where("status = ?", req.Status)
    }
    var lists []*CrmTicket
    if err := resp.Table(&lists, queryDB, &CrmTicket{}, &req.PageSortReq).Build(); err != nil {
        return err
    }
    // Build 后可遍历 lists，填计算字段、关联表展示字段（需先在 Build 前 Preload）、link 等
    return nil
}

func init() {
    packageContext.GET("crm_ticket.table", CrmTicketList, CrmTicketTemplate)
}
```

List 可在 **Build 前**对 `queryDB` 做 Where、Joins、Preload 等，`Table` 只保存 queryDB 和分页参数，**Build** 才执行 Count/Order/Offset/Limit/Find，**Build 后**遍历 `lists` 做后处理；详见「五、Table 回调函数 → 4. List 函数」。

单表完整示例（含所有常用组件与回调）：`read_doc("/system/prompt/case_catalog/table/ticket")`。

### Form 模式（POST，无 Table）

**Form 与 Chart 结构不可混用（必读）**：Form 的 Request/Response 必须是**普通表单/结果结构体**（input、select、table 子表、files、link 等），**禁止**在 Form 的 Request 或 Response 中使用 Chart 结构（如 `chart.LineChart`、`chart.BarChart`、`chart_type`、`series`、`x_axis` 等）。若需求是「按条件查询并展示图表」，应单独注册 **Chart 路由**（GET + ChartTemplate + `resp.Chart(chart).Build()`），不得在 Form 里返回图表数据；若需求是「用户提交数据后生成一张图」的一次性任务，仍用 Form，但 Response 应为普通结构体（如返回图片 URL、或 link 到前端页），不能把 Chart 结构体当作 Form 的 Response。

1. **定义请求/响应结构体**：字段加 `widget`、`validate`；请求体可含 files、input、select、table 等。
2. **写处理函数**：`ctx.ShouldBindValidate(&req)`，业务逻辑，`return resp.Form(&respStruct).Build()`；系统错误需加 `[系统错误]` 前缀并带详细参数（见第六节「系统错误」）。
3. **配置 FormTemplate**：`BaseConfig`（Name、Request、Response）+ 可选 `OnSelectFuzzyMap` 等。
4. **注册**：`init()` 中 `packageContext.POST("路由名", Handler, FormTemplate)`。

最小可用片段示例：

```go
type ExcelOrCsvReq struct {
    File string `json:"file" widget:"name:上传文件;type:files" validate:"required"`
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

单 Form 完整示例：`read_doc("/system/prompt/case_catalog/form/excelorcsv")`。

### Chart 模式（GET，统计/图表）

**Chart 拆分规则（必读）**：

1. SDK 没有 `resp.Charts`，也不支持 `resp.Chart(chart1, chart2)`。
2. 一个 Chart 路由只返回一张图，代码必须用 `resp.Chart(chart).Build()`。
3. 如果需求包含趋势图、分布图、仪表盘等多张图，PRD 功能清单和 Chart 设计必须拆成多个 `.chart` 路由。
4. 总数、占比、NPS 值、平均值等汇总指标放到图表 `Metadata`，不要为了返回多个指标编造多图响应。

图表只支持 4 种类型（`LineChart`/`BarChart`/`PieChart`/`GaugeChart`），详见第七节「图表类型说明」。

1. **定义请求结构体**：筛选条件加 `widget` 标签。
2. **写统计函数**：`ctx.ShouldBind(&req)` → 查库聚合 → 构造具体图表类型（只填 Title、XAxis、Series、Metadata，**响应体里的 ChartType 和 Series[].Type 由 `resp.Chart(...)` 注入，业务无需填**）→ `return resp.Chart(chart).Build()`。
3. **配置 ChartTemplate**：`BaseConfig`（Name、Request、Response 填**与返回值一致的具体类型**，如 `Response: &chart.LineChart{}`）+ `ChartType`（必须填 `app.ChartTypeLine` / `app.ChartTypeBar` / `app.ChartTypePie` / `app.ChartTypeGauge`，不要写死字符串）。图表类型请使用 **`sdk/agent-app/chart`** 包（`chart.LineChart`、`chart.BarChart` 等），勿使用 `types` 包下的图表类型。
4. **注册**：`init()` 中 `packageContext.GET("路由名", ChartHandler, ChartTemplate)`。

**多维图表推荐模式（重要）**：
- `LineChart` / `BarChart` 的 `Series` 是数组，**天然支持多系列**。
- 如果请求里有“状态 / 部门 / 门店 / 渠道”这类**可选聚焦维度**，推荐采用：
  - **不传维度**：返回多个 `ChartSeries` 做总览对比（例如不同状态的多条折线、分组柱状图）。
  - **传了维度值**：退回单 `ChartSeries`，只聚焦该维度。
- 对趋势图，建议把缺失日期补 `0`，避免线断掉或横轴漂移。
- 对柱状图，建议固定分类顺序（如 `低/中/高`、`待处理/处理中/已完成`），避免不同请求之间柱子顺序变化。

最小可用片段示例：

```go
import "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/chart"

type SalesStatisticsReq struct {
    StartTime string `json:"start_time" form:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
    EndTime   string `json:"end_time" form:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

func SalesTrendChart(ctx *app.Context, resp response.Response) error {
    var req SalesStatisticsReq
    if err := ctx.ShouldBind(&req); err != nil { return err }
    db := ctx.GetGormDB()
    // 聚合查询得到 dateLabels、seriesData...
    c := &chart.LineChart{
        Title:  "销售趋势",
        XAxis:  dateLabels,
        Series: []chart.ChartSeries{{Name: "销售额", Data: seriesData}},
        Metadata: map[string]interface{}{"总销售额": total},
    }
    return resp.Chart(c).Build()
}

func init() {
    packageContext.GET("sales_trend_statistics.chart", SalesTrendChart, &app.ChartTemplate{
        BaseConfig: app.BaseConfig{Name: "销售趋势", Request: &SalesStatisticsReq{}, Response: &chart.LineChart{}},
    })
}
```

多系列示例（默认总览）：

```go
type TicketTrendReq struct {
    StartTime string `json:"start_time" form:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
    EndTime   string `json:"end_time" form:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

func TicketTrendChart(ctx *app.Context, resp response.Response) error {
    var req TicketTrendReq
    if err := ctx.ShouldBind(&req); err != nil { return err }

    return resp.Chart(&chart.LineChart{
        Title: "工单趋势",
        XAxis: []string{"2026-04-01", "2026-04-02"},
        Series: []chart.ChartSeries{
            {Name: "待处理", Data: []interface{}{12, 15}},
            {Name: "处理中", Data: []interface{}{5, 7}},
            {Name: "已完成", Data: []interface{}{8, 11}},
        },
    }).Build()
}
```

完整 Chart 示例（折线图/饼图/仪表盘、时间筛选、多图表同包）：`read_doc("/system/prompt/case_catalog/form_table_chart/cashier")`。

---

## 三、结构体与标签

### 1. widget 标签

widget tag 内部格式是 `name:显示名;type:已支持组件类型;配置项:值`。生成代码时只使用**当前已支持**的组件类型；只要写了 `widget` 标签，就必须显式写 `type`。`type` 大小写敏感，主键/ID 展示必须写 `type:ID`，不要写成 `type:id`。常用配置：`render_default`、`placeholder`、`options`、`options_colors`、`min`/`max`/`step`/`unit`、`format`、`precision`、`accept`、`max_size`、`max_count`、`height`、`disabled`、`creatable` 等。`render_default` 是前端渲染默认值，只在新增/初始化界面填充，不等于 `gorm:"default:..."` 或数据库默认值。

同一层结构体字段的 `json` code 不能重复。可选 `data` 标签只允许 `format`、`example`，用于补充 schema 数据格式和示例，不替代 widget 配置。

**select / multiselect 与 options_colors（提示词约定，按必填处理）**：生成静态 `select` 或静态 `multiselect` 时配置 `options_colors`，与 `options` 一一对应（逗号分隔，顺序一致），前端会用颜色标签区分选项。动态 OnSelectFuzzy 下拉不写 `options`，也不要写 `options_colors`。`options_colors` 只支持 6 位十六进制 `RRGGBB`，不带 `#`，如 `FF9800`、`9C27B0`、`4CAF50`。不要生成 `primary`、`success`、`warning`、`danger`、`info`、`default`、`secondary`、`#FF9800` 或 `rgb(...)`。示例：`options:待处理,进行中,已完成` 对应 `options_colors:E6A23C,409EFF,67C23A`；`options:VIP,普通,体验` 对应 `options_colors:E91E63,9E9E9E,4CAF50`。

**select / multiselect 选项来源校验**：`select` 和 `multiselect` 必须至少有一种选项来源：静态 `options`，或字段 `callback:"OnSelectFuzzy"` + 模板 `OnSelectFuzzyMap`。`creatable:true` 只表示允许创建新选项，不能替代选项来源；配置了 `callback:"OnSelectFuzzy"` 时，字段必须是 `select` / `multiselect`，且 `OnSelectFuzzyMap` 的 key 必须和字段 `json` 名一致。

**静态枚举值一致性（重要）**：当前 widget tag 里的静态 `options`（包括 `select` / `multiselect` / `radio` / `checkbox`）默认就是**字符串列表**，前端实际提交值就是选项文本本身。生成 `validate:"oneof=..."`、`required_if`、`required_unless`、`excluded_if`、`excluded_unless` 等规则时，条件值必须与实际提交值**逐字一致**。不要写成“界面展示中文选项，但校验/条件值用英文 code”的混搭形式。

**自由输入列表**：当用户需要直接输入多个值（如 `1,2,3` 或多行文本列表）时使用 `type:list`，并显式指定 `item_type`：

```go
Numbers []int    `json:"numbers" widget:"name:数字列表;type:list;item_type:number;placeholder:例如 1,2,3"`
Names   []string `json:"names" widget:"name:文本列表;type:list;item_type:text;placeholder:例如 张三,李四"`
```

`type:list;item_type:number` 适配 `[]int`、`[]float64` 等数字切片；`type:list;item_type:text` 适配 `[]string`。如果是从候选项里选择多个值，仍然用 `multiselect`；如果是少量固定枚举平铺勾选，用 `checkbox`。

**switch 组件限制（必读）**：当前 `switch` 只支持 `render_default`，提示词和示例里**不要写** `true_label`、`false_label` 这类未实现参数。

**不要生成未支持的组件类型**：当前没有独立 `date`、`time`、`range`、`image`、`tag`、`tree`、`cascader`、`code` 等 widget type。日期时间统一使用 `datetime + types.Time`（真实数据库时间类型）。图片/媒体上传统一先用 `files`；标签类输入优先用 `multiselect` 或 `checkbox`。

组件参数速查结构体。下面是查参数用的全集示例，不是业务模型模板；生成真实代码时只复制当前业务需要的字段和组件：

```go
type WidgetLookupExample struct {
    ID             int        `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
    Title          string     `json:"title" gorm:"column:title" widget:"name:标题;type:input;placeholder:请输入标题;render_default:新建记录" validate:"required,min=2,max=200"`
    Password       string     `json:"password" gorm:"-" widget:"name:密码;type:input;password:true;placeholder:请输入密码"`
    Phone          string     `json:"phone" gorm:"column:phone" widget:"name:电话;type:input;prepend:+86;append:分机"`
    Description    string     `json:"description" gorm:"column:description" widget:"name:描述;type:text_area;placeholder:请输入详细描述;render_default:暂无" validate:"required,min=10"`
    ReadonlyText   string     `json:"readonly_text" gorm:"-" widget:"name:只读文本;type:text;format:markdown" hide:"create,update"`
    Content        string     `json:"content" gorm:"type:text;column:content" widget:"name:详细内容;type:richtext;height:360"`
    Source         string     `json:"source" gorm:"column:source" widget:"name:来源;type:radio;options:电话,邮件,在线;render_default:在线" validate:"oneof=电话 邮件 在线"`
    NotifyChannels []string   `json:"notify_channels" gorm:"-" widget:"name:通知渠道;type:checkbox;options:站内信,短信,邮件;render_default:站内信,邮件"`
    Status         string     `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:待处理,进行中,已完成;options_colors:E6A23C,409EFF,67C23A;placeholder:请选择状态;render_default:待处理;creatable:true"`
    Tags           []string   `json:"tags" gorm:"-" widget:"name:标签;type:multiselect;options:紧急,重要,客户;options_colors:F56C6C,E6A23C,409EFF;placeholder:请选择标签;render_default:紧急,客户;max_count:3;creatable:true"`
    Quantity       int        `json:"quantity" gorm:"column:quantity" widget:"name:数量;type:number;placeholder:请输入数量;min:0;max:999;step:1;render_default:1;unit:件"`
    Amount         float64    `json:"amount" gorm:"column:amount" widget:"name:金额;type:float;placeholder:请输入金额;min:0;max:999999;precision:2;step:0.01;render_default:0;unit:元"`
    ProgressEdit   int        `json:"progress_edit" gorm:"column:progress_edit" widget:"name:进度调整;type:slider;min:0;max:100;step:5;render_default:50;unit:%"`
    Rating         float64    `json:"rating" gorm:"column:rating" widget:"name:评分;type:rate;max:5;allow_half:true;render_default:4.5;texts:很差,较差,一般,满意,惊喜"`
    Enabled        bool       `json:"enabled" gorm:"column:enabled" widget:"name:启用;type:switch;render_default:true"`
    ThemeColor     string     `json:"theme_color" gorm:"column:theme_color" widget:"name:主题色;type:color;format:hex;show_alpha:false;render_default:#409EFF"`
    Deadline       types.Time `json:"deadline" gorm:"column:deadline;type:datetime" widget:"name:截止时间;type:datetime;format:YYYY-MM-DD HH:mm:ss;render_default:DATE_ADD(CURRENT_TIMESTAMP, INTERVAL 1 HOUR)"`
    Attachment     string     `json:"attachment" gorm:"type:text;column:attachment" widget:"name:附件;type:files;accept:.pdf,.doc,.docx,.png,.jpg;max_size:50MB;max_count:5"`
    Handler        string     `json:"handler" gorm:"column:handler" widget:"name:处理人;type:user;render_default:Me();disabled:true"`
    Reviewers      string     `json:"reviewers" gorm:"column:reviewers" widget:"name:审核人;type:users;render_default:Me(),MyLeader();max_count:5"`
    Department     string     `json:"department" gorm:"column:department" widget:"name:所属部门;type:department;render_default:MyDepartment()"`
    Departments    string     `json:"departments" gorm:"column:departments" widget:"name:关联部门;type:departments;render_default:MyDepartment();max_count:3"`
    NumberList     []int      `json:"number_list" gorm:"-" widget:"name:数字列表;type:list;item_type:number;separator:,;placeholder:例如 1,2,3;render_default:1,2,3;unique:true;max_count:10"`
    TextList       []string   `json:"text_list" gorm:"-" widget:"name:文本列表;type:list;item_type:text;separator:,;placeholder:每项用逗号分隔;render_default:张三,李四;unique:true;max_count:20"`
    DetailLink     string     `json:"detail_link" gorm:"-" widget:"name:查看详情;type:link;text:查看;target:_blank;link_type:primary;icon:Link" hide:"create,update"`
    Percentage     float64    `json:"percentage" gorm:"-" widget:"name:完成率;type:progress;min:0;max:100;unit:%" hide:"create,update"`
    Lines          []WidgetLookupLine `json:"lines" gorm:"-" widget:"name:明细行;type:table"`
    Extra          WidgetLookupExtra  `json:"extra" gorm:"-" widget:"name:附加信息;type:form"`
}

type WidgetLookupLine struct {
    Name      string  `json:"name" widget:"name:名称;type:input;placeholder:请输入名称"`
    Count     int     `json:"count" widget:"name:数量;type:number;min:1;max:999;step:1;render_default:1;unit:件"`
    UnitPrice float64 `json:"unit_price" widget:"name:单价;type:float;min:0;precision:2;step:0.01;unit:元"`
}

type WidgetLookupExtra struct {
    Remark string `json:"remark" widget:"name:备注;type:text_area;placeholder:请输入备注"`
    Link   string `json:"link" widget:"name:参考链接;type:link;text:打开;target:_blank;link_type:info"`
}
```

| 组件类型 | 常用配置 | 说明 | 典型用法 |
|----------|----------|------|----------|
| `ID` | — | 只读主键/标识；`type` 必须写大写 `ID` | 列表主键、详情 ID |
| `input` | `placeholder`、`password`、`prepend`、`append`、`render_default` | 单行文本输入 | 标题、电话、邮箱 |
| `text` | `format` | 只读文本展示 | JSON、Markdown、CSV、HTML 输出 |
| `text_area` | `placeholder`、`render_default` | 多行文本输入 | 描述、备注 |
| `richtext` | `height` | 富文本编辑 | 详细内容、公告正文 |
| `select` | `options`、`options_colors`、`render_default`、`placeholder`、`creatable` | 下拉单选 | 状态、优先级 |
| `radio` | `options`、`render_default` | 单选按钮 | 来源、性别（2-5 个选项） |
| `checkbox` | `options`、`render_default` | 固定选项复选 | 通知渠道、权限项 |
| `multiselect` | `options`、`options_colors`、`render_default`、`max_count`、`placeholder`、`creatable` | 下拉多选 | 标签、选项集合 |
| `list` | `item_type`、`separator`、`placeholder`、`render_default`、`unique`、`max_count` | 自由输入列表 | `[]int` 数字数组、`[]string` 文本数组 |
| `number` | `placeholder`、`min`、`max`、`step`、`render_default`、`unit` | 整数输入 | 数量、工时 |
| `float` | `placeholder`、`min`、`max`、`precision`、`step`、`render_default`、`unit` | 小数输入 | 价格、金额、折扣率 |
| `slider` | `min`、`max`、`step`、`render_default`、`unit` | 可编辑滑块 | 进度、评分、百分比 |
| `rate` | `max`、`allow_half`、`render_default`、`texts` | 星级评分 | 服务评价、满意度 |
| `switch` | `render_default` | 布尔开关；不要写未实现的 `true_label` / `false_label` | 是否启用、是否匿名 |
| `datetime` | `format`、`render_default`、`disabled` | 日期时间；raw value 为 `"YYYY-MM-DD HH:mm:ss"`，数据库推荐 `types.Time` + `type:datetime`；默认值推荐 `CURRENT_TIMESTAMP` / `DATE_ADD(CURRENT_TIMESTAMP, INTERVAL 1 HOUR)` | 创建时间、截止时间 |
| `color` | `format`、`render_default`、`show_alpha` | 颜色选择 | 主题色、颜色值 |
| `files` | `accept`、`max_size`、`max_count` | 文件上传/下载；字段类型必须为 `string` | 附件、图片、视频 |
| `user / users` | `render_default`、`disabled`、`max_count` | 用户选择 | 负责人、抄送人 |
| `department / departments` | `render_default`、`max_count` | 部门选择 | 所属部门、关联部门 |
| `progress` | `min`、`max`、`unit` | 只读进度展示 | 得票率、完成率 |
| `link` | `text`、`target`、`link_type`、`icon` | 只读跳转链接；`type` 是组件类型保留 key，链接样式使用 `link_type` | 查看详情、关联函数跳转 |
| `table` | — | Form 请求中的子表 | 明细行、商品清单 |
| `form` | — | Form 响应中的子表单 | 嵌套结构体展示 |

- `text`、`progress`、`link`、`ID` 多用于列表展示字段或响应字段；Table 中若只希望前端列表展示、不进入新增/编辑表单，配 `hide:"create,update"`。
- `select` / `multiselect` 的实际值类型由 Go 字段类型决定，SDK 会按字段类型推断 `string`、`int`、`[]string`、`[]int`、`[]float` 等。
- `checkbox` 更适合固定数量的勾选项；需要下拉式多选、远程搜索或可创建选项时优先使用 `multiselect`。
- `list` 表示自由输入多个值，不表示候选项选择；数字数组写 `item_type:number`，文本数组写 `item_type:text`。

**datetime / types.Time 约定（新业务必读）**：新建业务表默认使用 `types.Time` + `gorm:"type:datetime"` + `widget:"type:datetime"`。API/工作台/前端 raw value 都是 `"YYYY-MM-DD HH:mm:ss"` 字符串，数据库存真实时间类型。系统字段示例：`CreatedAt types.Time \`json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"\``。普通业务时间字段需要在新增/编辑填写时，不要加 `hide:"create,update"`。

`datetime` 的 `render_default` 可以写静态时间字符串（如 `2026-05-01 10:30:00`），也可以写前端可解析的动态表达式：`CURRENT_TIMESTAMP`、`CURRENT_DATE`、`DATE_ADD(CURRENT_TIMESTAMP, INTERVAL 1 HOUR)`、`DATE_SUB(CURRENT_DATE, INTERVAL 7 DAY)`。不要写 `NOW()` 或缺少 `INTERVAL` 的表达式，启动期会失败。

**系统审计字段启动期约束**：`id` 主键字段必须 `type:ID`、`hide:"create,update"`，且 `gorm` 包含 `primaryKey`、`autoIncrement`、`column:id`；`created_at` / `updated_at` 必须是 `datetime` + `format:YYYY-MM-DD HH:mm:ss` + `hide:"create,update"`，并分别包含 `autoCreateTime` / `autoUpdateTime`；`create_by` / `created_by` / `update_by` / `updated_by` 必须是 `type:user` + `hide:"create,update"`，且 `gorm column` 与 `json` 名一致；`deleted_at` 必须 `widget:"-"` 或 `json:"-"`，不要进入前端 schema。需要按这些字段筛选时，在 Table Request 里显式声明筛选字段。

`types.Time` 是对 `time.Time` 的包装类型，给结构体字段赋值时必须显式转换，不能直接把 `time.Now()` 赋给 `types.Time` 字段。
不要生成未在当前已读文档、案例或 SDK 源码中确认存在的 SDK 类型、函数、常量或结构体字段；遇到 `undefined: <sdk package>.<symbol>` 先回到对应知识点或源码确认真实 API。

```go
// 正确：当前时间赋给 types.Time 字段
row.ExpenseDate = types.Time(time.Now())

// 正确：字符串转 types.Time，适合后端生成或解析固定时间
expenseDate, err := types.ParseTime("2026-04-26 10:30:00")
if err != nil {
    return nil, err
}
row.ExpenseDate = expenseDate

// 正确：取出原生 time.Time 做比较、格式化、计算
if row.ExpenseDate.Time().Before(time.Now()) {
    // ...
}

// 正确：置空 types.Time 字段
row.ExpenseDate = types.Time{}

// 错误：time.Now() 是 time.Time，不能直接赋给 types.Time 字段
row.ExpenseDate = time.Now()
```

只有 `gorm.DeletedAt` 这类 GORM 软删除字段才常见 `deleted_at: time.Now()`；如果目标字段本身是 `types.Time`，优先写 `types.Time(time.Now())`，避免类型不匹配。

**form / table 结构约定（禁止 map，必读）**：使用 `type:form`（子表单）或 `type:table`（子表）时，**字段必须是具名、固定字段的结构体类型**，**禁止使用 `map[string]interface{}`**。前端和 SDK 依赖结构体标签（如 `widget`、`json`）来生成表单/表格列；map 的键在编译期不确定，无法解析出固定 schema，会导致展示异常或无法正确渲染。

- **错误（Badcase）**：`BasicInfo map[string]interface{} \`json:"basic_info" widget:"name:基本信息;type:form"\``、`FileInfo map[string]interface{} \`json:"file_info" widget:"name:文件信息;type:form"\`` —— 不要对 form/table 使用 map。
- **正确**：为「基本信息」「文件信息」等分别定义**具名结构体**，字段固定、带 widget 标签，例如：
  - 基本信息：`type BasicInfoStruct struct { Format string \`json:"format" widget:"name:图片格式;type:text"\`; Width string \`...\`; Height string \`...\`; ... }`，字段类型用 `BasicInfoStruct`。
  - 文件信息：`type FileInfoStruct struct { FileName string \`json:"file_name" widget:"name:文件名;type:text"\`; FileSize string \`...\`; ... }`，字段类型用 `FileInfoStruct`。

**files 类型约定**：使用 `type:files` 时字段类型为 `string`。字段值是稳定文件引用：`bucket/object_key`，多文件用英文逗号分隔。完整上传、下载与存储流程见第六节「文件上传、下载与存储」。图片、视频这类需要在 Table 列表中直接辨认内容的字段，可加 `thumbnail:true;list_preview:true`：浏览器上传时前端生成图片缩略图或视频封面，后端只保存缩略图引用；列表单元格使用缩略图/封面展示。普通合同、PDF、Excel、压缩包等附件不要默认开启列表预览。

#### link 组件（跳转链接，多函数联动）

用于在**列表**或**表单**中展示可点击链接，点击后**跳转到另一个函数**（Table 或 Form）或打开**外链**，实现多函数联动与带参跳转。Table 列表链接字段通常**不落库**（`gorm:"-"`），并配 `hide:"create,update"`，前端仅在列表展示，不进入新增/编辑表单；值由后端在 **List 函数 Build 之后**或 **Form 响应**里用 `ctx.BuildFunctionUrlWithText(target, params, linkText)` 赋值。

- **widget 配置**：`type:link`；可选 `target:_blank`（新窗口）或 `_self`（当前窗口）；可选 `text`、`link_type`（样式 primary/success 等）、`icon`。
- **赋值 API**：
  - **推荐**：`ctx.BuildFunctionUrlWithText(target string, params interface{}, linkText string) (string, error)` —— 带链接展示文案。
  - **无文案时**：`ctx.BuildFunctionUrl(target string, params interface{}) (string, error)` —— 返回的 url 会带参数，前端仍可展示目标页。
- **返回值格式（必读）**：上述 API 返回的是** JSON 字符串**，形如 `{"type":"table","name":"查看会议室详情","url":"/user/app/xxx?id=123"}`。前端会解析该字符串得到 `type`（table/form，外链为空）、`name`（展示文案）、`url`（path+query），点击后在工作空间内跳转到对应函数并应用 query（表格筛选、表单预填等）。业务侧只需把返回值赋给 link 字段即可，不要自行拼 JSON。
- **target**：函数路径（如 `"meeting_room_list.table"`、`"vote_submit.form"`、`"bangla_level_distribution.chart"`），或带查询（如 `"hr_resume_list.table?_tab=OnTableAddRow"`），或**外链**（如 `"https://example.com"` 或 `"www.example.com"`，无协议时自动补 https）。支持 Table、Form、**Chart**（图表为 GET + query，params 用该 Chart 的 Request 结构体）。  
  - **params**：见下「params 类型约定」；外链时传 `nil`。  
  - **linkText**：链接展示文本（如「查看会议室详情」「查看统计」）。

- **params 类型约定（必读，不可混用）**：  
  - **跳转到 Table（GET 列表）**：params 必须是**目标 Table 对应的列表 Model**，即该 GET 路由的 `AutoCrudTable` 指向的结构体。前端打开列表时会用 params 的字段（如 ID）做筛选/定位。  
    - 例：target 为 `"meeting_room_list.table"` 时，params 用 `MeetingRoom{ID: roomID}`，其中 **MeetingRoom** 是会议室表（meeting_room_list.table）的 **Model**，不能写成别的结构体。  
    - 例：target 为 `"hr_resume_list.table?_tab=OnTableAddRow"` 时，params 用 `HrResume{JobID: jobID}`（简历表 `hr_resume_list.table` 的 Model），打开简历列表并预填新增表单里的职位 ID。  
  - **跳转到 Form（POST 表单）**：params 必须是**目标 Form 的请求结构体**，即该 POST 路由的 `Request` 结构体。前端打开表单时会预填 params 的字段。  
    - 例：target 为 `"vote_result.form"` 时，params 用 `VoteResultReq{TopicID: topicID}`，其中 **VoteResultReq** 是查看结果 Form 的 **请求结构体**，不能写成 VoteTopic（Model）。  
    - 例：target 为 `"vote_submit.form"` 时，params 用 `VoteSubmitReq{TopicID: topicID}`（提交投票 Form 的请求结构体）。  
  - **跳转到 Chart（GET 图表）**：params 必须是**该 Chart 的 Request 结构体**，即该 GET 路由的 ChartTemplate 的 `Request` 结构体。前端打开图表时会用 params 转成 query（如 object_id=123）请求图表。  
    - 例：target 为 `"bangla_level_distribution.chart"` 时，params 用 `BanglaLevelDistributionReq{ObjectID: objectID}`（等级分布图请求结构体）。  
  - **外链**：params 传 `nil`。  
  - 总结：**跳 Table 用该表的 Model，跳 Form 用该 Form 的 Request，跳 Chart 用该 Chart 的 Request**，不要混用。

- **典型场景**：  
  1. **Table 列表「查看详情」列**：当前行关联另一张表，链接跳转到该表并带上当前行 ID（如预约列表的「会议室详情」→ 跳会议室列表，params 用 **MeetingRoom{ID: RoomID}**，MeetingRoom 是目标表的 Model）。  
  2. **Table 列表「操作」列**：根据状态动态生成链接（如投票主题列表「投票操作」→ 跳 `vote_submit.form` 用 **VoteSubmitReq**，跳 `vote_result.form` 用 **VoteResultReq**；职位列表「投递简历」→ 跳简历列表用 **HrResume{JobID: JobID}** 并 `_tab=OnTableAddRow`）。  
  3. **Form 响应**：提交后返回一个「查看结果」链接（如投票提交后返回「查看投票结果」，params 用 **VoteResultReq{TopicID: req.TopicID}**）。

```go
// Table 列表：不落库、只读，List Build 之后对每条记录赋值
RoomLink string `json:"room_link" gorm:"-" widget:"name:会议室详情;type:link;target:_blank" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。

// List 函数内，Build 之后：跳转到 Table 必须用目标表的 Model
// meeting_room_list.table 的 AutoCrudTable 是 MeetingRoom，故 params 用 MeetingRoom{ID: ...}
for i := range bookings {
    params := MeetingRoom{ID: bookings[i].RoomID}  // MeetingRoom 是会议室表的 Model
    bookings[i].RoomLink, _ = ctx.BuildFunctionUrlWithText("meeting_room_list.table", params, "查看会议室详情")
}

// 带 _tab 参数：打开列表并切到「新增」Tab（如投递简历），params 用目标表 Model，并按新增表单字段预填
params := HrResume{JobID: jobs[i].ID}  // HrResume 是简历表的 Model，新增表单会预填 JobID
jobs[i].ApplyLink, _ = ctx.BuildFunctionUrlWithText("hr_resume_list.table?_tab=OnTableAddRow", params, "投递简历")

// Form 响应：跳转到 Form 必须用该 Form 的请求结构体
// vote_result.form 的 Request 是 VoteResultReq，故 params 用 VoteResultReq{TopicID: ...}
params := VoteResultReq{TopicID: req.TopicID}
functionLink, _ := ctx.BuildFunctionUrlWithText("vote_result.form", params, "查看投票结果")
return resp.Form(&VoteSubmitResp{..., FunctionLink: functionLink}).Build()
```

完整示例：`read_doc("/system/prompt/case_catalog/tables/meeting")`（预约列表会议室详情 link）、`read_doc("/system/prompt/case_catalog/tables/hr")`（职位/简历列表 link、_tab=OnTableAddRow）、`read_doc("/system/prompt/case_catalog/formandtable/vote")`（投票操作/选项列表/提交结果 link）。

- **隐藏字段**：`widget:"-"` 表示该字段**被前端直接忽略**，不参与列表/表单的渲染，也不会被提交；常用于系统字段（如 DeletedAt、DeletedBy）或内部关联（如 `json:"-"` 的关联表）。
- **展示场景**：用 `hide.scenes` 声明隐藏场景，例如 `create,update` 表示仅列表展示、`list,update` 表示仅新增表单展示，见下节。

### 2. validate 标签

遵循 `github.com/go-playground/validator/v10`。常用：`required`、`min`/`max`、`oneof=值1 值2`（空格分隔，值含空格用单引号）、`email` 等。

```go
Title string `validate:"required,min=2,max=200"`
Status string `validate:"required,oneof=待处理 处理中 已完成"`
Email string `validate:"required,email"`
```

**条件必填 / 条件展示 / 条件排除（重要）**：

- 前端会基于一部分 `validator/v10` 的**存在性规则**做**动态显示、动态必填、提交前清理**，不只是提交时报错。
- 当前支持：
  - `required`
  - `required_if`
  - `required_unless`
  - `required_with`
  - `required_with_all`
  - `required_without`
  - `required_without_all`
  - `excluded_if`
  - `excluded_unless`
  - `excluded_with`
  - `excluded_with_all`
  - `excluded_without`
  - `excluded_without_all`
- 行为约定：
  - `required*`：条件成立时，字段会显示并标记为必填。
  - `excluded*`：条件成立时，字段会隐藏，并且提交时会从 payload 中剔除。
  - `required`：始终显示、始终必填。
- **字段引用必须写 Go 字段名，不是 json tag**。例如 `required_if=VoteType 多选` 中引用的是 `VoteType`，不是 `vote_type`。
- **条件值必须等于字段的实际提交值**。对于当前 widget tag 的静态 `select` / `multiselect` / `radio` / `checkbox`，实际提交值通常就是 `options` 中的字符串本身，因此应写 `required_if=VoteType 多选`、`oneof=单选 多选`，不要写成 `required_if=VoteType multiple`、`oneof=single multiple` 这种与选项值不一致的形式。
- **`validate` 多条规则使用英文逗号 `,` 分隔，不要使用分号 `;`**。例如：`validate:"required_if=VoteType 多选,min=1,max=10"` 是正确写法；`validate:"required_if=VoteType 多选;min=1;max=10"` 不要生成。
- **条件值尽量不要带空格**。虽然 `validator/v10` 某些场景支持引号，但前端动态规则目前按空格分词；如无必要，优先使用不含空格的中文值或稳定短枚举。

示例：

```go
type VoteSubmitReq struct {
    VoteType      string `json:"vote_type" widget:"name:投票类型;type:select;options:单选,多选;options_colors:409EFF,67C23A" validate:"required,oneof=单选 多选"`
    MaxSelections int    `json:"max_selections" widget:"name:最多选择数;type:number" validate:"required_if=VoteType 多选,min=1,max=10"`
}
```

说明：

- 当 `VoteType=多选` 时，`MaxSelections` 会显示且必填。
- 当 `VoteType!=多选` 时，`MaxSelections` 会隐藏。

再例如：

```go
type InvoiceReq struct {
    InvoiceType string `json:"invoice_type" widget:"name:发票类型;type:select;options:个人,企业;options_colors:909399,409EFF" validate:"required,oneof=个人 企业"`
    TaxNo       string `json:"tax_no" widget:"name:税号;type:input" validate:"excluded_unless=InvoiceType 企业"`
}
```

说明：

- 当 `InvoiceType=企业` 时，显示 `TaxNo`。
- 当 `InvoiceType!=企业` 时，隐藏 `TaxNo`，提交时也不会携带该字段。

**适用边界**：

- 这套动态行为只建议用于“字段当前是否该出现 / 是否当前必填”这类规则。
- `min`、`max`、`email`、`oneof` 等仍然只做校验，不驱动显示逻辑。

### 3. Table 筛选字段

Table 筛选统一在 Request 里显式声明业务字段，嵌入 `query.PageSortReq`，Handler 中手写 `Where` / `Joins` / `Preload`，最后调用 `Table`。

Table 筛选字段写在 Request 中，Handler 显式处理查询条件。常见筛选字段写法如下：

| 值 | 含义 | 适用 |
|----|------|------|
| like | 模糊 | input、text_area |
| in | 精确 IN | select、radio、user、department |
| contains | FIND_IN_SET | multiselect、users、departments |
| eq | 精确 = | ID、switch |
| gte,lte | 范围 | datetime、number、float、slider |

筛选字段按业务需要放入 Request；不要把所有 Model 字段都暴露成筛选项。

**组件值使用说明（重要）**：
- `type:user`、`type:users`、`type:department`、`type:departments` 这些组件提交到后端后，值可以直接当业务参数使用，不需要额外做组件层转换。
- 常见形态：`user/department` 通常是单值字符串；`users/departments` 通常是逗号分隔字符串，可在 Handler 里按业务需要手写拆分或包含查询。
- 当前请求上下文里也可直接拿到登录人信息：`ctx.GetRequestUser()`（请求用户）、`ctx.GetRequestUserDept()`（请求用户所在组织 full_code_path）。

示例：下面只演示 Table Model 字段建模。列表筛选字段另写在 Request 中，筛选逻辑不写在 Model 标签里。

```go
type CrmTicket struct {
    // 系统字段：前端仅在列表展示，不进入新增/编辑表单。
    ID        int   `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`           // 前端仅在列表展示，不进入新增/编辑表单。
    CreatedAt types.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
    UpdatedAt types.Time `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
    // 软删除：gorm.DeletedAt + widget:"-" 不在前端展示，GORM 查询时自动过滤已删除记录
    DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"` // 不做展示
    DeletedBy string         `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`       // 删除操作人，不做展示（可选）

    // 业务字段：未配 display 时前端会在列表/新增/编辑三个场景都展示。
    Title       string `json:"title" gorm:"column:title" widget:"name:工单标题;type:input"`
    Description string `json:"description" gorm:"column:description" widget:"name:问题描述;type:text_area"`
    Priority    string `json:"priority" gorm:"column:priority" widget:"name:优先级;type:select;options:低,中,高;options_colors:67C23A,E6A23C,F56C6C"`
    Status      string `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:待处理,处理中,已完成;options_colors:909399,E6A23C,67C23A"`
    IsUrgent    bool   `json:"is_urgent" gorm:"column:is_urgent" widget:"name:是否紧急;type:switch"`
    Progress    int    `json:"progress" gorm:"column:progress" widget:"name:完成进度;type:slider;min:0;max:100;unit:%"`
    Handler     string `json:"handler" gorm:"column:handler" widget:"name:处理人;type:user"`
    CcUsers     string `json:"cc_users" gorm:"column:cc_users" widget:"name:抄送人;type:users"`
    Deadline    types.Time `json:"deadline" gorm:"column:deadline;type:datetime" widget:"name:截止时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
    Remark      string `json:"remark" gorm:"column:remark" widget:"name:备注;type:text_area"`
}
```

### 4. hide.scenes 标签

用于控制字段在前端哪些界面隐藏。它不是权限控制：`hide:"create,update"` 表示前端只把字段渲染到列表列中，不会渲染到新增/编辑表单；不配置 `hide` 表示列表、新增表单、编辑表单都展示。`hide` 直接写逗号分隔的隐藏场景，值只能是 `list/create/update`，不能为空、不能重复。`table` / `form` 是容器组件，不作为 Table 列表列渲染；需要仅新增出现时可用 `hide:"list,update"`。

| scenes 值 | 新增表单 | 编辑表单 | 列表展示 | 适用场景 |
|-----------|----------|----------|----------|----------|
| `create,update` | ❌ 不展示 | ❌ 不展示 | ✅ 展示 | 主键、创建/更新时间、后端计算字段、列表链接 |
| `list,update` | ✅ 展示 | ❌ 不展示 | ❌ 不展示 | 仅创建时填写，创建后不再编辑 |
| `list,create` | ❌ 不展示 | ✅ 展示 | ❌ 不展示 | 仅编辑时填写 |
| `list` | ✅ 展示 | ✅ 展示 | ❌ 不展示 | 新增/编辑可填，但列表不展示 |
| 不设置 | ✅ 展示 | ✅ 展示 | ✅ 展示 | 普通业务字段 |

#### hide.scenes 最佳实践场景

**场景 1：仅列表展示、不落库的计算字段（hide:"create,update" + gorm:"-"）**

列表需要展示「剩余时间」等由后端根据其它字段计算出的值，不落库，且不进入新增/编辑表单。用 `gorm:"-"` 不落库，`hide:"create,update"` 表示前端仅在列表展示。

```go
RemainingTime string `json:"remaining_time" gorm:"-" widget:"name:剩余时间;type:text" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
```

在 List 函数 Build 之后遍历 `lists`，根据截止时间等计算并赋值给 `RemainingTime`。

**场景 2：后端在回调中赋值的字段**

创建人、提单部门等如果不希望用户在前端填写，通常不放到新增/编辑表单里，由 OnTableAddRow 用 `ctx.GetRequestUser()`、`ctx.GetRequestUserDept()` 等赋值。若业务字段本身就是 `user/users/department/departments` 组件，提交值也可直接用于入库或发消息等业务逻辑。

```go
Department string `json:"department" gorm:"column:department" widget:"name:提单部门;type:department" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
CreateBy   string `json:"create_by" gorm:"column:create_by" widget:"name:创建用户;type:user" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
```

在 OnTableAddRow 中：`row.Department = ctx.GetRequestUserDept()`；`row.CreateBy = ctx.GetRequestUser()`。

**场景 3：仅新增时展示、编辑/列表不展示（hide:"list,update"）**

某些字段只在「新增」时填写，编辑时不允许改（如投票主题的选项列表，创建后不可改）。用 `hide:"list,update"`，前端仅在新增表单展示，编辑表单和列表不展示该字段。

```go
Options []VoteOptionItem `json:"options" gorm:"-" widget:"name:投票选项;type:table" hide:"list,update" validate:"required,min=2"` // 前端仅在新增表单展示，列表和编辑不展示。
```

**场景 4：仅编辑时展示、新增/列表不展示（hide:"list,create"）**

某些字段只在「更新」时填写，新增时没有或不需要填。例如：实际完成时间（创建时未知）、关闭原因/处理备注（仅在结单或更新时填）、审核意见（仅审核人在更新时填）。用 `hide:"list,create"`，前端仅在编辑表单展示，新增表单和列表不展示该字段。

```go
ClosedReason  string `json:"closed_reason" gorm:"column:closed_reason" widget:"name:关闭原因;type:text_area" hide:"list,create"` // 前端仅在编辑表单展示，列表和新增不展示。
FinishedAt   types.Time  `json:"finished_at" gorm:"column:finished_at;type:datetime" widget:"name:实际完成时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"list,create"` // 前端仅在编辑表单展示，列表和新增不展示。
```

**场景 5：新增和编辑都展示、但列表不展示（hide:"list"）**

敏感或内部信息需要在新增/编辑表单里填写，但不在列表里展示，避免列表信息过载或泄露。例如：内部备注、成本价、二次确认密码等。用 `hide:"list"`，前端会在新增和编辑表单展示，列表不展示。

```go
InternalNote string  `json:"internal_note" gorm:"column:internal_note" widget:"name:内部备注;type:text_area" hide:"list"` // 前端在新增/编辑表单展示，列表不展示。
CostPrice    float64 `json:"cost_price" gorm:"column:cost_price" widget:"name:成本价;type:float;precision:2;unit:元" hide:"list"` // 前端在新增/编辑表单展示，列表不展示。
```

**小结**：`hide` 的值表示“隐藏在哪些场景”。`hide:"create,update"` = 仅列表展示；`hide:"list,update"` = 仅新增展示；`hide:"list,create"` = 仅编辑展示；`hide:"list"` = 新增/编辑展示但列表不展示；不设 = 三个场景均展示。配合 `widget:"-"` 可完全隐藏字段。

---

## 四、Table 模式要点

- **TableTemplate**：`BaseConfig` 含 Name、Request、CreateTables；**不要写 Table Response**，表格 schema 由 `AutoCrudTable` 推导。**`AutoCrudTable` 必须显式配置**（指向列表结构体，前端据此渲染列表字段、筛选、分页和表格 schema）。**不需要哪种操作就删掉对应回调**：不想要新增和批量导入 → 不配 `OnTableAddRow`；不想要更新 → 不配 `OnTableUpdateRow`；不允许删除 → 不配 `OnTableDeleteRows`（如消费记录、支付流水、操作日志通常应直接只读）。前端根据是否配置回调来显示或隐藏「新增」「编辑」「删除」按钮；工作台和服务端也会据此判断表是否允许写入。`OnTableCreateInBatches` 是系统内置批量导入能力，配置 `OnTableAddRow` 时自动暴露，不需要手写；若新增/编辑表单中有 select 需后端动态选项，配 `OnSelectFuzzyMap`（用法见「六、Form 模式要点 → OnSelectFuzzy」）。
- **AutoCrudTable 的 model 可落库字段类型**：model 里凡是有 **gorm 列**（会被 GORM 写入数据库）的字段，**只能是**以下可落库类型：**基础类型**（int、string、bool、int64、float64 等）、**string**（`gorm:"type:text"`，实际存 `bucket/object_key` 字符串，多文件逗号分隔）、**gorm.DeletedAt**（软删除，GORM 特例）。除此以外，**其他 struct、slice（如 type:table / type:form）不能作为一列写入数据库**；若在 model 里出现这类 struct/slice，须为：**外键关联**（如 `Room *MeetingRoom` 配 `gorm:"foreignKey:RoomID;references:ID"`，实际存的是 RoomID，不占一列）或 **gorm:"-"**（不落库，仅展示/表单用，如 RoomName、Status、Options、link 等）。否则 GORM 无法把该列写进数据库。
- **List 函数**：Request 显式声明筛选字段，并嵌入 `query.PageSortReq`（`widget:"-"`）隐藏分页字段；使用 `queryDB := db.Model(&Model{})` 后在 Build 前手写 `Where` / `Joins` / `Preload`，再调用 `resp.Table(&lists, queryDB, &Model{}, &req.PageSortReq).Build()`；Build 后可在内存中给计算字段赋值（如剩余时间、**link 跳转 URL**，见「三、结构体与标签 → link 组件」）。`Table` 只处理分页、排序、Count 和 Find；
- 主键、CreatedAt、UpdatedAt、DeletedAt、DeletedBy 等系统字段约定见案例；init_.go 由脚手架生成，不要手写。

完整 Table 示例（单表/多表/回调/OnSelectFuzzy/link）：`read_doc("/system/prompt/case_catalog/table/ticket")`、`read_doc("/system/prompt/case_catalog/tables/meeting")`、`read_doc("/system/prompt/case_catalog/tables/hr")`。

---

## 五、Table 回调函数

Table 的增删改由三个**可选**业务回调实现；**不配某个回调则对应操作不可用**（不配 `OnTableAddRow` 则无新增，也不暴露系统内置批量导入 `OnTableCreateInBatches`；不配 `OnTableUpdateRow` 则无编辑；不配 `OnTableDeleteRows` 则无删除）。配置了的回调用于新增、更新、删除时的业务逻辑。**流水/日志/审计类表默认不要配这些回调**；需要退款、冲正、撤销时，应做独立业务动作，不要直接修改流水记录本身。

### 1. OnTableAddRow（新增行）

- **作用**：绑定并校验请求体，填充后端负责赋值的字段（如创建人、部门），落库，可选调用第三方。
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
- **关键 API**：`req.BindChangedFields(&updateFields)`、`req.ChangedFields()`、`req.GetId()`、`req.IsFieldUpdated("fieldName")`。
- **注意**：必须用 `db.Model(&Model{}).Where("id = ?", req.GetId()).Updates(updates)` 更新，以便支持零值；本回调不校验 validate（仅部分字段更新）。

```go
OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
    db := ctx.GetGormDB()
    var updateFields CrmTicket
    if err := req.BindChangedFields(&updateFields); err != nil {
        return nil, err
    }
    updates := req.ChangedFields()
    if req.IsFieldUpdated("status") && updateFields.Status == "已关闭" {
        var current CrmTicket
        if err := db.First(&current, req.GetId()).Error; err != nil {
            return nil, err
        }
        durationMinutes := time.Since(current.CreatedAt.Time()).Minutes()
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

- **Build 之前**：在调用 `resp.Table(...).Build()` 之前，对 `queryDB` 做 Where（主表筛选、外表筛选、计算字段的筛选条件）、Joins、**Preload（GORM 预加载）** 等，再把 queryDB 作为参数传给 `resp.Table(&lists, queryDB, &Model{}, &req.PageSortReq).Build()`。
- **Build 之后**：对返回的 `lists` 逐条做计算、填充不落库字段（如剩余时间、状态、**关联表名称**、link URL）等。

#### GORM 预加载（Preload）

当列表需要展示**关联表字段**（如预约列表要显示「会议室名称」，而表里只存了 `room_id`）时，应使用 GORM 的 **Preload** 在查主表时一并加载关联，避免 N+1 查询。步骤：

1. **Model 上定义关联**：在列表结构体上声明关联字段，并设置 `gorm:"foreignKey:外键列"`（如 `Room *MeetingRoom` 配 `gorm:"foreignKey:RoomID"`），该关联字段可不落库、不展示（`json:"-"`、`widget:"-"`）。
2. **Build 前 Preload**：在调用 `resp.Table(...).Build()` 之前执行 `queryDB = queryDB.Preload("Room")`（参数为关联字段名），这样 Build 完成后每条记录的 `Room` 会被填充。
3. **Build 后处理**：遍历 `lists` 时，用预加载的关联填不落库的展示字段（如 `if item.Room != nil { item.RoomName = item.Room.Name }`），再填计算字段、link 等。

不预加载时，若在后处理里按 `room_id` 逐条查会议室会形成 N+1 查询；使用 Preload 后一次查询主表、一次查询关联表，性能更好。

下面先给一个**仅后处理**的最小示例（剩余时间），再给一个**前处理 + 后处理**的示例（会议室预约：外表/状态筛选 + 填充会议室名称/状态/link）。

```go
// 结构体：ID、标题、截止时间（落库），剩余时间（不落库，仅展示）
type Task struct {
    ID             int    `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
    Title          string `json:"title" gorm:"column:title" widget:"name:标题;type:input"`
    Deadline       types.Time `json:"deadline" gorm:"column:deadline;type:datetime" widget:"name:截止时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
    RemainingTime  string `json:"remaining_time" gorm:"-" widget:"name:剩余时间;type:input" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单；gorm:"-" 不落库。
}

func TaskList(ctx *app.Context, resp response.Response) error {
    var req struct {
        query.PageSortReq `widget:"-"`
    }
    if err := ctx.ShouldBind(&req); err != nil {
        return err
    }
    db := ctx.GetGormDB()
    var lists []*Task
    queryDB := db.Model(&Task{})
    if err := resp.Table(&lists, queryDB, &Task{}, &req.PageSortReq).Build(); err != nil {
        return err
    }
    // Build 之后：按截止时间计算「剩余时间」展示
    now := time.Now()
    for _, item := range lists {
        deadline := item.Deadline.Time()
        if deadline.IsZero() {
            item.RemainingTime = "-"
            continue
        }
        if !now.Before(deadline) {
            item.RemainingTime = "已过期"
            continue
        }
        diff := deadline.Sub(now)
        d, h := int(diff.Hours())/24, int(diff.Hours())%24
        if d > 0 {
            item.RemainingTime = fmt.Sprintf("%d天%d小时", d, h)
        } else {
            item.RemainingTime = fmt.Sprintf("%d小时", h)
        }
    }
    return nil
}
```

要点：计算字段用 `gorm:"-"`，不写库；`hide:"create,update"` 表示前端仅在列表展示，不进入新增/编辑表单。

**示例二：Build 前处理 + 后处理（会议室预约）**

请求里包含**外表筛选**（会议室名称）和**计算字段筛选**（预约状态：待开始/进行中/已结束，由开始/结束时间与当前时间算出）。需在 Build 前对 `queryDB` 做 Where；Build 后填充不落库字段（会议室名称、状态、详情 link）。参考：`read_doc("/system/prompt/case_catalog/tables/meeting")`（见 meeting_room_booking.go）。

```go
// 列表结构体：RoomName、Status、RoomLink 为不落库展示字段（gorm:"-"）
type MeetingRoomBooking struct {
    ID        int    `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:预约ID;type:ID" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
    RoomID    int    `json:"room_id" gorm:"column:room_id" widget:"name:会议室;type:select" callback:"OnSelectFuzzy"`
    Room      *MeetingRoom `json:"-" gorm:"foreignKey:RoomID"`
    RoomName  string `json:"room_name" gorm:"-" widget:"name:会议室名称;type:text" hide:"create,update"`   // 前端仅在列表展示，不进入新增/编辑表单；后处理从 Room 取。
    RoomLink  string `json:"room_link" gorm:"-" widget:"name:会议室详情;type:link" hide:"create,update"`  // 前端仅在列表展示，不进入新增/编辑表单；后处理 BuildFunctionUrlWithText。
    StartTime types.Time `json:"start_time" gorm:"column:start_time;type:datetime" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
    EndTime   types.Time `json:"end_time" gorm:"column:end_time;type:datetime" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
    Status    string `json:"status" gorm:"-" widget:"name:预约状态;type:select;options:待开始,进行中,已结束;options_colors:909399,409EFF,67C23A" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单；后处理按时间计算。
}

// 列表请求：RoomName、Status 为筛选条件（RoomName 来自外表，Status 为计算字段筛选），
// 需在 List 内手写 Where。字段要有 form 绑定 + widget 供前端展示筛选控件；非表字段可加 gorm:"-"。
// PageSortReq 只负责分页和排序；同名筛选字段由 Handler 显式处理。
type MeetingRoomBookingListReq struct {
    RoomName string `json:"room_name" form:"room_name" gorm:"-" widget:"name:会议室名称;type:input"`
    Status string `json:"status" form:"status" gorm:"-" widget:"name:预约状态;type:select;options:待开始,进行中,已结束;options_colors:909399,409EFF,67C23A"`
    query.PageSortReq `widget:"-"`
}

func MeetingRoomBookingList(ctx *app.Context, resp response.Response) error {
    db := ctx.GetGormDB()
    var req MeetingRoomBookingListReq
    if err := ctx.ShouldBind(&req); err != nil { return err }

    queryDB := db.Model(&MeetingRoomBooking{})

    // Build 前处理 1：按会议室名称筛选（外表字段，先查 MeetingRoom 得 roomIDs，再 Where room_id IN ?；无匹配时返回空表）
    if req.RoomName != "" {
        var roomIDs []int
        if err := db.Model(&MeetingRoom{}).Where("name LIKE ?", "%"+req.RoomName+"%").
            Pluck("id", &roomIDs).Error; err == nil && len(roomIDs) > 0 {
            queryDB = queryDB.Where("room_id IN ?", roomIDs)
        } else {
            queryDB = queryDB.Where("1 = 0")
        }
    }

    // Build 前处理 2：按预约状态筛选（计算字段：用 start_time/end_time 与当前时间比较；多表时建议加表名前缀如 crm_meeting_room_booking.start_time）
    if req.Status != "" {
        now := time.Now()
        switch req.Status {
        case "待开始": queryDB = queryDB.Where("start_time > ?", now)
        case "进行中": queryDB = queryDB.Where("start_time <= ? AND end_time > ?", now, now)
        case "已结束": queryDB = queryDB.Where("end_time <= ?", now)
        }
    }

    queryDB = queryDB.Preload("Room")
    var bookings []MeetingRoomBooking
    if err := resp.Table(&bookings, queryDB, &MeetingRoomBooking{}, &req.PageSortReq).Build(); err != nil {
        return err
    }

    // Build 后处理：填充不落库字段（会议室名称、状态、详情 link）
    for i := range bookings {
        if bookings[i].Room != nil {
            bookings[i].RoomName = bookings[i].Room.Name
        }
        bookings[i].Status = calculateBookingStatus(bookings[i].StartTime, bookings[i].EndTime)
        bookings[i].RoomLink, _ = ctx.BuildFunctionUrlWithText("meeting_room_list.table", MeetingRoom{ID: bookings[i].RoomID}, "查看会议室详情")
    }
    return nil
}

func calculateBookingStatus(startTime, endTime types.Time) string {
    now := time.Now()
    if now.Before(startTime.Time()) { return "待开始" }
    if now.Before(endTime.Time()) { return "进行中" }
    return "已结束"
}
```

要点：**前处理**用自定义 `queryDB`（外表 Where、计算字段 Where、**Preload 预加载关联**）再把 queryDB 作为参数传给 `resp.Table(&lists, queryDB, &Model{}, &req.PageSortReq).Build()`；**Build** 才真正执行 Count/Order/Offset/Limit/Find；**后处理**在 Build 之后遍历 `lists`，用预加载的 `Room` 填 `RoomName`，再填 `Status`、`RoomLink` 等不落库字段。

---

## 六、Form 模式要点

- **请求/响应结构体**：字段加 `widget`、`validate`；请求可含 `type:table`（子表）、`type:select` + `callback:"OnSelectFuzzy"` 等。
- **处理函数**：`ctx.ShouldBindValidate(&req)`；业务逻辑；成功 `return resp.Form(&respStruct).Build()`。涉及文件读写时见下「文件上传、下载与存储」。
- **事务要求**：一次提交若会同时改动多张表，尤其是余额、库存、数量、状态、流水这类强一致数据，必须使用 `db.Transaction(...)`，并在事务内做条件更新或再次校验，避免并发下超卖、透支或部分成功部分失败。
- **数据库兼容**：工作区应用默认是 **SQLite**；若后续可能切到 MySQL，涉及 Raw SQL、日期格式化、字符串拼接、upsert、JSON 函数时，必须先看 `db.Dialector.Name()` 分支处理，不要写死单一数据库语法。优先用 GORM 和 Go 代码做计算，实在需要 SQL 方言时再分支。**时间分组优先复用 SDK helper**：`app.DateTimeBucketExpr(db, "created_at", app.TimeBucketDay)`。
- **FormTemplate**：`BaseConfig` 含 Name、Request、Response；若请求中有下拉需联动后端数据，配 `OnSelectFuzzyMap`。
- **注册**：`packageContext.POST("路由名", Handler, FormTemplate)`。

#### 数据库兼容（SQLite 默认，MySQL 可切换）

工作区应用默认使用 SQLite。大部分 CRUD、筛选、排序、事务，直接用 GORM 就能同时兼容 SQLite/MySQL；真正容易踩坑的是**写 Raw SQL 或数据库函数**的时候。

**建议顺序**

1. 先用 GORM
2. GORM 不够，再用 Go 代码补计算
3. 还不够，再按 `db.Dialector.Name()` 分支写 SQL

**典型差异**

- **按天分组/时间格式化**
  - MySQL：`DATE_FORMAT(created_at, '%Y-%m-%d')`
  - SQLite：`strftime('%Y-%m-%d', created_at)`
- **字符串拼接**
  - MySQL：`CONCAT(a, b)`
  - SQLite：`a || b`
- **upsert**
  - MySQL：`ON DUPLICATE KEY UPDATE`
  - SQLite：`ON CONFLICT DO UPDATE`
- **JSON 函数**
  - 两边函数名和支持度不完全一致；不是刚需就尽量别在 Raw SQL 里直接写 JSON 函数

示例（日期分组，优先复用 SDK helper）：

```go
// dateExpr 用于 Select(... as date)，groupExpr 用于 Group(...)
dateExpr, groupExpr := app.DateTimeBucketExpr(db, "created_at", app.TimeBucketDay)
err := queryDB.
	Select(fmt.Sprintf("%s as date, COUNT(*) as count", dateExpr)).
	Group(groupExpr).
	Scan(&stats).Error
```

GORM `Count` 的参数必须是 `*int64`：

```go
var total int64
if err := queryDB.Count(&total).Error; err != nil {
    return err
}
```

需要业务 `int` 时，在计算处显式转换：`int(total)`。

#### 系统错误（必读）

系统错误（数据库异常、网络超时、Python/外部调用失败、未预期的 panic 等）需要**统一加上 `[系统错误]` 前缀**，并带上**报错信息和详细参数**（如请求体 `req`），方便大模型定位和排查问题。

- **规范写法**：`return nil, fmt.Errorf("[系统错误]-[函数名] 简短描述, req: %+v, err: %w", req, err)`；打日志时同样加上 `[系统错误]-[函数名]` 并输出 req、err。
- **参考实现**：`read_doc("/system/prompt/case_catalog/form/nlp")`（见 `jieba_segment.go` / `runJiebaOnText` 中 Python 执行失败分支）；使用 `pythonRuntime` 时须 **`defer executor.Close()`**，Go 与 Python 为同机子进程。

```go
// 系统错误：必须带 [系统错误]、函数名、req 与 err，方便大模型排查
// 使用 python runtime 默认临时目录时：创建 executor 后 defer executor.Close() 释放工作区
if err := executor.ExecuteJSON(ctx, &result); err != nil {
    logger.Errorf(ctx, "[系统错误]-[DoJiebaSegment] Python 执行失败, req: %+v, err: %v", req, err)
    return nil, fmt.Errorf("[系统错误]-[DoJiebaSegment] 执行中文分词失败, req: %+v, err: %w", req, err)
}
return resp.Form(&respStruct).Build()
```

#### 文件上传、下载与存储

- **上传**：请求或 Table 新增/编辑里用 `string` 字段，widget `type:files`；可选 `accept:.csv`、`max_size:50MB`、`max_count:10` 等。`string` 的持久化值是字符串：`bucket/object_key`，多文件用英文逗号分隔。Table 字段建议用 `gorm:"column:xxx;type:text"`。
- **列表缩略图/视频封面**：商品图、头像、照片、截图、凭证图、视频素材等需要在 Table 列表中直接辨认内容的字段，使用 `thumbnail:true;list_preview:true`。`thumbnail:true` 表示浏览器上传时前端生成轻量缩略图或视频封面并上传，后端不做媒体处理；`list_preview:true` 表示列表单元格优先用缩略图/封面展示。只对浏览器上传的图片/视频稳定生效；SDK/服务端生成的输出文件前端拿不到原始 `File`，不要假设一定有缩略图。
- **读上传的文件（Form 内）**：需要访问文件内容时（如解析 CSV、转 Excel），用 `fs := ctx.GetFS()`，`inputFiles := fs.DownloadFiles(req.xxx)` 得到运行时文件列表；遍历 `inputFiles`，用 `file`（如 `os.Open(file)`）读内容；用完后 **必须** `defer fs.RemoveFiles(inputFiles)` 清理临时文件。
- **响应里返回文件（供下载）**：业务生成文件到本地路径后，用 `outputFiles := fs.ResponseFiles([]string{outputPath})` 得到 `string` 填到响应结构体；返回值本身是 `bucket/object_key` 字符串，前端会通过 storage resolve 拿直连 URL 展示/下载。**路径建议始终用 `filepath.Abs` 得到绝对路径**再交给 `ResponseFiles` 或与 Python 互传（双方进程 cwd 不同）。若无上传、仅生成文件给用户（如 CSV 文本转 Excel），可先用 `ctx.GetFS().GetTraceOutputDir()` 得到当前 Trace 输出目录，在该目录下生成文件再 `ResponseFiles`。
- **Python runtime 生成可下载文件**：`read_doc("/system/prompt/case_catalog/form/python_output")`；Go 将 **绝对路径** 放入请求传给 Python（如 `savefig` 目标路径），**不要**再用 base64 绕一圈；须 **`defer executor.Close()`**。
- **参考实现**：Table 存储文件字段：`read_doc("/system/prompt/case_catalog/tables/hr")`（见 hr_resume_list.go 的 `ResumeFile`）；Form 上传读文件 + 响应返回文件：`read_doc("/system/prompt/case_catalog/form/excelorcsv")`（见 `DoCsvToExcel`、`DoCsvTextToExcel`）。

```go
// Form 内：读上传文件 → 处理 → 返回生成的文件
func DoCsvToExcel(ctx *app.Context, req *CsvToExcelReq) (*CsvToExcelResp, error) {
    fs := ctx.GetFS()
    inputFiles := fs.DownloadFiles(req.InputFiles)
    defer fs.RemoveFiles(inputFiles)

    var outputFilePaths []string
    for _, file := range inputFiles {
        if file == "" { continue }
        outPath, err := csvToExcel(ctx, file)
        if err != nil { /* 记录错误 */ continue }
        outputFilePaths = append(outputFilePaths, outPath)
    }

    var outputFiles string
    if len(outputFilePaths) > 0 {
        outputFiles = fs.ResponseFiles(outputFilePaths)
    }
    return &CsvToExcelResp{OutputFiles: outputFiles, ...}, nil
}
```

```go
// Table 模式：文件字段落库，无需 GetFS/Download/Remove
type HrResume struct {
    ResumeFile string `json:"resume_file" gorm:"column:resume_file;type:text" widget:"name:简历附件;type:files"`
}
// OnTableAddRow/OnTableUpdateRow 里直接 db.Create(&row) / db.Updates(updates)，ResumeFile 会按 bucket/object_key 字符串存储
```

```go
// Table 模式：图片/视频字段需要列表识别内容时，开启前端缩略图/封面和列表预览
type Product struct {
    Images string `json:"images" gorm:"column:images;type:text" widget:"name:商品图片;type:files;accept:image/*;max_count:5;thumbnail:true;list_preview:true"`
    Video  string `json:"video" gorm:"column:video;type:text" widget:"name:展示视频;type:files;accept:video/*;max_count:1;thumbnail:true;list_preview:true"`
}
```

#### OnSelectFuzzy（下拉联动后端数据）

当 **select** 或 **multiselect** 的选项需要从后端查库、按关键字模糊搜索或按业务条件过滤（如只显示「可用」会议室、只显示「上架」商品）时，使用 **OnSelectFuzzy**。前端在下拉里输入关键字或回显已选值时，会调用该回调，由后端返回选项列表。

- **适用**：Form 请求中的 select、Form 请求里 **table 子表**中的 select、**Table 模式**新增/编辑表单中的 select（如预约选会议室）。只要该 select 需要「后端动态选项」，就配 OnSelectFuzzy。
- **字段配置**：在字段上加 `callback:"OnSelectFuzzy"`；若选项依赖同表单其他字段，可加 `depend_on:字段json名`（如 `depend_on:topic_id`）提示前端该字段依赖上方字段。字段的 **json 名**（如 `product_id`、`member_id`、`room_id`）作为模板里 `OnSelectFuzzyMap` 的 key。
- **模板配置**：在 **FormTemplate** 或 **TableTemplate** 的 `BaseConfig` 里设置 `OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{"字段json名": handler}`，key 与请求/列表结构体里该字段的 json 名一致。
- **回调签名**：`func(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error)`。

**OnSelectFuzzyReq**：前端会传 `Code`（字段标识）、`Type`（`by_keyword` 用户输入关键字 / `by_value` 回显单个值 / `by_values` 回显多选值）、`Value`（关键字或已选值）。常用方法：`req.IsByKeyword()`、`req.IsByValue()`、`req.IsByValues()`；`req.Keyword()` 取关键字；`req.GetValue()`、`req.GetValues()` 取已选值（用于回显时查库）。**在回调中获取当前表单已填数据**：当某个下拉的选项依赖同表单其他字段时（例如「投票选项」依赖「投票主题」），可用 `req.BindCurrentFormData(&currentData)` 将当前用户已填写的表单数据绑定到与 Request 一致的结构体上，从而根据已填字段（如 `currentData.TopicID`）查库返回选项。**字段顺序很重要**：依赖的字段必须放在表单上面先填写（如投票主题在上、投票选项在下），这样用户先选主题后，选项回调里才能通过 `BindCurrentFormData` 拿到 `TopicID`；否则选项回调触发时依赖字段可能尚未填写，需在回调里校验并提示「请先选择 xxx」。

**OnSelectFuzzyResp**：返回 `Items []*SelectFuzzyItem`（每项含 `Value`、`Label`、可选 `DisplayInfo` 供详情展示）；`MaxSelections`（0 表示不限制，1 表示单选）；可选 `Statistics map[string]interface{}`，用于在表单旁**聚合展示**（见下「Statistics 与聚合计算」）。

**回调中获取当前表单数据示例（依赖字段 + 顺序）**：例如提交投票表单：先选「投票主题」、再选「投票选项」，选项列表依赖主题 ID。请求结构体上把 `TopicID` 放上面、`OptionIDs` 放下面；选项回调里用 `BindCurrentFormData` 拿到已填的 `TopicID`，再按 `topic_id` 查库返回该主题下的选项。若解析失败或 `TopicID == 0`，提示用户先选择投票主题。

```go
// 请求结构体：依赖的字段放上面，顺序重要
type VoteSubmitReq struct {
    TopicID   int   `json:"topic_id" widget:"name:选择投票主题;type:select" validate:"required" callback:"OnSelectFuzzy"`
    OptionIDs []int `json:"option_ids" widget:"name:选择投票选项;type:multiselect;depend_on:topic_id" validate:"required,min=1" callback:"OnSelectFuzzy"`
    Remark    string `json:"remark" widget:"name:投票备注;type:text_area" validate:"max=500"`
}

// 选项回调：依赖 topic_id，需先拿到当前表单已填数据
func voteOnSelectFuzzyOption(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
    db := ctx.GetGormDB()
    var currentData VoteSubmitReq
    err := req.BindCurrentFormData(&currentData)
    if err != nil {
        return nil, fmt.Errorf("表单解析失败，请刷新选择投票主题后再重试")
    }
    if currentData.TopicID == 0 {
        return nil, fmt.Errorf("请先选择投票主题，再选择投票选项")
    }
    var options []*VoteOption
    // 按 currentData.TopicID 查该主题下的选项...
    db.Where("topic_id = ?", currentData.TopicID).Find(&options)
    // 构造 Items、Statistics 后返回
    return &callback.OnSelectFuzzyResp{Items: items, ...}, nil
}
```

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

`OnSelectFuzzyResp.Statistics` 的键值对会在前端表单旁展示（如收银台「商品原价总额」「会员折扣后价格」「当前余额」等）。值可以是**静态字符串**，也可以是 **`statistics` 包**返回的表达式，由前端根据当前 **table 行数据**或**选中项**动态计算。仅当当前文件真实使用 `statistics.Value` 等符号时才导入 `github.com/ai-agent-os/ai-agent-os/sdk/agent-app/statistics`。

**1. table 子表场景（对当前 table 多行聚合）**

当 OnSelectFuzzy 用于 **table 子表**中的 select（如收银台商品清单里的「商品」）时，Statistics 里可用：

- **`statistics.Sum(expression)`**：对当前 table 所有行按表达式求和。表达式与 **MySQL/SQL 一致**：空格分隔、`*` 表示乘，字段名来自 `SelectFuzzyItem.DisplayInfo` 的 key 或行内字段名（如 `quantity`）。条件用 **MySQL IF(cond, thenExpr, elseExpr)**。**前端表达式解析器（ExpressionParserV2）支持该格式**，可直接使用。
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

完整收银台示例（商品清单 Sum/Count、会员卡 Value、表达式格式）：`read_doc("/system/prompt/case_catalog/form_table_chart/cashier")`。

**Table 模式**下同样可用 OnSelectFuzzy：在**列表结构体**（AutoCrudTable 指向的模型）里给新增/编辑需要动态选项的 select 字段加 `callback:"OnSelectFuzzy"`；只用于列表筛选时，把字段写在 Table Request 中并加同样的 callback。在 **TableTemplate** 的 `BaseConfig.OnSelectFuzzyMap` 里按「字段 json 名」注册回调即可。例如会议室预约表新增/编辑时选择会议室，评价记录表筛选区按评价对象筛选。外键字段 code 可以叫 `room_id` / `object_id`，但 `widget name` 应写“会议室”/“评价对象”，不要写“会议室ID”/“评价对象ID”；用户按名称搜索，前端实际提交 ID。

```go
// Table 模式：列表结构体（预约表）里会议室字段加 callback:"OnSelectFuzzy"
type MeetingRoomBooking struct {
    // ...
    RoomID   int    `json:"room_id" gorm:"column:room_id" widget:"name:会议室;type:select" validate:"required" callback:"OnSelectFuzzy"`
    RoomName string `json:"room_name" gorm:"-" widget:"name:会议室名称;type:text" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
    // ...
}

// TableTemplate 的 BaseConfig 里配置 OnSelectFuzzyMap，key 为 room_id
var MeetingRoomBookingListTemplate = &app.TableTemplate{
    BaseConfig: app.BaseConfig{
        Name: "会议室预约管理", Request: &MeetingRoomBookingListReq{},
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
    if req.IsByValue() {
        db = db.Where("id = ?", req.GetValue()).Limit(1)
    } else if req.IsByValues() {
        db = db.Where("id in ?", req.GetValues())
    } else {
        db = db.Where("name LIKE ? OR type LIKE ? OR location LIKE ?", "%"+req.Keyword()+"%", "%"+req.Keyword()+"%", "%"+req.Keyword()+"%").Limit(20)
    }
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

完整示例：`read_doc("/system/prompt/case_catalog/form_table_chart/cashier")`（Form + table 子表 OnSelectFuzzy + 聚合计算）、`read_doc("/system/prompt/case_catalog/tables/meeting")`（**Table 模式**预约选会议室 OnSelectFuzzy，见 meeting_room_booking.go）。

Form 请求中 table 子表、OnSelectFuzzy、多 POST 同目录等：`read_doc("/system/prompt/case_catalog/form/excelorcsv")`、`read_doc("/system/prompt/case_catalog/form_table_chart/cashier")`（收银台）、`read_doc("/system/prompt/case_catalog/formandtable/vote")`（投票提交：**BindCurrentFormData** 先选主题再选选项的依赖下拉示例）。

---

## 七、Chart 模式要点

Chart 用于**只读的统计/图表**（BI），GET 请求。ChartTemplate、请求结构体、处理函数、注册方式见第二节「快速开始 → Chart 模式」。

#### 图表开发 Badcase（务必避免）

以下为大模型常见错误，写图表代码时请勿出现：

1. **一个函数返回多张图**  
   - **错误**：写 `resp.Charts(...)`、`resp.Chart(chart1, chart2)` 等。SDK 没有 `resp.Charts`，`resp.Chart(chart).Build()` 只接受**一个**图表。  
   - **正确**：每张图一个 GET 路由。参考收银台：4 张图 = 4 个 `.chart` 路由、4 个函数。

2. **手填 ChartType 或 Series[].Type**  
   - **错误**：使用 `&chart.Chart{ ChartType: "line", ... }` 或给 Series 填 `Type: "line"`（chart 包无通用 Chart 结构体，只用具体类型）。
   - **正确**：使用具体类型 `&chart.LineChart{}`、`&chart.BarChart{}` 等，只填 Title、XAxis、Series（Name、Data、可选 Config），不填 ChartType 和 Series[].Type；框架会在 `resp.Chart()` 时自动注入。

3. **误用 sdk/agent-app 下的 query 包**  
   - **错误**：`import "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/query"`，导致编译报错「包找不到」。  
   - **正确**：查询/分页等应使用 `github.com/ai-agent-os/ai-agent-os/pkg/gormx/query`（或项目内实际提供的 query 包），不要使用 `sdk/agent-app/query`。

4. **不确定时先看案例**  
   - 图表个数、路由拆分、返回格式，以收银台案例为准：`read_doc("/system/prompt/case_catalog/form_table_chart/cashier")`，看每个图表是如何「一个 GET 路由 + 一个具体图表类型返回值」实现的。

#### 图表类型说明（4 种，唯一参考表）

| 必须使用的类型 | 说明 | 典型场景 | XAxis | Series Data 格式 |
|------|------|----------|-------|------------------|
| `chart.LineChart` | 折线图 | 时间趋势、多指标随时间的走势 | 必需，如日期列表 | `[]interface{}{y1, y2, ...}`，与 XAxis 一一对应 |
| `chart.BarChart` | 柱状图 | 分类对比、各维度数量/金额 | 必需，如分类名 | `[]interface{}{v1, v2, ...}`，与 XAxis 一一对应 |
| `chart.PieChart` | 饼图 | 占比分布、构成比例 | 不需要 | `[]interface{}{ map[string]interface{}{"name":"分类","value":数值}, ... }` |
| `chart.GaugeChart` | 仪表盘 | 单指标（完成率、平均值、达标值） | 不需要 | `[]interface{}{ 单值 }`，可选 Series.Config 的 min/max/detail |

**LineChart（折线图）**：按时间或类别展示趋势，可多系列。XAxis 为刻度（如日期），每个 Series 的 Data 与 XAxis 长度一致。

```go
c := &chart.LineChart{
    Title:  "工单趋势统计",
    XAxis:  dateLabels,  // []string{"2025-01-01", "2025-01-02", ...}
    Series: []chart.ChartSeries{
        {Name: "工单数量", Data: []interface{}{10, 25, 18, ...}},
        {Name: "已完成数", Data: []interface{}{5, 12, 10, ...}},
    },
    Metadata: map[string]interface{}{"总工单数": totalCount, "数据更新时间": time.Now().Format("2006-01-02 15:04:05")},
}
return resp.Chart(c).Build()
```

**BarChart（柱状图）**：分类对比，XAxis 为分类名，Data 为对应数值。

```go
c := &chart.BarChart{
    Title:  "工单优先级分布统计",
    XAxis:  []string{"低", "中", "高"},
    Series: []chart.ChartSeries{{Name: "工单数量", Data: []interface{}{8, 20, 5}}},
    Metadata: map[string]interface{}{"总工单数": totalCount, "完成率": "66.67%", ...},
}
return resp.Chart(c).Build()
```

**PieChart（饼图）**：展示占比，不需要 XAxis。Data 中每个元素为 `{"name": "分类名", "value": 数值}`。

```go
pieData := make([]interface{}, 0)
for _, stat := range statusStats {
    pieData = append(pieData, map[string]interface{}{"name": stat.Status, "value": stat.Count})
}
c := &chart.PieChart{
    Title:   "工单状态分布",
    Series:  []chart.ChartSeries{{Name: "工单状态", Data: pieData}},
    Metadata: map[string]interface{}{"总工单数": totalCount, "待处理数": statusMap["待处理"], ...},
}
return resp.Chart(c).Build()
```

**GaugeChart（仪表盘）**：单指标，Data 为单元素数组；可选 Config 指定 min、max、detail.formatter（如 `"¥{value}"`）。

```go
c := &chart.GaugeChart{
    Title: "工单完成率",
    Series: []chart.ChartSeries{
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
return resp.Chart(c).Build()
```

完整示例：收银台统计（LineChart/BarChart/PieChart/GaugeChart）`read_doc("/system/prompt/case_catalog/form_table_chart/cashier")`。

---

## 八、注册与目录约定

- **init()**：在业务 .go 中写；`packageContext.GET("路由名", ListFunc, TableTemplate)` 或 `packageContext.POST("路由名", Handler, FormTemplate)` 或 `packageContext.GET("路由名", ChartHandler, ChartTemplate)`。`packageContext` 由脚手架生成，不要重复声明。
- **init_.go**：由系统生成，不要用 write_go_file 创建或修改。
- **目录**：一个包一个目录，路由名与业务含义对应；多表/多 Form 可在同包多文件，各自 GET/POST 注册。创建类流程以系统注入的创建 SOP 和 `/system/prompt/case_catalog/*` 匹配案例为准。

### 路由命名约定（类型后缀，必须）

**生成代码时，路由名必须带类型后缀。** 这样从 full_code_path / URL 即可看出函数类型（自解释、大模型友好）：**看到后缀即知类型**，无需再查 DB 或文档。

| 类型 | 后缀 | 含义 | 示例 |
|------|------|------|------|
| Form | `.form` | 表单（POST） | `cashier_desk.form`、`vote_submit.form` |
| Table | `.table` | 表格列表（GET） | `ticket_list.table`、`meeting_room_list.table` |
| Chart | `.chart` | 图表（GET） | `cashier_sales_trend_statistics.chart` |

- **注册时必须带后缀**：`packageContext.POST("cashier_desk.form", ...)`；`packageContext.GET("ticket_list.table", ...)`；`packageContext.GET("sales_trend_statistics.chart", ...)`。禁止注册无后缀的路由名。
- **link 的 target**：跳转到其他函数时，target 需与注册的路由名一致（含后缀），如 `"meeting_room_list.table"`、`"vote_result.form"`。
- 示例与案例（如 `/system/prompt/case_catalog`）均按此约定命名。

---

## 九、完整案例（read_doc 路径）

以下路径均在系统消息「可读的目录」中；按需 read_doc 获取该案例的 PRD 与完整代码。

- **单 Table**：`/system/prompt/case_catalog/table/ticket`
- **单 Form**：`/system/prompt/case_catalog/form/excelorcsv`、`/system/prompt/case_catalog/form/images`、`/system/prompt/case_catalog/form/pdf`、`/system/prompt/case_catalog/form/nlp`、`/system/prompt/case_catalog/form/videos`
- **多 Table**：`/system/prompt/case_catalog/tables/meeting`、`/system/prompt/case_catalog/tables/hr`
- **Table + Form**：`/system/prompt/case_catalog/formandtable/vote`
- **Table + Form + Chart**：`/system/prompt/case_catalog/form_table_chart/cashier` （全部类型的图表都有在这个里面呈现）

写代码前，先按当前需求选择至少一个匹配案例读取；组合型需求读多个案例。完成代码后按当前身份 SOP build、验证并给出下一步建议。
