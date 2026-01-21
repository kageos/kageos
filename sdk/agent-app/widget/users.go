package widget

// Users 多用户选择器组件
//
// 功能：
// - 支持多个用户搜索和选择
// - 支持动态默认值：$me（当前登录用户）
// - 值使用逗号分隔的字符串格式存储（如 "user1,user2,user3"）
//
// 使用示例：
//   widget:"name:审核人;type:users;default:$me"
//   widget:"name:管理员;type:users;max_count:5"
//
// 动态默认值说明：
//   - $me: 自动填充当前登录用户的用户名，用户无需手动选择
//   - 支持多个默认值，用逗号分隔：$me,user2
//
// 注意：
//   - default 参数支持动态变量（以 $ 开头）
//   - 如果用户未登录，$me 会返回 null
//   - 值存储格式：逗号分隔的字符串（如 "user1,user2"），便于存储到数据库
//   - 前端会自动处理字符串和数组之间的转换
type Users struct {
	Default  string `json:"default,omitempty"`   // 默认值，支持动态变量 $me（当前登录用户），多个值用逗号分隔
	MaxCount int    `json:"max_count,omitempty"` // 最大选择数量，0表示不限制
}

func (u *Users) Config() interface{} {
	return u
}

func (u *Users) Type() string {
	return TypeUsers
}

func newUsers(widgetParsed map[string]string) *Users {
	users := &Users{}

	// 从widgetParsed中解析配置
	if defaultValue, exists := widgetParsed["default"]; exists {
		users.Default = defaultValue
	}
	if maxCount, exists := widgetParsed["max_count"]; exists {
		// 解析最大选择数量，支持 "0" 或 "" 表示不限制
		if maxCount == "0" || maxCount == "" {
			users.MaxCount = 0 // 0表示不限制
		} else {
			// 注意：如果需要在 widget 标签中设置具体数值，前端会解析字符串
			// Go 端只需要知道是否有限制即可，具体数值由前端处理
			users.MaxCount = 0 // 默认不限制，具体数值由前端解析
		}
	}

	return users
}

