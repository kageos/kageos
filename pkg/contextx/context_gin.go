package contextx

import (
	"context"
	"strings"

	"github.com/kageos/kageos/pkg/controlauth"

	"github.com/gin-gonic/gin"
)

// ToContext 将 gin.Context 转换为标准 context.Context
// 从 header 或 gin 上下文（如中间件 c.Set）读取关键信息，写入 context.Value，并同步回 c.Request.Header，保证请求头为权威来源。
func ToContext(c *gin.Context) context.Context {
	// Start from a clean context so stale string-key identity and SSE
	// cancellation semantics do not leak in. Copy only the private, typed Agent
	// delegation capability installed after strict HTTP authentication.
	ctx := context.Background()
	if c != nil && c.Request != nil {
		ctx = controlauth.PropagateDelegatedHTTPRequestSigner(c.Request.Context(), ctx)
	}

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
	for _, key := range []string{UserIDHeader, UserEmailHeader, LeaderUsernameHeader} {
		value := c.GetHeader(key)
		if value == "" {
			if raw, exists := c.Get(key); exists {
				value, _ = raw.(string)
			}
		}
		if value != "" {
			ctx = context.WithValue(ctx, key, value)
			c.Request.Header.Set(key, value)
		}
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

	for _, key := range []string{
		SourcePathHeader,
		SourceTitleHeader,
		SourceParentPathHeader,
		SourceParentTitleHeader,
		SourceTemplateTypeHeader,
		WorkspaceSessionIDHeader,
		WorkspaceSessionTitleHeader,
		WorkspaceRoleHeader,
		InitiatorUserHeader,
		WorkspaceMessageIDHeader,
		ToolCallIDHeader,
		ToolNameHeader,
	} {
		if value := c.GetHeader(key); value != "" {
			ctx = context.WithValue(ctx, key, value)
			c.Request.Header.Set(key, value)
		}
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
