package widget

import (
	"errors"
	"strconv"
)

func init() {
	RegisterWidgetValidator(TypeRichText, validateRichTextWidget)
}

// RichText 富文本编辑组件。
//
// 使用示例：
//
//	Content string `json:"content" widget:"name:正文;type:richtext;height:420"`
//
// 校验规则：
// - 注册的是本文件的 validateRichTextWidget；
// - Go 字段必须是 string 或 *string，富文本 HTML/Delta 内容都按字符串协议传输；
// - height 必须是正整数；
// - 内容安全、HTML 白名单、长度限制不在 widget validator 内处理，应在业务层或前端富文本组件中处理。
type RichText struct {
	// 可选参数（有合理默认值）
	Height int `json:"height,omitempty"` // 编辑器高度（单位：px，默认300）
}

func (r *RichText) Config() interface{} {
	return r
}

func (r *RichText) Type() string {
	return TypeRichText
}

// validateRichTextWidget 校验 richtext 的字符串协议。
//
// 富文本内容最终仍是字符串；编辑器高度和内容安全策略不在启动期校验。
func validateRichTextWidget(ctx ValidateContext) error {
	var errs []error
	if err := requireStringLikeGoType(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validatePositiveIntTag(ctx, "height"); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
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
