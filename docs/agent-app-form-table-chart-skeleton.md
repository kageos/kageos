# Agent-App SDK Form / Table / Chart 骨架说明

本文档给大模型使用，目标是让模型在工作台里写 AgentOS 应用时，先理解 `agent-app` SDK 的组织方式，再按稳定骨架生成代码。

## 一、Agent-App SDK 是什么

`agent-app` SDK 是 `ai-agent-os` 系统的应用开发 SDK。

`ai-agent-os` 不是传统“手写前端页面 + 后端接口”的模式，而是：

- Go `package` 对应一个应用目录。
- Go 文件里的处理函数对应一个可执行 API。
- API 在前端自动渲染成三种形态：
  - `Form`：表单，一次输入，一次输出。
  - `Table`：表格，管理一批长期记录，支持搜索、分页、增删改。
  - `Chart`：图表，基于数据聚合后展示 ECharts 图表。

开发者只需要通过 `write_go_file` 写入满足 SDK 约定的 Go 代码，然后调用 `build_workspace`。编译后系统会解析 Go struct tag 和 Template 配置，自动生成前端页面，不需要写 HTML/CSS/JS。

最重要的心智模型：

```text
Go struct tag = 前端 UI 协议
FormTemplate / TableTemplate / ChartTemplate = 前端页面类型声明
packageContext.POST/GET = 注册函数路由
```

## 二、通用规则

### 路由和 Template 必须匹配

| 类型 | 路由后缀 | 注册方式 | Template |
| --- | --- | --- | --- |
| Form | `.form` | `packageContext.POST` | `*app.FormTemplate` |
| Table | `.table` | `packageContext.GET` | `*app.TableTemplate` |
| Chart | `.chart` | `packageContext.GET` | `*app.ChartTemplate` |

正确示例：

```go
packageContext.POST("demo_submit.form", DemoSubmit, DemoSubmitTemplate)
packageContext.GET("demo_item_list.table", DemoItemList, DemoItemListTemplate)
packageContext.GET("demo_trend.chart", DemoTrendChart, DemoTrendChartTemplate)
```

错误示例：

```go
// 错误：Go 文件名不要带 .form/.table/.chart
demo_submit.form.go

// 错误：Form 路由不要注册成 GET table
packageContext.GET("demo_submit.form", DemoSubmit, DemoSubmitTemplate)
```

### import 规则

不要复制“通用 import 模板”。每个 Go 文件只导入当前文件真实使用的包，先写代码再按本文件里的符号补 import。

常见包的使用条件：

| 包 | 什么时候导入 |
| --- | --- |
| `pkg/gormx/query` | 当前文件的 Request 嵌入 `query.PageSortReq`，或代码真实使用 `query.` |
| `pkg/logger` | 当前文件真实调用 `logger.Errorf` 等日志函数 |
| `sdk/agent-app/app` | 当前文件使用 `app.Context`、`app.FormTemplate`、`app.TableTemplate`、`app.ChartTemplate`、`app.ChartTypeLine` 等 |
| `sdk/agent-app/callback` | 当前文件定义 Table 回调、OnSelectFuzzy 回调，或使用 `callback.` 类型 |
| `sdk/agent-app/chart` | 当前文件定义 Chart 响应，如 `chart.LineChart`、`chart.PieChart` |
| `sdk/agent-app/response` | 当前文件入口函数参数使用 `response.Response` |
| `sdk/agent-app/types` | 当前文件字段或代码使用 `types.Time` |
| `sdk/agent-app/statistics` | 当前文件真实使用 `statistics.Value` 等 OnSelectFuzzy 统计表达式 |
| `gorm.io/gorm` | 当前文件使用 `gorm.DeletedAt`、`gorm.DB`、`gorm.Expr`、`gorm.ErrRecordNotFound` 等 |

禁止：

- 不要导入 `github.com/ai-agent-os/ai-agent-os/sdk/agent-app` 根包。
- 不要为了“可能会用到”提前导入包。
- 不要创建 `nps_types.go` 这类 import shim 文件来消除诊断。
- 一个文件没有 `query.`，就不要导入 `query`；没有 `callback.`，就不要导入 `callback`。

时间字段必须用 `types.Time`，不要直接用 `time.Time` 做表单/表格字段。

## 三、Form 函数骨架

Form 适合一次性动作：

- 提交评价
- 上传文件并转换
- 生成报告
- 发送通知
- 执行一次计算

