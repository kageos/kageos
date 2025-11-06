package contextx

import (
	"context"

	"github.com/gin-gonic/gin"
)

// TraceIdHeader HTTP Header 中的 TraceId key（统一使用此名称）
const TraceIdHeader = "X-Trace-Id"

// GetTraceId 获取追踪ID
// 优先级：HTTP Header (X-Trace-Id) > Context (trace_id)
func GetTraceId(c context.Context) string {
	v, ok := c.(*gin.Context)
	if ok {
		// ✨ 优先从 HTTP header 读取（如果网关已设置）
		if traceId := v.GetHeader(TraceIdHeader); traceId != "" {
			return traceId
		}
		
		// 从 context 读取（由中间件设置）
		if value := c.Value("trace_id"); value != nil {
			if traceId, ok := value.(string); ok && traceId != "" {
				return traceId
			}
		}
		
		// 从 gin context 读取（兼容旧方式）
		return v.GetString("trace_id")
	}
	return ""
}

func GetUserInfo(c context.Context) string {
	v, ok := c.(*gin.Context)
	if ok {
		value := c.Value("user")
		return value.(string)
	}
	return v.GetString("user")
}

func GetTenantUser(c context.Context) string {
	v, ok := c.(*gin.Context)
	if ok {
		value := c.Value("tenant_user")
		if value != nil {
			return value.(string)
		}
	}
	if v != nil {
		return v.GetString("tenant_user")
	}
	return ""
}

func GetRequestUser(c context.Context) string {
	v, ok := c.(*gin.Context)
	if ok {
		value := c.Value("request_user")
		if value != nil {
			return value.(string)
		}
	}
	if v != nil {
		return v.GetString("request_user")
	}
	return ""
}

// GetToken 获取认证 Token（从 HTTP header 或 context）
func GetToken(c context.Context) string {
	v, ok := c.(*gin.Context)
	if ok {
		// 优先从 context 获取（由 JWT 中间件设置）
		value := c.Value("token")
		if value != nil {
			if token, ok := value.(string); ok && token != "" {
				return token
			}
		}
		// 从 HTTP header 获取
		return v.GetHeader("X-Token")
	}
	return ""
}
