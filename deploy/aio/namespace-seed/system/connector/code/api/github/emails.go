package github

import (
	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type GitHubEmailsReq struct {
	query.PageSortReq `widget:"-"`
}

type GitHubEmail struct {
	Email      string `json:"email" widget:"name:邮箱;type:input"`
	Primary    bool   `json:"primary" widget:"name:主邮箱;type:switch"`
	Verified   bool   `json:"verified" widget:"name:已验证;type:switch"`
	Visibility string `json:"visibility" widget:"name:可见性;type:input"`
}

func GitHubEmails(ctx *app.Context, resp response.Response) error {
	var req GitHubEmailsReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	var emails []GitHubEmail
	if _, err := callGitHubJSON(ctx, "/user/emails", nil, &emails); err != nil {
		return err
	}
	items := emails
	total := int64(len(emails))
	start := req.PageSortReq.GetOffset()
	if start > len(items) {
		start = len(items)
	}
	end := start + req.PageSortReq.GetLimit()
	if end > len(items) {
		end = len(items)
	}
	items = items[start:end]
	return resp.Table(response.TableResult{
		Items:      items,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

var GitHubEmailsTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:       "GitHub 邮箱列表",
		Desc:       "使用当前用户已授权的 GitHub 连接器读取 /user/emails 邮箱列表。",
		Tags:       []string{"GitHub", "连接器", "邮箱"},
		Connectors: []string{"github"},
		ConnectorEndpoints: []app.ConnectorEndpoint{
			githubEndpoint("GET", "/user/emails", "读取当前用户邮箱", "user:email"),
		},
		Request: &GitHubEmailsReq{},
	},
	AutoCrudTable: &GitHubEmail{},
}

func init() {
	packageContext.GET("emails.table", GitHubEmails, GitHubEmailsTemplate)
}
