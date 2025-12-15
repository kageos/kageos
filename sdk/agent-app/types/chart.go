package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Chart 图表数据结构（统一标准，支持所有图表类型）
type Chart struct {
	// 图表类型（必需）
	// 可选值：bar（柱状图）、line（折线图）、pie（饼图）、gauge（仪表盘）、scatter（散点图）、area（面积图）
	ChartType string `json:"chart_type"`

	// 图表标题
	Title string `json:"title,omitempty"`

	// X 轴数据（可选，某些图表类型不需要）
	// 用于 bar、line、scatter、area 等需要 X 轴的图表
	XAxis []string `json:"x_axis,omitempty"`

	// 数据系列（必需）
	// 所有图表类型都使用 Series 来存储数据
	Series []ChartSeries `json:"series"`

	// ECharts 配置（可选，用于高级定制）
	// 可以覆盖默认配置，实现更复杂的图表效果
	EChartsConfig map[string]interface{} `json:"echarts_config,omitempty"`

	// 元数据（可选，用于扩展）
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// 标识字段（用于类型识别，类似 types.Files 的 WidgetType）
	WidgetType string `json:"widget_type,omitempty"` // 固定为 "chart"
	DataType   string `json:"data_type,omitempty"`   // 固定为 "chart"
}

// ChartSeries 数据系列
type ChartSeries struct {
	// 系列名称
	Name string `json:"name"`

	// 数据点（必需）
	// 不同类型图表的数据格式：
	// - bar/line/area: []interface{}，如 [100, 200, 150]
	// - pie: []map[string]interface{}，如 [{"name": "A", "value": 100}, {"name": "B", "value": 200}]
	// - gauge: []interface{}，如 [75]（单个数值，表示百分比）
	// - scatter: []interface{}，如 [[10, 20], [15, 25]]（二维数组，表示坐标点）
	Data []interface{} `json:"data"`

	// 系列类型（可选，默认使用 ChartType）
	// 用于混合图表（如折线图+柱状图组合）
	Type string `json:"type,omitempty"`

	// 系列配置（可选，用于单个系列的样式配置）
	Config map[string]interface{} `json:"config,omitempty"`
}

// GetChartType 获取图表类型
func (c *Chart) GetChartType() string {
	return c.ChartType
}

// GetSeries 获取数据系列
func (c *Chart) GetSeries() []ChartSeries {
	return c.Series
}

// GetXAxis 获取 X 轴数据
func (c *Chart) GetXAxis() []string {
	return c.XAxis
}

// Scan 实现 sql.Scanner 接口，用于从数据库读取
func (c *Chart) Scan(value interface{}) error {
	if value == nil {
		*c = Chart{}
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return fmt.Errorf("cannot scan %T into Chart", value)
	}

	return json.Unmarshal(data, c)
}

// Value 实现 driver.Valuer 接口，用于存储到数据库
func (c Chart) Value() (driver.Value, error) {
	// 设置标识字段
	c.WidgetType = "chart"
	c.DataType = "chart"
	return json.Marshal(c)
}

