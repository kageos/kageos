package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/openapitoken"
)

type AuthOption func(*authOptions)

type authOptions struct {
	openAPITokenStore *openapitoken.Store
}

func WithOpenAPITokenStore(store *openapitoken.Store) AuthOption {
	return func(options *authOptions) {
		options.openAPITokenStore = store
	}
}

func resolveAuthOptions(options ...AuthOption) authOptions {
	resolved := authOptions{}
	for _, option := range options {
		if option != nil {
			option(&resolved)
		}
	}
	return resolved
}

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
func JWTAuth(options ...AuthOption) gin.HandlerFunc {
	tokenStore := resolveAuthOptions(options...).openAPITokenStore
	return func(c *gin.Context) {
		if rawOpenAPIToken := openapitoken.BearerToken(c.GetHeader("Authorization")); rawOpenAPIToken != "" {
			principal, err := tokenStore.Validate(rawOpenAPIToken, c.ClientIP(), c.GetHeader("User-Agent"))
			if err != nil {
				logger.Errorf(c, "[JWTAuth] OpenAPI token validation failed: %v", err)
				response.FailWithMessage(c, "OpenAPI Token 无效或已过期")
				c.Abort()
				return
			}
			c.Set("user_id", principal.UserID)
			c.Set("username", principal.Username)
			c.Set("email", principal.Email)
			c.Set("openapi_token_id", principal.TokenID)
			c.Set(contextx.RequestUserHeader, principal.Username)
			c.Set(contextx.TokenHeader, rawOpenAPIToken)
			c.Request.Header.Set(contextx.RequestUserHeader, principal.Username)
			c.Request.Header.Set(contextx.TokenHeader, rawOpenAPIToken)
			if principal.DepartmentFullPath != "" {
				c.Set(contextx.DepartmentFullPathHeader, principal.DepartmentFullPath)
				c.Request.Header.Set(contextx.DepartmentFullPathHeader, principal.DepartmentFullPath)
			}
			c.Set(contextx.ClientSourceHeader, contextx.ClientSourceOpenAPI)
			c.Request.Header.Set(contextx.ClientSourceHeader, contextx.ClientSourceOpenAPI)
			c.Request.Header.Set(contextx.SourceTypeHeader, contextx.SourceTypeOpenAPIToken)
			c.Request.Header.Set(contextx.SourceRefHeader, principal.Username)

			c.Next()
			return
		}

		// 网关会移除客户端伪造的身份 Header，并仅在认证成功后重新写入。
		requestUser := c.GetHeader(contextx.RequestUserHeader)
		if requestUser == "" {
			requestUser = c.GetHeader("X-Username")
		}
		if requestUser != "" {
			c.Set(contextx.RequestUserHeader, requestUser)
			if deptPath := c.GetHeader(contextx.DepartmentFullPathHeader); deptPath != "" {
				c.Set(contextx.DepartmentFullPathHeader, deptPath)
			}
			logger.Debugf(c, "[JWTAuth] 从可信网关 header 获取用户信息 - User: %s, Path: %s", requestUser, c.Request.URL.Path)
			c.Next()
			return
		}

		// 如果header中没有username，尝试解析token（向后兼容）
		// ⭐ 只从 header 读取 token
		token := c.GetHeader(contextx.TokenHeader)

		// ✅ 如果有token，使用token验证（Web端调用）
		if token != "" {
			// 验证Token
			jwtService := auth.NewJWTService()
			claims, err := jwtService.ValidateAccessToken(token)
			if err != nil {
				logger.Errorf(c, "[JWTAuth] Token validation failed: %v", err)
				response.FailWithMessage(c, "认证令牌无效或已过期")
				c.Abort()
				return
			}

			// ⭐ 设置用户信息到上下文（统一使用常量）
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("email", claims.Email)
			c.Set(contextx.RequestUserHeader, claims.Username) // ⭐ 统一使用常量 RequestUserHeader
			c.Set(contextx.TokenHeader, token)                 // ⭐ 统一使用常量 TokenHeader

			// ⭐ 设置组织架构信息到上下文（token 中一定包含这些字段，如果用户有组织架构信息）
			// ⭐ 统一使用 DepartmentFullPathHeader 常量
			if claims.DepartmentFullPath != nil && *claims.DepartmentFullPath != "" {
				c.Set(contextx.DepartmentFullPathHeader, *claims.DepartmentFullPath)
			}
			c.Next()
			return
		}

		// ✅ 如果没有token，检查是否为内网请求（SDK内部调用）
		if isInternalRequest(c) {
			// 从header获取用户信息（SDK传入）
			requestUser := c.GetHeader(contextx.RequestUserHeader)
			if requestUser == "" {
				logger.Warnf(c, "[JWTAuth] 内网请求缺少 %s header - Path: %s, IP: %s", contextx.RequestUserHeader, c.Request.URL.Path, c.ClientIP())
				response.FailWithMessage(c, "内网请求必须提供X-Request-User头")
				c.Abort()
				return
			}

			// ⭐ 设置用户信息（统一使用常量 RequestUserHeader）
			c.Set(contextx.RequestUserHeader, requestUser)

			c.Next()
			return
		}

		// 外部请求且没有token，拒绝
		logger.Warnf(c, "[JWTAuth] 外部请求缺少认证令牌 - Path: %s, IP: %s", c.Request.URL.Path, c.ClientIP())
		response.FailWithMessage(c, "未提供认证令牌")
		c.Abort()
	}
}

// JWTAuthOptional 可选 JWT 认证：有 token 则解析并设置用户，无 token 不拦截，始终 c.Next()
// 用于公开接口（如详情页）需要「有登录则返回 has_starred 等」的场景
func JWTAuthOptional() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestUser := c.GetHeader(contextx.RequestUserHeader)
		if requestUser == "" {
			requestUser = c.GetHeader("X-Username")
		}
		if requestUser != "" {
			c.Set(contextx.RequestUserHeader, requestUser)
			c.Next()
			return
		}
		token := c.GetHeader(contextx.TokenHeader)
		if token != "" {
			jwtService := auth.NewJWTService()
			claims, err := jwtService.ValidateAccessToken(token)
			if err == nil {
				c.Set(contextx.RequestUserHeader, claims.Username)
			}
		}
		c.Next()
	}
}
