package widget

import (
	"errors"
	"strconv"
)

func init() {
	RegisterWidgetValidator(TypeSlider, validateSliderWidget)
}

// Slider 滑块输入组件。
//
// 使用示例：
//
//	Progress int `json:"progress" widget:"name:进度;type:slider;min:0;max:100;step:5;unit:%"`
//
// 校验规则：
// - 注册的是本文件的 validateSliderWidget；
// - Go 字段必须是任意数值类型，整数和浮点数都可以；
// - min/max/step/render_default 必须能解析为 float64；
// - step 必须是正数；
// - 同时配置 min/max 时，min 不能大于 max；
// - render_default 必须落在 min/max 范围内，未配置 min/max 时按默认 0-100 校验。
type Slider struct {
	// 核心参数（必需）
	Min float64 `json:"min"` // 最小值（必需）
	Max float64 `json:"max"` // 最大值（必需）

	// 可选参数（有合理默认值）
	Step          float64  `json:"step,omitempty"`           // 步长（可选，默认1）
	RenderDefault *float64 `json:"render_default,omitempty"` // 前端渲染默认值（可选）
	Unit          string   `json:"unit,omitempty"`           // 单位（可选，如：%、元、kg等）

	// 注意：以下参数都有合理的默认值，前端自动处理，不需要配置
	// - show_input: 默认 false（简单场景不需要输入框）
	// - show_stops: 默认 false（简单场景不需要刻度）
	// - show_tooltip: 默认 true（拖动时显示提示）
	// - show_percentage: 输出模式默认 true（进度条显示百分比）
	// - status: 根据值自动判断（>80% success, 50-80% warning, <50% danger）
	// - stroke_width: 默认 6（进度条粗细）
}

func (s *Slider) Config() interface{} {
	return s
}

func (s *Slider) Type() string {
	return TypeSlider
}

// validateSliderWidget 校验 slider 的数值协议。
//
// slider 允许绑定整数或浮点数字段；范围配置是前端交互约束，不替代业务校验。
func validateSliderWidget(ctx ValidateContext) error {
	var errs []error
	if err := requireNumericGoType(ctx); err != nil {
		errs = append(errs, err)
	}
	for _, key := range []string{"min", "max", "step", renderDefaultTagKey} {
		if err := validateFloatTag(ctx, key); err != nil {
			errs = append(errs, err)
		}
	}
	if err := validatePositiveFloatTag(ctx, "step"); err != nil {
		errs = append(errs, err)
	}
	if err := validateNumberRange(ctx); err != nil {
		errs = append(errs, err)
	}
	defaultMin, defaultMax := 0.0, 100.0
	if err := validateRenderDefaultNumberRange(ctx, &defaultMin, &defaultMax); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func newSlider(widgetParsed map[string]string) *Slider {
	slider := &Slider{
		// 默认值
		Min:  0,
		Max:  100,
		Step: 1,
	}

	// 从widgetParsed中解析配置（只解析核心参数）
	if min, exists := widgetParsed["min"]; exists {
		if val, err := strconv.ParseFloat(min, 64); err == nil {
			slider.Min = val
		}
	}
	if max, exists := widgetParsed["max"]; exists {
		if val, err := strconv.ParseFloat(max, 64); err == nil {
			slider.Max = val
		}
	}
	if step, exists := widgetParsed["step"]; exists {
		if val, err := strconv.ParseFloat(step, 64); err == nil {
			slider.Step = val
		}
	}
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		if val, err := strconv.ParseFloat(defaultValue, 64); err == nil {
			slider.RenderDefault = &val
		}
	}
	if unit, exists := widgetParsed["unit"]; exists {
		slider.Unit = unit
	}

	return slider
}
