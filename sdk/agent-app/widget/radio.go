package widget

import (
	"errors"
	"strings"
)

func init() {
	RegisterWidgetValidator(TypeRadio, validateRadioWidget)
}

// Radio 单选按钮组件。
//
// 使用示例：
//
//	Gender string `json:"gender" widget:"name:性别;type:radio;options:男,女;render_default:男"`
//
// 校验规则：
// - Go 字段必须是标量类型；
// - options 必须非空，radio 不支持只靠 OnSelectFuzzyMap 动态加载；
// - options 不能有重复值；
// - render_default 必须属于 options；
// - 如果配置 validate:"oneof=..."，options 必须与 oneof 集合完全一致。
type Radio struct {
	Options       []string `json:"options,omitempty"`        // 选项列表
	RenderDefault string   `json:"render_default,omitempty"` // 前端渲染默认选中项
}

func (r *Radio) Config() interface{} {
	return r
}

func (r *Radio) Type() string {
	return TypeRadio
}

func (r *Radio) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := make([]SemanticFact, 0, 3)
	if len(r.Options) > 0 {
		facts = append(facts, SemanticFact{Key: "enum", Value: strings.Join(r.Options, "|")})
	}
	if strings.TrimSpace(r.RenderDefault) != "" {
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: r.RenderDefault})
		facts = append(facts, SemanticFact{Key: "example", Value: quoteExampleValue(r.RenderDefault)})
	} else if len(r.Options) > 0 {
		facts = append(facts, SemanticFact{Key: "example", Value: quoteExampleValue(r.Options[0])})
	}
	return facts
}

func newRadio(widgetParsed map[string]string) *Radio {
	radio := &Radio{}

	// 从widgetParsed中解析配置
	if options, exists := widgetParsed["options"]; exists {
		// 解析逗号分隔的选项
		radio.Options = parseOptions(options)
	}
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		radio.RenderDefault = defaultValue
	}

	return radio
}

// validateRadioWidget 校验 radio 的值类型和静态选项。
//
// radio 是强静态枚举展示，必须有 options；如果需要远程搜索或大数据量选项，应使用 select + OnSelectFuzzyMap。
func validateRadioWidget(ctx ValidateContext) error {
	var errs []error
	if !isScalarType(ctx.GoType) {
		errs = append(errs, fieldError(ctx, "radio widget requires scalar Go type, got %s", typeName(ctx.GoType)))
	}
	if len(parseOptions(ctx.Field.WidgetParsed["options"])) == 0 {
		errs = append(errs, fieldError(ctx, "radio widget requires non-empty options"))
	}
	if err := validateUniqueOptions(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateOptionsMatchOneOf(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateChoiceDefaultsInOptions(ctx, []string{ctx.Field.WidgetParsed[renderDefaultTagKey]}, false); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
