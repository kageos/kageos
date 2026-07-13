package contextx

import (
	"context"

	"github.com/nats-io/nats.go"
)

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
		if value := msg.Header.Get(key); value != "" {
			ctx = context.WithValue(ctx, key, value)
		}
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
	for _, item := range []struct {
		key   string
		value string
	}{
		{SourcePathHeader, GetSourcePath(c)},
		{SourceTitleHeader, GetSourceTitle(c)},
		{SourceParentPathHeader, GetSourceParentPath(c)},
		{SourceParentTitleHeader, GetSourceParentTitle(c)},
		{SourceTemplateTypeHeader, GetSourceTemplateType(c)},
		{WorkspaceSessionIDHeader, GetWorkspaceSessionID(c)},
		{WorkspaceSessionTitleHeader, GetWorkspaceSessionTitle(c)},
		{WorkspaceRoleHeader, GetWorkspaceRole(c)},
		{InitiatorUserHeader, GetInitiatorUser(c)},
		{WorkspaceMessageIDHeader, GetWorkspaceMessageID(c)},
		{ToolCallIDHeader, GetToolCallID(c)},
		{ToolNameHeader, GetToolName(c)},
	} {
		if item.value != "" {
			msg.Header.Set(item.key, item.value)
		}
	}
	return msg

}
