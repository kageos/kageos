package widget

// Timestamp 时间戳组件
//
// 功能：
// - 支持日期时间选择
// - 支持动态默认值：$now、$today、$tomorrow、$yesterday
//
// 使用示例：
//   widget:"name:开始时间;type:timestamp;default:$now;format:YYYY-MM-DD HH:mm:ss"
//   widget:"name:创建时间;type:timestamp;default:$today;format:YYYY-MM-DD HH:mm:ss"
//   widget:"name:截止日期;type:timestamp;default:$tomorrow;format:YYYY-MM-DD"
//
// 动态默认值说明：
//   基础时间：
//   - $now: 当前时间（毫秒时间戳），适用于：开始时间、创建时间等
//   - $today: 今天开始时间 00:00:00（毫秒时间戳），适用于：创建日期、开始日期等
//   - $tomorrow: 明天开始时间 00:00:00（毫秒时间戳），适用于：截止日期、到期日期等
//   - $yesterday: 昨天开始时间 00:00:00（毫秒时间戳），适用于：历史记录查询等
//   相对时间（此刻）：
//   - $yesterday_now: 昨天此刻
//   - $tomorrow_now: 明天此刻
//   相对时间（小时）：
//   - $after_1h, $after_2h, $after_3h, $after_6h, $after_12h: 1/2/3/6/12小时后
//   - $before_1h, $before_2h, $before_3h: 1/2/3小时前
//   相对时间（天）：
//   - $after_1d, $after_2d, $after_3d, $after_7d, $after_30d: 1/2/3/7/30天后
//   - $before_1d, $before_2d, $before_7d, $before_30d: 1/2/7/30天前
//   相对时间（周）：
//   - $next_week: 下周一开始（00:00:00）
//   - $last_week: 上周一开始（00:00:00）
//   相对时间（月）：
//   - $next_month: 下个月1号（00:00:00）
//   - $last_month: 上个月1号（00:00:00）
//   相对时间（年）：
//   - $next_year: 明年1月1日（00:00:00）
//   - $last_year: 去年1月1日（00:00:00）
//
// 参数说明：
//   - format: 日期格式，如 YYYY-MM-DD HH:mm:ss、YYYY-MM-DD 等
//   - disabled: 是否禁用（只读模式）
//   - default: 默认值，支持动态变量（以 $ 开头）或具体时间戳
//
// 注意：
//   - 所有动态变量返回的是毫秒级时间戳
//   - 如果字段已有值（编辑模式），不会覆盖已有值
//   - 只有在字段为空时，才会使用默认值
type Timestamp struct {
	Format   string `json:"format,omitempty"`   // 日期格式，如 YYYY-MM-DD HH:mm:ss
	Disabled bool   `json:"disabled,omitempty"` // 是否禁用
	Default  string `json:"default,omitempty"`  // 默认值，支持动态变量 $now、$today、$tomorrow、$yesterday
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
	if defaultValue, exists := widgetParsed["default"]; exists {
		timestamp.Default = defaultValue
	}

	return timestamp
}
