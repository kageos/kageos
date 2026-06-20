package github

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type GitHubAPIGetReq struct {
	Path      string `json:"path" widget:"name:API 路径;type:input;placeholder:/user" validate:"required"`
	QueryText string `json:"query_text" widget:"name:Query JSON;type:text_area;placeholder:{\"per_page\":20}"`
}

type GitHubAPIGetResp struct {
	StatusCode    int    `json:"status_code" widget:"name:状态码;type:integer"`
	ResolvedFrom  string `json:"resolved_from" widget:"name:命中绑定路径;type:input"`
	RequestedPath string `json:"requested_path" widget:"name:请求资源路径;type:input"`
	HeadersText   string `json:"headers_text" widget:"name:响应头;type:text_area"`
	BodyText      string `json:"body_text" widget:"name:响应体;type:text_area"`
}

func GitHubAPIGet(ctx *app.Context, resp response.Response) error {
	var req GitHubAPIGetReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	query, err := parseGitHubQueryText(req.QueryText)
	if err != nil {
		return err
	}
	proxyResp, err := ctx.CallConnector("github", app.ConnectorRequest{
		Method: http.MethodGet,
		Path:   strings.TrimSpace(req.Path),
		Query:  query,
	})
	if err != nil {
		return err
	}
	headerBytes, _ := json.MarshalIndent(proxyResp.Headers, "", "  ")
	return resp.Form(&GitHubAPIGetResp{
		StatusCode:    proxyResp.StatusCode,
		ResolvedFrom:  proxyResp.ResolvedFrom,
		RequestedPath: proxyResp.RequestedPath,
		HeadersText:   string(headerBytes),
		BodyText:      githubPrettyJSON(proxyResp.Body),
	}).Build()
}

var GitHubAPIGetTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:       "GitHub 通用 GET",
		Desc:       "用当前 GitHub 连接器调用任意只读 REST API 路径，适合作为新接口沉淀前的验证入口。",
		Tags:       []string{"GitHub", "连接器", "REST", "模板"},
		Connectors: []string{"github"},
		ConnectorEndpoints: []app.ConnectorEndpoint{
			githubEndpoint("GET", "/*", "通用只读 GitHub REST API"),
		},
		Request:  &GitHubAPIGetReq{},
		Response: &GitHubAPIGetResp{},
	},
}

func init() {
	packageContext.POST("api_get.form", GitHubAPIGet, GitHubAPIGetTemplate)
}
