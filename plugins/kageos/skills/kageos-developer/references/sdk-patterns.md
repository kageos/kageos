# kageos SDK 开发模式

## packageContext

每个目录包应有 `init_.go`：

```go
package sample

import "github.com/kageos/kageos-sdk/agent-app/app"

var packageContext = &app.PackageContext{
	RouterGroup: "/sample",
	Name:        "示例目录",
	Desc:        "示例业务能力集合。",
}
```

嵌套包：

```go
package child

import "github.com/kageos/kageos-sdk/agent-app/app"

var packageContext = &app.PackageContext{
	RouterGroup: "/sample/child",
	Name:        "子目录",
	Desc:        "示例子目录能力。",
}
```

落地时必须把 `sample`、`child`、`RouterGroup`、`Name`、`Desc` 替换成目标业务目录。

只有子目录本身是独立业务 package、拥有函数或独立生命周期时才使用嵌套 `PackageContext`。单纯创建 `docs/readme.docs` 不属于这种情况，应继续使用原 package 的 `AddDocs` 相对多级路径。

## 默认文档种子

需要让 package 在首次 build/update 后自带 `runbook.docs` 或其他默认文档时，在该 package 下新建 `kageos_manifest.go`，只使用 `packageContext.AddDocs(...)`：

```go
package sample

import "github.com/kageos/kageos-sdk/agent-app/app"

func init() {
	packageContext.AddDocs(app.DocManifest{
		Code:    "runbook.docs",
		Name:    "运行手册",
		Content: `# 运行手册

这里写本 package 的默认运行说明。
`,
		Format: "markdown",
	})
}
```

约定：

- 文件名固定为 `kageos_manifest.go`；不要使用 `_init_data.go`，Go 不会编译 `_*.go` 文件。
- 不要使用 `kageos_seed.go`、`AddRunbook`、`AddDoc` 或其他别名；runbook 只是 `Code: "runbook.docs"` 的普通 `DocManifest`。
- 新代码的 `Code` 统一带 `.docs` 后缀；SDK 会规范化后参与目录对账。
- 平台只在 docs 不存在时创建，不覆盖树上已有文档；后续运行态权威内容以 service tree docs 为准。
- `kageos_manifest.go` 是本地开发种子声明文件，工作台目录读取和能力包导出会隐藏它。

同一 package 的子目录文档仍挂在原 `packageContext`：

```go
packageContext.AddDocs(app.DocManifest{
	Code:    "./docs/readme.docs",
	Name:    "文档/目录说明",
	Content: docsReadme,
	Format:  "markdown",
})
```

不要为 docs 创建 Go 子包或第二个 `PackageContext`。SDK 会自动补齐中间目录并随 package 分发。

## 默认 Agent 任务种子

需要让 package 在首次 build/update 后自带一条可选无人值守 Agent 定时会话时，也放在 `kageos_manifest.go`：

```go
package sample

import "github.com/kageos/kageos-sdk/agent-app/app"

func init() {
	packageContext.AddAgentTask(app.AgentTask{
		Code:               "sample_daily_report",
		Title:              "每日复盘报告",
		Description:        "每天读取业务数据并生成复盘报告。",
		Message:            sampleDailyReportRunbook,
		CronExpr:           "0 8 * * *",
		Timezone:           "Asia/Shanghai",
		Enabled:            false,
		MaxDurationSeconds: 900,
	})
}
```

约定：

- 不要填写 `Policy`；默认就是 `create_if_missing`。
- 平台只在任务不存在时创建；如果任务已经存在，不覆盖用户后续修改过的 message、cron、启停状态、模型配置或附件配置。
- `Code` 必须稳定且在当前 package 内唯一，用于幂等识别。
- `CronExpr` 和 `EverySeconds` 必须二选一；国内业务建议显式写 `Timezone: "Asia/Shanghai"`。
- Agent 会话成本高于固定定时函数，出厂模板通常 `Enabled: false`，让用户在平台里选择性开启。
- `Message` 和 runbook 里引用同 package 资源时必须写 `<./xxx.table>`、`<./xxx.form>`、`<./xxx.chart>`、`<./runbook.docs>` 这类带 `./` 的尖括号资源引用，不要写 `./xxx.table` 或 `<xxx.table>`。跨 package 或跨 workspace 资源才写绝对工作台路径并说明原因。引用内置 Agent 工具时写 `<tool:send_notification>`，不要写 `<send_notification>`；真实工具调用名仍是 `send_notification`。

## main.go import

新增 package 后必须更新 `code/cmd/app/main.go`：

```go
import (
	"github.com/kageos/kageos-sdk/agent-app/app"
	_ "github.com/kageos/kageos/namespace/<user>/<app>/code/api/<package_path>"
)
```

如果忘记 import，Go 会编译成功但 `init()` 不执行，函数不会注册，目录也不会上报。

