package widget

func init() {
	RegisterWidgetValidator(TypeText, validateTextWidget)
}

// Text 只读文本展示组件。
//
// 使用场景：
// - response 字段展示后端计算结果；
// - 表格或详情页展示 markdown/json/html 等格式化文本。
//
// 使用示例：
//
//	Summary string `json:"summary" widget:"name:摘要;type:text;format:markdown"`
//
// 校验规则：
// - 注册的是本文件的 validateTextWidget；
// - Go 字段必须是 string 或 *string；
// - format 只告诉前端如何展示，不会做内容格式校验；
// - 如果字段需要用户编辑，应优先使用 input/text_area/richtext。
type Text struct {
	Format string `json:"format,omitempty"` // json，yaml，xml，markdown，html，csv 等等
}

func (t *Text) Config() interface{} {
	return t
}

func (t *Text) Type() string {
	return TypeText
}

// validateTextWidget 校验 text 的只读文本协议。
//
// text 只展示字符串内容；format 是展示提示，不改变字段类型。
func validateTextWidget(ctx ValidateContext) error {
	return requireStringLikeGoType(ctx)
}

func newText(widgetParsed map[string]string) *Text {
	text := &Text{}

	// 从widgetParsed中解析配置
	if format, exists := widgetParsed["format"]; exists {
		text.Format = format
	}

	return text
}
