package widget

type Timestamp struct {
	Format   string `json:"format"`   // 日期格式，如 YYYY-MM-DD HH:mm:ss
	Disabled bool   `json:"disabled"` // 是否禁用
}

func (t *Timestamp) Config() interface{} {
	return t
}

func (t *Timestamp) Type() string {
	return TypeTimestamp
}

func newTimestamp(widgetParsed map[string]string) *Timestamp {
	timestamp := &Timestamp{}

	// 从widgetParsed中解析配置
	if format, exists := widgetParsed["format"]; exists {
		timestamp.Format = format
	}
	if disabled, exists := widgetParsed["disabled"]; exists {
		timestamp.Disabled = disabled == "true"
	}

	return timestamp
}
