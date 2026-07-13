package contextx

import (
	"context"
)

// RequestInfo 无 HTTP 请求时的请求信息，与 ToContext 透传字段一致
type RequestInfo struct {
	TraceId               string
	RequestUser           string
	Token                 string
	DepartmentFullPath    string
	CompanyCode           string
	CompanyName           string
	CompanyLogoURL        string
	UserID                string
	UserEmail             string
	LeaderUsername        string
	ClientSource          string
	SourceType            string
	SourceRef             string
	SourcePath            string
	SourceTitle           string
	SourceParentPath      string
	SourceParentTitle     string
	SourceTemplateType    string
	WorkspaceSessionID    string
	WorkspaceSessionTitle string
	WorkspaceRole         string
	InitiatorUser         string
	WorkspaceMessageID    int64
	ToolCallID            string
	ToolName              string
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
	if info.UserID != "" {
		ctx = context.WithValue(ctx, UserIDHeader, info.UserID)
	}
	if info.UserEmail != "" {
		ctx = context.WithValue(ctx, UserEmailHeader, info.UserEmail)
	}
	if info.LeaderUsername != "" {
		ctx = context.WithValue(ctx, LeaderUsernameHeader, info.LeaderUsername)
	}
	if info.InitiatorUser != "" {
		ctx = context.WithValue(ctx, InitiatorUserHeader, info.InitiatorUser)
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
	ctx = WithSourceDisplay(ctx, info.SourcePath, info.SourceTitle, info.SourceParentPath, info.SourceParentTitle, info.SourceTemplateType)
	ctx = WithWorkspaceSession(ctx, info.WorkspaceSessionID, info.WorkspaceSessionTitle, info.WorkspaceRole)
	ctx = WithWorkspaceMessageID(ctx, info.WorkspaceMessageID)
	ctx = WithToolCallInfo(ctx, info.ToolCallID, info.ToolName)
	return ctx
}
