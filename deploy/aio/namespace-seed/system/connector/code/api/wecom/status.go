package wecom

import (
	"fmt"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
)

type WeComConnectionStatusReq struct {
	ConfigID int `json:"config_id" widget:"name:企业微信配置;type:select;placeholder:留空使用第一个启用配置" callback:"OnSelectFuzzy"`
}

type WeComConnectionStatusResp struct {
	Status         string     `json:"status" widget:"name:连接状态;type:select;options:已连接,未连接;options_colors:67C23A,F56C6C"`
	ConfigID       int        `json:"config_id" widget:"name:配置ID;type:integer"`
	ConfigName     string     `json:"config_name" widget:"name:配置名称;type:input"`
	CorpID         string     `json:"corp_id" widget:"name:企业ID;type:input"`
	AgentID        int        `json:"agent_id" widget:"name:应用AgentId;type:integer"`
	TokenExpiresAt types.Time `json:"token_expires_at" widget:"name:Token过期时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	Summary        string     `json:"summary" widget:"name:说明;type:text_area"`
}

func WeComConnectionStatus(ctx *app.Context, resp response.Response) error {
	var req WeComConnectionStatusReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	cfg, err := loadWeComConfig(ctx, req.ConfigID)
	if err != nil {
		return err
	}
	if _, err := ensureWeComAccessToken(ctx, cfg, false); err != nil {
		return err
	}
	return resp.Form(&WeComConnectionStatusResp{
		Status:         "已连接",
		ConfigID:       cfg.ID,
		ConfigName:     cfg.Name,
		CorpID:         cfg.CorpID,
		AgentID:        cfg.AgentID,
		TokenExpiresAt: cfg.TokenExpiresAt,
		Summary: fmt.Sprintf(
			"企业微信自建应用连接可用。\n当前函数资源: %s\n配置: %s (#%d)\naccess_token 已缓存，过期前会自动刷新。",
			ctx.GetFullCodePath(),
			cfg.Name,
			cfg.ID,
		),
	}).Build()
}

var WeComConnectionStatusTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "企业微信连接状态",
		Desc:         "使用已保存的企业微信自建应用配置获取 access_token，验证 corp_id、agent_id、corp_secret 是否可用。",
		Tags:         []string{"企业微信", "连接状态", "access_token"},
		Request:      &WeComConnectionStatusReq{},
		Response:     &WeComConnectionStatusResp{},
		CreateTables: []interface{}{&WeComConfig{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"config_id": wecomConfigOnSelectFuzzy,
		},
	},
}

func init() {
	packageContext.POST("connection_status.form", WeComConnectionStatus, WeComConnectionStatusTemplate)
}
