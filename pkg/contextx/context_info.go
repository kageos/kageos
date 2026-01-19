package contextx

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/nats-io/nats.go"

	"github.com/gin-gonic/gin"
)

// TraceIdHeader HTTP Header 中的 TraceId key（统一使用此名称）
const TraceIdHeader = "X-Trace-Id"

// RequestUserHeader HTTP Header 中的 RequestUser key（统一使用此名称）
const RequestUserHeader = "X-Request-User"

// DepartmentFullPathHeader HTTP Header 和 Context 中的 DepartmentFullPath key（统一使用此名称）
// ⭐ 统一使用此常量，不要硬编码字符串（既用于 HTTP Header，也用于 Context）
const DepartmentFullPathHeader = "X-Department-Full-Path"

// TokenHeader HTTP Header 中的 Token key（统一使用此名称）
const TokenHeader = "X-Token"

// GetTraceId 获取追踪ID
// 优先级：HTTP Header (X-Trace-Id) > Context (TraceIdHeader)
// 支持从 *gin.Context 或标准 context.Context 读取
// ⭐ 只从常量 TraceIdHeader 读取，不再使用 "trace_id" 字符串 key
func GetTraceId(c context.Context) string {
	v, ok := c.(*gin.Context)
	if ok {
		// ✨ 优先从 HTTP header 读取（如果网关已设置）
		if traceId := v.GetHeader(TraceIdHeader); traceId != "" {
			return traceId
		}

		// 从 context 读取（使用常量 TraceIdHeader）
		if value := c.Value(TraceIdHeader); value != nil {
			if traceId, ok := value.(string); ok && traceId != "" {
				return traceId
			}
		}

		// 从 gin context 读取（使用常量 TraceIdHeader）
		if traceId := v.GetString(TraceIdHeader); traceId != "" {
			return traceId
		}

		// ⭐ 如果都没取到值，打印警告日志
		logger.Warnf(c, "[GetTraceId] 无法获取 TraceId - Path: %s", v.Request.URL.Path)
		return ""
	}

	// 从标准 context.Value 读取（使用常量 TraceIdHeader）
	if value := c.Value(TraceIdHeader); value != nil {
		if traceId, ok := value.(string); ok && traceId != "" {
			return traceId
		}
	}

	// ⭐ 如果都没取到值，打印警告日志
	logger.Warnf(context.Background(), "[GetTraceId] 无法获取 TraceId - Context 类型: %T", c)
	return ""
}

// GetRequestUser 获取请求用户（从 HTTP header 或 context 读取）
// 优先级：HTTP Header (X-Request-User) > Gin Context > Context.Value
// 网关已解析 token 并设置到 X-Request-User header，直接从 header 读取即可
// 支持从 *gin.Context 或标准 context.Context 读取
// ⭐ 统一使用常量 RequestUserHeader
func GetRequestUser(c context.Context) string {
	// 首先尝试转换为 *gin.Context（可以读取 header）
	v, ok := c.(*gin.Context)
	if ok {
		// ✨ 优先从 HTTP header 读取（网关已设置）
		if requestUser := v.GetHeader(RequestUserHeader); requestUser != "" {
			return requestUser
		}
		// ⭐ 降级：从 gin context 读取（使用常量 RequestUserHeader）
		if requestUser := v.GetString(RequestUserHeader); requestUser != "" {
			return requestUser
		}
		// ⭐ 降级：从 context.Value 读取
		if value := c.Value(RequestUserHeader); value != nil {
			if requestUser, ok := value.(string); ok && requestUser != "" {
				return requestUser
			}
		}
		// ⭐ 如果都没取到值，打印警告日志
		logger.Warnf(c, "[GetRequestUser] 无法获取 RequestUser - Path: %s", v.Request.URL.Path)
		return ""
	}

	// 从标准 context.Value 读取（可能是 ToContext 转换后的标准 context，或 context.WithValue 包装的）
	// context.WithValue 创建的新 context 的 Value() 会向上查找父 context
	if value := c.Value(RequestUserHeader); value != nil {
		if requestUser, ok := value.(string); ok && requestUser != "" {
			return requestUser
		}
	}

	// ⭐ 如果都没取到值，打印警告日志
	logger.Warnf(context.Background(), "[GetRequestUser] 无法获取 RequestUser - Context 类型: %T", c)
	return ""
}

// GetRequestDepartmentFullPath 获取请求用户的组织架构路径（从 HTTP header 或 context 读取）
// 优先级：HTTP Header (X-Department-Full-Path) > Context (X-Department-Full-Path)
// 支持从 *gin.Context 或标准 context.Context 读取
// ⭐ 统一使用 DepartmentFullPathHeader 常量
func GetRequestDepartmentFullPath(c context.Context) string {
	// 首先尝试转换为 *gin.Context（可以读取 header）
	v, ok := c.(*gin.Context)
	if ok {
		// ✨ 优先从 HTTP header 读取（网关已设置）
		if deptPath := v.GetHeader(DepartmentFullPathHeader); deptPath != "" {
			return deptPath
		}
		// 从 gin context 读取（JWTAuth 中间件通过 c.Set() 设置）
		if deptPath := v.GetString(DepartmentFullPathHeader); deptPath != "" {
			return deptPath
		}
		// 从 context.Value 读取
		if value := c.Value(DepartmentFullPathHeader); value != nil {
			if deptPath, ok := value.(string); ok && deptPath != "" {
				return deptPath
			}
		}
		return ""
	}

	// 从标准 context.Value 读取（可能是 ToContext 转换后的标准 context，或 context.WithValue 包装的）
	if value := c.Value(DepartmentFullPathHeader); value != nil {
		if deptPath, ok := value.(string); ok && deptPath != "" {
			return deptPath
		}
	}

	return ""
}

