# Examples And SDK Quick Start

> 状态：执行口径
> 更新时间：2026-05-17
> 负责人窗口：事项 9 / codex/examples-and-sdk-docs

The generated app model is centered on three shapes:

- `Form`: collect data or trigger an action.
- `Table`: list, filter, create, update, and delete structured records.
- `Chart`: return read-only chart data for dashboard views.

The examples below are minimal skeletons for the Go SDK in `sdk/agent-app`.

## Form

```go
package main

import "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"

type FeedbackReq struct {
	Title   string `json:"title" widget:"name:标题;type:input" validate:"required"`
	Content string `json:"content" widget:"name:内容;type:text_area" validate:"required"`
}

type FeedbackResp struct {
	Message string `json:"message" widget:"name:处理结果;type:input"`
}

var FeedbackForm = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "反馈提交",
		Request:  &FeedbackReq{},
		Response: &FeedbackResp{},
	},
}

func main() {
	if err := app.Run(); err != nil {
		panic(err)
	}
}
```

## Table

```go
package main

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
)

type TicketListReq struct {
	Title string `json:"title" form:"title" widget:"name:标题;type:input"`
	query.PageSortReq `widget:"-"`
}

type Ticket struct {
	ID       int    `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	Title    string `json:"title" gorm:"column:title" widget:"name:标题;type:input" validate:"required"`
	Priority string `json:"priority" gorm:"column:priority" widget:"name:优先级;type:select;options:低,中,高;render_default:中"`
	Status   string `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:待处理,处理中,已完成;render_default:待处理"`
}

func (Ticket) TableName() string {
	return "demo_ticket"
}

var TicketTable = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "工单表",
		CreateTables: []interface{}{&Ticket{}},
		Request:      &TicketListReq{},
		Response:     []*Ticket{},
	},
	AutoCrudTable: &Ticket{},
}

func main() {
	if err := app.Run(); err != nil {
		panic(err)
	}
}
```

## Chart

```go
package main

import "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"

type SalesChartReq struct {
	Range string `json:"range" form:"range" widget:"name:时间范围;type:select;options:本周,本月,本年;render_default:本月"`
}

type SalesChartResp struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

var SalesChart = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "销售统计",
		Request:  &SalesChartReq{},
		Response: []*SalesChartResp{},
	},
	ChartType: app.ChartTypeBar,
}

func main() {
	if err := app.Run(); err != nil {
		panic(err)
	}
}
```

## Walkthrough

1. Create a generated app directory from `template/main.go.template`.
2. Add one or more templates like the examples above.
3. Run the platform locally with [Local Development](../local-development.md).
4. Let the Agent workstation register, build, and run the app through `app-runtime`.
5. Use the frontend Service Tree to open the generated Form, Table, or Chart.

For deeper SDK behavior, see `sdk/agent-app/*_test.go`, `sdk/agent-app/TABLER_INTERFACE_EXAMPLE.md`, and `sdk/agent-app/runtime/python/README.md`.