前端渲染逻辑：

```text
Request struct  -> 输入表单
点击提交        -> 后端执行业务逻辑
Response struct -> 输出结果区域
```

### 带注释 Form 示例

```go
package demo

import (
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
)

// DemoSubmitReq 是 Form 的输入结构。
// 每个字段都会被前端渲染成一个表单组件。
type DemoSubmitReq struct {
	// widget name 是前端字段名称。
	// type:input 表示普通输入框。
	// validate:"required" 表示提交时必填。
	Name string `json:"name" widget:"name:姓名;type:input;placeholder:请输入姓名" validate:"required"`

	// type:select 表示静态下拉框。
	// 静态 select 必须写 options。
	// options_colors 数量必须和 options 一致，颜色必须是不带 # 的 6 位十六进制。
	Level string `json:"level" widget:"name:等级;type:select;options:A,B,C;options_colors:67C23A,409EFF,E6A23C;render_default:A" validate:"required"`

	// type:text_area 表示多行文本。
	Remark string `json:"remark" widget:"name:备注;type:text_area;placeholder:请输入备注"`
}

// DemoSubmitResp 是 Form 的输出结构。
// 提交成功后，前端会把这里的字段渲染到结果区域。
type DemoSubmitResp struct {
	Success bool   `json:"success" widget:"name:是否成功;type:switch"`
	Message string `json:"message" widget:"name:处理结果;type:text_area"`
	Time    string `json:"time" widget:"name:处理时间;type:text"`
}

// DemoSubmit 是 Form 的入口函数。
// 入口函数负责：
// 1. 绑定请求参数
// 2. 调用业务逻辑
// 3. 通过 resp.Form 返回结果
func DemoSubmit(ctx *app.Context, resp response.Response) error {
	var req DemoSubmitReq

	// ShouldBindValidate 会先把前端提交的数据绑定到 req，
	// 然后执行 validate 标签校验。
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	res, err := DoDemoSubmit(ctx, &req)
	if err != nil {
		return err
	}

	// Form 必须使用 resp.Form(res).Build() 返回。
	return resp.Form(res).Build()
}

// DoDemoSubmit 是实际业务逻辑。
// 建议把业务逻辑拆出来，入口函数只负责绑定和响应。
func DoDemoSubmit(ctx *app.Context, req *DemoSubmitReq) (*DemoSubmitResp, error) {
	// ctx.GetRequestUser() 获取当前请求用户。
	// 创建人、提交人、操作人一般都从这里取，不要让用户手填。
	currentUser := ctx.GetRequestUser()
	if currentUser == "" {
		return nil, fmt.Errorf("获取当前用户失败，请重新登录")
	}

	logger.Infof(ctx, "demo submit by user=%s req=%+v", currentUser, req)

	return &DemoSubmitResp{
		Success: true,
		Message: fmt.Sprintf("用户 %s 已提交：%s，等级 %s", currentUser, req.Name, req.Level),
		Time:    time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// DemoSubmitTemplate 是 Form 的 schema 声明。
// 前端根据 Request / Response 的 struct tag 自动渲染页面。
var DemoSubmitTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "示例提交表单",
		Desc:     "演示 Form 的输入、提交和输出",
		Tags:     []string{"示例"},
		Request:  &DemoSubmitReq{},
		Response: &DemoSubmitResp{},
	},
}

func init() {
	// Form 路由必须以 .form 结尾。
	// Form 通常用 POST，因为它代表一次提交动作。
	packageContext.POST("demo_submit.form", DemoSubmit, DemoSubmitTemplate)
}
```

### Form 常见注意事项

- `Request` 只放用户需要填写的字段。
- `Response` 只放需要展示给用户看的结果。
- 当前用户用 `ctx.GetRequestUser()` 获取，不要默认做成用户选择字段。
- 有副作用的逻辑，例如写表、扣库存、发消息，应该放在业务函数里，必要时用数据库事务。
- Form 不适合管理长期记录；长期记录用 Table。

## 四、Table 函数骨架

Table 适合长期记录管理：

- 客户档案
- 工单列表
- 商品列表
- 问卷列表
- 评价记录
- 支付流水

前端渲染逻辑：

