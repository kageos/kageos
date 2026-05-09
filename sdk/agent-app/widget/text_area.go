package widget

import (
	"strconv"
	"strings"
)

func init() {
	RegisterWidgetValidator(TypeTextArea, validateTextAreaWidget)
}

// TextArea 多行文本输入组件。
//
// 使用示例：
//
//	Description string `json:"description" widget:"name:描述;type:text_area;placeholder:请输入详细描述" validate:"required,min=10"`
//
// 校验规则：
// - 注册的是本文件的 validateTextAreaWidget；
// - Go 字段必须是 string 或 *string；
// - placeholder/render_default/rows 只影响前端渲染；
// - 多行内容的长度、必填、格式等业务规则仍然写 validate tag。
type TextArea struct {
	Placeholder   string `json:"placeholder,omitempty"`    // 占位符文本
	RenderDefault string `json:"render_default,omitempty"` // 前端渲染默认值
	Rows          int    `json:"rows,omitempty"`           // 文本域行数
}

func (t *TextArea) Config() interface{} {
	return t
}

func (t *TextArea) Type() string {
	return TypeTextArea
}

func (t *TextArea) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := make([]SemanticFact, 0, 3)
	if fact, ok := placeholderFact(t.Placeholder); ok {
		facts = append(facts, fact)
	}
	if strings.TrimSpace(t.RenderDefault) != "" {
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: t.RenderDefault})
		if field != nil && field.Data != nil && strings.TrimSpace(field.Data.Example) == "" {
			facts = append(facts, SemanticFact{Key: "example", Value: quoteExampleValue(t.RenderDefault)})
		}
	}
	return facts
}

// validateTextAreaWidget 校验 text_area 的值协议。
//
// text_area 仍然是字符串字段，多行只是前端输入形态，不改变后端 schema 类型。
func validateTextAreaWidget(ctx ValidateContext) error {
	if err := requireStringLikeGoType(ctx); err != nil {
		return err
	}
	return validatePositiveIntTag(ctx, "rows")
}

func newTextArea(widgetParsed map[string]string) *TextArea {
	textArea := &TextArea{}

	// 从widgetParsed中解析配置
	if placeholder, exists := widgetParsed["placeholder"]; exists {
		textArea.Placeholder = placeholder
	}
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		textArea.RenderDefault = defaultValue
	}
	if rows, exists := widgetParsed["rows"]; exists {
		if val, err := strconv.Atoi(rows); err == nil && val > 0 {
			textArea.Rows = val
		}
	}

	return textArea
}
