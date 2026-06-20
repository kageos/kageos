package notion

import (
	"net/http"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type NotionMeReq struct{}

type notionUser struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Type      string `json:"type"`
	Person    *struct {
		Email string `json:"email"`
	} `json:"person"`
	Bot *struct {
		Owner         map[string]interface{} `json:"owner"`
		WorkspaceName string                 `json:"workspace_name"`
	} `json:"bot"`
}

type NotionMeResp struct {
	ID            string `json:"id" widget:"name:用户ID;type:input"`
	Object        string `json:"object" widget:"name:对象类型;type:input"`
	Name          string `json:"name" widget:"name:名称;type:input"`
	Type          string `json:"type" widget:"name:用户类型;type:select;options:person,bot;options_colors:409EFF,67C23A"`
	Email         string `json:"email" widget:"name:邮箱;type:input"`
	AvatarURL     string `json:"avatar_url" widget:"name:头像;type:link;target:_blank;link_type:info"`
	WorkspaceName string `json:"workspace_name" widget:"name:工作空间;type:input"`
	OwnerJSON     string `json:"owner_json" widget:"name:Bot Owner;type:text_area"`
}

func NotionMe(ctx *app.Context, resp response.Response) error {
	var req NotionMeReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	var user notionUser
	if _, err := callNotionJSON(ctx, http.MethodGet, "/v1/users/me", nil, nil, &user); err != nil {
		return err
	}
	out := NotionMeResp{
		ID:        user.ID,
		Object:    user.Object,
		Name:      user.Name,
		Type:      user.Type,
		AvatarURL: user.AvatarURL,
	}
	if user.Person != nil {
		out.Email = user.Person.Email
	}
	if user.Bot != nil {
		out.WorkspaceName = user.Bot.WorkspaceName
		out.OwnerJSON = notionPrettyJSON(user.Bot.Owner)
	}
	return resp.Form(&out).Build()
}

var NotionMeTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:       "Notion 当前机器人用户",
		Desc:       "使用当前用户已授权的 Notion 连接器读取 /v1/users/me，通常返回该 token 对应的 bot user。",
		Tags:       []string{"Notion", "连接器", "账号资料"},
		Connectors: []string{"notion"},
		ConnectorEndpoints: []app.ConnectorEndpoint{
			notionEndpoint("GET", "/v1/users/me", "读取当前 token 的 bot user"),
		},
		Request:  &NotionMeReq{},
		Response: &NotionMeResp{},
	},
}

func init() {
	packageContext.POST("me.form", NotionMe, NotionMeTemplate)
}
