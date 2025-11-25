package contextx

import (
	"context"

	"github.com/gin-gonic/gin"
)

// TraceIdHeader HTTP Header 中的 TraceId key（统一使用此名称）
const TraceIdHeader = "X-Trace-Id"

// RequestUserHeader HTTP Header 中的 RequestUser key（统一使用此名称）
const RequestUserHeader = "X-Request-User"

// TokenHeader HTTP Header 中的 Token key（统一使用此名称）
const TokenHeader = "X-Token"

// GetTraceId 获取追踪ID
// 优先级：HTTP Header (X-Trace-Id) > Context (trace_id)
// 支持从 *gin.Context 或标准 context.Context 读取
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
	
	// 从标准 context.Value 读取（可能是 ToContext 转换后的标准 context）
	if value := c.Value("trace_id"); value != nil {
		if traceId, ok := value.(string); ok && traceId != "" {
			return traceId
		}
	}
	
	return ""
}

// GetRequestUser 获取请求用户（从 HTTP header 或 context 读取）
// 优先级：HTTP Header (X-Request-User) > Context (request_user)
// 网关已解析 token 并设置到 X-Request-User header，直接从 header 读取即可
// 支持从 *gin.Context 或标准 context.Context 读取
func GetRequestUser(c context.Context) string {
	// 首先尝试转换为 *gin.Context（可以读取 header）
	v, ok := c.(*gin.Context)
	if ok {
		// ✨ 优先从 HTTP header 读取（网关已设置）
		if requestUser := v.GetHeader(RequestUserHeader); requestUser != "" {
			return requestUser
		}
		// 备用：从 X-Username header 读取
		if requestUser := v.GetHeader("X-Username"); requestUser != "" {
			return requestUser
		}
		// 从 gin context 读取（兼容旧方式）
		if requestUser := v.GetString("request_user"); requestUser != "" {
			return requestUser
		}
		// 从 context.Value 读取（JWTAuth 中间件通过 c.Set() 设置到 Keys 中）
		if value := c.Value("request_user"); value != nil {
			if requestUser, ok := value.(string); ok && requestUser != "" {
				return requestUser
			}
		}
		return ""
	}
	
	// 从标准 context.Value 读取（可能是 ToContext 转换后的标准 context，或 context.WithValue 包装的）
	// context.WithValue 创建的新 context 的 Value() 会向上查找父 context
	if value := c.Value("request_user"); value != nil {
		if requestUser, ok := value.(string); ok && requestUser != "" {
			return requestUser
		}
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
		return v.GetHeader(TokenHeader)
	}
	// 从 context.Value 获取（可能是 ToContext 转换后的标准 context）
	if value := c.Value("token"); value != nil {
		if token, ok := value.(string); ok && token != "" {
			return token
		}
	}
	return ""
}

// ToContext 将 gin.Context 转换为标准 context.Context
// 解析 header 中的关键信息（trace_id, request_user, token）并放入 context.Value
// 这样即使内部使用 context.WithValue 包装，也能通过 context.Value 获取到这些值
func ToContext(c *gin.Context) context.Context {
	ctx := context.Background()
	
	// 1. 解析 TraceId（优先从 header，然后从 gin context）
	traceId := c.GetHeader(TraceIdHeader)
	if traceId == "" {
		traceId = c.GetString("trace_id")
	}
	if traceId != "" {
		ctx = context.WithValue(ctx, "trace_id", traceId)
	}
	
	// 2. 解析 RequestUser（优先从 header，然后从 gin context）
	requestUser := c.GetHeader(RequestUserHeader)
	if requestUser == "" {
		requestUser = c.GetHeader("X-Username")
	}
	if requestUser == "" {
		requestUser = c.GetString("request_user")
	}
	if requestUser != "" {
		ctx = context.WithValue(ctx, "request_user", requestUser)
	}
	
	// 3. 解析 Token（优先从 gin context，然后从 header）
	token := c.GetString("token")
	if token == "" {
		token = c.GetHeader(TokenHeader)
	}
	if token != "" {
		ctx = context.WithValue(ctx, "token", token)
	}
	
	return ctx
}
