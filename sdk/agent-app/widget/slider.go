package widget

import "strconv"

type Slider struct {
	// 核心参数（必需）
	Min float64 `json:"min"` // 最小值（必需）
	Max float64 `json:"max"` // 最大值（必需）

	// 可选参数（有合理默认值）
	Step    float64 `json:"step"`    // 步长（可选，默认1）
	Default float64 `json:"default"` // 默认值（可选）
	Unit    string  `json:"unit"`    // 单位（可选，如：%、元、kg等）

	// 注意：以下参数都有合理的默认值，前端自动处理，不需要配置
	// - show_input: 默认 false（简单场景不需要输入框）
	// - show_stops: 默认 false（简单场景不需要刻度）
	// - show_tooltip: 默认 true（拖动时显示提示）
	// - show_percentage: 输出模式默认 true（进度条显示百分比）
	// - status: 根据值自动判断（>80% success, 50-80% warning, <50% danger）
	// - stroke_width: 默认 6（进度条粗细）
}

func (s *Slider) Config() interface{} {
	return s
}

func (s *Slider) Type() string {
	return TypeSlider
}

func newSlider(widgetParsed map[string]string) *Slider {
	slider := &Slider{
		// 默认值
		Min:  0,
		Max:  100,
		Step: 1,
	}

	// 从widgetParsed中解析配置（只解析核心参数）
	if min, exists := widgetParsed["min"]; exists {
		if val, err := strconv.ParseFloat(min, 64); err == nil {
			slider.Min = val
		}
	}
	if max, exists := widgetParsed["max"]; exists {
		if val, err := strconv.ParseFloat(max, 64); err == nil {
			slider.Max = val
		}
	}
	if step, exists := widgetParsed["step"]; exists {
		if val, err := strconv.ParseFloat(step, 64); err == nil {
			slider.Step = val
		}
	}
	if defaultValue, exists := widgetParsed["default"]; exists {
		if val, err := strconv.ParseFloat(defaultValue, 64); err == nil {
			slider.Default = val
		}
	}
	if unit, exists := widgetParsed["unit"]; exists {
		slider.Unit = unit
	}

	return slider
}

