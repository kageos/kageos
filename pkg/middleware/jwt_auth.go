package middleware

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// JWTAuth JWT认证中间件
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从X-Token头获取Token
		token := c.GetHeader("X-Token")
		if token == "" {
			response.FailWithMessage(c, "未提供认证令牌")
			c.Abort()
			return
		}

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

		logger.Infof(c, "[JWTAuth] User authenticated: %s (ID: %d)", claims.Username, claims.UserID)
		c.Next()
	}
}
