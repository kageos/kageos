package widget

import "strings"

type Select struct {
	Options     []string `json:"options"`     // 选项列表
	Placeholder string   `json:"placeholder"` // 占位符文本
	Default     string   `json:"default"`     // 默认值
	Creatable   bool     `json:"creatable"`   // 是否支持创建新选项
}

func (s *Select) Config() interface{} {
	return s
}

func (s *Select) Type() string {
	return TypeSelect
}

func newSelect(widgetParsed map[string]string) *Select {
	selectWidget := &Select{}

	// 从widgetParsed中解析配置
	if options, exists := widgetParsed["options"]; exists {
		// 解析逗号分隔的选项
		selectWidget.Options = parseOptions(options)
	}
	if placeholder, exists := widgetParsed["placeholder"]; exists {
		selectWidget.Placeholder = placeholder
	}
	if defaultValue, exists := widgetParsed["default"]; exists {
		selectWidget.Default = defaultValue
	}
	if creatable, exists := widgetParsed["creatable"]; exists {
		selectWidget.Creatable = creatable == "true"
	}

	return selectWidget
}

// parseOptions 解析选项字符串 "低,中,高" -> []string{"低", "中", "高"}
func parseOptions(optionsStr string) []string {
	if optionsStr == "" {
		return []string{}
	}

	// 简单分割，可以后续优化为更复杂的解析
	options := strings.Split(optionsStr, ",")
	var result []string
	for _, option := range options {
		option = strings.TrimSpace(option)
		if option != "" {
			result = append(result, option)
		}
	}
	return result
}
