package v1

import (
	"github.com/kageos/kageos/core/hr-server/service"
)

// User 用户相关API
type User struct {
	userService       *service.UserService
	departmentService *service.DepartmentService
}

// NewUser 创建用户API（依赖注入）
func NewUser(userService *service.UserService, departmentService *service.DepartmentService) *User {
	return &User{
		userService:       userService,
		departmentService: departmentService,
	}
}
