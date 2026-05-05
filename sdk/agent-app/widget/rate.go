package widget

import (
	"errors"
	"strconv"
	"strings"
)

func init() {
	RegisterWidgetValidator(TypeRate, validateRateWidget)
}

// Rate 星级评分组件。
//
// 使用示例：
//
//	Score float64 `json:"score" widget:"name:评分;type:rate;max:5;allow_half:true;texts:很差,差,一般,好,很好"`
//
// 校验规则：
// - 注册的是本文件的 validateRateWidget；
// - Go 字段必须是任意数值类型；
// - max 必须是正整数；
// - allow_half 必须是 true/false；
// - render_default 必须是数值，不能小于 0，且不能大于 max；未配置 max 时按默认 5 校验；
// - 如果评分范围影响业务逻辑，应在 validate tag 或 handler 中补充约束。
type Rate struct {
	// 核心参数（必需）
	Max int `json:"max,omitempty"` // 最大星级（默认5）

	// 可选参数（有合理默认值）
	AllowHalf     bool     `json:"allow_half,omitempty"`     // 是否允许半星（默认false）
	RenderDefault *float64 `json:"render_default,omitempty"` // 前端渲染默认评分（可选）
	Texts         []string `json:"texts,omitempty"`          // 自定义文字数组（可选，如：["很差", "差", "一般", "好", "很好"]）
	// 注意：如果配置了 texts，会自动显示文字；如果没有配置 texts，则不显示文字
}

func (r *Rate) Config() interface{} {
	return r
}

func (r *Rate) Type() string {
	return TypeRate
}

// validateRateWidget 校验 rate 的数值协议。
//
// rate 的视觉星级配置不改变底层值类型，字段仍然必须是数值。
func validateRateWidget(ctx ValidateContext) error {
	var errs []error
	if err := requireNumericGoType(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validatePositiveIntTag(ctx, "max"); err != nil {
		errs = append(errs, err)
	}
	if err := validateBoolTag(ctx, "allow_half"); err != nil {
		errs = append(errs, err)
	}
	if err := validateFloatTag(ctx, renderDefaultTagKey); err != nil {
		errs = append(errs, err)
	}
	if err := validateRateDefaultRange(ctx); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func validateRateDefaultRange(ctx ValidateContext) error {
	rawDefault := strings.TrimSpace(ctx.Field.WidgetParsed[renderDefaultTagKey])
	if rawDefault == "" {
		return nil
	}
	defaultValue, err := strconv.ParseFloat(rawDefault, 64)
	if err != nil {
		return nil
	}
	if defaultValue < 0 {
		return fieldError(ctx, "widget render_default must be >= 0, got %s", rawDefault)
	}
	rawMax := strings.TrimSpace(ctx.Field.WidgetParsed["max"])
	if rawMax == "" {
		rawMax = "5"
	}
	maxValue, err := strconv.ParseFloat(rawMax, 64)
	if err != nil {
		return nil
	}
	if defaultValue > maxValue {
		return fieldError(ctx, "widget render_default must be <= max, got render_default=%s max=%s", rawDefault, rawMax)
	}
	return nil
}

func newRate(widgetParsed map[string]string) *Rate {
	rate := &Rate{
		// 默认值
		Max:       5,
		AllowHalf: false,
	}

	// 从widgetParsed中解析配置
	if max, exists := widgetParsed["max"]; exists {
		if val, err := strconv.Atoi(max); err == nil && val > 0 {
			rate.Max = val
		}
	}
	if allowHalf, exists := widgetParsed["allow_half"]; exists {
		rate.AllowHalf = allowHalf == "true"
	}
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		if val, err := strconv.ParseFloat(defaultValue, 64); err == nil {
			rate.RenderDefault = &val
		}
	}
	if texts, exists := widgetParsed["texts"]; exists {
		// 解析逗号分隔的文字数组
		rate.Texts = parseOptions(texts)
	}

	return rate
}
