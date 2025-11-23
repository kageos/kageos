package widget

import "strconv"

type Slider struct {
	// 基础参数
	Min     float64 `json:"min"`      // 最小值（必需）
	Max     float64 `json:"max"`      // 最大值（必需）
	Step    float64 `json:"step"`     // 步长（可选，默认1）
	Default float64 `json:"default"`  // 默认值（可选）
	Unit    string  `json:"unit"`     // 单位（可选，如：%、元、kg等）

	// 显示参数
	ShowInput     bool   `json:"show_input"`      // 是否显示输入框（可选，默认false）
	ShowStops     bool   `json:"show_stops"`      // 是否显示刻度（可选，默认false）
	ShowTooltip   bool   `json:"show_tooltip"`    // 是否显示提示（可选，默认true）
	FormatTooltip string `json:"format_tooltip"`  // 自定义提示格式（可选，如：{value}%）

	// 输出模式（进度条）参数
	ShowPercentage bool   `json:"show_percentage"` // 是否显示百分比（可选，默认true）
	Status         string `json:"status"`          // 状态颜色（可选，success/warning/danger/info）
	StrokeWidth    int    `json:"stroke_width"`    // 进度条粗细（可选，默认6）
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
		Min:           0,
		Max:           100,
		Step:          1,
		ShowTooltip:   true,
		ShowPercentage: true,
		StrokeWidth:   6,
	}

	// 从widgetParsed中解析配置
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
	if showInput, exists := widgetParsed["show_input"]; exists {
		slider.ShowInput = showInput == "true"
	}
	if showStops, exists := widgetParsed["show_stops"]; exists {
		slider.ShowStops = showStops == "true"
	}
	if showTooltip, exists := widgetParsed["show_tooltip"]; exists {
		slider.ShowTooltip = showTooltip != "false" // 默认true
	}
	if formatTooltip, exists := widgetParsed["format_tooltip"]; exists {
		slider.FormatTooltip = formatTooltip
	}
	if showPercentage, exists := widgetParsed["show_percentage"]; exists {
		slider.ShowPercentage = showPercentage != "false" // 默认true
	}
	if status, exists := widgetParsed["status"]; exists {
		slider.Status = status
	}
	if strokeWidth, exists := widgetParsed["stroke_width"]; exists {
		if val, err := strconv.Atoi(strokeWidth); err == nil {
			slider.StrokeWidth = val
		}
	}

	return slider
}

