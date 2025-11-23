package widget

type Radio struct {
	Options []string `json:"options"` // 选项列表
	Default string   `json:"default"` // 默认选中项
}

func (r *Radio) Config() interface{} {
	return r
}

func (r *Radio) Type() string {
	return TypeRadio
}

func newRadio(widgetParsed map[string]string) *Radio {
	radio := &Radio{}

	// 从widgetParsed中解析配置
	if options, exists := widgetParsed["options"]; exists {
		// 解析逗号分隔的选项
		radio.Options = parseOptions(options)
	}
	if defaultValue, exists := widgetParsed["default"]; exists {
		radio.Default = defaultValue
	}

	return radio
}



