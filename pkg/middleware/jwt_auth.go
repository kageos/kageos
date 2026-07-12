package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/controlauth"
)

// JWTAuth is the compatibility name for the strict backend boundary. It does
// not trust loopback or identity headers set by the caller.
func JWTAuth(gatewayVerifiers ...*controlauth.Verifier) gin.HandlerFunc {
	return GatewayOrCredentialAuth(gatewayVerifiers...)
}

// JWTAuthOptional 可选 JWT 认证：有 token 则解析并设置用户，无 token 不拦截，始终 c.Next()
// 用于公开接口（如详情页）需要「有登录则返回 has_starred 等」的场景
func JWTAuthOptional() gin.HandlerFunc {
	return WithUserInfo()
}
