package widget

import "strings"

type Checkbox struct {
	Options       []string `json:"options,omitempty"`        // 选项列表
	RenderDefault []string `json:"render_default,omitempty"` // 前端渲染默认选中项（逗号分隔）
}

func (c *Checkbox) Config() interface{} {
	return c
}

func (c *Checkbox) Type() string {
	return TypeCheckbox
}

func (c *Checkbox) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := make([]SemanticFact, 0, 3)
	if len(c.Options) > 0 {
		facts = append(facts, SemanticFact{Key: "enum", Value: strings.Join(c.Options, "|")})
	}
	if len(c.RenderDefault) > 0 {
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: strings.Join(c.RenderDefault, "|")})
		facts = append(facts, SemanticFact{Key: "example", Value: quoteJSONArrayExample(c.RenderDefault)})
	} else if len(c.Options) > 0 {
		limit := 2
		if len(c.Options) < limit {
			limit = len(c.Options)
		}
		facts = append(facts, SemanticFact{Key: "example", Value: quoteJSONArrayExample(c.Options[:limit])})
	}
	return facts
}

func newCheckbox(widgetParsed map[string]string) *Checkbox {
	checkbox := &Checkbox{}

	// 从widgetParsed中解析配置
	if options, exists := widgetParsed["options"]; exists {
		// 解析逗号分隔的选项
		checkbox.Options = parseOptions(options)
	}
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		// 解析默认选中项（逗号分隔）
		checkbox.RenderDefault = parseOptions(defaultValue)
	}

	return checkbox
}
