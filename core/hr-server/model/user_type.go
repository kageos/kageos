package model

// UserType 用户类型
type UserType int

const (
	// UserTypeNormal 普通用户（默认）
	UserTypeNormal UserType = 0
	
	// UserTypeSystem 系统用户（内置官方库）
	UserTypeSystem UserType = 1
	
	// UserTypeAgent 智能体用户（智能助手，未来扩展）
	UserTypeAgent UserType = 2
)

// String 返回用户类型的字符串表示
func (t UserType) String() string {
	switch t {
	case UserTypeNormal:
		return "normal"
	case UserTypeSystem:
		return "system"
	case UserTypeAgent:
		return "agent"
	default:
		return "unknown"
	}
}

// IsSystem 判断是否为系统用户
func (t UserType) IsSystem() bool {
	return t == UserTypeSystem
}

// IsAgent 判断是否为智能体用户
func (t UserType) IsAgent() bool {
	return t == UserTypeAgent
}

// IsNormal 判断是否为普通用户
func (t UserType) IsNormal() bool {
	return t == UserTypeNormal
}
