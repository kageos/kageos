package widget

// DateTime 日期时间组件。
//
// raw value 是 "YYYY-MM-DD HH:mm:ss" 字符串；
// 数据库存储推荐使用 sdk/agent-app/types.Time + gorm:"type:datetime" 的真实时间列。
type DateTime struct {
	Format        string `json:"format,omitempty"`         // 日期格式，如 YYYY-MM-DD HH:mm:ss
	Disabled      bool   `json:"disabled,omitempty"`       // 是否禁用
	RenderDefault string `json:"render_default,omitempty"` // 前端渲染默认值，支持 CURRENT_TIMESTAMP、DATE_ADD 等白名单 SQL 风格表达式
}

func (t *DateTime) Config() interface{} {
	return t
}

func (t *DateTime) Type() string {
	return TypeDatetime
}

func (t *DateTime) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := []SemanticFact{
		{Key: "example", Value: `"2026-04-21 16:30:00"`},
		{Key: "storage", Value: "database datetime"},
	}
	if t.Format != "" {
		facts = append(facts, SemanticFact{Key: "display_format", Value: t.Format})
	}
	if t.RenderDefault != "" {
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: t.RenderDefault})
	}
	if t.Disabled && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "disabled", Value: "true"})
	}
	return facts
}

func newDateTime(widgetParsed map[string]string) *DateTime {
	datetime := &DateTime{}

	if format, exists := widgetParsed["format"]; exists {
		datetime.Format = format
	}
	if disabled, exists := widgetParsed["disabled"]; exists {
		datetime.Disabled = disabled == "true"
	}
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		datetime.RenderDefault = defaultValue
	}

	return datetime
}
