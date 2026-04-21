package widget

import "strconv"

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
