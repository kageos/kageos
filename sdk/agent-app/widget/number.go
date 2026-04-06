package widget

import (
	"fmt"
	"strconv"
)

type Number struct {
	Placeholder string `json:"placeholder,omitempty"` // 占位符文本
	Step        string `json:"step,omitempty"`        // 步长（点击增减按钮的步进值）
	Default     int    `json:"default,omitempty"`     // 默认值
	Unit        string `json:"unit,omitempty"`        // 单位（如：件、个、元、kg等）
}

func (n *Number) Config() interface{} {
	return n
}

func (n *Number) Type() string {
	return TypeNumber
}

func (n *Number) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := make([]SemanticFact, 0, 5)
	if fact, ok := placeholderFact(n.Placeholder); ok {
		facts = append(facts, fact)
	}
	if n.Default != 0 {
		defaultValue := fmt.Sprintf("%d", n.Default)
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: defaultValue})
		if field != nil && field.Data != nil && field.Data.Example == "" {
			facts = append(facts, SemanticFact{Key: "example", Value: defaultValue})
		}
	}
	if n.Unit != "" {
		facts = append(facts, SemanticFact{Key: "unit", Value: n.Unit})
	}
	if n.Step != "" && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "step", Value: n.Step})
	}
	return facts
}

func newNumber(widgetParsed map[string]string) *Number {
	number := &Number{}

	// 从widgetParsed中解析配置
	if placeholder, exists := widgetParsed["placeholder"]; exists {
		number.Placeholder = placeholder
	}
	if step, exists := widgetParsed["step"]; exists {
		number.Step = step
	}
	if defaultValue, exists := widgetParsed["default"]; exists {
		if val, err := strconv.Atoi(defaultValue); err == nil {
			number.Default = val
		}
	}
	if unit, exists := widgetParsed["unit"]; exists {
		number.Unit = unit
	}

	return number
}
