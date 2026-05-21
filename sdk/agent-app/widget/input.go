package widget

import (
	"errors"
	"strings"
)

func init() {
	RegisterWidgetValidator(TypeInput, validateInputWidget)
}

// Input 单行文本输入组件。
//
// 使用示例：
//
//	Title string `json:"title" widget:"name:标题;type:input;placeholder:请输入标题" validate:"required,min=2"`
//
// 校验规则：
// - 注册的是本文件的 validateInputWidget；
// - Go 字段必须是 string 或 *string；
// - 业务 schema 不支持 password 输入；密钥/密码类数据应作为普通业务值自行处理，或后续接入 Secrets 能力；
// - placeholder/prepend/append/render_default 都是前端渲染配置，不改变 Go 类型；
// - 必填、长度、格式等业务约束应写在 validate tag，例如 validate:"required,email"。
type Input struct {
	Placeholder   string `json:"placeholder,omitempty"`    // 占位符文本
	Password      bool   `json:"password,omitempty"`       // 密码框
	Prepend       string `json:"prepend,omitempty"`        // 前置
	Append        string `json:"append,omitempty"`         // 后置
	RenderDefault string `json:"render_default,omitempty"` // 前端渲染默认值
}

func (i *Input) Config() interface{} {
	return i
}

func (i *Input) Type() string {
	return TypeInput
}

func (i *Input) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := make([]SemanticFact, 0, 4)
	if fact, ok := placeholderFact(i.Placeholder); ok {
		facts = append(facts, fact)
	}
	if strings.TrimSpace(i.RenderDefault) != "" {
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: i.RenderDefault})
		if field != nil && field.Data != nil && strings.TrimSpace(field.Data.Example) == "" {
			facts = append(facts, SemanticFact{Key: "example", Value: quoteExampleValue(i.RenderDefault)})
		}
	}
	if i.Password && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "password", Value: "true"})
	}
	return facts
}

// validateInputWidget 校验 input 的值协议。
//
// input 是单行文本输入，底层复用 requireStringLikeGoType，所有业务格式限制都应通过 validate tag 表达。
func validateInputWidget(ctx ValidateContext) error {
	var errs []error
	if err := requireStringLikeGoType(ctx); err != nil {
		errs = append(errs, err)
	}
	if strings.TrimSpace(ctx.Field.WidgetParsed["password"]) != "" {
		errs = append(errs, fieldError(ctx, "input widget does not support password; use a normal input and handle encryption in application code if needed"))
	}
	return errors.Join(errs...)
}

func newInput(widgetParsed map[string]string) *Input {
	input := &Input{}

	// 从widgetParsed中解析配置
	if placeholder, exists := widgetParsed["placeholder"]; exists {
		input.Placeholder = placeholder
	}
	if password, exists := widgetParsed["password"]; exists {
		input.Password = password == "true"
	}
	if prepend, exists := widgetParsed["prepend"]; exists {
		input.Prepend = prepend
	}
	if append, exists := widgetParsed["append"]; exists {
		input.Append = append
	}
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		input.RenderDefault = defaultValue
	}

	return input
}
