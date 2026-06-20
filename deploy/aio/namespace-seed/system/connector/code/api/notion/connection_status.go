package notion

import (
	"fmt"
	"strings"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type NotionConnectionStatusReq struct{}

type NotionConnectionStatusResp struct {
	Status            string `json:"status" widget:"name:连接状态;type:select;options:已连接,未连接;options_colors:67C23A,F56C6C"`
	Provider          string `json:"provider" widget:"name:平台;type:input"`
	DisplayName       string `json:"display_name" widget:"name:显示名称;type:input"`
	WorkspaceName     string `json:"workspace_name" widget:"name:工作空间;type:input"`
	WorkspaceID       string `json:"workspace_id" widget:"name:工作空间ID;type:input"`
	AccountName       string `json:"account_name" widget:"name:授权账号;type:input"`
	ExternalAccountID string `json:"external_account_id" widget:"name:外部账号ID;type:input"`
	ConnectionID      string `json:"connection_id" widget:"name:连接ID;type:input"`
	ResolvedFrom      string `json:"resolved_from" widget:"name:命中绑定路径;type:input"`
	RequestedPath     string `json:"requested_path" widget:"name:请求资源路径;type:input"`
	Summary           string `json:"summary" widget:"name:说明;type:text_area"`
}

func NotionConnectionStatus(ctx *app.Context, resp response.Response) error {
	var req NotionConnectionStatusReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	resolved, err := ctx.GetConnector("notion")
	if err != nil {
		return err
	}

	profile := resolved.Connection.Profile
	displayName := strings.TrimSpace(resolved.Connection.DisplayName)
	externalAccountID := strings.TrimSpace(resolved.Connection.ExternalAccountID)
	workspaceName := ""
	workspaceID := ""
	accountName := ""
	if profile != nil {
		displayName = notionDisplayValue(displayName, profile.DisplayName, profile.WorkspaceName, profile.AccountName)
		workspaceName = profile.WorkspaceName
		workspaceID = profile.WorkspaceID
		accountName = profile.AccountName
	}

	return resp.Form(&NotionConnectionStatusResp{
		Status:            "已连接",
		Provider:          resolved.Connection.Provider,
		DisplayName:       displayName,
		WorkspaceName:     workspaceName,
		WorkspaceID:       workspaceID,
		AccountName:       accountName,
		ExternalAccountID: externalAccountID,
		ConnectionID:      resolved.Connection.ConnectionID,
		ResolvedFrom:      resolved.ResolvedFrom,
		RequestedPath:     resolved.RequestedPath,
		Summary: fmt.Sprintf(
			"Notion 连接器已经可用。\n当前函数资源: %s\n绑定命中路径: %s\n工作空间: %s\n授权账号: %s",
			ctx.GetFullCodePath(),
			resolved.ResolvedFrom,
			notionDisplayValue(workspaceName, displayName, externalAccountID, resolved.Connection.ConnectionID),
			notionDisplayValue(accountName, externalAccountID, resolved.Connection.ConnectionID),
		),
	}).Build()
}

var NotionConnectionStatusTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:       "Notion 连接状态",
		Desc:       "验证当前用户是否已经完成 Notion OAuth 授权，并展示平台解析到的连接器绑定信息。",
		Tags:       []string{"Notion", "连接器", "OAuth", "状态检查"},
		Connectors: []string{"notion"},
		Request:    &NotionConnectionStatusReq{},
		Response:   &NotionConnectionStatusResp{},
	},
}

func init() {
	packageContext.POST("connection_status.form", NotionConnectionStatus, NotionConnectionStatusTemplate)
}
