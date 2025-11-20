package widget

import "strconv"

type Float struct {
	Precision string  `json:"precision"` // 小数位数（显示和输入精度）
	Step      string  `json:"step"`      // 步长（点击增减按钮的步进值）
	Default   float64 `json:"default"`  // 默认值
	Unit      string  `json:"unit"`     // 单位（如：元、kg、%等）
}

func (f *Float) Config() interface{} {
	return f
}

func (f *Float) Type() string {
	return TypeFloat
}

func newFloat(widgetParsed map[string]string) *Float {
	floatWidget := &Float{}

	// 从widgetParsed中解析配置
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


