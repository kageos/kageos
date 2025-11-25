package widget

import "strconv"

type Rate struct {
	// 核心参数（必需）
	Max int `json:"max"` // 最大星级（默认5）

	// 可选参数（有合理默认值）
	AllowHalf bool     `json:"allow_half"` // 是否允许半星（默认false）
	Default   float64  `json:"default"`    // 默认评分（可选）
	Texts     []string `json:"texts"`      // 自定义文字数组（可选，如：["很差", "差", "一般", "好", "很好"]）
	// 注意：如果配置了 texts，会自动显示文字；如果没有配置 texts，则不显示文字
}

func (r *Rate) Config() interface{} {
	return r
}

func (r *Rate) Type() string {
	return TypeRate
}

func newRate(widgetParsed map[string]string) *Rate {
	rate := &Rate{
		// 默认值
		Max:       5,
		AllowHalf: false,
	}

	// 从widgetParsed中解析配置
	if max, exists := widgetParsed["max"]; exists {
		if val, err := strconv.Atoi(max); err == nil && val > 0 {
			rate.Max = val
		}
	}
	if allowHalf, exists := widgetParsed["allow_half"]; exists {
		rate.AllowHalf = allowHalf == "true"
	}
	if defaultValue, exists := widgetParsed["default"]; exists {
		if val, err := strconv.ParseFloat(defaultValue, 64); err == nil {
			rate.Default = val
		}
	}
	if texts, exists := widgetParsed["texts"]; exists {
		// 解析逗号分隔的文字数组
		rate.Texts = parseOptions(texts)
	}

	return rate
}

