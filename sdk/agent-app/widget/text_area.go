package widget

import "strings"

type TextArea struct {
	Placeholder string `json:"placeholder,omitempty"` // 占位符文本
	Default     string `json:"default,omitempty"`     // 默认值
}

func (t *TextArea) Config() interface{} {
	return t
}

func (t *TextArea) Type() string {
	return TypeTextArea
}

func (t *TextArea) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := make([]SemanticFact, 0, 3)
	if fact, ok := placeholderFact(t.Placeholder); ok {
		facts = append(facts, fact)
	}
	if strings.TrimSpace(t.Default) != "" {
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: t.Default})
		if field != nil && field.Data != nil && strings.TrimSpace(field.Data.Example) == "" {
			facts = append(facts, SemanticFact{Key: "example", Value: quoteExampleValue(t.Default)})
		}
	}
	return facts
}

func newTextArea(widgetParsed map[string]string) *TextArea {
	textArea := &TextArea{}

	// 从widgetParsed中解析配置
	if placeholder, exists := widgetParsed["placeholder"]; exists {
		textArea.Placeholder = placeholder
	}
	if defaultValue, exists := widgetParsed["default"]; exists {
		textArea.Default = defaultValue
	}

	return textArea
}
