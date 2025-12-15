package app

type ChartTemplate struct {
	BaseConfig
	// 注意：ChartTemplate 不需要回调函数（OnTableAddRow 等）
	// 因为 BI 图表是只读的，不需要增删改操作
}

func (t *ChartTemplate) GetBaseConfig() *BaseConfig {
	return &t.BaseConfig
}

func (t *ChartTemplate) TemplateType() TemplateType {
	return TemplateTypeChart
}

