package widget

import (
	"fmt"
	"strconv"
)

type Float struct {
	Placeholder string  `json:"placeholder,omitempty"` // 占位符文本
	Precision   string  `json:"precision,omitempty"`   // 小数位数（显示和输入精度）
	Step        string  `json:"step,omitempty"`        // 步长（点击增减按钮的步进值）
	Default     float64 `json:"default,omitempty"`     // 默认值
	Unit        string  `json:"unit,omitempty"`        // 单位（如：元、kg、%等）
}

func (f *Float) Config() interface{} {
	return f
}

func (f *Float) Type() string {
	return TypeFloat
}

func (f *Float) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := make([]SemanticFact, 0, 6)
	if fact, ok := placeholderFact(f.Placeholder); ok {
		facts = append(facts, fact)
	}
	if f.Default != 0 {
		defaultValue := fmt.Sprintf("%v", f.Default)
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: defaultValue})
		if field != nil && field.Data != nil && field.Data.Example == "" {
			facts = append(facts, SemanticFact{Key: "example", Value: defaultValue})
		}
	}
	if f.Unit != "" {
		facts = append(facts, SemanticFact{Key: "unit", Value: f.Unit})
	}
	if f.Precision != "" && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "precision", Value: f.Precision})
	}
	if f.Step != "" && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "step", Value: f.Step})
	}
	return facts
}

func newFloat(widgetParsed map[string]string) *Float {
	floatWidget := &Float{}

	// 从widgetParsed中解析配置
	if placeholder, exists := widgetParsed["placeholder"]; exists {
		floatWidget.Placeholder = placeholder
	}
	if precision, exists := widgetParsed["precision"]; exists {
		floatWidget.Precision = precision
	}
	if step, exists := widgetParsed["step"]; exists {
		floatWidget.Step = step
	}
	if defaultValue, exists := widgetParsed["default"]; exists {
		if val, err := strconv.ParseFloat(defaultValue, 64); err == nil {
			floatWidget.Default = val
		}
	}
	if unit, exists := widgetParsed["unit"]; exists {
		floatWidget.Unit = unit
	}

	return floatWidget
}
