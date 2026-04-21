package widget

import "strings"

type Radio struct {
	Options       []string `json:"options,omitempty"`        // 选项列表
	RenderDefault string   `json:"render_default,omitempty"` // 前端渲染默认选中项
}

func (r *Radio) Config() interface{} {
	return r
}

func (r *Radio) Type() string {
	return TypeRadio
}

func (r *Radio) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := make([]SemanticFact, 0, 3)
	if len(r.Options) > 0 {
		facts = append(facts, SemanticFact{Key: "enum", Value: strings.Join(r.Options, "|")})
	}
	if strings.TrimSpace(r.RenderDefault) != "" {
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: r.RenderDefault})
		facts = append(facts, SemanticFact{Key: "example", Value: quoteExampleValue(r.RenderDefault)})
	} else if len(r.Options) > 0 {
		facts = append(facts, SemanticFact{Key: "example", Value: quoteExampleValue(r.Options[0])})
	}
	return facts
}

func newRadio(widgetParsed map[string]string) *Radio {
	radio := &Radio{}

	// 从widgetParsed中解析配置
	if options, exists := widgetParsed["options"]; exists {
		// 解析逗号分隔的选项
		radio.Options = parseOptions(options)
	}
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		radio.RenderDefault = defaultValue
	}

	return radio
}