// GetToken 获取认证 Token（从 HTTP header 或 context）
// ⭐ 只从常量 TokenHeader 读取，不再使用 "token" 字符串 key
func GetToken(c context.Context) string {
	v, ok := c.(*gin.Context)
	if ok {
		// 优先从 gin context 获取（使用常量 TokenHeader）
		if token := v.GetString(TokenHeader); token != "" {
			return token
		}
		// 降级：从 context.Value 获取
		if value := c.Value(TokenHeader); value != nil {
			if token, ok := value.(string); ok && token != "" {
				return token
			}
		}
		// 降级：从 HTTP header 获取
		if token := v.GetHeader(TokenHeader); token != "" {
			return token
		}
		// ⭐ 如果都没取到值，打印警告日志
		logger.Warnf(c, "[GetToken] 无法获取 Token - Path: %s", v.Request.URL.Path)
		return ""
	}
	// 从 context.Value 获取（使用常量 TokenHeader）
	if value := c.Value(TokenHeader); value != nil {
		if token, ok := value.(string); ok && token != "" {
			return token
		}
	}
	// ⭐ 如果都没取到值，打印警告日志
	logger.Warnf(context.Background(), "[GetToken] 无法获取 Token - Context 类型: %T", c)
	return ""
}

// ToContext 将 gin.Context 转换为标准 context.Context
// 解析 header 中的关键信息并放入 context.Value
// ⭐ 统一使用常量 key（TraceIdHeader、RequestUserHeader、TokenHeader、DepartmentFullPathHeader）
// 这样即使内部使用 context.WithValue 包装，也能通过 context.Value 获取到这些值
func ToContext(c *gin.Context) context.Context {
	ctx := context.Background()

	// 1. 解析 TraceId（优先从 header，然后从 gin context）
	traceId := c.GetHeader(TraceIdHeader)
	if traceId == "" {
		traceId = c.GetString(TraceIdHeader) // ⭐ 使用常量 TraceIdHeader
	}
	if traceId != "" {
		ctx = context.WithValue(ctx, TraceIdHeader, traceId) // ⭐ 只使用常量 TraceIdHeader
	}

	// 2. 解析 RequestUser（优先从 header，然后从 gin context）
	requestUser := c.GetHeader(RequestUserHeader)
	if requestUser == "" {
		requestUser = c.GetString(RequestUserHeader) // ⭐ 使用常量 RequestUserHeader
	}
	if requestUser != "" {
		ctx = context.WithValue(ctx, RequestUserHeader, requestUser)
	}

	// 3. 解析 Token（优先从 gin context，然后从 header）
	token := c.GetString(TokenHeader) // ⭐ 使用常量 TokenHeader
	if token == "" {
		token = c.GetHeader(TokenHeader)
	}
	if token != "" {
		ctx = context.WithValue(ctx, TokenHeader, token) // ⭐ 只使用常量 TokenHeader
	}

	// 4. 解析 DepartmentFullPath（优先从 header，然后从 gin context）
	// ⭐ 统一使用 DepartmentFullPathHeader 常量
	deptPath := c.GetHeader(DepartmentFullPathHeader)
	if deptPath == "" {
		deptPath = c.GetString(DepartmentFullPathHeader)
	}
	if deptPath != "" {
		ctx = context.WithValue(ctx, DepartmentFullPathHeader, deptPath)
	}

	return ctx
}

func NatsTraceContext(msg *nats.Msg) context.Context {
	//从nats 取出用户信息相关
	background := context.Background()
	ctx := context.WithValue(background, RequestUserHeader, msg.Header.Get(RequestUserHeader))
	ctx = context.WithValue(ctx, TokenHeader, msg.Header.Get(TokenHeader))
	ctx = context.WithValue(ctx, TraceIdHeader, msg.Header.Get(TraceIdHeader))

	return ctx
}

// NewNatsMsg 需要携带尽可能多的信息，例如请求用户，trace_id
func NewNatsTraceMsg(subject string, requestUser string, traceID string, token string) *nats.Msg {
	msg := nats.NewMsg(subject)
	msg.Header.Set(TraceIdHeader, traceID)
	msg.Header.Set(TokenHeader, token)
	msg.Header.Set(RequestUserHeader, requestUser)
	return msg
}

func CtxToTraceNats(c context.Context, subject string) *nats.Msg {
	user := GetRequestUser(c)
	token := GetToken(c)
	trace := GetTraceId(c)

	msg := nats.NewMsg(subject)
	msg.Header.Set(TraceIdHeader, trace)
	msg.Header.Set(TokenHeader, token)
	msg.Header.Set(RequestUserHeader, user)
	return msg

}
