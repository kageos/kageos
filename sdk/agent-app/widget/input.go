package widget

type Input struct {
	Placeholder string `json:"placeholder"` // 占位符文本
	Password    bool   `json:"password"`    //密码框框
	Prepend     string `json:"prepend"`     //前置
	Append      string `json:"append"`      //后置
	Default     string `json:"default"`     //默认值
}

func (i *Input) Config() interface{} {
	return i
}

func (i *Input) Type() string {
	return TypeInput
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
	if defaultValue, exists := widgetParsed["default"]; exists {
		input.Default = defaultValue
	}

	return input
}
