package widget

import (
	"errors"
	"reflect"
	"strconv"
)

func init() {
	RegisterWidgetValidator(TypeSwitch, validateSwitchWidget)
}

// Switch 布尔开关组件。
//
// 使用示例：
//
//	Enabled bool `json:"enabled" widget:"name:启用;type:switch;render_default:true"`
//
// 校验规则：
// - Go 字段必须是 bool 或 *bool；
// - render_default 必须显式写 true/false；
// - 当前组件只表达布尔值，不支持 true_label/false_label 之类展示枚举。
type Switch struct {
	RenderDefault *bool `json:"render_default,omitempty"` // 前端渲染默认值
}

func (s *Switch) Config() interface{} {
	return s
}

func (s *Switch) Type() string {
	return TypeSwitch
}

func newSwitch(widgetParsed map[string]string) *Switch {
	switchWidget := &Switch{}
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		if val, err := strconv.ParseBool(defaultValue); err == nil {
			switchWidget.RenderDefault = &val
		}
	}
	return switchWidget
}

// validateSwitchWidget 保证开关只绑定布尔字段。
//
// 如果业务字段是 "启用/禁用" 这种字符串枚举，应使用 select/radio，而不是 switch。
func validateSwitchWidget(ctx ValidateContext) error {
	var errs []error
	if derefType(ctx.GoType).Kind() != reflect.Bool {
		errs = append(errs, fieldError(ctx, "switch widget requires bool Go type, got %s", typeName(ctx.GoType)))
	}
	if err := validateBoolTag(ctx, renderDefaultTagKey); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
