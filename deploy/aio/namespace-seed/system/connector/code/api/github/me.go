package github

import (
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type GitHubMeReq struct{}

type GitHubMeResp struct {
	Login       string `json:"login" widget:"name:登录名;type:input"`
	ID          int64  `json:"id" widget:"name:账号ID;type:integer"`
	Name        string `json:"name" widget:"name:姓名;type:input"`
	Email       string `json:"email" widget:"name:邮箱;type:input"`
	Company     string `json:"company" widget:"name:公司;type:input"`
	Location    string `json:"location" widget:"name:地区;type:input"`
	Blog        string `json:"blog" widget:"name:主页;type:link;target:_blank;link_type:info"`
	HTMLURL     string `json:"html_url" widget:"name:GitHub 主页;type:link;target:_blank;link_type:primary"`
	PublicRepos int    `json:"public_repos" widget:"name:公开仓库数;type:integer"`
	Followers   int    `json:"followers" widget:"name:关注者;type:integer"`
	Following   int    `json:"following" widget:"name:正在关注;type:integer"`
	Bio         string `json:"bio" widget:"name:简介;type:text_area"`
}

func GitHubMe(ctx *app.Context, resp response.Response) error {
	var req GitHubMeReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	var me GitHubMeResp
	if _, err := callGitHubJSON(ctx, "/user", nil, &me); err != nil {
		return err
	}
	me.Blog = normalizeURL(me.Blog)
	return resp.Form(&me).Build()
}

func normalizeURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	return "https://" + value
}

var GitHubMeTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:       "GitHub 我的资料",
		Desc:       "使用当前用户已授权的 GitHub 连接器读取 /user 账号资料。",
		Tags:       []string{"GitHub", "连接器", "账号资料"},
		Connectors: []string{"github"},
		ConnectorEndpoints: []app.ConnectorEndpoint{
			githubEndpoint("GET", "/user", "读取当前用户资料", "read:user"),
		},
		Request:  &GitHubMeReq{},
		Response: &GitHubMeResp{},
	},
}

func init() {
	packageContext.POST("me.form", GitHubMe, GitHubMeTemplate)
}
