package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Charter 图表响应接口：GetChartType 返回类型，SetChartType 由框架调用以注入 ChartType（及 Series.Type），无需反射
type Charter interface {
	GetChartType() string
	SetChartType(typ string)
}

// Chart 图表数据结构（统一标准，支持所有图表类型）
// 实现 Charter：可直接 resp.Chart(chart)，ChartType 由业务填写
type Chart struct {
	// ChartType：必填，且代表图表类型。请使用 app 包常量（如 app.ChartTypeBar、app.ChartTypeLine、app.ChartTypePie、app.ChartTypeGauge），勿写死字符串
	ChartType string `json:"chart_type"`

	// 图表标题
	Title string `json:"title,omitempty"`

	// X 轴数据（可选，某些图表类型不需要）
	// 用于 bar、line 等需要 X 轴的图表
	XAxis []string `json:"x_axis,omitempty"`

	// 数据系列（必需）
	// 所有图表类型都使用 Series 来存储数据
	Series []ChartSeries `json:"series"`

	// 元数据（可选，用于扩展）
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// 标识字段（用于类型识别，类似 string 的 WidgetType）
	WidgetType string `json:"widget_type,omitempty"` // 固定为 "chart"
	DataType   string `json:"data_type,omitempty"`   // 固定为 "chart"
}

// ChartSeries 数据系列
type ChartSeries struct {
	// 系列名称
	Name string `json:"name"`

	// 数据点（必需）
	// 不同类型图表的数据格式：
	// - bar/line: []interface{}，如 [100, 200, 150]
	// - pie: []map[string]interface{}，如 [{"name": "A", "value": 100}, {"name": "B", "value": 200}]
	// - gauge: []interface{}，如 [75]（单个数值，表示百分比）
	Data []interface{} `json:"data"`

	// Type：必填，且代表该系列的图表类型。请使用 app 包常量（如 app.ChartTypeBar、app.ChartTypePie）；混合图时每系列可不同
	Type string `json:"type,omitempty"`

	// 系列配置（可选，用于单个系列的样式配置）
	Config map[string]interface{} `json:"config,omitempty"`
}

// GetChartType 实现 Charter，返回图表类型
func (c *Chart) GetChartType() string {
	return c.ChartType
}

// SetChartType 实现 Charter，由框架调用以注入 ChartType 和 Series[].Type
func (c *Chart) SetChartType(typ string) {
	c.ChartType = typ
	for i := range c.Series {
		c.Series[i].Type = typ
	}
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

// ========== 具体图表类型（实现 Charter，resp.Chart() 时调用 SetChartType 注入 ChartType / Series.Type） ==========
// 各类型数据结构可完全独立，后续新增类型只需实现 GetChartType + SetChartType 即可

// LineChart 折线图
type LineChart struct {
	ChartType string                 `json:"chart_type"` // 由 resp.Chart() 调用 SetChartType 注入，业务无需填
	Title     string                 `json:"title,omitempty"`
	XAxis     []string               `json:"x_axis,omitempty"`
	Series    []ChartSeries          `json:"series"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func (c *LineChart) GetChartType() string { return "line" }
func (c *LineChart) SetChartType(typ string) {
	c.ChartType = typ
	for i := range c.Series {
		c.Series[i].Type = typ
	}
}

// BarChart 柱状图
type BarChart struct {
	ChartType string                 `json:"chart_type"` // 由 resp.Chart() 调用 SetChartType 注入，业务无需填
	Title     string                 `json:"title,omitempty"`
	XAxis     []string               `json:"x_axis,omitempty"`
	Series    []ChartSeries          `json:"series"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func (c *BarChart) GetChartType() string { return "bar" }
func (c *BarChart) SetChartType(typ string) {
	c.ChartType = typ
	for i := range c.Series {
		c.Series[i].Type = typ
	}
}

// PieChart 饼图
type PieChart struct {
	ChartType string                 `json:"chart_type"` // 由 resp.Chart() 调用 SetChartType 注入，业务无需填
	Title     string                 `json:"title,omitempty"`
	Series    []ChartSeries          `json:"series"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func (c *PieChart) GetChartType() string { return "pie" }
func (c *PieChart) SetChartType(typ string) {
	c.ChartType = typ
	for i := range c.Series {
		c.Series[i].Type = typ
	}
}

// GaugeChart 仪表盘
type GaugeChart struct {
	ChartType string                 `json:"chart_type"` // 由 resp.Chart() 调用 SetChartType 注入，业务无需填
	Title     string                 `json:"title,omitempty"`
	Series    []ChartSeries          `json:"series"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func (c *GaugeChart) GetChartType() string { return "gauge" }
func (c *GaugeChart) SetChartType(typ string) {
	c.ChartType = typ
	for i := range c.Series {
		c.Series[i].Type = typ
	}
}
