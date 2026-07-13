package middleware

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/config"
)

const corsAllowedOriginsEnv = "KAGEOS_CORS_ALLOWED_ORIGINS"

// Cors 只允许显式配置的浏览器来源。无 Origin 的服务间请求不受影响。
// 未传 allowedOrigins 时使用主站地址和 KAGEOS_CORS_ALLOWED_ORIGINS（逗号分隔）。
func Cors(allowedOrigins ...string) gin.HandlerFunc {
	allowed := buildCORSOriginSet(allowedOrigins)
	return func(c *gin.Context) {
		origin := normalizeCORSOrigin(c.GetHeader("Origin"))
		if origin == "" {
			c.Next()
			return
		}
		if _, ok := allowed[origin]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    "origin_not_allowed",
				"message": "请求来源不在允许列表中",
			})
			return
		}

		c.Writer.Header().Add("Vary", "Origin")
		c.Writer.Header().Add("Vary", "Access-Control-Request-Method")
		c.Writer.Header().Add("Vary", "Access-Control-Request-Headers")
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Token, Accept, Origin, Cache-Control, X-Requested-With, X-Function-ID, X-Trace-ID, X-Public-Anonymous-Token, Idempotency-Key")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func buildCORSOriginSet(explicit []string) map[string]struct{} {
	values := append([]string(nil), explicit...)
	if len(values) == 0 {
		values = append(values, config.GetPublicSiteBaseURL())
		values = append(values, strings.Split(os.Getenv(corsAllowedOriginsEnv), ",")...)
	}
	allowed := make(map[string]struct{}, len(values))
	for _, value := range values {
		if origin := normalizeCORSOrigin(value); origin != "" {
			allowed[origin] = struct{}{}
		}
	}
	return allowed
}

func normalizeCORSOrigin(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "*" || raw == "null" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return ""
	}
	return strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host)
}
