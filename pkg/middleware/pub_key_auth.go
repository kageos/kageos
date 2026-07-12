package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/logger"
)

// PubKeyValidator 验证 pub key 的回调函数，返回用户名
type PubKeyValidator func(key string) (username string, err error)

// JWTOrPubKeyAuth 支持 JWT 或 Pub Key 认证的中间件
// 优先校验 X-Pub-Key（validator），再走 JWT；用户身份只信 Token 或 Pub Key，不信任裸的 X-Request-User
// validator 由调用方提供，通常直接校验调用方自己的 key 存储
func JWTOrPubKeyAuth(validator PubKeyValidator, gatewayVerifiers ...*controlauth.Verifier) gin.HandlerFunc {
	gatewayVerifier := resolveGatewayBackendVerifier(gatewayVerifiers)
	return func(c *gin.Context) {
		if controlauth.HasHTTPMetadata(c.Request.Header) {
			if strings.TrimSpace(c.GetHeader(contextx.PubKeyHerder)) != "" || verifyGatewayBackendIdentity(c, gatewayVerifier) != nil {
				rejectGatewayOrCredential(c)
				return
			}
			c.Next()
			return
		}

		// 1. 优先 Pub Key（X-Pub-Key + validator，防伪造）
		pubKey := strings.TrimSpace(c.GetHeader(contextx.PubKeyHerder))
		if pubKey != "" {
			clearStrictCredentialIdentity(c)
			if validator == nil {
				response.FailWithMessage(c, "无效的 Pub Key")
				c.Abort()
				return
			}
			username, err := validator(pubKey)
			username = strings.TrimSpace(username)
			if err != nil || username == "" {
				logger.Warnf(c, "[JWTOrPubKeyAuth] Pub Key 验证失败: %v", err)
				response.FailWithMessage(c, "无效的 Pub Key")
				c.Abort()
				return
			}
			identity := map[string]string{contextx.RequestUserHeader: username}
			contextx.ApplyTrustedIdentityHeaders(c.Request.Header, identity)
			c.Set(contextx.RequestUserHeader, username)
			c.Set("username", username)
			logger.Infof(c, "[JWTOrPubKeyAuth] Pub Key 认证成功 - User: %s, Path: %s", username, c.Request.URL.Path)
			c.Next()
			return
		}

		// 2. Access/OpenAPI credential. Caller-supplied identity headers are
		// cleared and rebuilt by the shared strict helper.
		if applyStrictExternalCredential(c) {
			c.Next()
			return
		}

		response.NoAuth(c, "未提供认证令牌或 Pub Key")
		c.Abort()
	}
}
