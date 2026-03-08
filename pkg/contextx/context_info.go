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
const PubKeyHerder = "X-Pub-Key"

// PresignHostKey 用于生成预签名 URL 时使用的 Host（与请求 Host 一致，避免 Nginx 代理后签名 403）
type presignHostKeyType struct{}

var PresignHostKey = presignHostKeyType{}

// GetTraceId 获取追踪ID
// ⭐ 只从 HTTP Header 读取（统一方式，避免混乱）
// 支持从 *gin.Context 或标准 context.Context 读取
func GetTraceId(c context.Context) string {
	v, ok := c.(*gin.Context)
	if ok {
		// ✨ 只从 HTTP header 读取
		return v.GetHeader(TraceIdHeader)
	}

	// 从标准 context.Value 读取（可能是 ToContext 转换后的标准 context，或 context.WithValue 包装的）
	if value := c.Value(TraceIdHeader); value != nil {
		if traceId, ok := value.(string); ok && traceId != "" {
			return traceId
		}
	}

	return ""
}

// GetRequestUser 获取请求用户
// ⭐ 只从 HTTP Header 读取（统一方式，避免混乱）
// 支持从 *gin.Context 或标准 context.Context 读取
func GetRequestUser(c context.Context) string {
	// 首先尝试转换为 *gin.Context（可以读取 header）
	v, ok := c.(*gin.Context)
	if ok {
		// ✨ 只从 HTTP header 读取
		requestUser := v.GetHeader(RequestUserHeader)
		if requestUser == "" {
			// ⭐ 如果 header 为空，打印警告日志（包含更多调试信息）
			token := v.GetHeader(TokenHeader)
			logger.Warnf(c, "[GetRequestUser] 无法获取 RequestUser - Path: %s, IP: %s, HasToken: %v, TokenLength: %d, X-Request-User Header: %s",
				v.Request.URL.Path, v.ClientIP(), token != "", len(token), v.GetHeader(RequestUserHeader))
		}
		return requestUser
	}

	// 从标准 context.Value 读取（可能是 ToContext 转换后的标准 context，或 context.WithValue 包装的）
	if value := c.Value(RequestUserHeader); value != nil {
		if requestUser, ok := value.(string); ok && requestUser != "" {
			return requestUser
		}
	}

	return ""
}

// GetRequestDepartmentFullPath 获取请求用户的组织架构路径
// ⭐ 只从 HTTP Header 读取（统一方式，避免混乱）
// 支持从 *gin.Context 或标准 context.Context 读取
func GetRequestDepartmentFullPath(c context.Context) string {
	// 首先尝试转换为 *gin.Context（可以读取 header）
	v, ok := c.(*gin.Context)
	if ok {
		// ✨ 只从 HTTP header 读取
		return v.GetHeader(DepartmentFullPathHeader)
	}

	// 从标准 context.Value 读取（可能是 ToContext 转换后的标准 context，或 context.WithValue 包装的）
	if value := c.Value(DepartmentFullPathHeader); value != nil {
		if deptPath, ok := value.(string); ok && deptPath != "" {
			return deptPath
		}
	}

	return ""
}

// GetToken 获取认证 Token
// ⭐ 只从 HTTP Header 读取（统一方式，避免混乱）
func GetToken(c context.Context) string {
	v, ok := c.(*gin.Context)
	if ok {
		// ✨ 只从 HTTP header 读取
		return v.GetHeader(TokenHeader)
	}
	// 从标准 context.Value 读取（可能是 ToContext 转换后的标准 context，或 context.WithValue 包装的）
	if value := c.Value(TokenHeader); value != nil {
		if token, ok := value.(string); ok && token != "" {
			return token
		}
	}
	return ""
}

// GetPresignHost 获取用于生成预签名 URL 的 Host（浏览器上传时需与请求 Host 一致）
func GetPresignHost(c context.Context) string {
	if v, ok := c.(*gin.Context); ok {
		return v.Request.Host
	}
	if value := c.Value(PresignHostKey); value != nil {
		if host, ok := value.(string); ok && host != "" {
			return host
		}
	}
	return ""
}

// ToContext 将 gin.Context 转换为标准 context.Context
// 从 header 或 gin 上下文（如中间件 c.Set）读取关键信息，写入 context.Value，并同步回 c.Request.Header，保证请求头为权威来源。
func ToContext(c *gin.Context) context.Context {
	ctx := context.Background()

	// 1. TraceId：header 或 context，取到后 set 回 header + context
	traceId := c.GetHeader(TraceIdHeader)
	if traceId != "" {
		ctx = context.WithValue(ctx, TraceIdHeader, traceId)
		c.Request.Header.Set(TraceIdHeader, traceId)
	}

	// 2. RequestUser：优先 header，若无则从 gin 上下文取（中间件从 JWT/PubKey 解析后 c.Set 的值），取到后 set 回 header + context
	requestUser := c.GetHeader(RequestUserHeader)
	if requestUser == "" {
		if v, exists := c.Get(RequestUserHeader); exists {
			if s, ok := v.(string); ok && s != "" {
				requestUser = s
			}
		}
	}
	if requestUser != "" {
		ctx = context.WithValue(ctx, RequestUserHeader, requestUser)
		c.Request.Header.Set(RequestUserHeader, requestUser)
	}

	// 3. Token：header 或 context，取到后 set 回 header + context
	token := c.GetHeader(TokenHeader)
	if token == "" {
		if v, exists := c.Get(TokenHeader); exists {
			if s, ok := v.(string); ok && s != "" {
				token = s
			}
		}
	}
	if token != "" {
		ctx = context.WithValue(ctx, TokenHeader, token)
		c.Request.Header.Set(TokenHeader, token)
	}

	// 4. DepartmentFullPath：header 或 context，取到后 set 回 header + context
	deptPath := c.GetHeader(DepartmentFullPathHeader)
	if deptPath == "" {
		if v, exists := c.Get(DepartmentFullPathHeader); exists {
			if s, ok := v.(string); ok && s != "" {
				deptPath = s
			}
		}
	}
	if deptPath != "" {
		ctx = context.WithValue(ctx, DepartmentFullPathHeader, deptPath)
		c.Request.Header.Set(DepartmentFullPathHeader, deptPath)
	}

	// 5. PresignHost：请求的 Host，用于存储服务生成预签名 URL 时与 Nginx 转发 Host 一致，避免 403
	if c.Request.Host != "" {
		ctx = context.WithValue(ctx, PresignHostKey, c.Request.Host)
	}

	return ctx
}

func NatsTraceContext(msg *nats.Msg) context.Context {
	//从nats 取出用户信息相关
	background := context.Background()
	ctx := context.WithValue(background, RequestUserHeader, msg.Header.Get(RequestUserHeader))
	ctx = context.WithValue(ctx, TokenHeader, msg.Header.Get(TokenHeader))
	ctx = context.WithValue(ctx, TraceIdHeader, msg.Header.Get(TraceIdHeader))
	ctx = context.WithValue(ctx, DepartmentFullPathHeader, msg.Header.Get(DepartmentFullPathHeader))

	return ctx
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
