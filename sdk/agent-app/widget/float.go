package widget

import (
	"errors"
	"fmt"
	"strconv"
)

func init() {
	RegisterWidgetValidator(TypeFloat, validateFloatWidget)
}

// Float 小数输入组件。
//
// 使用示例：
//
//	Amount float64 `json:"amount" widget:"name:金额;type:float;min:0;precision:2;unit:元" validate:"required"`
//
// 校验规则：
// - 注册的是本文件的 validateFloatWidget；
// - Go 字段必须是 float32/float64 或对应指针；
// - min/max/step 必须是可解析的小数；
// - step 必须是正数；
// - precision 必须是非负整数；
// - 同时配置 min/max 时，min 不能大于 max；
// - render_default 必须能解析为 float64，且配置 min/max 时必须落在范围内。
type Float struct {
	Placeholder   string   `json:"placeholder,omitempty"`    // 占位符文本
	Min           *float64 `json:"min,omitempty"`            // 最小值
	Max           *float64 `json:"max,omitempty"`            // 最大值
	Precision     string   `json:"precision,omitempty"`      // 小数位数（显示和输入精度）
	Step          string   `json:"step,omitempty"`           // 步长（点击增减按钮的步进值）
	RenderDefault *float64 `json:"render_default,omitempty"` // 前端渲染默认值
	Unit          string   `json:"unit,omitempty"`           // 单位（如：元、kg、%等）
}

func (f *Float) Config() interface{} {
	return f
}

func (f *Float) Type() string {
	return TypeFloat
}

// validateFloatWidget 是 float 组件的校验。
//
// 规则：
// - Go 类型必须是 float32/float64；
// - min/max/step 必须能解析为浮点数；
// - step 必须是正数；
// - precision 必须是非负整数；
// - 同时配置 min/max 时，min 不能大于 max；
// - render_default 必须落在 min/max 范围内。
//
// 需要整数输入时不要使用 float，应使用 number。
func validateFloatWidget(ctx ValidateContext) error {
	var errs []error
	if !isFloatType(ctx.GoType) {
		errs = append(errs, fieldError(ctx, "float widget requires float32/float64 Go type, got %s", typeName(ctx.GoType)))
	}
	for _, key := range []string{"min", "max", "step"} {
		if err := validateFloatTag(ctx, key); err != nil {
			errs = append(errs, err)
		}
	}
	if err := validatePositiveFloatTag(ctx, "step"); err != nil {
		errs = append(errs, err)
	}
	if err := validateFloatTag(ctx, renderDefaultTagKey); err != nil {
		errs = append(errs, err)
	}
	if err := validateNonNegativeIntTag(ctx, "precision"); err != nil {
		errs = append(errs, err)
	}
	if err := validateNumberRange(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateRenderDefaultNumberRange(ctx, nil, nil); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (f *Float) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := make([]SemanticFact, 0, 6)
	if fact, ok := placeholderFact(f.Placeholder); ok {
		facts = append(facts, fact)
	}
	if f.Min != nil && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "min", Value: fmt.Sprintf("%v", *f.Min)})
	}
	if f.Max != nil && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "max", Value: fmt.Sprintf("%v", *f.Max)})
	}
	if f.RenderDefault != nil {
		defaultValue := fmt.Sprintf("%v", *f.RenderDefault)
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: defaultValue})
		if field != nil && field.Data != nil && field.Data.Example == "" {
			facts = append(facts, SemanticFact{Key: "example", Value: defaultValue})
		}
	}
	if f.Unit != "" {
		facts = append(facts, SemanticFact{Key: "unit", Value: f.Unit})
	}
	if f.Precision != "" && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "precision", Value: f.Precision})
	}
	if f.Step != "" && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "step", Value: f.Step})
	}
	return facts
}

func newFloat(widgetParsed map[string]string) *Float {
	floatWidget := &Float{}

	// 从widgetParsed中解析配置
	if placeholder, exists := widgetParsed["placeholder"]; exists {
		floatWidget.Placeholder = placeholder
	}
	if min, exists := widgetParsed["min"]; exists {
		if val, err := strconv.ParseFloat(min, 64); err == nil {
			floatWidget.Min = &val
		}
	}
	if max, exists := widgetParsed["max"]; exists {
		if val, err := strconv.ParseFloat(max, 64); err == nil {
			floatWidget.Max = &val
		}
	}
	if precision, exists := widgetParsed["precision"]; exists {
		floatWidget.Precision = precision
	}
	if step, exists := widgetParsed["step"]; exists {
		floatWidget.Step = step
	}
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		if val, err := strconv.ParseFloat(defaultValue, 64); err == nil {
			floatWidget.RenderDefault = &val
		}
	}
	if unit, exists := widgetParsed["unit"]; exists {
		floatWidget.Unit = unit
	}

	return floatWidget
}
