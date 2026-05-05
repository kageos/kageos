package widget

import (
	"errors"
	"strconv"
)

func init() {
	RegisterWidgetValidator(TypeProgress, validateProgressWidget)
}

// Progress 进度展示组件。
//
// 使用示例：
//
//	Completion float64 `json:"completion" widget:"name:完成率;type:progress;min:0;max:100;unit:%"`
//
// 校验规则：
// - 注册的是本文件的 validateProgressWidget；
// - Go 字段必须是任意数值类型；
// - min/max/unit 只影响前端展示区间和单位；
// - min/max 必须能解析为 float64；
// - 同时配置 min/max 时，min 不能大于 max；
// - 业务需要时仍应在 handler 或 validate tag 中约束实际数值。
type Progress struct {
	Min  float64 `json:"min,omitempty"`  // 最小值，默认 0
	Max  float64 `json:"max,omitempty"`  // 最大值，默认 100
	Unit string  `json:"unit,omitempty"` // 单位（如：%、人、次等），默认 %
}

func (p *Progress) Config() interface{} {
	return p
}

func (p *Progress) Type() string {
	return TypeProgress
}

// validateProgressWidget 校验 progress 的数值协议。
//
// progress 是展示型数值组件，只要求 Go 字段为数值；展示区间配置由 newProgress 解析默认值。
func validateProgressWidget(ctx ValidateContext) error {
	var errs []error
	if err := requireNumericGoType(ctx); err != nil {
		errs = append(errs, err)
	}
	for _, key := range []string{"min", "max"} {
		if err := validateFloatTag(ctx, key); err != nil {
			errs = append(errs, err)
		}
	}
	if err := validateNumberRange(ctx); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func newProgress(widgetParsed map[string]string) *Progress {
	progress := &Progress{
		Min:  0,
		Max:  100,
		Unit: "%",
	}

	// 从widgetParsed中解析配置
	if min, exists := widgetParsed["min"]; exists {
		if val, err := parseFloat(min); err == nil {
			progress.Min = val
		}
	}
	if max, exists := widgetParsed["max"]; exists {
		if val, err := parseFloat(max); err == nil {
			progress.Max = val
		}
	}
	if unit, exists := widgetParsed["unit"]; exists {
		progress.Unit = unit
	}

	return progress
}

// parseFloat 解析浮点数
func parseFloat(s string) (float64, error) {
	// 这里可以后续优化为更复杂的解析
	// 暂时使用简单的 strconv.ParseFloat
	return strconv.ParseFloat(s, 64)
}
