package widget

type Color struct {
	// 可选参数（有合理默认值）
	Format  string `json:"format"`  // 颜色格式：hex, rgb, rgba（默认hex）
	Default string `json:"default"` // 默认颜色（可选，如：#409EFF）
	ShowAlpha bool `json:"show_alpha"` // 是否显示透明度选择器（默认false，仅在format为rgba时有效）
}

func (c *Color) Config() interface{} {
	return c
}

func (c *Color) Type() string {
	return TypeColor
}

func newColor(widgetParsed map[string]string) *Color {
	color := &Color{
		// 默认值
		Format:    "hex",
		ShowAlpha: false,
	}

	// 从widgetParsed中解析配置
	if format, exists := widgetParsed["format"]; exists {
		// 验证格式是否有效
		if format == "hex" || format == "rgb" || format == "rgba" {
			color.Format = format
		}
	}
	if defaultValue, exists := widgetParsed["default"]; exists {
		color.Default = defaultValue
	}
	if showAlpha, exists := widgetParsed["show_alpha"]; exists {
		color.ShowAlpha = showAlpha == "true"
		// 如果启用透明度，自动设置为rgba格式
		if color.ShowAlpha && color.Format == "hex" {
			color.Format = "rgba"
		}
	}

	return color
}

