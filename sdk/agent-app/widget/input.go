package widget

import "strings"

type Input struct {
	Placeholder   string `json:"placeholder,omitempty"`    // 占位符文本
	Password      bool   `json:"password,omitempty"`       // 密码框
	Prepend       string `json:"prepend,omitempty"`        // 前置
	Append        string `json:"append,omitempty"`         // 后置
	RenderDefault string `json:"render_default,omitempty"` // 前端渲染默认值
}

func (i *Input) Config() interface{} {
	return i
}

func (i *Input) Type() string {
	return TypeInput
}

func (i *Input) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := make([]SemanticFact, 0, 4)
	if fact, ok := placeholderFact(i.Placeholder); ok {
		facts = append(facts, fact)
	}
	if strings.TrimSpace(i.RenderDefault) != "" {
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: i.RenderDefault})
		if field != nil && field.Data != nil && strings.TrimSpace(field.Data.Example) == "" {
			facts = append(facts, SemanticFact{Key: "example", Value: quoteExampleValue(i.RenderDefault)})
		}
	}
	if i.Password && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "password", Value: "true"})
	}
	return facts
}

func newInput(widgetParsed map[string]string) *Input {
	input := &Input{}

	// 从widgetParsed中解析配置
	if placeholder, exists := widgetParsed["placeholder"]; exists {
		input.Placeholder = placeholder
	}
	if password, exists := widgetParsed["password"]; exists {
		input.Password = password == "true"
	}
	if prepend, exists := widgetParsed["prepend"]; exists {
		input.Prepend = prepend
	}
	if append, exists := widgetParsed["append"]; exists {
		input.Append = append
	}
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		input.RenderDefault = defaultValue
	}

	return input
}
