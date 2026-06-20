package github

import (
	"fmt"
	"strings"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type ConnectionStatusReq struct{}

type ConnectionStatusResp struct {
	Status            string `json:"status" widget:"name:连接状态;type:select;options:已连接,未连接;options_colors:67C23A,F56C6C"`
	Provider          string `json:"provider" widget:"name:平台;type:input"`
	DisplayName       string `json:"display_name" widget:"name:显示名称;type:input"`
	ExternalAccountID string `json:"external_account_id" widget:"name:外部账号ID;type:input"`
	ConnectionID      string `json:"connection_id" widget:"name:连接ID;type:input"`
	ResolvedFrom      string `json:"resolved_from" widget:"name:命中绑定路径;type:input"`
	RequestedPath     string `json:"requested_path" widget:"name:请求资源路径;type:input"`
	Summary           string `json:"summary" widget:"name:说明;type:text_area"`
}

func ConnectionStatus(ctx *app.Context, resp response.Response) error {
	var req ConnectionStatusReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	resolved, err := ctx.GetConnector("github")
	if err != nil {
		return err
	}

	displayName := strings.TrimSpace(resolved.Connection.DisplayName)
	externalAccountID := strings.TrimSpace(resolved.Connection.ExternalAccountID)
	if displayName == "" {
		displayName = externalAccountID
	}

	return resp.Form(&ConnectionStatusResp{
		Status:            "已连接",
		Provider:          resolved.Connection.Provider,
		DisplayName:       displayName,
		ExternalAccountID: externalAccountID,
		ConnectionID:      resolved.Connection.ConnectionID,
		ResolvedFrom:      resolved.ResolvedFrom,
		RequestedPath:     resolved.RequestedPath,
		Summary: fmt.Sprintf(
			"GitHub 连接器已经可用。\n当前函数资源: %s\n绑定命中路径: %s\n账号: %s",
			ctx.GetFullCodePath(),
			resolved.ResolvedFrom,
			displayValue(displayName, externalAccountID, resolved.Connection.ConnectionID),
		),
	}).Build()
}

func displayValue(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return "-"
}

var ConnectionStatusTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:       "GitHub 连接状态",
		Desc:       "验证当前用户是否已经完成 GitHub OAuth 授权，并展示平台解析到的连接器绑定信息。这个函数用于测试连接器闭环，不直接读取 GitHub API 数据。",
		Tags:       []string{"GitHub", "连接器", "OAuth", "状态检查"},
		Connectors: []string{"github"},
		Request:    &ConnectionStatusReq{},
		Response:   &ConnectionStatusResp{},
	},
}

func init() {
	packageContext.POST("connection_status.form", ConnectionStatus, ConnectionStatusTemplate)
}
