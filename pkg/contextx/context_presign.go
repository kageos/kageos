package contextx

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
)

// PresignHostKey 用于生成预签名 URL 时使用的 Host（与请求 Host 一致，避免 Nginx 代理后签名 403）
type presignHostKeyType struct{}

var PresignHostKey = presignHostKeyType{}

// PresignDefaultPort 当 Host 无端口且未收到 X-Forwarded-Port 时的默认端口（与当前默认 Web 入口端口一致）
const PresignDefaultPort = "8999"

// GetPresignHost 获取用于生成预签名 URL 的 Host（浏览器上传时需与请求 Host 一致，含端口）
// 优先 X-Forwarded-Host；若无端口则用 X-Forwarded-Port 补全，都无则用 PresignDefaultPort 兜底，避免 403
func GetPresignHost(c context.Context) string {
	if v, ok := c.(*gin.Context); ok {
		host := v.GetHeader("X-Forwarded-Host")
		if host == "" {
			host = v.Request.Host
		}
		if host != "" && !strings.Contains(host, ":") {
			port := v.GetHeader("X-Forwarded-Port")
			if port == "" {
				port = PresignDefaultPort
			}
			host = host + ":" + port
		}
		return host
	}
	if value := c.Value(PresignHostKey); value != nil {
		if host, ok := value.(string); ok && host != "" {
			return host
		}
	}
	return ""
}
