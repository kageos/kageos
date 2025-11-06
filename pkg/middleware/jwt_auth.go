package middleware

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// isInternalRequest 检查是否为内网请求（SDK内部调用）
func isInternalRequest(c *gin.Context) bool {
	clientIP := c.ClientIP()
	// 检查是否为内网IP：localhost、127.0.0.1、容器内网地址
	return clientIP == "127.0.0.1" || clientIP == "localhost" || clientIP == "::1" ||
		clientIP == "172.17.0.1" || // Docker默认网关
		clientIP == "host.docker.internal" || // Docker Desktop
		c.GetHeader("X-Forwarded-For") == "" // 没有代理，说明是直连
}

// JWTAuth JWT认证中间件（支持内网免验证）
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从X-Token头获取Token
		token := c.GetHeader("X-Token")

		// ✅ 如果有token，使用token验证（Web端调用）
		if token != "" {
			// 验证Token
			jwtService := service.NewJWTService()
			claims, err := jwtService.ValidateToken(token)
			if err != nil {
				logger.Errorf(c, "[JWTAuth] Token validation failed: %v", err)
				response.FailWithMessage(c, "认证令牌无效或已过期")
				c.Abort()
				return
			}

			// 设置用户信息到上下文
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("email", claims.Email)
			c.Set("request_user", claims.Username) // 保持向后兼容
			c.Set("user", claims.Username)         // 保持向后兼容
			c.Set("token", token)                  // ✅ 保存 token 到 context，供透传使用

			logger.Infof(c, "[JWTAuth] User authenticated: %s (ID: %d)", claims.Username, claims.UserID)
			c.Next()
			return
		}

		// ✅ 如果没有token，检查是否为内网请求（SDK内部调用）
		if isInternalRequest(c) {
			// 从header获取用户信息（SDK传入）
			requestUser := c.GetHeader("X-Request-User")
			if requestUser == "" {
				response.FailWithMessage(c, "内网请求必须提供X-Request-User头")
				c.Abort()
				return
			}

			// 设置用户信息（仅设置用户名，不设置user_id等）
			c.Set("request_user", requestUser)
			c.Set("user", requestUser) // 保持向后兼容

			logger.Infof(c, "[JWTAuth] Internal request authenticated: %s (IP: %s)", requestUser, c.ClientIP())
			c.Next()
			return
		}

		// 外部请求且没有token，拒绝
		response.FailWithMessage(c, "未提供认证令牌")
		c.Abort()
	}
}
