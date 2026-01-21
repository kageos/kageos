package middleware

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// WithUserInfo 为请求添加用户信息的中间件
// ⭐ 统一使用常量 RequestUserHeader 和 DepartmentFullPathHeader，与 GetRequestUser 保持一致
// ⭐ 如果 X-Request-User header 为空，尝试从 token 中解析用户信息（作为降级方案）
func WithUserInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
		// ✨ 优先从 X-Request-User header 读取（网关已设置）
		requestUser := c.GetHeader(contextx.RequestUserHeader)

		// ⭐ 如果 header 中没有用户信息，尝试从 token 中解析（降级方案）
		if requestUser == "" {
			token := c.GetHeader(contextx.TokenHeader)
			if token != "" {
				// 尝试解析 token 获取用户信息
				jwtService := service.NewJWTService()
				claims, err := jwtService.ValidateToken(token)
				if err == nil {
					// token 解析成功，使用 token 中的用户信息
					requestUser = claims.Username
					logger.Debugf(c, "[WithUserInfo] 从 token 解析用户信息 - User: %s, Path: %s", requestUser, c.Request.URL.Path)

					// ⭐ 设置组织架构信息（从 token 中获取）
					if claims.DepartmentFullPath != nil && *claims.DepartmentFullPath != "" {
						c.Set(contextx.DepartmentFullPathHeader, *claims.DepartmentFullPath)
						logger.Debugf(c, "[WithUserInfo] 从 token 设置部门信息 - User: %s, DepartmentPath: %s", requestUser, *claims.DepartmentFullPath)
					}
				} else {
					// token 解析失败，记录警告日志
					logger.Warnf(c, "[WithUserInfo] X-Request-User header 为空，且 token 解析失败 - Path: %s, IP: %s, TokenLength: %d, Error: %v",
						c.Request.URL.Path, c.ClientIP(), len(token), err)
				}
			} else {
				// 没有 token，记录警告日志
				logger.Warnf(c, "[WithUserInfo] X-Request-User header 为空，且没有 X-Token - Path: %s, IP: %s",
					c.Request.URL.Path, c.ClientIP())
			}
		} else {
			// ⭐ 从 header 获取组织架构信息（网关已设置）
			// ⭐ 统一使用 DepartmentFullPathHeader 常量
			if deptPath := c.GetHeader(contextx.DepartmentFullPathHeader); deptPath != "" {
				c.Set(contextx.DepartmentFullPathHeader, deptPath)
				logger.Debugf(c, "[WithUserInfo] 从 header 设置部门信息 - User: %s, DepartmentPath: %s", requestUser, deptPath)
			}
		}

		// ⭐ 设置请求用户信息到 context（使用常量 RequestUserHeader）
		c.Set(contextx.RequestUserHeader, requestUser)

		c.Next()
	}
}
