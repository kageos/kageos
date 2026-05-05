package widget

import (
	"errors"
	"strings"
)

func init() {
	RegisterWidgetValidator(TypeColor, validateColorWidget)
}

// Color 颜色选择组件。
//
// 使用示例：
//
//	ThemeColor string `json:"theme_color" widget:"name:主题色;type:color;format:hex;render_default:#409EFF"`
//
// 校验规则：
// - 注册的是本文件的 validateColorWidget；
// - Go 字段必须是 string 或 *string；
// - format 只能是 hex/rgb/rgba；
// - show_alpha 必须显式写 true/false；
// - show_alpha:true 会自动把 hex 升级成 rgba；
// - render_default 必须是前端颜色组件能识别的基础颜色值：hex、rgb/rgba 或少量常见颜色名。
type Color struct {
	// 可选参数（有合理默认值）
	Format        string `json:"format,omitempty"`         // 颜色格式：hex, rgb, rgba（默认hex）
	RenderDefault string `json:"render_default,omitempty"` // 前端渲染默认颜色（可选，如：#409EFF）
	ShowAlpha     bool   `json:"show_alpha,omitempty"`     // 是否显示透明度选择器（默认false，仅在format为rgba时有效）
}

func (c *Color) Config() interface{} {
	return c
}

func (c *Color) Type() string {
	return TypeColor
}

// validateColorWidget 校验 color 的字符串协议。
//
// 颜色值以字符串传输；这里按前端 ColorWidget 的基础识别范围拦截明显非法默认值。
func validateColorWidget(ctx ValidateContext) error {
	var errs []error
	if err := requireStringLikeGoType(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateEnumTag(ctx, "format", "hex", "rgb", "rgba"); err != nil {
		errs = append(errs, err)
	}
	if err := validateBoolTag(ctx, "show_alpha"); err != nil {
		errs = append(errs, err)
	}
	if err := validateColorRenderDefault(ctx); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func validateColorRenderDefault(ctx ValidateContext) error {
	rawDefault := strings.TrimSpace(ctx.Field.WidgetParsed[renderDefaultTagKey])
	if rawDefault == "" {
		return nil
	}
	if isValidColorDefaultValue(rawDefault) {
		return nil
	}
	return fieldError(ctx, "widget render_default must be a valid color value, got %q", rawDefault)
}

func isValidColorDefaultValue(value string) bool {
	normalized := normalizeColorDefaultValue(value)
	if isValidHexColor(normalized) {
		return true
	}
	lower := strings.ToLower(normalized)
	if strings.HasPrefix(lower, "rgb(") || strings.HasPrefix(lower, "rgba(") {
		return strings.HasSuffix(lower, ")")
	}
	switch lower {
	case "red", "blue", "green", "yellow", "orange", "purple", "pink", "black", "white", "gray", "grey":
		return true
	default:
		return false
	}
}

func isValidHexColor(value string) bool {
	if !strings.HasPrefix(value, "#") {
		return false
	}
	hex := value[1:]
	if len(hex) != 3 && len(hex) != 6 && len(hex) != 8 {
		return false
	}
	for _, r := range hex {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}

func normalizeColorDefaultValue(value string) string {
	value = strings.TrimSpace(value)
	hex := value
	if strings.HasPrefix(hex, "#") {
		return hex
	}
	if len(hex) == 3 || len(hex) == 6 || len(hex) == 8 {
		for _, r := range hex {
			if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
				return value
			}
		}
		return "#" + hex
	}
	return value
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
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		color.RenderDefault = defaultValue
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
