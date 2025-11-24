package widget

import "strconv"

type RichText struct {
	// 可选参数（有合理默认值）
	Height int `json:"height"` // 编辑器高度（单位：px，默认300）
}

func (r *RichText) Config() interface{} {
	return r
}

func (r *RichText) Type() string {
	return TypeRichText
}

func newRichText(widgetParsed map[string]string) *RichText {
	richText := &RichText{
		// 默认值
		Height: 300,
	}

	// 从widgetParsed中解析配置
	if height, exists := widgetParsed["height"]; exists {
		if val, err := strconv.Atoi(height); err == nil && val > 0 {
			richText.Height = val
		}
	}

	return richText
}