```text
Model struct               -> 表格列、新增/编辑表单字段
Request struct             -> 列表筛选字段
query.PageSortReq              -> 分页和排序
Table 筛选字段             -> 写在 Request 中，Handler 显式处理查询条件
hide:"create,update"      -> 只在列表展示
hide:"update"             -> 只在列表和新增弹窗展示，更新时候不展示
hide:"list,create"        -> 只在更新时候展示，其他时候不展示
OnTableAddRow != nil       -> 前端显示新增入口
OnTableUpdateRow != nil    -> 前端允许编辑
OnTableDeleteRows != nil   -> 前端允许删除
```

如果 `OnTableAddRow` 不实例化，前端就禁止新增。适合收银记录、支付流水、评价提交记录这类“只能由业务动作产生，不能手工新增”的事实表。

如果 `OnTableUpdateRow` 不实例化，前端就禁止编辑。

如果 `OnTableDeleteRows` 不实例化，前端就禁止删除。

### 带注释 Table 示例

```go
package demo

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
	"gorm.io/gorm"
)

// DemoItem 是 Table 的数据模型。
// 这个 struct 同时承担三件事：
// 1. GORM 数据库模型
// 2. 前端表格列定义
// 3. 前端新增/编辑表单字段定义
type DemoItem struct {
	// ID 必须是主键。
	// widget type:ID 表示 ID 展示。
	// hide:"create,update" 表示只在列表展示，不进入新增/编辑表单。
	ID int `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`

	// CreatedAt / UpdatedAt 必须用 sdk/agent-app/types.Time。
	// 不要直接用 time.Time，否则前端 schema 和 SDK 时间控件可能不匹配。
	// autoCreateTime / autoUpdateTime 由 GORM 自动维护。
	CreatedAt types.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`

	// DeletedAt 是 GORM 软删除字段。
	// widget:"-" 表示前端完全不渲染。
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	// 普通业务字段。
	// widget 控制新增/编辑表单和列表列的组件类型。
	// Table 筛选字段写在 Request 里，Handler 里手写 Where。
	Name string `json:"name" gorm:"column:name;comment:名称" widget:"name:名称;type:input" validate:"required"`

	// 静态 select 必须写 options。
	Status string `json:"status" gorm:"column:status;comment:状态" widget:"name:状态;type:select;options:启用,停用;options_colors:67C23A,F56C6C;render_default:启用" validate:"required"`

	// type:text_area 表示多行文本。
	Remark string `json:"remark" gorm:"column:remark;comment:备注" widget:"name:备注;type:text_area"`

	// 创建人通常由后端 ctx.GetRequestUser() 自动写入。
	// hide:"create,update" 表示只展示，不让用户在新增/编辑时填写。
	CreatedBy string `json:"created_by" gorm:"column:created_by;comment:创建人" widget:"name:创建人;type:user" hide:"create,update"`

	// gorm:"-" 表示这不是数据库字段，只是列表展示用的计算字段。
	// type:link 表示前端渲染成可点击链接。
	// 这类字段通常在 Build() 查询完成之后再填充。
	DetailLink string `json:"detail_link" gorm:"-" widget:"name:查看详情;type:link;target:_blank" hide:"create,update"`
}

func (DemoItem) TableName() string {
	return "demo_item"
}

// DemoItemListReq 是 Table 查询请求。
// 业务筛选字段显式写在 Request 里，并在 Handler 中手写 Where。
// query.PageSortReq 只负责分页和排序，不参与业务筛选。
type DemoItemListReq struct {
	Name   string `json:"name" form:"name" widget:"name:名称;type:input"`
	Status string `json:"status" form:"status" widget:"name:状态;type:select;options:启用,停用;options_colors:67C23A,F56C6C"`

	OnlyEnabled bool `json:"only_enabled" form:"only_enabled" widget:"name:只看启用;type:switch"`

	query.PageSortReq `widget:"-"`
}

