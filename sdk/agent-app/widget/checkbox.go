package widget

type Checkbox struct {
	Options []string `json:"options,omitempty"` // 选项列表
	Default []string `json:"default,omitempty"` // 默认选中项（逗号分隔）
}

func (c *Checkbox) Config() interface{} {
	return c
}

func (c *Checkbox) Type() string {
	return TypeCheckbox
}

func newCheckbox(widgetParsed map[string]string) *Checkbox {
	checkbox := &Checkbox{}

	// 从widgetParsed中解析配置
	if options, exists := widgetParsed["options"]; exists {
		// 解析逗号分隔的选项
		checkbox.Options = parseOptions(options)
	}
	if defaultValue, exists := widgetParsed["default"]; exists {
		// 解析默认选中项（逗号分隔）
		checkbox.Default = parseOptions(defaultValue)
	}

	return checkbox
}



