package model

// AppType 应用类型
type AppType int

const (
	// AppTypeUser 用户空间（默认）
	AppTypeUser AppType = 0
	
	// AppTypeSystem 系统空间（内置官方库）
	AppTypeSystem AppType = 1
)

// String 返回应用类型的字符串表示
func (t AppType) String() string {
	switch t {
	case AppTypeUser:
		return "user"
	case AppTypeSystem:
		return "system"
	default:
		return "unknown"
	}
}

// IsSystem 判断是否为系统空间
func (t AppType) IsSystem() bool {
	return t == AppTypeSystem
}

// IsUser 判断是否为用户空间
func (t AppType) IsUser() bool {
	return t == AppTypeUser
}
