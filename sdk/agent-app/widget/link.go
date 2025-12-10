package widget

// Link 链接组件配置
type Link struct {
	// Text 链接文本（可选，如果不设置则使用字段名称）
	Text string `json:"text,omitempty"`
	// Target 链接打开方式（_self, _blank）
	Target string `json:"target,omitempty"`
	// LinkType 链接类型（primary, success, warning, danger, info）
	LinkType string `json:"type,omitempty"`
	// Icon 链接图标（可选）
	Icon string `json:"icon,omitempty"`
}

// Config 返回配置
func (l *Link) Config() interface{} {
	return l
}

// Type 返回组件类型
func (l *Link) Type() string {
	return TypeLink
}

// newLink 创建链接组件
func newLink(widgetParsed map[string]string) Widget {
	link := &Link{
		Target:   "_self",
		LinkType: "primary",
	}

	// 解析 text 参数
	if text, ok := widgetParsed["text"]; ok {
		link.Text = text
	}

	// 解析 target 参数
	if target, ok := widgetParsed["target"]; ok {
		link.Target = target
	}

	// 解析 type 参数
	if linkType, ok := widgetParsed["type"]; ok {
		link.LinkType = linkType
	}

	// 解析 icon 参数
	if icon, ok := widgetParsed["icon"]; ok {
		link.Icon = icon
	}

	return link
}