// DemoItemList 是 Table 查询入口。
func DemoItemList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req DemoItemListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	// Build() 之前：可以继续改 queryDB。
	// 这里适合做：
	// 1. 外表搜索：先查外表 ID，再 queryDB.Where("xxx_id IN ?", ids)
	// 2. 计算字段筛选：例如“待开始/进行中/已结束”转成 start_time/end_time 条件
	// 3. 固定业务条件：例如只看启用、只看当前用户的数据
	// 4. GORM 预加载：queryDB = queryDB.Preload("Relation")
	queryDB := db.Model(&DemoItem{})
	if req.Name != "" {
		queryDB = queryDB.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}
	if req.OnlyEnabled {
		queryDB = queryDB.Where("status = ?", "启用")
	}

	var rows []DemoItem

	// Table 只是把 rows、queryDB、Model 和分页参数先保存到 builder 里。
	// 注意：这里还没有真正查询数据库。
	builder := resp.Table(&rows, queryDB, &DemoItem{}, &req.PageSortReq)

	// Build() 才是真正执行查询的地方。
	// Build() 内部会做：
	// 1. Count 查询总数
	// 2. 根据 PageSortReq.Sorts 追加排序
	// 3. Offset / Limit
	// 4. Find 查询当前页数据
	// 5. 写入分页信息
	if err := builder.Build(); err != nil {
		return err
	}

	// Build() 之后：rows 已经有当前页数据。
	// 这里适合做不参与 SQL 的后处理：
	// 1. 填充 gorm:"-" 的计算字段
	// 2. 填充 link 字段
	// 3. 根据关联对象补展示名称
	// 4. 隐藏或脱敏某些展示值
	for i := range rows {
		rows[i].DetailLink, _ = ctx.BuildFunctionUrlWithText(
			"demo_item_list.table",
			DemoItem{ID: rows[i].ID},
			"查看详情",
		)
	}

	return nil
}

// DemoItemListTemplate 是 Table 的 schema 声明。
var DemoItemListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name: "示例表格",
		Desc: "演示 Table 的列表、搜索、分页、增删改",
		Tags: []string{"示例"},

		// Request 是查询请求。
		Request: &DemoItemListReq{},

		// CreateTables 表示 build 时自动建表。
		// 多表场景可以放多个模型。
		CreateTables: []interface{}{&DemoItem{}},
	},

	// AutoCrudTable 是这张 Table 的主模型。
	// 前端会根据它解析列、搜索、新增/编辑表单。
	AutoCrudTable: &DemoItem{},

	// OnTableAddRow 不为 nil 时，前端显示“新增”入口。
	// 如果不配置 OnTableAddRow，前端禁止新增。
	// 收银记录、支付流水、评价提交记录等事实表通常不要配置新增。
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()

		var row DemoItem

		// Table 新增时，也用 ShouldBindValidate 绑定前端新增表单。
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}

		// 创建人由后端自动写入，不让用户手填。
		row.CreatedBy = ctx.GetRequestUser()

		if err := db.Create(&row).Error; err != nil {
			logger.Errorf(ctx, "create demo item err: %v", err)
			return nil, err
		}

		// Data 返回新增后的行数据。
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},

	// OnTableUpdateRow 不为 nil 时，前端允许编辑。
	// 如果不配置 OnTableUpdateRow，前端禁止编辑。
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()

		// BindChangedFields 会把前端提交的更新字段绑定到结构体。
		// 注意：它只包含本次变更的字段，未变更字段会是零值。
		// 适合在校验某个更新字段类型、读取更新时间字段时使用。
		var updateFields DemoItem
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, fmt.Errorf("绑定更新字段失败: %w", err)
		}

		// IsFieldUpdated 判断某个字段本次是否被改了。
		// 不要直接用 updateFields 的零值判断，因为未更新字段也是零值。
		if req.IsFieldUpdated("status") && updateFields.Status == "" {
			return nil, fmt.Errorf("状态不能为空")
		}

		// ChangedFields 返回 map[string]interface{}，
		// 只包含前端本次变更字段，适合直接配合 GORM Updates。
		updates := req.ChangedFields()

		// GetId 获取本次编辑的行 ID。
		if req.GetId() == 0 {
			return nil, fmt.Errorf("缺少记录 ID")
		}

		if err := db.Model(&DemoItem{}).
			Where("id = ?", req.GetId()).
			Updates(updates).Error; err != nil {
			logger.Errorf(ctx, "update demo item err: %v", err)
			return nil, err
		}

		return &callback.OnTableUpdateRowResp{}, nil
	},

	// OnTableDeleteRows 不为 nil 时，前端允许删除。
	// 如果不配置 OnTableDeleteRows，前端禁止删除。
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()

		// GetIds 获取前端选择删除的 ID 数组。
		ids := req.GetIds()
		if len(ids) == 0 {
			return nil, fmt.Errorf("请选择要删除的记录")
		}

		if err := db.Delete(&DemoItem{}, "id in ?", ids).Error; err != nil {
			logger.Errorf(ctx, "delete demo item err: %v", err)
			return nil, err
		}

		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

