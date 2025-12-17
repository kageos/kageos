package widget

import "strconv"

type Number struct {
	Placeholder string `json:"placeholder,omitempty"` // 占位符文本
	Step        string `json:"step,omitempty"`        // 步长（点击增减按钮的步进值）
	Default     int    `json:"default,omitempty"`    // 默认值
	Unit        string `json:"unit,omitempty"`        // 单位（如：件、个、元、kg等）
}

func (n *Number) Config() interface{} {
	return n
}

func (n *Number) Type() string {
	return TypeNumber
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

