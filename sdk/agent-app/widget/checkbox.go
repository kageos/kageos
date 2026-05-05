package widget

import (
	"errors"
	"reflect"
	"strings"
)

func init() {
	RegisterWidgetValidator(TypeCheckbox, validateCheckboxWidget)
}

// Checkbox 复选组件。
//
// 使用示例：
//
//	Agree bool `json:"agree" widget:"name:同意协议;type:checkbox"`
//	Tags []string `json:"tags" widget:"name:标签;type:checkbox;options:前端,后端;render_default:前端"`
//
// 值形态：
// - bool：适合单个是否勾选；
// - string：适合用逗号分隔保存多个选项；
// - slice/array：适合接口结构中的多选值。
//
// 校验规则：
// - Go 字段必须是 bool/string/slice/array；
// - slice/array 元素必须是标量；
// - 当前启动期不强制 options 非空，因此 bool checkbox 可以不配置 options；
// - 如果配置了 options，options 不能重复，render_default 必须来自 options；
// - 如果配置 validate:"oneof=..."，静态 options 必须与 oneof 集合完全一致，包含 validate:"dive,oneof=..."。
type Checkbox struct {
	Options       []string `json:"options,omitempty"`        // 选项列表
	RenderDefault []string `json:"render_default,omitempty"` // 前端渲染默认选中项（逗号分隔）
}

func (c *Checkbox) Config() interface{} {
	return c
}

func (c *Checkbox) Type() string {
	return TypeCheckbox
}

func (c *Checkbox) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := make([]SemanticFact, 0, 3)
	if len(c.Options) > 0 {
		facts = append(facts, SemanticFact{Key: "enum", Value: strings.Join(c.Options, "|")})
	}
	if len(c.RenderDefault) > 0 {
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: strings.Join(c.RenderDefault, "|")})
		facts = append(facts, SemanticFact{Key: "example", Value: quoteJSONArrayExample(c.RenderDefault)})
	} else if len(c.Options) > 0 {
		limit := 2
		if len(c.Options) < limit {
			limit = len(c.Options)
		}
		facts = append(facts, SemanticFact{Key: "example", Value: quoteJSONArrayExample(c.Options[:limit])})
	}
	return facts
}

func newCheckbox(widgetParsed map[string]string) *Checkbox {
	checkbox := &Checkbox{}

	// 从widgetParsed中解析配置
	if options, exists := widgetParsed["options"]; exists {
		// 解析逗号分隔的选项
		checkbox.Options = parseOptions(options)
	}
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		// 解析默认选中项（逗号分隔）
		checkbox.RenderDefault = parseOptions(defaultValue)
	}

	return checkbox
}

// validateCheckboxWidget 根据 Go 类型判断 checkbox 是单布尔还是多值复选。
//
// 与 radio/select 不同，checkbox 允许 bool 且不要求 options，因为它可以表示单个开关勾选。
func validateCheckboxWidget(ctx ValidateContext) error {
	var errs []error
	typ := derefType(ctx.GoType)
	if typ.Kind() != reflect.Bool && typ.Kind() != reflect.String && typ.Kind() != reflect.Slice && typ.Kind() != reflect.Array {
		errs = append(errs, fieldError(ctx, "checkbox widget requires bool, string, or slice/array Go type, got %s", typeName(ctx.GoType)))
	}
	if typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array {
		if err := validateScalarElement(ctx, typ.Elem(), "scalar"); err != nil {
			errs = append(errs, err)
		}
	}
	if err := validateUniqueOptions(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateOptionsMatchOneOf(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateChoiceDefaultsInOptions(ctx, parseOptions(ctx.Field.WidgetParsed[renderDefaultTagKey]), false); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
