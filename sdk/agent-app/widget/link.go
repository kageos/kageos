package widget

import "errors"

func init() {
	RegisterWidgetValidator(TypeLink, validateLinkWidget)
}

// Link 链接组件配置。
//
// 使用示例：
//
//	Homepage string `json:"homepage" widget:"name:官网;type:link;text:打开官网;target:_blank;link_type:primary"`
//
// 校验规则：
// - 注册的是本文件的 validateLinkWidget；
// - Go 字段必须是 string 或 *string，字段值通常是 URL；
// - text/target/link_type/icon 只是展示配置；
// - target 只能是 _self 或 _blank；
// - link_type 只能是 primary/success/warning/danger/info；
// - 不要在 widget tag 里写第二个 type，type 是组件类型保留 key；link_type 会输出为 schema JSON 的 type 字段；
// - 当前启动期不校验 URL 格式，业务需要时应加 validate tag 或后端逻辑校验。
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

// validateLinkWidget 校验 link 的字符串协议。
//
// URL 合法性不是 widget 层职责；需要强约束时在 validate tag 或业务代码中处理。
func validateLinkWidget(ctx ValidateContext) error {
	var errs []error
	if err := requireStringLikeGoType(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateEnumTag(ctx, "target", "_self", "_blank"); err != nil {
		errs = append(errs, err)
	}
	if err := validateEnumTag(ctx, "link_type", "primary", "success", "warning", "danger", "info"); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
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

	// 解析 link_type 参数。widget tag 的 type 是组件类型保留 key，所以链接样式不能复用 type。
	if linkType, ok := widgetParsed["link_type"]; ok {
		link.LinkType = linkType
	}

	// 解析 icon 参数
	if icon, ok := widgetParsed["icon"]; ok {
		link.Icon = icon
	}

	return link
}
