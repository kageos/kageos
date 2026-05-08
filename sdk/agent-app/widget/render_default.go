package widget

const renderDefaultTagKey = "render_default"

// getRenderDefault 统一读取 render_default tag。
//
// render_default 是“前端渲染默认值”协议：
// - 它会进入 widget.config.render_default；
// - 它不等价于数据库默认值，也不会自动写入 gorm default；
// - 动态函数如 CURRENT_TIMESTAMP、Me()、MyDepartment() 在前端初始化阶段解析；
// - SDK 这里只负责保留原始字符串，具体组件再决定是否解析成 int/float/bool/slice。
func getRenderDefault(widgetParsed map[string]string) (string, bool) {
	value, exists := widgetParsed[renderDefaultTagKey]
	return value, exists
}