func init() {
	// Table 路由必须以 .table 结尾。
	// Table 通常用 GET，因为它主要承载列表查询。
	packageContext.GET("demo_item_list.table", DemoItemList, DemoItemListTemplate)
}
```

### 只读 Table 骨架

事实表、流水表、提交记录表通常只读。只读表不要配置新增、编辑、删除回调。

```go
var PaymentRecordListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "支付记录",
		Desc:         "支付记录只允许查询，不允许手工新增、编辑、删除",
		Tags:         []string{"支付记录"},
		Request:      &PaymentRecordListReq{},
		CreateTables: []interface{}{&PaymentRecord{}},
	},

	// AutoCrudTable 是只读表的列表 schema 来源。
	AutoCrudTable: &PaymentRecord{},

	// 不写 OnTableAddRow：前端没有新增按钮。
	// 不写 OnTableUpdateRow：前端不能编辑。
	// 不写 OnTableDeleteRows：前端不能删除。
}
```

## 五、Chart 函数骨架

Chart 适合统计展示：

- 趋势图
- 分布图
- 占比图
- 排行榜
- 仪表盘

前端渲染逻辑：

```text
Request struct -> 图表筛选条件
后端聚合数据   -> chart.LineChart / BarChart / PieChart / GaugeChart
resp.Chart     -> 前端自动用 ECharts 渲染
```

### 带注释 Chart 示例

```go
package demo

import (
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/chart"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

// DemoTrendReq 是图表筛选条件。
// 图表请求也通过 widget tag 渲染筛选表单。
type DemoTrendReq struct {
	// 时间字段必须使用 types.Time。
	// 读取时用 req.StartTime.Time() 转成 time.Time。
	StartTime types.Time `json:"start_time" form:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   types.Time `json:"end_time" form:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

// DemoTrendChart 是图表入口函数。
func DemoTrendChart(ctx *app.Context, resp response.Response) error {
	var req DemoTrendReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	startTime := req.StartTime.Time()
	endTime := req.EndTime.Time()

	// 图表通常给默认时间范围，避免空筛选导致查询过大。
	if endTime.IsZero() {
		endTime = time.Now()
	}
	if startTime.IsZero() {
		startTime = endTime.AddDate(0, 0, -30)
	}

	// 这里演示静态数据。
	// 真实项目中一般用 db.Model(...).Select(...).Where(...).Group(...).Scan(...)
	// 先聚合出 xAxis 和 seriesData。
	xAxis := []string{"05-01", "05-02", "05-03"}
	seriesData := []interface{}{12, 18, 25}

	c := &chart.LineChart{
		// Title 是图表标题。
		Title: "示例趋势图",

		// XAxis 是横轴分类，折线图/柱状图常用。
		XAxis: xAxis,

		// Series 是数据系列。
		// 一个 LineChart 可以有多条线。
		Series: []chart.ChartSeries{
			{Name: "数量", Data: seriesData},
		},

		// Metadata 是图表旁边的摘要信息。
		Metadata: map[string]interface{}{
			"开始时间": startTime.Format("2006-01-02"),
			"结束时间": endTime.Format("2006-01-02"),
			"数据点数": len(seriesData),
		},
	}

	// Chart 必须使用 resp.Chart(c).Build() 返回。
	// ChartType 和 Series.Type 由 SDK 注入，业务代码不要手动填。
	return resp.Chart(c).Build()
}

// DemoTrendChartTemplate 是图表 schema。
var DemoTrendChartTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "示例趋势图",
		Desc:     "演示 Chart 折线图",
		Tags:     []string{"示例", "统计"},
		Request:  &DemoTrendReq{},
		Response: &chart.LineChart{},
	},

	// ChartType 必须和 Response 类型匹配。
	// 可选：app.ChartTypeLine / app.ChartTypeBar / app.ChartTypePie / app.ChartTypeGauge。
	ChartType: app.ChartTypeLine,
}

