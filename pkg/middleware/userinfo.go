package middleware

import (
	"github.com/gin-gonic/gin"
)

// WithUserInfo 为请求添加用户信息的中间件
func WithUserInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只从请求头中获取请求用户信息
		requestUser := c.GetHeader("X-Request-User")
		if requestUser == "" {
			requestUser = c.GetHeader("X-User")
		}
		if requestUser == "" {
			requestUser = c.GetHeader("X-Username")
		}
		if requestUser == "" {
			requestUser = c.GetHeader("User")
		}
		if requestUser == "" {
			requestUser = c.GetHeader("Username")
		}

		// 如果都没有，使用默认值（开发环境）
		if requestUser == "" {
			requestUser = "beiluo" // 开发环境默认用户
		}

		// 设置请求用户信息到 context
		c.Set("request_user", requestUser)
		c.Set("user", requestUser) // 保持向后兼容

		c.Next()
	}
}
