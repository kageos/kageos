package widget

import (
	"errors"
	"fmt"
	"strconv"
)

func init() {
	RegisterWidgetValidator(TypeNumber, validateNumberWidget)
}

// Number 整数输入组件。
//
// 使用示例：
//
//	Count int `json:"count" widget:"name:数量;type:number;min:1;max:100;step:1;unit:个" validate:"required,min=1"`
//
// 校验规则：
// - 注册的是本文件的 validateNumberWidget；
// - Go 字段必须是整数类型或整数指针，float 字段应使用 float 组件；
// - min/max 必须是整数；
// - 同时配置 min/max 时，min 不能大于 max；
// - step 必须是正数；
// - render_default 必须是整数，且配置 min/max 时必须落在范围内。
type Number struct {
	Placeholder   string `json:"placeholder,omitempty"`    // 占位符文本
	Min           *int   `json:"min,omitempty"`            // 最小值
	Max           *int   `json:"max,omitempty"`            // 最大值
	Step          string `json:"step,omitempty"`           // 步长（点击增减按钮的步进值）
	RenderDefault *int   `json:"render_default,omitempty"` // 前端渲染默认值
	Unit          string `json:"unit,omitempty"`           // 单位（如：件、个、元、kg等）
}

func (n *Number) Config() interface{} {
	return n
}

func (n *Number) Type() string {
	return TypeNumber
}

// validateNumberWidget 是 number 组件的校验。
//
// 规则：
// - Go 类型必须是整数类型，float 应使用 float 组件；
// - widget tag 中的 min/max 必须能解析为整数；
// - 同时配置 min/max 时，min 不能大于 max。
// - step 必须是正数，render_default 必须是整数且落在 min/max 范围内。
//
// 注意：step 允许小数形式，因为前端输入步长可以比 Go 整数承载值更细；最终提交值仍由 Go 字段整数类型约束。
func validateNumberWidget(ctx ValidateContext) error {
	var errs []error
	if !isIntegerType(ctx.GoType) {
		errs = append(errs, fieldError(ctx, "number widget requires integer Go type, got %s", typeName(ctx.GoType)))
	}
	if err := validateIntTag(ctx, "min"); err != nil {
		errs = append(errs, err)
	}
	if err := validateIntTag(ctx, "max"); err != nil {
		errs = append(errs, err)
	}
	if err := validateNumberRange(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validatePositiveFloatTag(ctx, "step"); err != nil {
		errs = append(errs, err)
	}
	if err := validateIntTag(ctx, renderDefaultTagKey); err != nil {
		errs = append(errs, err)
	}
	if err := validateRenderDefaultNumberRange(ctx, nil, nil); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (n *Number) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := make([]SemanticFact, 0, 5)
	if fact, ok := placeholderFact(n.Placeholder); ok {
		facts = append(facts, fact)
	}
	if n.Min != nil && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "min", Value: fmt.Sprintf("%d", *n.Min)})
	}
	if n.Max != nil && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "max", Value: fmt.Sprintf("%d", *n.Max)})
	}
	if n.RenderDefault != nil {
		defaultValue := fmt.Sprintf("%d", *n.RenderDefault)
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: defaultValue})
		if field != nil && field.Data != nil && field.Data.Example == "" {
			facts = append(facts, SemanticFact{Key: "example", Value: defaultValue})
		}
	}
	if n.Unit != "" {
		facts = append(facts, SemanticFact{Key: "unit", Value: n.Unit})
	}
	if n.Step != "" && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "step", Value: n.Step})
	}
	return facts
}

func newNumber(widgetParsed map[string]string) *Number {
	number := &Number{}

	// 从widgetParsed中解析配置
	if placeholder, exists := widgetParsed["placeholder"]; exists {
		number.Placeholder = placeholder
	}
	if min, exists := widgetParsed["min"]; exists {
		if val, err := strconv.Atoi(min); err == nil {
			number.Min = &val
		}
	}
	if max, exists := widgetParsed["max"]; exists {
		if val, err := strconv.Atoi(max); err == nil {
			number.Max = &val
		}
	}
	if step, exists := widgetParsed["step"]; exists {
		number.Step = step
	}
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		if val, err := strconv.Atoi(defaultValue); err == nil {
			number.RenderDefault = &val
		}
	}
	if unit, exists := widgetParsed["unit"]; exists {
		number.Unit = unit
	}

	return number
}
