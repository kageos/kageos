package wecom

import (
	"fmt"
	"strings"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type WeComSendMessageBaseReq struct {
	ConfigID               int    `json:"config_id" widget:"name:企业微信配置;type:select;placeholder:留空使用第一个启用配置" callback:"OnSelectFuzzy"`
	ToAll                  bool   `json:"to_all" widget:"name:发送给应用可见范围内全部成员;type:switch"`
	ToUser                 string `json:"touser" widget:"name:接收成员;type:text_area;placeholder:多个 UserID 用 | 分隔，例如 zhangsan|lisi"`
	ToParty                string `json:"toparty" widget:"name:接收部门;type:input;placeholder:多个部门 ID 用 | 分隔"`
	ToTag                  string `json:"totag" widget:"name:接收标签;type:input;placeholder:多个标签 ID 用 | 分隔"`
	Safe                   bool   `json:"safe" widget:"name:保密消息;type:switch"`
	EnableDuplicateCheck   bool   `json:"enable_duplicate_check" widget:"name:启用重复消息检查;type:switch"`
	DuplicateCheckInterval int    `json:"duplicate_check_interval" widget:"name:重复检查间隔;type:integer;min:0;max:14400;step:60;unit:秒"`
}

type WeComSendTextReq struct {
	ConfigID               int    `json:"config_id" widget:"name:企业微信配置;type:select;placeholder:留空使用第一个启用配置" callback:"OnSelectFuzzy"`
	ToAll                  bool   `json:"to_all" widget:"name:发送给应用可见范围内全部成员;type:switch"`
	ToUser                 string `json:"touser" widget:"name:接收成员;type:text_area;placeholder:多个 UserID 用 | 分隔，例如 zhangsan|lisi"`
	ToParty                string `json:"toparty" widget:"name:接收部门;type:input;placeholder:多个部门 ID 用 | 分隔"`
	ToTag                  string `json:"totag" widget:"name:接收标签;type:input;placeholder:多个标签 ID 用 | 分隔"`
	Safe                   bool   `json:"safe" widget:"name:保密消息;type:switch"`
	EnableDuplicateCheck   bool   `json:"enable_duplicate_check" widget:"name:启用重复消息检查;type:switch"`
	DuplicateCheckInterval int    `json:"duplicate_check_interval" widget:"name:重复检查间隔;type:integer;min:0;max:14400;step:60;unit:秒"`
	Content                string `json:"content" widget:"name:消息内容;type:text_area;placeholder:请输入要发送的文本消息" validate:"required"`
}

type WeComSendMarkdownReq struct {
	ConfigID               int    `json:"config_id" widget:"name:企业微信配置;type:select;placeholder:留空使用第一个启用配置" callback:"OnSelectFuzzy"`
	ToAll                  bool   `json:"to_all" widget:"name:发送给应用可见范围内全部成员;type:switch"`
	ToUser                 string `json:"touser" widget:"name:接收成员;type:text_area;placeholder:多个 UserID 用 | 分隔，例如 zhangsan|lisi"`
	ToParty                string `json:"toparty" widget:"name:接收部门;type:input;placeholder:多个部门 ID 用 | 分隔"`
	ToTag                  string `json:"totag" widget:"name:接收标签;type:input;placeholder:多个标签 ID 用 | 分隔"`
	Safe                   bool   `json:"safe" widget:"name:保密消息;type:switch"`
	EnableDuplicateCheck   bool   `json:"enable_duplicate_check" widget:"name:启用重复消息检查;type:switch"`
	DuplicateCheckInterval int    `json:"duplicate_check_interval" widget:"name:重复检查间隔;type:integer;min:0;max:14400;step:60;unit:秒"`
	Content                string `json:"content" widget:"name:Markdown内容;type:text_area;placeholder:请输入企业微信 Markdown 消息内容" validate:"required"`
}

type wecomMessageSendResp struct {
	wecomBaseResp
	InvalidUser  string `json:"invaliduser"`
	InvalidParty string `json:"invalidparty"`
	InvalidTag   string `json:"invalidtag"`
	MsgID        string `json:"msgid"`
}

func (r wecomMessageSendResp) Base() wecomBaseResp {
	return r.wecomBaseResp
}

type WeComSendMessageResp struct {
	Status       string `json:"status" widget:"name:发送状态;type:select;options:成功,失败;options_colors:67C23A,F56C6C"`
	ConfigID     int    `json:"config_id" widget:"name:配置ID;type:integer"`
	ConfigName   string `json:"config_name" widget:"name:配置名称;type:input"`
	MsgID        string `json:"msgid" widget:"name:消息ID;type:input"`
	InvalidUser  string `json:"invaliduser" widget:"name:无效成员;type:text_area"`
	InvalidParty string `json:"invalidparty" widget:"name:无效部门;type:text_area"`
	InvalidTag   string `json:"invalidtag" widget:"name:无效标签;type:text_area"`
	Summary      string `json:"summary" widget:"name:说明;type:text_area"`
}

func WeComSendText(ctx *app.Context, resp response.Response) error {
	var req WeComSendTextReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	result, err := sendWeComMessage(ctx, req.ConfigID, req.baseReq(), "text", map[string]string{"content": req.Content})
	if err != nil {
		return err
	}
	return resp.Form(result).Build()
}

func WeComSendMarkdown(ctx *app.Context, resp response.Response) error {
	var req WeComSendMarkdownReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	result, err := sendWeComMessage(ctx, req.ConfigID, req.baseReq(), "markdown", map[string]string{"content": req.Content})
	if err != nil {
		return err
	}
	return resp.Form(result).Build()
}

func sendWeComMessage(ctx *app.Context, configID int, base *WeComSendMessageBaseReq, msgType string, content map[string]string) (*WeComSendMessageResp, error) {
	cfg, err := loadWeComConfig(ctx, configID)
	if err != nil {
		return nil, err
	}
	payload, err := buildWeComMessagePayload(cfg, base, msgType, content)
	if err != nil {
		return nil, err
	}
	var apiResp wecomMessageSendResp
	if err := postWeComAPI(ctx, cfg, "/cgi-bin/message/send", payload, &apiResp); err != nil {
		updateWeComConfigStatus(ctx, cfg.ID, "失败", err.Error(), cfg.TokenExpiresAt.Time(), "")
		return nil, err
	}
	summary := fmt.Sprintf("企业微信%s消息发送成功。", msgType)
	if apiResp.InvalidUser != "" || apiResp.InvalidParty != "" || apiResp.InvalidTag != "" {
		summary += "\n企业微信返回了部分无效接收人，请检查无效成员、部门或标签。"
	}
	updateWeComConfigStatus(ctx, cfg.ID, "正常", summary, cfg.TokenExpiresAt.Time(), "")
	return &WeComSendMessageResp{
		Status:       "成功",
		ConfigID:     cfg.ID,
		ConfigName:   cfg.Name,
		MsgID:        apiResp.MsgID,
		InvalidUser:  apiResp.InvalidUser,
		InvalidParty: apiResp.InvalidParty,
		InvalidTag:   apiResp.InvalidTag,
		Summary:      summary,
	}, nil
}

func (r *WeComSendTextReq) baseReq() *WeComSendMessageBaseReq {
	return &WeComSendMessageBaseReq{
		ConfigID:               r.ConfigID,
		ToAll:                  r.ToAll,
		ToUser:                 r.ToUser,
		ToParty:                r.ToParty,
		ToTag:                  r.ToTag,
		Safe:                   r.Safe,
		EnableDuplicateCheck:   r.EnableDuplicateCheck,
		DuplicateCheckInterval: r.DuplicateCheckInterval,
	}
}

func (r *WeComSendMarkdownReq) baseReq() *WeComSendMessageBaseReq {
	return &WeComSendMessageBaseReq{
		ConfigID:               r.ConfigID,
		ToAll:                  r.ToAll,
		ToUser:                 r.ToUser,
		ToParty:                r.ToParty,
		ToTag:                  r.ToTag,
		Safe:                   r.Safe,
		EnableDuplicateCheck:   r.EnableDuplicateCheck,
		DuplicateCheckInterval: r.DuplicateCheckInterval,
	}
}

func buildWeComMessagePayload(cfg *WeComConfig, req *WeComSendMessageBaseReq, msgType string, content map[string]string) (map[string]interface{}, error) {
	if req == nil {
		return nil, fmt.Errorf("消息参数不能为空")
	}
	touser := strings.TrimSpace(req.ToUser)
	toparty := strings.TrimSpace(req.ToParty)
	totag := strings.TrimSpace(req.ToTag)
	if req.ToAll {
		touser = "@all"
	}
	if touser == "" && toparty == "" && totag == "" {
		return nil, fmt.Errorf("请至少填写接收成员、接收部门、接收标签，或开启发送给全部成员")
	}
	payload := map[string]interface{}{
		"msgtype": msgType,
		"agentid": cfg.AgentID,
		"safe":    boolToInt(req.Safe),
		msgType:   content,
	}
	if touser != "" {
		payload["touser"] = touser
	}
	if toparty != "" {
		payload["toparty"] = toparty
	}
	if totag != "" {
		payload["totag"] = totag
	}
	if req.EnableDuplicateCheck {
		payload["enable_duplicate_check"] = 1
		if req.DuplicateCheckInterval > 0 {
			payload["duplicate_check_interval"] = req.DuplicateCheckInterval
		}
	}
	return payload, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

var WeComSendTextTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "企业微信发送文本消息",
		Desc:         "使用企业微信自建应用配置发送 text 应用消息。接收人必须在应用可见范围内。",
		Tags:         []string{"企业微信", "应用消息", "文本"},
		Request:      &WeComSendTextReq{},
		Response:     &WeComSendMessageResp{},
		CreateTables: []interface{}{&WeComConfig{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"config_id": wecomConfigOnSelectFuzzy,
		},
	},
}

var WeComSendMarkdownTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "企业微信发送 Markdown 消息",
		Desc:         "使用企业微信自建应用配置发送 markdown 应用消息。适合告警、通知和状态汇总。",
		Tags:         []string{"企业微信", "应用消息", "Markdown"},
		Request:      &WeComSendMarkdownReq{},
		Response:     &WeComSendMessageResp{},
		CreateTables: []interface{}{&WeComConfig{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"config_id": wecomConfigOnSelectFuzzy,
		},
	},
}

func init() {
	packageContext.POST("send_text.form", WeComSendText, WeComSendTextTemplate)
	packageContext.POST("send_markdown.form", WeComSendMarkdown, WeComSendMarkdownTemplate)
}
