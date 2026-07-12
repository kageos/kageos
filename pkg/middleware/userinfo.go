package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/publicshare"
)

// WithUserInfo is optional identity enrichment. It may rebuild identity from a
// real access/OpenAPI credential, but never from caller-supplied headers. An
// attempted internal signature is rejected instead of silently downgraded.
func WithUserInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
		if controlauth.HasHTTPMetadata(c.Request.Header) {
			rejectGatewayOrCredential(c)
			return
		}
		// Optional routes continue anonymously when a credential is absent or
		// invalid, but all forged identity metadata is cleared either way.
		applyOptionalExternalCredential(c)
		c.Next()
	}
}

func applyOptionalExternalCredential(c *gin.Context) {
	if c == nil || c.Request == nil {
		return
	}
	preserved := make(map[string][]string, 4)
	for _, name := range []string{
		contextx.TokenHeader,
		"Authorization",
		contextx.PubKeyHerder,
		publicshare.AnonymousTokenHeader,
	} {
		if values, ok := c.Request.Header[name]; ok {
			preserved[name] = append([]string(nil), values...)
		}
	}
	if applyStrictExternalCredential(c) {
		return
	}
	for name, values := range preserved {
		c.Request.Header[name] = values
	}
}
