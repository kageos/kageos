package middleware

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// AccessLog 访问日志中间件
// 记录所有经过网关的请求，包括：请求方法、路径、状态码、响应时间、TraceId、用户信息等
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录请求开始时间
		startTime := time.Now()

		// 处理请求
		c.Next()

		// 计算响应时间
		latency := time.Since(startTime)

		// 获取 TraceId
		traceId := contextx.GetTraceId(c)
		if traceId == "" {
			traceId = c.GetString("trace_id")
		}

		// 获取用户信息
		requestUser := contextx.GetRequestUser(c)
		if requestUser == "" {
			requestUser = c.GetString("request_user")
		}
		if requestUser == "" {
			requestUser = "-"
		}

		// 获取客户端IP
		clientIP := c.ClientIP()

		// 记录访问日志
		// 格式：方法 路径 状态码 响应时间 TraceId 用户 IP
		logger.Infof(c, "[AccessLog] %s %s %d %v TraceId:%s User:%s IP:%s",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			latency,
			traceId,
			requestUser,
			clientIP,
		)
	}
}

