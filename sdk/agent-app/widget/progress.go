package widget

import "strconv"

type Progress struct {
	Min  float64 `json:"min,omitempty"`  // 最小值，默认 0
	Max  float64 `json:"max,omitempty"`  // 最大值，默认 100
	Unit string  `json:"unit,omitempty"` // 单位（如：%、人、次等），默认 %
}

func (p *Progress) Config() interface{} {
	return p
}

func (p *Progress) Type() string {
	return TypeProgress
}

func newProgress(widgetParsed map[string]string) *Progress {
	progress := &Progress{
		Min:  0,
		Max:  100,
		Unit: "%",
	}

	// 从widgetParsed中解析配置
	if min, exists := widgetParsed["min"]; exists {
		if val, err := parseFloat(min); err == nil {
			progress.Min = val
		}
	}
	if max, exists := widgetParsed["max"]; exists {
		if val, err := parseFloat(max); err == nil {
			progress.Max = val
		}
	}
	if unit, exists := widgetParsed["unit"]; exists {
		progress.Unit = unit
	}

	return progress
}

// parseFloat 解析浮点数
func parseFloat(s string) (float64, error) {
	// 这里可以后续优化为更复杂的解析
	// 暂时使用简单的 strconv.ParseFloat
	return strconv.ParseFloat(s, 64)
}

