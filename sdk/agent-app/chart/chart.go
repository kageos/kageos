package chart

// Charter 图表响应接口：GetChartType 返回类型，SetChartType 由框架调用以注入 ChartType（及 Series.Type），无需反射
type Charter interface {
	GetChartType() string
	SetChartType(typ string)
}

// ChartSeries 数据系列
type ChartSeries struct {
	// 系列名称
	Name string `json:"name"`
	// 数据点（必需）
	// - bar/line: []interface{}，如 [100, 200, 150]
	//   支持多系列：例如 3 个状态维度可返回 3 个 ChartSeries，前端会并列渲染为多根柱子或多条折线
	// - pie: []map[string]interface{}，如 [{"name": "A", "value": 100}]
	// - gauge: []interface{}，如 [75]（单值）
	Data []interface{} `json:"data"`
	// Type：由 resp.Chart() 注入，业务无需填
	Type string `json:"type,omitempty"`
	// 系列配置（可选）
	Config map[string]interface{} `json:"config,omitempty"`
}

// LineChart 折线图
// - XAxis 为横轴分类（常见为日期）
// - Series 支持单条线或多条线；多维趋势对比时，通常一个维度值对应一个 ChartSeries
type LineChart struct {
	ChartType string                 `json:"chart_type"` // 由 resp.Chart() 注入，业务无需填
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
// - XAxis 为横轴分类（常见为优先级、部门、月份等）
// - Series 支持单系列或多系列；多维对比时可返回多个 ChartSeries 做分组柱状图
type BarChart struct {
	ChartType string                 `json:"chart_type"`
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
	ChartType string                 `json:"chart_type"`
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
	ChartType string                 `json:"chart_type"`
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
