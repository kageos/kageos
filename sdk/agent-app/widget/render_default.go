package widget

const renderDefaultTagKey = "render_default"

func getRenderDefault(widgetParsed map[string]string) (string, bool) {
	value, exists := widgetParsed[renderDefaultTagKey]
	return value, exists
}
