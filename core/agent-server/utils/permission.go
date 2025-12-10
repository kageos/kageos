package utils

import "strings"

// IsAdmin 判断用户是否是管理员
// adminList: 管理员列表，逗号分隔，如："user1,user2,user3"
// user: 当前用户
func IsAdmin(adminList, user string) bool {
	if adminList == "" || user == "" {
		return false
	}
	// 分割管理员列表
	admins := strings.Split(adminList, ",")
	for _, admin := range admins {
		if strings.TrimSpace(admin) == user {
			return true
		}
	}
	return false
}