## Table 基本形状

```go
type Item struct {
	ID     int    `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	Name   string `json:"name" gorm:"column:name" widget:"name:名称;type:input" validate:"required"`
	Status string `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:待处理,进行中,完成;render_default:待处理"`
}

func (Item) TableName() string { return "sample_item" }
```

Table template 常用字段：

- `Request`: 查询请求结构体
- `Response`: Table 通常不填；Table schema 由 `AutoCrudTable` 解码，列表运行时响应使用 `response.TableResult`
- `CreateTables`: 当前函数真实依赖的自动迁移表，优先直接写最小 `[]interface{}{...}`，不要用“全包所有表”的共享 helper
- `AutoCrudTable`: 工作台列 schema 和自动 CRUD 的目标结构体；可编辑表必须显式设置，不要依赖 SDK 从 `CreateTables` 第一个表推断
- `OnTableAddRow` / `OnTableUpdateRow` / `OnTableDeleteRows`: 自定义写入逻辑

不要给 Table 写 `Response: &[]Model{}`。这不会成为表格列 schema，只会误导后续维护者以为列表响应是裸数组。列 schema 看 `AutoCrudTable`，分页协议看 Handler 里的 `resp.Table(response.TableResult{...})`。

如果列表展示结构和真实存储表不同，显式把 `AutoCrudTable` 指向工作台应该渲染的行结构，并用回调把写入落到真实表。`CreateTables` 只负责迁移真实表，不能承担“顺便告诉前端 schema”的职责。

列表 Handler 推荐显式分页：

```go
return resp.Table(response.TableResult{
	Items:      rows,
	TotalCount: total,
	PageInfo:   &req.PageSortReq,
}).Build()
```

## Table 回调优先

新增、编辑、状态流转优先放在 Table 回调里，不要默认拆成 Form：

- 新增记录：用 `OnTableAddRow`。例如 AI 写入草稿，本质是给草稿表新增一行，回调里补账号、负责人、默认 CTA、状态。
- 编辑记录：用 `OnTableUpdateRow`。例如预览审核通过/退回/废弃，本质是更新预览状态和备注，回调里同步草稿状态。
- 删除记录：用 `OnTableDeleteRows`，按业务校验是否允许删除；删除动作默认用 `UPDATE deleted_at/deleted_by` 软删除，不要直接 `db.Delete`。
- 只有生成文件、调用外部系统、发布包生成、复盘报告这类“不是在编辑一行数据”的动作才单独建 Form。
- 如果平台操作日志已经能记录谁改了状态，不要再默认建审核记录表；除非业务要独立报表、导出或外部审计。

列表排序不要直接信任前端字段。对带 `gorm:"-"` 的展示字段或别名字段，使用白名单或映射；否则前端按 `preview_id`、`account_name`、`topic_title` 排序时会生成不存在的 `ORDER BY` 列。

完整模式见 `references/examples/table-crud-case.md`。

## 外键选择与名称展示

涉及外键时，用户选择字段显示业务名，后端仍提交 ID。列表和响应不要只展示 ID，要补出名称字段：

```go
type Booking struct {
	ID       int          `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	RoomID   int          `json:"room_id" gorm:"column:room_id;index" widget:"name:会议室;type:select" callback:"OnSelectFuzzy"`
	Room     *MeetingRoom `json:"-" gorm:"foreignKey:RoomID;references:ID" widget:"-"`
	RoomName string       `json:"room_name" gorm:"-" widget:"name:会议室名称;type:text" hide:"create,update"`
}
```

查询列表时用 `Preload("Room")` 读取关联对象，再把 `RoomName` 写回；主工作台里内部 ID 字段可用 `widget:"-"` 隐藏，动作链接用 `ctx.BuildFunctionUrlWithText(...)` 携带 ID。

工作空间应用默认可以用 `foreignKey + Preload` 表达 GORM 关联；SDK 的 app DB 连接启用 `DisableForeignKeyConstraintWhenMigrating`，因此 `AutoMigrate` 不会创建数据库外键约束。关联对象只服务查询，不要暴露到 JSON、表格列或表单字段，展示字段仍用 `gorm:"-"` 单独承载。

`OnSelectFuzzy` 要同时返回清晰 label、可展示详情和统计字段：

```go
items = append(items, &callback.SelectFuzzyItem{
	Value: room.ID,
	Label: fmt.Sprintf("%s - %s", room.Name, room.Location),
	DisplayInfo: map[string]interface{}{
		"会议室名称": room.Name,
		"位置":    room.Location,
	},
})
return &callback.OnSelectFuzzyResp{
	MaxSelections: 1,
	Items: items,
	Statistics: map[string]interface{}{
		"选中会议室": statistics.Value("会议室名称"),
		"位置":    statistics.Value("位置"),
	},
}, nil
```

## Form 基本形状

```go
type SubmitReq struct {
	Title string `json:"title" widget:"name:标题;type:input" validate:"required"`
}

type SubmitResp struct {
	Message string `json:"message" widget:"name:结果;type:text"`
}

func Submit(ctx *app.Context, resp response.Response) error {
	var req SubmitReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	return resp.Form(&SubmitResp{Message: "ok"}).Build()
}

func init() {
	packageContext.POST("submit.form", Submit, &app.FormTemplate{
		BaseConfig: app.BaseConfig{
			Name:     "提交",
			Request:  &SubmitReq{},
			Response: &SubmitResp{},
		},
	})
}
```

## 工作台可见说明和默认值

- 需要给操作员看的配置流程、外部控制台链接、字段填写说明，写进 `BaseConfig.Desc`；Markdown 可用于链接和分段。Go 注释不会出现在工作台。
- 字段级提示优先用 `widget` 的 `placeholder`、`desc`、`options` 等能力表达。
- `render_default` 是前端默认渲染值，不是后端强制默认值。不要在 `ShouldBind` 之后无条件覆盖字段，否则用户提交的 `false`、空字符串或自定义值会被吃掉。

## Chart 基本形状

优先使用 SDK 的 `chart.LineChart`、`chart.BarChart`、`chart.PieChart` 等结构，不要手写前端图表。完整模式见 `references/examples/chart-case.md`。

时间趋势图优先复用 SDK 粒度能力，不要在业务里到处手写按天/按小时判断：

- 默认窗口宜短，监控、价格、行情、实时质量等波动型图表优先默认最近 1 天。
- 前端可传聚合粒度：自动、按分钟、按5分钟、按小时、按天、按月；业务映射到 `app.TimeBucketAuto/Minute/5Minute/Hour/Day/Month`。
- `app.ResolveChartBucket` 默认只推荐和估算，不硬拦细粒度；只有显式传 `MaxValues` 时才自动放粗，适合默认总览保护前端。
- 查询分桶使用 `app.DateTimeBucketExpr(db, column, decision.Bucket)`，返回 metadata 时合并 `app.ChartBucketMetadata(decision)`。
- `ChartBucketPolicy` 字段要写清楚：`Requested` 是前端请求粒度，`WindowStart/WindowEnd` 是真实查询窗口，`SeriesCount` 是返回系列数而不是数据库行数，`MaxValues` 是可选保护预算且默认不填。

`YAxis` 是可选配置，不是所有图表都要加。只有需要覆盖默认数字展示时才按需声明；普通数量图可以不写 `YAxis`，默认相当于 `chart.ValueFormatCompact`。需要声明坐标轴数值格式时，不要让前端或 AI 从系列名、元数据里猜：

```go
return resp.Chart(&chart.LineChart{
	Title: "接口耗时趋势",
	XAxis: xAxis,
	YAxis: &chart.AxisConfig{
		// ValueFormat 控制 Y 轴标签、十字准星标签和 tooltip 的数值显示。
		// 不填 YAxis 时保持默认展示，相当于 chart.ValueFormatCompact。
		// 可选值：
		// - chart.ValueFormatCompact：默认大数字缩写，如 1200 显示为 1.2K
		// - chart.ValueFormatPlain：普通数字，不做 K/M 缩写
		// - chart.ValueFormatDurationMS：数据原始单位是毫秒，前端显示为 ms/s/min
		// - chart.ValueFormatPercent：百分比数字，前端追加 %
		ValueFormat: chart.ValueFormatDurationMS,
	},
	Series: []chart.ChartSeries{
		// Data 保持业务原始单位；这里 1200 表示 1200ms，不要预先换算成秒。
		{Name: "平均耗时", Data: []interface{}{1200, 860, 1530}},
	},
}).Build()
```

## 常用 Context 能力

- `ctx.GetGormDB()`: 当前 package 对应业务数据库。
- `ctx.GetRequestUser()`: 当前用户。
- `ctx.SendNotification(...)`: 发平台通知，业务代码不要直接写 message-server 表，也不要直连具体外部渠道。附件用 `Files` 传平台文件引用 `bucket/object_key`，可直接使用 `ResponseFiles`、工具 `output_files` 或文件组件值；站内信和移动处理页展示完整附件，外部 webhook 卡片只展示摘要并回到 kageos 详情查看，不做渠道原生文件上传。
- `ctx.BuildFunctionUrlWithText(...)`: 生成跳转到其他函数的链接。
- `ctx.GetFS()`: 文件下载、输出和响应文件引用。

## 命名建议

- 文件名用下划线：`item_list.go`、`item_submit.go`。
- 表名带目录前缀：`sample_item`、`sample_event`。
- 函数路由带类型后缀：`item_list.table`、`item_submit.form`、`item_trend.chart`。
