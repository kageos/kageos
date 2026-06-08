package contextx

import (
	"context"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
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

const CompanyCodeHeader = "X-Company-Code"
const CompanyNameHeader = "X-Company-Name"
const CompanyLogoURLHeader = "X-Company-Logo-Url"

// TokenHeader HTTP Header 中的 Token key（统一使用此名称）
const TokenHeader = "X-Token"

// ClientSourceHeader HTTP Header 中的客户端来源 key（统一使用此名称）
const ClientSourceHeader = "X-Client-Source"

const (
	ClientSourceBrowser     = "browser"
	ClientSourceAgent       = "agent"
	ClientSourceOpenAPI     = "openapi"
	ClientSourcePublicShare = "public_share"
	ClientSourceUnknown     = "unknown"
)

// SourceTypeHeader / SourceRefHeader 标记后台自动化、函数触发等调用来源。
// 定时 Agent 会话先埋 ref，后续工具白名单可基于这些字段在工具入口统一控制。
const SourceTypeHeader = "X-Source-Type"
const SourceRefHeader = "X-Source-Ref"

const (
	SourceTypeOpenAPIToken = "openapi_token"
	SourceTypePublicShare  = "public_share"
	SourceTypeAgentTool    = "agent_tool"
)

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

func GetRequestCompanyCode(c context.Context) string {
	return getStringFromContextOrHeader(c, CompanyCodeHeader)
}

func GetRequestCompanyName(c context.Context) string {
	return getStringFromContextOrHeader(c, CompanyNameHeader)
}

func GetRequestCompanyLogoURL(c context.Context) string {
	return getStringFromContextOrHeader(c, CompanyLogoURLHeader)
}

func getStringFromContextOrHeader(c context.Context, key string) string {
	if v, ok := c.(*gin.Context); ok {
		return v.GetHeader(key)
	}
	if value := c.Value(key); value != nil {
		if s, ok := value.(string); ok && s != "" {
			return s
		}
	}
	return ""
}

// GetClientSource 获取客户端来源
// 支持从 *gin.Context 或标准 context.Context 读取
func GetClientSource(c context.Context) string {
	v, ok := c.(*gin.Context)
	if ok {
		return v.GetHeader(ClientSourceHeader)
	}

	if value := c.Value(ClientSourceHeader); value != nil {
		if source, ok := value.(string); ok && source != "" {
			return source
		}
	}

	return ""
}

// ResolveClientSource 返回审计使用的入口来源。优先使用 X-Client-Source；
// 没有显式来源时根据 SourceType 推断 OpenAPI、公开分享或智能体工具。
func ResolveClientSource(c context.Context) string {
	source := strings.TrimSpace(GetClientSource(c))
	switch strings.ToLower(source) {
	case "api":
		return ClientSourceOpenAPI
	case "":
	default:
		return source
	}

	switch strings.TrimSpace(GetSourceType(c)) {
	case SourceTypeOpenAPIToken:
		return ClientSourceOpenAPI
	case SourceTypePublicShare:
		return ClientSourcePublicShare
	case SourceTypeAgentTool:
		return ClientSourceAgent
	default:
		return ""
	}
}

// GetAuditClientSource 返回非空审计来源，避免 operate_logs.source 出现空值。
func GetAuditClientSource(c context.Context) string {
	if source := ResolveClientSource(c); source != "" {
		return source
	}
	return ClientSourceUnknown
}

// GetSourceType 获取调用来源类型。
func GetSourceType(c context.Context) string {
	v, ok := c.(*gin.Context)
	if ok {
		return v.GetHeader(SourceTypeHeader)
	}
	if value := c.Value(SourceTypeHeader); value != nil {
		if sourceType, ok := value.(string); ok && sourceType != "" {
			return sourceType
		}
	}
	return ""
}

// GetSourceRef 获取调用来源引用。
func GetSourceRef(c context.Context) string {
	v, ok := c.(*gin.Context)
	if ok {
		return v.GetHeader(SourceRefHeader)
	}
	if value := c.Value(SourceRefHeader); value != nil {
		if sourceRef, ok := value.(string); ok && sourceRef != "" {
			return sourceRef
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

// WithClientSource 为标准 context 写入客户端来源；空值时返回原 context
func WithClientSource(ctx context.Context, source string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	source = strings.TrimSpace(source)
	if source == "" {
		return ctx
	}
	return context.WithValue(ctx, ClientSourceHeader, source)
}

// WithSourceInfo 为标准 context 写入调用来源类型和引用。
func WithSourceInfo(ctx context.Context, sourceType, sourceRef string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	sourceType = strings.TrimSpace(sourceType)
	sourceRef = strings.TrimSpace(sourceRef)
	if sourceType != "" {
		ctx = context.WithValue(ctx, SourceTypeHeader, sourceType)
	}
	if sourceRef != "" {
		ctx = context.WithValue(ctx, SourceRefHeader, sourceRef)
	}
	return ctx
}

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

	// 5. Company：header 或 context，取到后 set 回 header + context
	companyCode := c.GetHeader(CompanyCodeHeader)
	if companyCode == "" {
		if v, exists := c.Get(CompanyCodeHeader); exists {
			if s, ok := v.(string); ok && s != "" {
				companyCode = s
			}
		}
	}
	if companyCode != "" {
		ctx = context.WithValue(ctx, CompanyCodeHeader, companyCode)
		c.Request.Header.Set(CompanyCodeHeader, companyCode)
	}
	companyName := c.GetHeader(CompanyNameHeader)
	if companyName == "" {
		if v, exists := c.Get(CompanyNameHeader); exists {
			if s, ok := v.(string); ok && s != "" {
				companyName = s
			}
		}
	}
	if companyName != "" {
		ctx = context.WithValue(ctx, CompanyNameHeader, companyName)
		c.Request.Header.Set(CompanyNameHeader, companyName)
	}
	companyLogoURL := c.GetHeader(CompanyLogoURLHeader)
	if companyLogoURL == "" {
		if v, exists := c.Get(CompanyLogoURLHeader); exists {
			if s, ok := v.(string); ok && s != "" {
				companyLogoURL = s
			}
		}
	}
	if companyLogoURL != "" {
		ctx = context.WithValue(ctx, CompanyLogoURLHeader, companyLogoURL)
		c.Request.Header.Set(CompanyLogoURLHeader, companyLogoURL)
	}

	// 6. ClientSource：header 或 context，取到后 set 回 header + context，供操作日志和下游调用链识别入口来源
	clientSource := c.GetHeader(ClientSourceHeader)
	if clientSource == "" {
		if v, exists := c.Get(ClientSourceHeader); exists {
			if s, ok := v.(string); ok && s != "" {
				clientSource = s
			}
		}
	}
	if clientSource != "" {
		ctx = context.WithValue(ctx, ClientSourceHeader, clientSource)
		c.Request.Header.Set(ClientSourceHeader, clientSource)
	}

	// 7. Source ref：后台自动化/函数触发来源，供工具调用链审计和后续白名单使用
	sourceType := c.GetHeader(SourceTypeHeader)
	if sourceType != "" {
		ctx = context.WithValue(ctx, SourceTypeHeader, sourceType)
		c.Request.Header.Set(SourceTypeHeader, sourceType)
	}
	sourceRef := c.GetHeader(SourceRefHeader)
	if sourceRef != "" {
		ctx = context.WithValue(ctx, SourceRefHeader, sourceRef)
		c.Request.Header.Set(SourceRefHeader, sourceRef)
	}

	// 8. PresignHost：优先 X-Forwarded-Host（含端口），无端口时用 X-Forwarded-Port 或 PresignDefaultPort 补全，与浏览器 PUT 的 Host 一致避免 403
	presignHost := c.GetHeader("X-Forwarded-Host")
	if presignHost == "" {
		presignHost = c.Request.Host
	}
	if presignHost != "" && !strings.Contains(presignHost, ":") {
		port := c.GetHeader("X-Forwarded-Port")
		if port == "" {
			port = PresignDefaultPort
		}
		presignHost = presignHost + ":" + port
	}
	if presignHost != "" {
		ctx = context.WithValue(ctx, PresignHostKey, presignHost)
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
	ctx = context.WithValue(ctx, CompanyCodeHeader, msg.Header.Get(CompanyCodeHeader))
	ctx = context.WithValue(ctx, CompanyNameHeader, msg.Header.Get(CompanyNameHeader))
	ctx = context.WithValue(ctx, CompanyLogoURLHeader, msg.Header.Get(CompanyLogoURLHeader))
	if clientSource := msg.Header.Get(ClientSourceHeader); clientSource != "" {
		ctx = context.WithValue(ctx, ClientSourceHeader, clientSource)
	}
	if sourceType := msg.Header.Get(SourceTypeHeader); sourceType != "" {
		ctx = context.WithValue(ctx, SourceTypeHeader, sourceType)
	}
	if sourceRef := msg.Header.Get(SourceRefHeader); sourceRef != "" {
		ctx = context.WithValue(ctx, SourceRefHeader, sourceRef)
	}

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
	if departmentFullPath := GetRequestDepartmentFullPath(c); departmentFullPath != "" {
		msg.Header.Set(DepartmentFullPathHeader, departmentFullPath)
	}
	if companyCode := GetRequestCompanyCode(c); companyCode != "" {
		msg.Header.Set(CompanyCodeHeader, companyCode)
	}
	if companyName := GetRequestCompanyName(c); companyName != "" {
		msg.Header.Set(CompanyNameHeader, companyName)
	}
	if companyLogoURL := GetRequestCompanyLogoURL(c); companyLogoURL != "" {
		msg.Header.Set(CompanyLogoURLHeader, companyLogoURL)
	}
	if clientSource := GetClientSource(c); clientSource != "" {
		msg.Header.Set(ClientSourceHeader, clientSource)
	}
	if sourceType := GetSourceType(c); sourceType != "" {
		msg.Header.Set(SourceTypeHeader, sourceType)
	}
	if sourceRef := GetSourceRef(c); sourceRef != "" {
		msg.Header.Set(SourceRefHeader, sourceRef)
	}
	return msg

}

// RequestInfo 无 HTTP 请求时的请求信息，与 ToContext 透传字段一致
type RequestInfo struct {
	TraceId            string
	RequestUser        string
	Token              string
	DepartmentFullPath string
	CompanyCode        string
	CompanyName        string
	CompanyLogoURL     string
	ClientSource       string
	SourceType         string
	SourceRef          string
}

// WithRequestInfo 一次性注入与 ToContext 一致的 context（用于后台任务等无 HTTP 请求场景）
func WithRequestInfo(ctx context.Context, info RequestInfo) context.Context {
	if info.TraceId != "" {
		ctx = context.WithValue(ctx, TraceIdHeader, info.TraceId)
	}
	if info.RequestUser != "" {
		ctx = context.WithValue(ctx, RequestUserHeader, info.RequestUser)
	}
	if info.Token != "" {
		ctx = context.WithValue(ctx, TokenHeader, info.Token)
	}
	if info.DepartmentFullPath != "" {
		ctx = context.WithValue(ctx, DepartmentFullPathHeader, info.DepartmentFullPath)
	}
	if info.CompanyCode != "" {
		ctx = context.WithValue(ctx, CompanyCodeHeader, info.CompanyCode)
	}
	if info.CompanyName != "" {
		ctx = context.WithValue(ctx, CompanyNameHeader, info.CompanyName)
	}
	if info.CompanyLogoURL != "" {
		ctx = context.WithValue(ctx, CompanyLogoURLHeader, info.CompanyLogoURL)
	}
	if info.ClientSource != "" {
		ctx = context.WithValue(ctx, ClientSourceHeader, info.ClientSource)
	}
	if info.SourceType != "" {
		ctx = context.WithValue(ctx, SourceTypeHeader, info.SourceType)
	}
	if info.SourceRef != "" {
		ctx = context.WithValue(ctx, SourceRefHeader, info.SourceRef)
	}
	return ctx
}

// WithRequestUser 注入请求用户到 context（用于后台任务等无 HTTP 请求场景）
func WithRequestUser(ctx context.Context, username string) context.Context {
	if username == "" {
		return ctx
	}
	return context.WithValue(ctx, RequestUserHeader, username)
}

// WithToken 注入 Token 到 context
func WithToken(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return context.WithValue(ctx, TokenHeader, token)
}

// WithDepartmentFullPath 注入部门路径到 context
func WithDepartmentFullPath(ctx context.Context, deptPath string) context.Context {
	if deptPath == "" {
		return ctx
	}
	return context.WithValue(ctx, DepartmentFullPathHeader, deptPath)
}

// WithTraceId 注入 TraceId 到 context
func WithTraceId(ctx context.Context, traceId string) context.Context {
	if traceId == "" {
		return ctx
	}
	return context.WithValue(ctx, TraceIdHeader, traceId)
}
