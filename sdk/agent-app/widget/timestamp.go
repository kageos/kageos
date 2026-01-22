package widget

// Timestamp 时间戳组件
//
// 功能：
// - 支持日期时间选择
// - 支持动态默认值函数：Now()、Today()、Tomorrow()、Yesterday() 等
//
// 使用示例：
//   widget:"name:开始时间;type:timestamp;default:Now();format:YYYY-MM-DD HH:mm:ss"
//   widget:"name:创建时间;type:timestamp;default:Today();format:YYYY-MM-DD HH:mm:ss"
//   widget:"name:截止日期;type:timestamp;default:Tomorrow();format:YYYY-MM-DD"
//   widget:"name:一小时后;type:timestamp;default:Now(\"+1h\");format:YYYY-MM-DD HH:mm:ss"
//   widget:"name:两天后;type:timestamp;default:Now(\"+2d\");format:YYYY-MM-DD HH:mm:ss"
//
// 动态默认值函数说明：
//   基础时间函数：
//   - Now(): 当前时间（毫秒时间戳），适用于：开始时间、创建时间等
//   - Today(): 今天开始时间 00:00:00（毫秒时间戳），适用于：创建日期、开始日期等
//   - Tomorrow(): 明天开始时间 00:00:00（毫秒时间戳），适用于：截止日期、到期日期等
//   - Yesterday(): 昨天开始时间 00:00:00（毫秒时间戳），适用于：历史记录查询等
//
//   相对时间函数（Now() 支持参数）：
//   - Now("+1h"): 一小时后（当前时间 + 1小时）
//   - Now("-1h"): 一小时前（当前时间 - 1小时）
//   - Now("+2d"): 两天后（当前时间 + 2天）
//   - Now("-2d"): 两天前（当前时间 - 2天）
//   - Now("+1w"): 一周后（当前时间 + 1周）
//   - Now("-1w"): 一周前（当前时间 - 1周）
//   - Now("+1m"): 一个月后（当前时间 + 1月）
//   - Now("-1m"): 一个月前（当前时间 - 1月）
//   - Now("+1y"): 一年后（当前时间 + 1年）
//   - Now("-1y"): 一年前（当前时间 - 1年）
//   - Now("+3600s"): 3600秒后（当前时间 + 3600秒）
//   - Now("-3600s"): 3600秒前（当前时间 - 3600秒）
//   - Now("+2"): 2小时后（如果只写数字，默认单位是小时）
//
//   参数格式说明：
//   - 支持正负号：+ 表示未来，- 表示过去
//   - 支持单位：s(秒)、h(小时)、d(天)、w(周)、m(月)、y(年)
//   - 如果只写数字（如 "+2"），默认单位是小时
//   - 示例：Now("+1h"), Now("-2d"), Now("+1w"), Now("-1m"), Now("+3600s")
//
// 参数说明：
//   - format: 日期格式，如 YYYY-MM-DD HH:mm:ss、YYYY-MM-DD 等
//   - disabled: 是否禁用（只读模式）
//   - default: 默认值，支持函数调用（如 Now()、Today()）或具体时间戳
//
// 注意：
//   - 所有函数返回的是毫秒级时间戳
//   - 如果字段已有值（编辑模式），不会覆盖已有值
//   - 只有在字段为空时，才会使用默认值
type Timestamp struct {
	Format   string `json:"format,omitempty"`   // 日期格式，如 YYYY-MM-DD HH:mm:ss
	Disabled bool   `json:"disabled,omitempty"` // 是否禁用
	Default  string `json:"default,omitempty"`  // 默认值，支持函数调用 Now()、Today()、Tomorrow()、Yesterday() 等
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