func init() {
	// Chart 路由必须以 .chart 结尾。
	// Chart 通常用 GET，因为它主要是查询和展示。
	packageContext.GET("demo_trend.chart", DemoTrendChart, DemoTrendChartTemplate)
}
```

### PieChart 数据格式

饼图的 `Data` 通常是 `[]interface{}{map[string]interface{}{"name": "...", "value": ...}}`。

```go
c := &chart.PieChart{
	Title: "分类占比",
	Series: []chart.ChartSeries{
		{
			Name: "分类",
			Data: []interface{}{
				map[string]interface{}{"name": "A类", "value": 10},
				map[string]interface{}{"name": "B类", "value": 20},
				map[string]interface{}{"name": "C类", "value": 5},
			},
		},
	},
	Metadata: map[string]interface{}{
		"总数": 35,
	},
}
return resp.Chart(c).Build()
```

## 六、动态下拉 OnSelectFuzzy 骨架

当 `select` / `multiselect` 选项来自数据库时，使用 `callback:"OnSelectFuzzy"`。

```go
type DemoSubmitReq struct {
	// callback:"OnSelectFuzzy" 表示这个字段的选项由后端回调提供。
	// Template 里必须配置 OnSelectFuzzyMap，key 必须等于 json 字段名 item_id。
	ItemID int `json:"item_id" widget:"name:选择条目;type:select" validate:"required" callback:"OnSelectFuzzy"`
}

func demoOnSelectFuzzyItem(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接失败")
	}

	var rows []DemoItem

	if req.IsByValue() {
		// 前端回显单个已选值时会走这里。
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		// multiselect 回显多个已选值时会走这里。
		db = db.Where("id in ?", req.GetValues())
	} else {
		// 用户输入关键词搜索时会走这里。
		db = db.Where("name LIKE ?", "%"+req.Keyword()+"%").Limit(20)
	}

	if err := db.Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]*callback.SelectFuzzyItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &callback.SelectFuzzyItem{
			Value: row.ID,
			Label: row.Name,
			DisplayInfo: map[string]interface{}{
				"名称": row.Name,
				"状态": row.Status,
			},
		})
	}

	return &callback.OnSelectFuzzyResp{
		Items: items,
	}, nil
}

var DemoSubmitTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "示例提交",
		Request:  &DemoSubmitReq{},
		Response: &DemoSubmitResp{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			// key 必须和字段 json 名一致。
			"item_id": demoOnSelectFuzzyItem,
		},
	},
}
```

注意：

- `callback:"OnSelectFuzzy"` 只能挂在 `select` / `multiselect` 字段上。
- `OnSelectFuzzyMap` 的 key 必须对应真实字段 code。
- 如果字段只是只读外键 ID，不要写 `type:select`；用 `type:ID` 或隐藏，再额外展示关联名称。

## 七、常用 ctx 能力

```go
// 获取数据库连接。
db := ctx.GetGormDB()

// 获取当前请求用户。
// 创建人、提交人、处理人默认应该从这里取。
user := ctx.GetRequestUser()

// 获取当前请求用户所在部门 full_code_path。
dept := ctx.GetRequestUserDept()

// 构建跳转到另一个函数的 link 字段。
link, _ := ctx.BuildFunctionUrlWithText("demo_item_list.table", DemoItem{ID: 1}, "查看详情")

// 发送消息。业务需要通知时优先用 SDK 能力，不要自建消息表。
err := ctx.SendMessage(&app.SendMessageOpts{
	ToUsers:     "zhangsan,lisi",
	Title:       "处理提醒",
	Content:     "你有一条记录需要处理",
	ContentType: "markdown",
})
```

## 八、常见错误清单

- 不要写独立前端页面。
- 不要把 `.form` / `.table` / `.chart` 写进 Go 文件名。
- 不要导入 `sdk/agent-app` 根包。
- `CreatedAt` / `UpdatedAt` 用 `types.Time`。
- `types.Time` 格式化时写 `t.Time().Format(...)`，不要写 `t.Format(...)`。
- Table 筛选字段写在 Request 中；如果和 Model 业务字段语义相同，优先使用 `xxx_filter` 这类不冲突的 json/form 名称。
- `select` / `multiselect` 必须有静态 `options`，或 `callback:"OnSelectFuzzy"` + `OnSelectFuzzyMap`。
- 事实记录表默认只读，不要默认配置新增/编辑/删除回调。
- `OnTableAddRow` 不配置，前端就不能新增。
- `OnTableUpdateRow` 不配置，前端就不能编辑。
- `OnTableDeleteRows` 不配置，前端就不能删除。
- `BindChangedFields` 只绑定本次变更字段，未变更字段是零值。
- `ChangedFields` 适合直接配合 GORM `Updates`。
- 当前用户用 `ctx.GetRequestUser()`，不要让用户手填创建人/提交人。
- Chart 不要塞进 Form Response，必须用 `.chart` 路由和 `resp.Chart`。
