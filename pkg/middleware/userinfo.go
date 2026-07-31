package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/openapitoken"
)

// WithUserInfo 为请求添加用户信息的中间件
// ⭐ 统一使用常量 RequestUserHeader 和 DepartmentFullPathHeader，与 GetRequestUser 保持一致
// ⭐ 如果 X-Request-User header 为空，尝试从 token 中解析用户信息（作为降级方案）
func WithUserInfo(options ...AuthOption) gin.HandlerFunc {
	tokenStore := resolveAuthOptions(options...).openAPITokenStore
	return func(c *gin.Context) {
		// ✨ 优先从 X-Request-User header 读取（网关已设置）
		requestUser := c.GetHeader(contextx.RequestUserHeader)

		if requestUser == "" {
			if rawOpenAPIToken := openapitoken.BearerToken(c.GetHeader("Authorization")); rawOpenAPIToken != "" {
				principal, err := tokenStore.Validate(rawOpenAPIToken, c.ClientIP(), c.GetHeader("User-Agent"))
				if err == nil {
					requestUser = principal.Username
					c.Set("user_id", principal.UserID)
					c.Set("username", principal.Username)
					c.Set("email", principal.Email)
					c.Set("openapi_token_id", principal.TokenID)
					c.Set(contextx.TokenHeader, rawOpenAPIToken)
					c.Request.Header.Set(contextx.TokenHeader, rawOpenAPIToken)
					if principal.DepartmentFullPath != "" {
						c.Set(contextx.DepartmentFullPathHeader, principal.DepartmentFullPath)
						c.Request.Header.Set(contextx.DepartmentFullPathHeader, principal.DepartmentFullPath)
					}
					if principal.CompanyCode != "" {
						c.Set(contextx.CompanyCodeHeader, principal.CompanyCode)
						c.Request.Header.Set(contextx.CompanyCodeHeader, principal.CompanyCode)
					}
					if principal.CompanyName != "" {
						c.Set(contextx.CompanyNameHeader, principal.CompanyName)
						c.Request.Header.Set(contextx.CompanyNameHeader, principal.CompanyName)
					}
					if principal.CompanyLogoURL != "" {
						c.Set(contextx.CompanyLogoURLHeader, principal.CompanyLogoURL)
						c.Request.Header.Set(contextx.CompanyLogoURLHeader, principal.CompanyLogoURL)
					}
					c.Set(contextx.ClientSourceHeader, contextx.ClientSourceOpenAPI)
					c.Request.Header.Set(contextx.ClientSourceHeader, contextx.ClientSourceOpenAPI)
					c.Request.Header.Set(contextx.SourceTypeHeader, contextx.SourceTypeOpenAPIToken)
					c.Request.Header.Set(contextx.SourceRefHeader, principal.Username)
				} else {
					logger.Warnf(c, "[WithUserInfo] OpenAPI token 解析失败 - Path: %s, Error: %v", c.Request.URL.Path, err)
				}
			}
		}

		// ⭐ 如果 header 中没有用户信息，尝试从 token 中解析（降级方案）
		if requestUser == "" {
			token := c.GetHeader(contextx.TokenHeader)
			if token != "" {
				// 尝试解析 token 获取用户信息
				jwtService := auth.NewJWTService()
				claims, err := jwtService.ValidateAccessToken(token)
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
