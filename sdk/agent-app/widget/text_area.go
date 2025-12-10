package widget

type TextArea struct {
	Placeholder string `json:"placeholder,omitempty"` // 占位符文本
	Default     string `json:"default,omitempty"`     // 默认值
}

func (t *TextArea) Config() interface{} {
	return t
}

func (t *TextArea) Type() string {
	return TypeTextArea
}

func newTextArea(widgetParsed map[string]string) *TextArea {
	textArea := &TextArea{}

	// 从widgetParsed中解析配置
	if placeholder, exists := widgetParsed["placeholder"]; exists {
		textArea.Placeholder = placeholder
	}
	if defaultValue, exists := widgetParsed["default"]; exists {
		textArea.Default = defaultValue
	}

	return textArea
}
