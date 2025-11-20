package widget

// Text 这个参数一般在输出参数中用
type Text struct {
	Format string `json:"format"` //json，yaml，xml，markdown，html，csv 等等
}

func (t *Text) Config() interface{} {
	return t
}

func (t *Text) Type() string {
	return TypeText
}

func newText(widgetParsed map[string]string) *Text {
	text := &Text{}

	// 从widgetParsed中解析配置
	if format, exists := widgetParsed["format"]; exists {
		text.Format = format
	}

	return text
}
