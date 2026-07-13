package contextx

import (
	"context"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

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
	case SourceTypeScheduledTask:
		return ClientSourceScheduledTask
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
	return getStringFromContextOrHeader(c, SourceRefHeader)
}

func GetSourcePath(c context.Context) string {
	return getStringFromContextOrHeader(c, SourcePathHeader)
}

func GetSourceTitle(c context.Context) string {
	return getStringFromContextOrHeader(c, SourceTitleHeader)
}

func GetSourceParentPath(c context.Context) string {
	return getStringFromContextOrHeader(c, SourceParentPathHeader)
}

func GetSourceParentTitle(c context.Context) string {
	return getStringFromContextOrHeader(c, SourceParentTitleHeader)
}

func GetSourceTemplateType(c context.Context) string {
	return getStringFromContextOrHeader(c, SourceTemplateTypeHeader)
}

func GetWorkspaceSessionID(c context.Context) string {
	return getStringFromContextOrHeader(c, WorkspaceSessionIDHeader)
}

func GetWorkspaceSessionTitle(c context.Context) string {
	return getStringFromContextOrHeader(c, WorkspaceSessionTitleHeader)
}

func GetWorkspaceRole(c context.Context) string {
	return getStringFromContextOrHeader(c, WorkspaceRoleHeader)
}

func GetInitiatorUser(c context.Context) string {
	if initiator := getStringFromContextOrHeader(c, InitiatorUserHeader); initiator != "" {
		return initiator
	}
	return GetRequestUser(c)
}

func GetWorkspaceMessageID(c context.Context) string {
	return getStringFromContextOrHeader(c, WorkspaceMessageIDHeader)
}

func GetToolCallID(c context.Context) string {
	return getStringFromContextOrHeader(c, ToolCallIDHeader)
}

func GetToolName(c context.Context) string {
	return getStringFromContextOrHeader(c, ToolNameHeader)
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

func WithSourceDisplay(ctx context.Context, sourcePath, sourceTitle, parentPath, parentTitle, templateType string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	values := map[string]string{
		SourcePathHeader:         strings.TrimSpace(sourcePath),
		SourceTitleHeader:        strings.TrimSpace(sourceTitle),
		SourceParentPathHeader:   strings.TrimSpace(parentPath),
		SourceParentTitleHeader:  strings.TrimSpace(parentTitle),
		SourceTemplateTypeHeader: strings.TrimSpace(templateType),
	}
	for key, value := range values {
		if value != "" {
			ctx = context.WithValue(ctx, key, value)
		}
	}
	return ctx
}

func WithWorkspaceSession(ctx context.Context, sessionID, sessionTitle, role string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	values := map[string]string{
		WorkspaceSessionIDHeader:    strings.TrimSpace(sessionID),
		WorkspaceSessionTitleHeader: strings.TrimSpace(sessionTitle),
		WorkspaceRoleHeader:         strings.TrimSpace(role),
	}
	for key, value := range values {
		if value != "" {
			ctx = context.WithValue(ctx, key, value)
		}
	}
	return ctx
}

func WithInitiatorUser(ctx context.Context, initiatorUser string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	initiatorUser = strings.TrimSpace(initiatorUser)
	if initiatorUser == "" {
		return ctx
	}
	return context.WithValue(ctx, InitiatorUserHeader, initiatorUser)
}

func WithWorkspaceMessageID(ctx context.Context, messageID int64) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if messageID <= 0 {
		return ctx
	}
	return context.WithValue(ctx, WorkspaceMessageIDHeader, strconv.FormatInt(messageID, 10))
}

func WithToolCallInfo(ctx context.Context, toolCallID, toolName string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	values := map[string]string{
		ToolCallIDHeader: strings.TrimSpace(toolCallID),
		ToolNameHeader:   strings.TrimSpace(toolName),
	}
	for key, value := range values {
		if value != "" {
			ctx = context.WithValue(ctx, key, value)
		}
	}
	return ctx
}
