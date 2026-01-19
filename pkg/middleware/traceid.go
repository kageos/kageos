package middleware

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WithTraceId 为请求添加跟踪ID的中间件
// 如果请求中已有 TraceId（通过 X-Trace-Id header），则复用；否则生成新的
// ⭐ 统一使用常量 TraceIdHeader，不再使用 "trace_id" 字符串 key
func WithTraceId() gin.HandlerFunc {
	return func(c *gin.Context) {
		// ✨ 优先从 HTTP header 读取 TraceId（如果客户端已提供）
		traceId := c.GetHeader(contextx.TraceIdHeader)
		
		// 如果请求中没有 TraceId，则生成新的
		if traceId == "" {
			traceId = uuid.New().String()
			// 将新生成的 TraceId 设置到响应头（方便客户端追踪）
			c.Header(contextx.TraceIdHeader, traceId)
		}
		
		// ⭐ 设置到 context 中（使用常量 TraceIdHeader）
		c.Set(contextx.TraceIdHeader, traceId)
	}
}
