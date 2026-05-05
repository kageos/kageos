package app

// 图表类型常量（与 chart 包图表类型、前端 ChartRenderer 约定一致）
// 生成/手写图表代码时请使用以下常量，勿写死字符串
const (
	ChartTypeBar   = "bar"   // 柱状图
	ChartTypeLine  = "line"  // 折线图
	ChartTypePie   = "pie"   // 饼图
	ChartTypeGauge = "gauge" // 仪表盘
)

type ChartTemplate struct {
	BaseConfig
	// ChartType 标识该图表接口返回的图表类型，请使用本包常量（ChartTypeBar、ChartTypeLine 等）
	ChartType string `json:"chart_type,omitempty"`
	// 注意：ChartTemplate 不需要回调函数（OnTableAddRow 等）
	// 因为 BI 图表是只读的，不需要增删改操作
}

func (t *ChartTemplate) GetBaseConfig() *BaseConfig {
	return &t.BaseConfig
}

func (t *ChartTemplate) TemplateType() TemplateType {
	return TemplateTypeChart
}
