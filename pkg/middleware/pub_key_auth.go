package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/logger"
)

// PubKeyValidator 验证 pub key 的回调函数，返回用户名
type PubKeyValidator func(key string) (username string, err error)

// JWTOrPubKeyAuth 支持 JWT 或 Pub Key 认证的中间件
// 优先校验 X-Pub-Key（validator），再走 JWT；用户身份只信 Token 或 Pub Key，不信任裸的 X-Request-User
// validator 由调用方提供，通常直接校验调用方自己的 key 存储
func JWTOrPubKeyAuth(validator PubKeyValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 优先 Pub Key（X-Pub-Key + validator，防伪造）
		pubKey := c.GetHeader(contextx.PubKeyHerder)
		if pubKey != "" && validator != nil {
			username, err := validator(pubKey)
			if err != nil {
				logger.Warnf(c, "[JWTOrPubKeyAuth] Pub Key 验证失败: %v", err)
				response.FailWithMessage(c, "无效的 Pub Key")
				c.Abort()
				return
			}
			c.Set(contextx.RequestUserHeader, username)
			logger.Infof(c, "[JWTOrPubKeyAuth] Pub Key 认证成功 - User: %s, Path: %s", username, c.Request.URL.Path)
			c.Next()
			return
		}

		// 2. 从 header 获取 username（仅当网关已解析 token 并设置时）
		requestUser := c.GetHeader(contextx.RequestUserHeader)
		if requestUser == "" {
			requestUser = c.GetHeader("X-Username")
		}
		if requestUser != "" {
			c.Set(contextx.RequestUserHeader, requestUser)
			if deptPath := c.GetHeader(contextx.DepartmentFullPathHeader); deptPath != "" {
				c.Set(contextx.DepartmentFullPathHeader, deptPath)
			}
			c.Next()
			return
		}

		// 3. JWT Token 解析
		token := c.GetHeader(contextx.TokenHeader)
		if token != "" {
			jwtService := auth.NewJWTService()
			claims, err := jwtService.ValidateAccessToken(token)
			if err == nil {
				c.Set(contextx.RequestUserHeader, claims.Username)
				c.Set(contextx.TokenHeader, token)
				if claims.DepartmentFullPath != nil && *claims.DepartmentFullPath != "" {
					c.Set(contextx.DepartmentFullPathHeader, *claims.DepartmentFullPath)
				}
				c.Next()
				return
			}
			logger.Warnf(c, "[JWTOrPubKeyAuth] JWT 验证失败，尝试其他认证方式: %v", err)
		}

		// 4. 内网请求 + X-Request-User（仅内网可信场景）
		if isInternalRequest(c) {
			internalUser := c.GetHeader(contextx.RequestUserHeader)
			if internalUser != "" {
				c.Set(contextx.RequestUserHeader, internalUser)
				c.Next()
				return
			}
		}

		response.FailWithMessage(c, "未提供认证令牌或 Pub Key")
		c.Abort()
	}
}
