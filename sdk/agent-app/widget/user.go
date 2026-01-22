package widget

// User 用户选择器组件
//
// 功能：
// - 支持用户搜索和选择
// - 支持动态默认值函数：Me()（当前登录用户）
//
// 使用示例：
//
//	widget:"name:预约人;type:user;default:Me()"
//
// 动态默认值函数说明：
//   - Me(): 自动填充当前登录用户的用户名，用户无需手动选择
//     适用于：预约人、创建人、负责人等字段，大部分情况下默认是自己
//
// 注意：
//   - default 参数支持函数调用（如 Me()）
//   - 如果用户未登录，Me() 会返回 null
type User struct {
	Default string `json:"default,omitempty"` // 默认值，支持函数调用 Me()（当前登录用户）
}

func (u *User) Config() interface{} {
	return u
}

func (u *User) Type() string {
	return TypeUser
}

func newUser(widgetParsed map[string]string) *User {
	user := &User{}

	// 从widgetParsed中解析配置
	if defaultValue, exists := widgetParsed["default"]; exists {
		user.Default = defaultValue
	}

	return user
}
