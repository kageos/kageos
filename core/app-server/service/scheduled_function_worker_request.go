package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/functionschema"
)

func (a *AppService) requireScheduledFunctionAccess(ctx context.Context, fullCodePath, action string) error {
	if a == nil || a.teamAccess == nil {
		return nil
	}
	resourcePath := access.NormalizeResourcePath(fullCodePath)
	tenantUser, app, err := access.ParseUserApp(resourcePath)
	if err != nil {
		return err
	}
	return a.teamAccess.Check(ctx, tenantUser, app, contextx.GetRequestUser(ctx), resourcePath, scheduledFunctionRequiredAccessAction(fullCodePath, action))
}

func (a *AppService) ensureScheduledTableCallbackEnabled(ctx context.Context, fullCodePath, callbackType, denyMessage string) error {
	function, err := a.GetFunctionByFullCodePath(ctx, fullCodePath)
	if err != nil {
		return err
	}
	if functionschema.Type(function.Schema) != functionschema.TypeTable {
		return fmt.Errorf("目标函数不是 Table 类型，不支持该操作")
	}
	if !function.HasCallback(callbackType) {
		return fmt.Errorf("%s", denyMessage)
	}
	return nil
}

func buildScheduledRequestAppReq(ctx context.Context, fullCodePath, method string, body []byte, rawQuery string) (*dto.RequestAppReq, error) {
	user, app, router, err := parseScheduledFunctionFullCodePath(fullCodePath)
	if err != nil {
		return nil, err
	}
	req := buildScheduledBaseRequestAppReq(ctx, user, app, router, method)
	req.Body = body
	req.UrlQuery = rawQuery
	return req, nil
}

func buildScheduledCallbackAppReq(ctx context.Context, fullCodePath, method, callbackType string, body []byte, rawQuery string) (*dto.RequestAppReq, error) {
	user, app, router, err := parseScheduledFunctionFullCodePath(fullCodePath)
	if err != nil {
		return nil, err
	}
	req := buildScheduledBaseRequestAppReq(ctx, user, app, "/_callback", method)
	req.TargetRouter = router
	req.UrlQuery = rawQuery
	envelope := scheduledCallbackRequestEnvelope{
		Method: method,
		Router: router,
		Body:   body,
		Type:   callbackType,
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	req.Body = data
	return req, nil
}

func buildScheduledBaseRequestAppReq(ctx context.Context, user, app, router, method string) *dto.RequestAppReq {
	return &dto.RequestAppReq{
		User:                  user,
		App:                   app,
		Router:                router,
		Method:                method,
		TraceId:               contextx.GetTraceId(ctx),
		RequestUser:           contextx.GetRequestUser(ctx),
		RequestUserDept:       contextx.GetRequestDepartmentFullPath(ctx),
		Token:                 contextx.GetToken(ctx),
		ClientSource:          contextx.GetClientSource(ctx),
		SourceType:            contextx.GetSourceType(ctx),
		SourceRef:             contextx.GetSourceRef(ctx),
		SourcePath:            contextx.GetSourcePath(ctx),
		SourceTitle:           contextx.GetSourceTitle(ctx),
		SourceParentPath:      contextx.GetSourceParentPath(ctx),
		SourceParentTitle:     contextx.GetSourceParentTitle(ctx),
		SourceTemplateType:    contextx.GetSourceTemplateType(ctx),
		WorkspaceSessionID:    contextx.GetWorkspaceSessionID(ctx),
		WorkspaceSessionTitle: contextx.GetWorkspaceSessionTitle(ctx),
		WorkspaceRole:         contextx.GetWorkspaceRole(ctx),
		InitiatorUser:         contextx.GetInitiatorUser(ctx),
		WorkspaceMessageID:    parseOperateLogAuditInt64(contextx.GetWorkspaceMessageID(ctx)),
		ToolCallID:            contextx.GetToolCallID(ctx),
		ToolName:              contextx.GetToolName(ctx),
	}
}

func parseScheduledFunctionFullCodePath(fullCodePath string) (user, app, router string, err error) {
	fullCodePath = strings.TrimPrefix(access.NormalizeResourcePath(fullCodePath), "/")
	parts := strings.Split(fullCodePath, "/")
	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("full-code-path 格式错误，至少需要包含 user/app/function")
	}
	return parts[0], parts[1], strings.Join(parts[2:], "/"), nil
}

func isScheduledFunctionAction(action string) bool {
	switch strings.TrimSpace(action) {
	case "execute", "table_create", "table_update", "table_delete":
		return true
	default:
		return false
	}
}

func scheduledFunctionRequiredAccessAction(fullCodePath, action string) access.Action {
	switch strings.TrimSpace(action) {
	case "table_update":
		return access.ActionUpdate
	case "table_delete":
		return access.ActionDelete
	case "execute":
		switch scheduledFunctionTemplateType(fullCodePath) {
		case "chart", "table":
			return access.ActionRead
		default:
			return access.ActionWrite
		}
	default:
		return access.ActionWrite
	}
}

func scheduledFunctionHTTPMethod(payload scheduledFunctionPayload) string {
	if method := strings.TrimSpace(payload.Method); method != "" {
		return strings.ToUpper(method)
	}
	switch strings.TrimSpace(payload.Action) {
	case "table_update":
		return http.MethodPut
	case "table_delete":
		return http.MethodDelete
	case "execute":
		switch payload.TemplateType {
		case "table", "chart":
			return http.MethodGet
		default:
			return http.MethodPost
		}
	default:
		return http.MethodPost
	}
}

func scheduledFunctionTemplateType(fullCodePath string) string {
	path := strings.ToLower(strings.TrimSpace(fullCodePath))
	switch {
	case strings.Contains(path, ".form"):
		return "form"
	case strings.Contains(path, ".table"):
		return "table"
	case strings.Contains(path, ".chart"):
		return "chart"
	default:
		return ""
	}
}
