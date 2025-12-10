package widget

type MultiSelect struct {
	Options       []string `json:"options,omitempty"`        // 选项列表
	OptionsColors []string `json:"options_colors,omitempty"` // 选项的颜色，支持warning，info，success，danger，primary 还支持自定义颜色例如：#FF9800 橙色，#9C27B0 紫色，每个颜色都可以可以重复
	Placeholder   string   `json:"placeholder,omitempty"`    // 占位符文本
	Default       []string `json:"default,omitempty"`        // 默认选中的值（多个，逗号分隔）
	MaxCount      int      `json:"max_count,omitempty"`      // 最大选择数量，0表示不限制
	Creatable     bool     `json:"creatable,omitempty"`      // 是否支持创建新选项
}

func (m *MultiSelect) Config() interface{} {
	return m
}

func (m *MultiSelect) Type() string {
	return TypeMultiSelect
}

func newMultiSelect(widgetParsed map[string]string) *MultiSelect {
	multiSelect := &MultiSelect{}

	// 从widgetParsed中解析配置
	if options, exists := widgetParsed["options"]; exists {
		// 解析逗号分隔的选项
		multiSelect.Options = parseOptions(options)
	}
	if placeholder, exists := widgetParsed["placeholder"]; exists {
		multiSelect.Placeholder = placeholder
	}
	if defaultValue, exists := widgetParsed["default"]; exists {
		// 解析默认值，支持多个值用逗号分隔
		if defaultValue != "" {
			multiSelect.Default = parseOptions(defaultValue)
		}
	}
	if maxCount, exists := widgetParsed["max_count"]; exists {
		// 解析最大选择数量，支持 "0" 或 "" 表示不限制
		if maxCount == "0" || maxCount == "" {
			multiSelect.MaxCount = 0 // 0表示不限制
		}
		// 注意：如果需要在 widget 标签中设置具体数值，前端会解析字符串
		// Go 端只需要知道是否有限制即可，具体数值由前端处理
	}
	if creatable, exists := widgetParsed["creatable"]; exists {
		multiSelect.Creatable = creatable == "true"
	}
	if optionsColors, exists := widgetParsed["options_colors"]; exists {
		// 解析逗号分隔的颜色选项
		multiSelect.OptionsColors = parseOptions(optionsColors)
	}

	return multiSelect
}
