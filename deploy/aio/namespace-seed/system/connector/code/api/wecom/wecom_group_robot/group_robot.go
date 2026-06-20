package wecom_group_robot

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/env"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

const (
	wecomGroupRobotSendPath        = "/cgi-bin/webhook/send"
	wecomGroupRobotUploadMediaPath = "/cgi-bin/webhook/upload_media"
	groupRobotSecretCipherPrefix   = "v1:"
	maxRobotImageBytes             = 2 * 1024 * 1024
	maxRobotFileBytes              = 20 * 1024 * 1024
	maxRobotVoiceBytes             = 2 * 1024 * 1024
)

var wecomGroupRobotHTTPClient = &http.Client{Timeout: 12 * time.Second}

type groupRobotBaseResp struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

type WeComGroupRobotConfig struct {
	ID               int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:配置ID;type:ID" hide:"create,update"`
	CreatedAt        types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt        types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	CreatedBy        string         `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
	Name             string         `json:"name" gorm:"column:name;comment:机器人名称;index" widget:"name:机器人名称;type:input" validate:"required"`
	GroupName        string         `json:"group_name" gorm:"column:group_name;comment:群名称" widget:"name:群名称;type:input"`
	WebhookURLCipher string         `json:"-" gorm:"column:webhook_url_cipher;type:text" widget:"-"`
	Enabled          bool           `json:"enabled" gorm:"column:enabled;comment:是否启用" widget:"name:启用;type:switch;render_default:true"`
	LastStatus       string         `json:"last_status" gorm:"column:last_status;comment:最近状态" widget:"name:最近状态;type:select;options:未测试,正常,失败;options_colors:909399,67C23A,F56C6C" hide:"create,update"`
	LastMessage      string         `json:"last_message" gorm:"column:last_message;type:text;comment:最近消息" widget:"name:最近消息;type:text_area" hide:"create,update"`
}

func (WeComGroupRobotConfig) TableName() string {
	return "wecom_group_robot_config"
}

type WeComGroupRobotConfigListReq struct {
	Name              string `json:"name" form:"name" widget:"name:机器人名称;type:input"`
	GroupName         string `json:"group_name" form:"group_name" widget:"name:群名称;type:input"`
	LastStatus        string `json:"last_status" form:"last_status" widget:"name:状态;type:select;options:未测试,正常,失败;options_colors:909399,67C23A,F56C6C"`
	query.PageSortReq `widget:"-"`
}

type WeComGroupRobotConfigListItem struct {
	ID             int        `json:"id" widget:"name:配置ID;type:integer"`
	Name           string     `json:"name" widget:"name:机器人名称;type:input"`
	GroupName      string     `json:"group_name" widget:"name:群名称;type:input"`
	HasWebhook     bool       `json:"has_webhook" widget:"name:已配置Webhook;type:switch"`
	Enabled        bool       `json:"enabled" widget:"name:启用;type:switch"`
	LastStatus     string     `json:"last_status" widget:"name:最近状态;type:select;options:未测试,正常,失败;options_colors:909399,67C23A,F56C6C"`
	LastMessage    string     `json:"last_message" widget:"name:最近消息;type:text_area"`
	RobotUpdatedAt types.Time `json:"robot_updated_at" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

func WeComGroupRobotConfigList(ctx *app.Context, resp response.Response) error {
	db, err := wecomGroupRobotDB(ctx)
	if err != nil {
		return err
	}
	var req WeComGroupRobotConfigListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	queryDB := db.Model(&WeComGroupRobotConfig{})
	if strings.TrimSpace(req.Name) != "" {
		queryDB = queryDB.Where("name LIKE ?", "%"+strings.TrimSpace(req.Name)+"%")
	}
	if strings.TrimSpace(req.GroupName) != "" {
		queryDB = queryDB.Where("group_name LIKE ?", "%"+strings.TrimSpace(req.GroupName)+"%")
	}
	if strings.TrimSpace(req.LastStatus) != "" {
		queryDB = queryDB.Where("last_status = ?", strings.TrimSpace(req.LastStatus))
	}
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	} else {
		queryDB = queryDB.Order("id ASC")
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}
	var rows []WeComGroupRobotConfig
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&rows).Error; err != nil {
		return err
	}
	items := make([]WeComGroupRobotConfigListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, WeComGroupRobotConfigListItem{
			ID:             row.ID,
			Name:           row.Name,
			GroupName:      row.GroupName,
			HasWebhook:     strings.TrimSpace(row.WebhookURLCipher) != "",
			Enabled:        row.Enabled,
			LastStatus:     firstNonEmptyRobot(row.LastStatus, "未测试"),
			LastMessage:    row.LastMessage,
			RobotUpdatedAt: row.UpdatedAt,
		})
	}
	return resp.Table(response.TableResult{
		Items:      items,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

type WeComGroupRobotConfigSaveReq struct {
	ConfigID   int    `json:"config_id" widget:"name:更新已有机器人;type:select;placeholder:留空则创建新机器人" callback:"OnSelectFuzzy"`
	Name       string `json:"name" widget:"name:机器人名称;type:input;placeholder:如 KageOS 通知群" validate:"required"`
	GroupName  string `json:"group_name" widget:"name:群名称;type:input;placeholder:方便自己识别，非必填"`
	WebhookURL string `json:"webhook_url" widget:"name:Webhook地址;type:input;placeholder:https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx" sensitive:"true"`
	Enabled    bool   `json:"enabled" widget:"name:启用;type:switch;render_default:true"`
}

type WeComGroupRobotConfigSaveResp struct {
	ConfigID       int    `json:"config_id" widget:"name:配置ID;type:integer"`
	Status         string `json:"status" widget:"name:状态;type:select;options:正常,失败;options_colors:67C23A,F56C6C"`
	Name           string `json:"name" widget:"name:机器人名称;type:input"`
	GroupName      string `json:"group_name" widget:"name:群名称;type:input"`
	WebhookPreview string `json:"webhook_preview" widget:"name:Webhook预览;type:input"`
	Message        string `json:"message" widget:"name:说明;type:text_area"`
}

func WeComGroupRobotConfigSave(ctx *app.Context, resp response.Response) error {
	db, err := wecomGroupRobotDB(ctx)
	if err != nil {
		return err
	}
	var req WeComGroupRobotConfigSaveReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	req.Name = strings.TrimSpace(req.Name)
	req.GroupName = strings.TrimSpace(req.GroupName)
	req.WebhookURL = strings.TrimSpace(req.WebhookURL)
	if req.Name == "" {
		return fmt.Errorf("机器人名称不能为空")
	}

	var cfg WeComGroupRobotConfig
	if req.ConfigID > 0 {
		if err := db.First(&cfg, req.ConfigID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("未找到要更新的企业微信群机器人 ID=%d", req.ConfigID)
			}
			return err
		}
	} else {
		cfg.Enabled = true
		cfg.LastStatus = "未测试"
		cfg.CreatedBy = ctx.GetRequestUser()
	}
	if req.WebhookURL == "" && strings.TrimSpace(cfg.WebhookURLCipher) == "" {
		return fmt.Errorf("首次创建企业微信群机器人时必须填写 Webhook 地址")
	}
	if req.WebhookURL != "" {
		if err := validateWeComGroupRobotWebhook(req.WebhookURL); err != nil {
			return err
		}
		cipherText, err := encryptGroupRobotSecret(req.WebhookURL)
		if err != nil {
			return err
		}
		cfg.WebhookURLCipher = cipherText
	}
	cfg.Name = req.Name
	cfg.GroupName = req.GroupName
	cfg.Enabled = req.Enabled
	if cfg.LastStatus == "" {
		cfg.LastStatus = "未测试"
	}

	if cfg.ID == 0 {
		if err := db.Create(&cfg).Error; err != nil {
			return err
		}
	} else if err := db.Save(&cfg).Error; err != nil {
		return err
	}

	webhookPreview := "沿用已保存的 Webhook"
	if req.WebhookURL != "" {
		webhookPreview = maskWebhookURL(req.WebhookURL)
	}
	return resp.Form(&WeComGroupRobotConfigSaveResp{
		ConfigID:       cfg.ID,
		Status:         "正常",
		Name:           cfg.Name,
		GroupName:      cfg.GroupName,
		WebhookPreview: webhookPreview,
		Message:        "企业微信群机器人配置已保存。可使用发送文本或发送 Markdown 表单测试实际推送。",
	}).Build()
}

type WeComGroupRobotStatusReq struct {
	ConfigID int `json:"config_id" widget:"name:群机器人配置;type:select;placeholder:留空使用第一个启用配置" callback:"OnSelectFuzzy"`
}

type WeComGroupRobotStatusResp struct {
	Status         string `json:"status" widget:"name:配置状态;type:select;options:可用,不可用;options_colors:67C23A,F56C6C"`
	ConfigID       int    `json:"config_id" widget:"name:配置ID;type:integer"`
	ConfigName     string `json:"config_name" widget:"name:机器人名称;type:input"`
	GroupName      string `json:"group_name" widget:"name:群名称;type:input"`
	WebhookPreview string `json:"webhook_preview" widget:"name:Webhook预览;type:input"`
	Summary        string `json:"summary" widget:"name:说明;type:text_area"`
}

func WeComGroupRobotStatus(ctx *app.Context, resp response.Response) error {
	var req WeComGroupRobotStatusReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	cfg, webhookURL, err := loadWeComGroupRobotConfig(ctx, req.ConfigID)
	if err != nil {
		return err
	}
	if err := validateWeComGroupRobotWebhook(webhookURL); err != nil {
		return err
	}
	return resp.Form(&WeComGroupRobotStatusResp{
		Status:         "可用",
		ConfigID:       cfg.ID,
		ConfigName:     cfg.Name,
		GroupName:      cfg.GroupName,
		WebhookPreview: maskWebhookURL(webhookURL),
		Summary:        "企业微信群机器人配置可用。配置检查不会主动发消息；真实可用性以发送文本/Markdown 的企业微信返回为准。",
	}).Build()
}

type WeComGroupRobotSendTextReq struct {
	ConfigID         int    `json:"config_id" widget:"name:群机器人配置;type:select;placeholder:留空使用第一个启用配置" callback:"OnSelectFuzzy"`
	Content          string `json:"content" widget:"name:消息内容;type:text_area;placeholder:请输入要发送到企业微信群的文本" validate:"required"`
	MentionAll       bool   `json:"mention_all" widget:"name:@所有人;type:switch"`
	MentionedUsers   string `json:"mentioned_users" widget:"name:@成员UserID;type:input;placeholder:多个 UserID 用 |、逗号或换行分隔"`
	MentionedMobiles string `json:"mentioned_mobiles" widget:"name:@成员手机号;type:input;placeholder:多个手机号用 |、逗号或换行分隔"`
}

type WeComGroupRobotSendMarkdownReq struct {
	ConfigID int    `json:"config_id" widget:"name:群机器人配置;type:select;placeholder:留空使用第一个启用配置" callback:"OnSelectFuzzy"`
	Content  string `json:"content" widget:"name:Markdown内容;type:text_area;placeholder:请输入企业微信 Markdown 消息内容" validate:"required"`
}

type WeComGroupRobotSendMarkdownV2Req struct {
	ConfigID int    `json:"config_id" widget:"name:群机器人配置;type:select;placeholder:留空使用第一个启用配置" callback:"OnSelectFuzzy"`
	Content  string `json:"content" widget:"name:Markdown V2内容;type:text_area;placeholder:支持表格、图片链接、代码块等新版 Markdown 语法" validate:"required"`
}

type WeComGroupRobotSendImageReq struct {
	ConfigID  int    `json:"config_id" widget:"name:群机器人配置;type:select;placeholder:留空使用第一个启用配置" callback:"OnSelectFuzzy"`
	ImageFile string `json:"image_file" widget:"name:图片文件;type:files;accept:.jpg,.jpeg,.png,image/jpeg,image/png;max_size:2MB;max_count:1" validate:"required"`
}

type WeComGroupRobotSendNewsReq struct {
	ConfigID          int    `json:"config_id" widget:"name:群机器人配置;type:select;placeholder:留空使用第一个启用配置" callback:"OnSelectFuzzy"`
	Title             string `json:"title" widget:"name:标题;type:input;placeholder:图文主标题" validate:"required"`
	Description       string `json:"description" widget:"name:描述;type:text_area;placeholder:图文摘要，非必填"`
	URL               string `json:"url" widget:"name:跳转链接;type:input;placeholder:https://example.com" validate:"required"`
	PicURL            string `json:"picurl" widget:"name:图片链接;type:input;placeholder:https://example.com/image.png"`
	ExtraArticlesJSON string `json:"extra_articles_json" widget:"name:附加图文JSON;type:text_area;placeholder:可粘贴图文数组文本，最多补足到 8 条"`
}

type WeComGroupRobotUploadMediaReq struct {
	ConfigID  int    `json:"config_id" widget:"name:群机器人配置;type:select;placeholder:留空使用第一个启用配置" callback:"OnSelectFuzzy"`
	MediaType string `json:"media_type" widget:"name:素材类型;type:select;options:普通文件,语音;render_default:普通文件" validate:"required"`
	MediaFile string `json:"media_file" widget:"name:素材文件;type:files;accept:*/*;max_size:20MB;max_count:1" validate:"required"`
}

type WeComGroupRobotSendFileReq struct {
	ConfigID int    `json:"config_id" widget:"name:群机器人配置;type:select;placeholder:留空使用第一个启用配置" callback:"OnSelectFuzzy"`
	MediaID  string `json:"media_id" widget:"name:已有 media_id;type:input;placeholder:可直接填写三天内有效的文件 media_id"`
	File     string `json:"file" widget:"name:上传并发送文件;type:files;accept:*/*;max_size:20MB;max_count:1"`
}

type WeComGroupRobotSendVoiceReq struct {
	ConfigID  int    `json:"config_id" widget:"name:群机器人配置;type:select;placeholder:留空使用第一个启用配置" callback:"OnSelectFuzzy"`
	MediaID   string `json:"media_id" widget:"name:已有 media_id;type:input;placeholder:可直接填写三天内有效的语音 media_id"`
	VoiceFile string `json:"voice_file" widget:"name:上传并发送语音;type:files;accept:.amr,audio/amr;max_size:2MB;max_count:1"`
}

type WeComGroupRobotSendTextNoticeCardReq struct {
	ConfigID       int    `json:"config_id" widget:"name:群机器人配置;type:select;placeholder:留空使用第一个启用配置" callback:"OnSelectFuzzy"`
	SourceDesc     string `json:"source_desc" widget:"name:来源名称;type:input;placeholder:如 KageOS"`
	SourceIconURL  string `json:"source_icon_url" widget:"name:来源图标URL;type:input;placeholder:https://example.com/icon.png"`
	SourceColor    int    `json:"source_color" widget:"name:来源颜色;type:integer;min:0;max:3;step:1;render_default:0"`
	MainTitle      string `json:"main_title" widget:"name:主标题;type:input;placeholder:建议不超过 26 个字"`
	MainDesc       string `json:"main_desc" widget:"name:主标题描述;type:input;placeholder:建议不超过 30 个字"`
	EmphasisTitle  string `json:"emphasis_title" widget:"name:关键数据;type:input;placeholder:如 100"`
	EmphasisDesc   string `json:"emphasis_desc" widget:"name:关键数据说明;type:input;placeholder:如 待处理"`
	QuoteTitle     string `json:"quote_title" widget:"name:引用标题;type:input"`
	QuoteText      string `json:"quote_text" widget:"name:引用内容;type:text_area"`
	QuoteURL       string `json:"quote_url" widget:"name:引用跳转URL;type:input"`
	SubTitleText   string `json:"sub_title_text" widget:"name:二级正文;type:text_area;placeholder:主标题为空时这里必须填写"`
	HorizontalKey  string `json:"horizontal_key" widget:"name:横向条目名;type:input;placeholder:如 状态"`
	HorizontalText string `json:"horizontal_text" widget:"name:横向条目值;type:input;placeholder:如 已完成"`
	HorizontalURL  string `json:"horizontal_url" widget:"name:横向条目URL;type:input"`
	JumpTitle      string `json:"jump_title" widget:"name:跳转入口标题;type:input"`
	JumpURL        string `json:"jump_url" widget:"name:跳转入口URL;type:input"`
	CardActionURL  string `json:"card_action_url" widget:"name:整卡跳转URL;type:input;placeholder:https://example.com" validate:"required"`
}

type WeComGroupRobotSendNewsNoticeCardReq struct {
	ConfigID          int     `json:"config_id" widget:"name:群机器人配置;type:select;placeholder:留空使用第一个启用配置" callback:"OnSelectFuzzy"`
	SourceDesc        string  `json:"source_desc" widget:"name:来源名称;type:input;placeholder:如 KageOS"`
	SourceIconURL     string  `json:"source_icon_url" widget:"name:来源图标URL;type:input;placeholder:https://example.com/icon.png"`
	SourceColor       int     `json:"source_color" widget:"name:来源颜色;type:integer;min:0;max:3;step:1;render_default:0"`
	MainTitle         string  `json:"main_title" widget:"name:主标题;type:input;placeholder:建议不超过 26 个字" validate:"required"`
	MainDesc          string  `json:"main_desc" widget:"name:主标题描述;type:input;placeholder:建议不超过 30 个字"`
	CardImageURL      string  `json:"card_image_url" widget:"name:卡片图片URL;type:input;placeholder:https://example.com/banner.png" validate:"required"`
	CardImageRatio    float64 `json:"card_image_ratio" widget:"name:图片宽高比;type:float;min:1.3;max:2.25;precision:2;step:0.01;render_default:1.3"`
	ImageTextTitle    string  `json:"image_text_title" widget:"name:左图右文标题;type:input"`
	ImageTextDesc     string  `json:"image_text_desc" widget:"name:左图右文描述;type:input"`
	ImageTextImageURL string  `json:"image_text_image_url" widget:"name:左图右文图片URL;type:input"`
	ImageTextURL      string  `json:"image_text_url" widget:"name:左图右文跳转URL;type:input"`
	QuoteTitle        string  `json:"quote_title" widget:"name:引用标题;type:input"`
	QuoteText         string  `json:"quote_text" widget:"name:引用内容;type:text_area"`
	QuoteURL          string  `json:"quote_url" widget:"name:引用跳转URL;type:input"`
	VerticalTitle     string  `json:"vertical_title" widget:"name:垂直内容标题;type:input"`
	VerticalDesc      string  `json:"vertical_desc" widget:"name:垂直内容描述;type:text_area"`
	HorizontalKey     string  `json:"horizontal_key" widget:"name:横向条目名;type:input;placeholder:如 状态"`
	HorizontalText    string  `json:"horizontal_text" widget:"name:横向条目值;type:input;placeholder:如 已完成"`
	HorizontalURL     string  `json:"horizontal_url" widget:"name:横向条目URL;type:input"`
	JumpTitle         string  `json:"jump_title" widget:"name:跳转入口标题;type:input"`
	JumpURL           string  `json:"jump_url" widget:"name:跳转入口URL;type:input"`
	CardActionURL     string  `json:"card_action_url" widget:"name:整卡跳转URL;type:input;placeholder:https://example.com" validate:"required"`
}

type WeComGroupRobotSendRawJSONReq struct {
	ConfigID    int    `json:"config_id" widget:"name:群机器人配置;type:select;placeholder:留空使用第一个启用配置" callback:"OnSelectFuzzy"`
	PayloadJSON string `json:"payload_json" widget:"name:请求JSON;type:text_area;placeholder:粘贴企业微信群机器人请求体，用于高级消息类型" validate:"required"`
}

type WeComGroupRobotSendResp struct {
	Status     string `json:"status" widget:"name:发送状态;type:select;options:成功,失败;options_colors:67C23A,F56C6C"`
	ConfigID   int    `json:"config_id" widget:"name:配置ID;type:integer"`
	ConfigName string `json:"config_name" widget:"name:机器人名称;type:input"`
	GroupName  string `json:"group_name" widget:"name:群名称;type:input"`
	Summary    string `json:"summary" widget:"name:说明;type:text_area"`
}

type WeComGroupRobotUploadMediaResp struct {
	Status         string `json:"status" widget:"name:上传状态;type:select;options:成功,失败;options_colors:67C23A,F56C6C"`
	ConfigID       int    `json:"config_id" widget:"name:配置ID;type:integer"`
	ConfigName     string `json:"config_name" widget:"name:机器人名称;type:input"`
	MediaType      string `json:"media_type" widget:"name:素材类型;type:input"`
	MediaID        string `json:"media_id" widget:"name:media_id;type:input"`
	MediaCreatedAt string `json:"media_created_at" widget:"name:素材创建时间戳;type:input"`
	Summary        string `json:"summary" widget:"name:说明;type:text_area"`
}

type WeComGroupRobotSendMediaResp struct {
	Status     string `json:"status" widget:"name:发送状态;type:select;options:成功,失败;options_colors:67C23A,F56C6C"`
	ConfigID   int    `json:"config_id" widget:"name:配置ID;type:integer"`
	ConfigName string `json:"config_name" widget:"name:机器人名称;type:input"`
	GroupName  string `json:"group_name" widget:"name:群名称;type:input"`
	MediaID    string `json:"media_id" widget:"name:media_id;type:input"`
	Summary    string `json:"summary" widget:"name:说明;type:text_area"`
}

type groupRobotNewsArticle struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
	PicURL      string `json:"picurl,omitempty"`
}

type wecomGroupRobotUploadMediaAPIResp struct {
	groupRobotBaseResp
	Type           string `json:"type"`
	MediaID        string `json:"media_id"`
	MediaCreatedAt string `json:"created_at"`
}

func WeComGroupRobotSendText(ctx *app.Context, resp response.Response) error {
	var req WeComGroupRobotSendTextReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	cfg, webhookURL, err := loadWeComGroupRobotConfig(ctx, req.ConfigID)
	if err != nil {
		return err
	}
	mentionedUsers := splitRobotMentionList(req.MentionedUsers)
	if req.MentionAll {
		mentionedUsers = append([]string{"@all"}, mentionedUsers...)
	}
	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]interface{}{
			"content":               req.Content,
			"mentioned_list":        uniqueStrings(mentionedUsers),
			"mentioned_mobile_list": uniqueStrings(splitRobotMentionList(req.MentionedMobiles)),
		},
	}
	return sendWeComGroupRobotPayload(ctx, resp, cfg, webhookURL, payload, "文本")
}

func WeComGroupRobotSendMarkdown(ctx *app.Context, resp response.Response) error {
	var req WeComGroupRobotSendMarkdownReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	cfg, webhookURL, err := loadWeComGroupRobotConfig(ctx, req.ConfigID)
	if err != nil {
		return err
	}
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": req.Content,
		},
	}
	return sendWeComGroupRobotPayload(ctx, resp, cfg, webhookURL, payload, "Markdown")
}

func WeComGroupRobotSendMarkdownV2(ctx *app.Context, resp response.Response) error {
	var req WeComGroupRobotSendMarkdownV2Req
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	cfg, webhookURL, err := loadWeComGroupRobotConfig(ctx, req.ConfigID)
	if err != nil {
		return err
	}
	payload := map[string]interface{}{
		"msgtype": "markdown_v2",
		"markdown_v2": map[string]string{
			"content": req.Content,
		},
	}
	return sendWeComGroupRobotPayload(ctx, resp, cfg, webhookURL, payload, "Markdown V2")
}

func WeComGroupRobotSendImage(ctx *app.Context, resp response.Response) error {
	var req WeComGroupRobotSendImageReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	cfg, webhookURL, err := loadWeComGroupRobotConfig(ctx, req.ConfigID)
	if err != nil {
		return err
	}
	localPath, cleanup, err := downloadSingleWeComGroupRobotFile(ctx, req.ImageFile, "图片文件")
	if err != nil {
		return err
	}
	defer cleanup()
	imageBytes, err := readWeComGroupRobotFileBytes(localPath, maxRobotImageBytes, "图片文件")
	if err != nil {
		return err
	}
	if err := validateWeComGroupRobotImage(imageBytes); err != nil {
		return err
	}
	sum := md5.Sum(imageBytes)
	payload := map[string]interface{}{
		"msgtype": "image",
		"image": map[string]string{
			"base64": base64.StdEncoding.EncodeToString(imageBytes),
			"md5":    fmt.Sprintf("%x", sum),
		},
	}
	return sendWeComGroupRobotPayload(ctx, resp, cfg, webhookURL, payload, "图片")
}

func WeComGroupRobotSendNews(ctx *app.Context, resp response.Response) error {
	var req WeComGroupRobotSendNewsReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	cfg, webhookURL, err := loadWeComGroupRobotConfig(ctx, req.ConfigID)
	if err != nil {
		return err
	}
	articles, err := buildWeComGroupRobotNewsArticles(req)
	if err != nil {
		return err
	}
	payload := map[string]interface{}{
		"msgtype": "news",
		"news": map[string]interface{}{
			"articles": articles,
		},
	}
	return sendWeComGroupRobotPayload(ctx, resp, cfg, webhookURL, payload, "图文")
}

func WeComGroupRobotUploadMedia(ctx *app.Context, resp response.Response) error {
	var req WeComGroupRobotUploadMediaReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	cfg, webhookURL, err := loadWeComGroupRobotConfig(ctx, req.ConfigID)
	if err != nil {
		return err
	}
	mediaType := normalizeWeComGroupRobotMediaType(req.MediaType)
	localPath, cleanup, err := downloadSingleWeComGroupRobotFile(ctx, req.MediaFile, "素材文件")
	if err != nil {
		return err
	}
	defer cleanup()
	if err := validateWeComGroupRobotMediaFile(localPath, mediaType); err != nil {
		return err
	}
	uploadResp, err := uploadWeComGroupRobotMedia(webhookURL, mediaType, localPath)
	if err != nil {
		updateWeComGroupRobotStatus(ctx, cfg.ID, "失败", err.Error())
		return err
	}
	summary := "企业微信群机器人素材上传成功，media_id 三天内有效且只能用于当前 webhook。"
	updateWeComGroupRobotStatus(ctx, cfg.ID, "正常", summary)
	return resp.Form(&WeComGroupRobotUploadMediaResp{
		Status:         "成功",
		ConfigID:       cfg.ID,
		ConfigName:     cfg.Name,
		MediaType:      uploadResp.Type,
		MediaID:        uploadResp.MediaID,
		MediaCreatedAt: uploadResp.MediaCreatedAt,
		Summary:        summary,
	}).Build()
}

func WeComGroupRobotSendFile(ctx *app.Context, resp response.Response) error {
	var req WeComGroupRobotSendFileReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	cfg, webhookURL, err := loadWeComGroupRobotConfig(ctx, req.ConfigID)
	if err != nil {
		return err
	}
	mediaID := strings.TrimSpace(req.MediaID)
	if mediaID == "" && strings.TrimSpace(req.File) == "" {
		return fmt.Errorf("请填写 media_id 或上传要发送的文件")
	}
	if mediaID == "" {
		localPath, cleanup, err := downloadSingleWeComGroupRobotFile(ctx, req.File, "文件")
		if err != nil {
			return err
		}
		defer cleanup()
		if err := validateWeComGroupRobotMediaFile(localPath, "file"); err != nil {
			return err
		}
		uploadResp, err := uploadWeComGroupRobotMedia(webhookURL, "file", localPath)
		if err != nil {
			updateWeComGroupRobotStatus(ctx, cfg.ID, "失败", err.Error())
			return err
		}
		mediaID = uploadResp.MediaID
	}
	if mediaID == "" {
		return fmt.Errorf("请填写 media_id 或上传要发送的文件")
	}
	payload := map[string]interface{}{
		"msgtype": "file",
		"file": map[string]string{
			"media_id": mediaID,
		},
	}
	return sendWeComGroupRobotMediaPayload(ctx, resp, cfg, webhookURL, payload, "文件", mediaID)
}

func WeComGroupRobotSendVoice(ctx *app.Context, resp response.Response) error {
	var req WeComGroupRobotSendVoiceReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	cfg, webhookURL, err := loadWeComGroupRobotConfig(ctx, req.ConfigID)
	if err != nil {
		return err
	}
	mediaID := strings.TrimSpace(req.MediaID)
	if mediaID == "" && strings.TrimSpace(req.VoiceFile) == "" {
		return fmt.Errorf("请填写 media_id 或上传要发送的 AMR 语音文件")
	}
	if mediaID == "" {
		localPath, cleanup, err := downloadSingleWeComGroupRobotFile(ctx, req.VoiceFile, "语音文件")
		if err != nil {
			return err
		}
		defer cleanup()
		if err := validateWeComGroupRobotMediaFile(localPath, "voice"); err != nil {
			return err
		}
		uploadResp, err := uploadWeComGroupRobotMedia(webhookURL, "voice", localPath)
		if err != nil {
			updateWeComGroupRobotStatus(ctx, cfg.ID, "失败", err.Error())
			return err
		}
		mediaID = uploadResp.MediaID
	}
	if mediaID == "" {
		return fmt.Errorf("请填写 media_id 或上传要发送的 AMR 语音文件")
	}
	payload := map[string]interface{}{
		"msgtype": "voice",
		"voice": map[string]string{
			"media_id": mediaID,
		},
	}
	return sendWeComGroupRobotMediaPayload(ctx, resp, cfg, webhookURL, payload, "语音", mediaID)
}

func WeComGroupRobotSendTextNoticeCard(ctx *app.Context, resp response.Response) error {
	var req WeComGroupRobotSendTextNoticeCardReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	cfg, webhookURL, err := loadWeComGroupRobotConfig(ctx, req.ConfigID)
	if err != nil {
		return err
	}
	card, err := buildWeComGroupRobotTextNoticeCard(req)
	if err != nil {
		return err
	}
	payload := map[string]interface{}{
		"msgtype":       "template_card",
		"template_card": card,
	}
	return sendWeComGroupRobotPayload(ctx, resp, cfg, webhookURL, payload, "文本通知卡片")
}

func WeComGroupRobotSendNewsNoticeCard(ctx *app.Context, resp response.Response) error {
	var req WeComGroupRobotSendNewsNoticeCardReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	cfg, webhookURL, err := loadWeComGroupRobotConfig(ctx, req.ConfigID)
	if err != nil {
		return err
	}
	card, err := buildWeComGroupRobotNewsNoticeCard(req)
	if err != nil {
		return err
	}
	payload := map[string]interface{}{
		"msgtype":       "template_card",
		"template_card": card,
	}
	return sendWeComGroupRobotPayload(ctx, resp, cfg, webhookURL, payload, "图文通知卡片")
}

func WeComGroupRobotSendRawJSON(ctx *app.Context, resp response.Response) error {
	var req WeComGroupRobotSendRawJSONReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	cfg, webhookURL, err := loadWeComGroupRobotConfig(ctx, req.ConfigID)
	if err != nil {
		return err
	}
	payload, err := parseWeComGroupRobotRawPayload(req.PayloadJSON)
	if err != nil {
		return err
	}
	messageType := fmt.Sprintf("自定义 %s", strings.TrimSpace(fmt.Sprint(payload["msgtype"])))
	return sendWeComGroupRobotPayload(ctx, resp, cfg, webhookURL, payload, messageType)
}

func sendWeComGroupRobotPayload(ctx *app.Context, resp response.Response, cfg *WeComGroupRobotConfig, webhookURL string, payload map[string]interface{}, messageType string) error {
	if err := postWeComGroupRobot(webhookURL, payload); err != nil {
		updateWeComGroupRobotStatus(ctx, cfg.ID, "失败", err.Error())
		return err
	}
	summary := fmt.Sprintf("企业微信群机器人%s消息发送成功。", messageType)
	updateWeComGroupRobotStatus(ctx, cfg.ID, "正常", summary)
	return resp.Form(&WeComGroupRobotSendResp{
		Status:     "成功",
		ConfigID:   cfg.ID,
		ConfigName: cfg.Name,
		GroupName:  cfg.GroupName,
		Summary:    summary,
	}).Build()
}

func sendWeComGroupRobotMediaPayload(ctx *app.Context, resp response.Response, cfg *WeComGroupRobotConfig, webhookURL string, payload map[string]interface{}, messageType, mediaID string) error {
	if err := postWeComGroupRobot(webhookURL, payload); err != nil {
		updateWeComGroupRobotStatus(ctx, cfg.ID, "失败", err.Error())
		return err
	}
	summary := fmt.Sprintf("企业微信群机器人%s消息发送成功。", messageType)
	updateWeComGroupRobotStatus(ctx, cfg.ID, "正常", summary)
	return resp.Form(&WeComGroupRobotSendMediaResp{
		Status:     "成功",
		ConfigID:   cfg.ID,
		ConfigName: cfg.Name,
		GroupName:  cfg.GroupName,
		MediaID:    mediaID,
		Summary:    summary,
	}).Build()
}

func buildWeComGroupRobotNewsArticles(req WeComGroupRobotSendNewsReq) ([]groupRobotNewsArticle, error) {
	main := groupRobotNewsArticle{
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		URL:         strings.TrimSpace(req.URL),
		PicURL:      strings.TrimSpace(req.PicURL),
	}
	if main.Title == "" || main.URL == "" {
		return nil, fmt.Errorf("图文标题和跳转链接不能为空")
	}
	articles := []groupRobotNewsArticle{main}
	if strings.TrimSpace(req.ExtraArticlesJSON) != "" {
		var extras []groupRobotNewsArticle
		dec := json.NewDecoder(strings.NewReader(req.ExtraArticlesJSON))
		if err := dec.Decode(&extras); err != nil {
			return nil, fmt.Errorf("附加图文 JSON 解析失败: %w", err)
		}
		for _, item := range extras {
			item.Title = strings.TrimSpace(item.Title)
			item.Description = strings.TrimSpace(item.Description)
			item.URL = strings.TrimSpace(item.URL)
			item.PicURL = strings.TrimSpace(item.PicURL)
			if item.Title == "" || item.URL == "" {
				return nil, fmt.Errorf("附加图文的 title 和 url 不能为空")
			}
			articles = append(articles, item)
		}
	}
	if len(articles) > 8 {
		return nil, fmt.Errorf("企业微信群机器人图文消息最多支持 8 条")
	}
	return articles, nil
}

func buildWeComGroupRobotTextNoticeCard(req WeComGroupRobotSendTextNoticeCardReq) (map[string]interface{}, error) {
	req.MainTitle = strings.TrimSpace(req.MainTitle)
	req.MainDesc = strings.TrimSpace(req.MainDesc)
	req.SubTitleText = strings.TrimSpace(req.SubTitleText)
	req.CardActionURL = strings.TrimSpace(req.CardActionURL)
	if req.MainTitle == "" && req.SubTitleText == "" {
		return nil, fmt.Errorf("文本通知卡片的主标题和二级正文至少填写一个")
	}
	if req.CardActionURL == "" {
		return nil, fmt.Errorf("文本通知卡片必须填写整卡跳转 URL")
	}
	card := map[string]interface{}{
		"card_type": "text_notice",
		"main_title": map[string]string{
			"title": req.MainTitle,
			"desc":  req.MainDesc,
		},
		"card_action": map[string]interface{}{
			"type": 1,
			"url":  req.CardActionURL,
		},
	}
	addWeComGroupRobotCardSource(card, req.SourceDesc, req.SourceIconURL, req.SourceColor)
	if strings.TrimSpace(req.EmphasisTitle) != "" || strings.TrimSpace(req.EmphasisDesc) != "" {
		card["emphasis_content"] = map[string]string{
			"title": strings.TrimSpace(req.EmphasisTitle),
			"desc":  strings.TrimSpace(req.EmphasisDesc),
		}
	}
	addWeComGroupRobotQuoteArea(card, req.QuoteTitle, req.QuoteText, req.QuoteURL)
	if req.SubTitleText != "" {
		card["sub_title_text"] = req.SubTitleText
	}
	addWeComGroupRobotHorizontalContent(card, req.HorizontalKey, req.HorizontalText, req.HorizontalURL)
	addWeComGroupRobotJump(card, req.JumpTitle, req.JumpURL)
	return card, nil
}

func buildWeComGroupRobotNewsNoticeCard(req WeComGroupRobotSendNewsNoticeCardReq) (map[string]interface{}, error) {
	req.MainTitle = strings.TrimSpace(req.MainTitle)
	req.MainDesc = strings.TrimSpace(req.MainDesc)
	req.CardImageURL = strings.TrimSpace(req.CardImageURL)
	req.CardActionURL = strings.TrimSpace(req.CardActionURL)
	if req.MainTitle == "" || req.CardImageURL == "" || req.CardActionURL == "" {
		return nil, fmt.Errorf("图文通知卡片必须填写主标题、卡片图片 URL 和整卡跳转 URL")
	}
	if req.CardImageRatio == 0 {
		req.CardImageRatio = 1.3
	}
	if req.CardImageRatio < 1.3 || req.CardImageRatio > 2.25 {
		return nil, fmt.Errorf("图文通知卡片图片宽高比必须在 1.3 到 2.25 之间")
	}
	card := map[string]interface{}{
		"card_type": "news_notice",
		"main_title": map[string]string{
			"title": req.MainTitle,
			"desc":  req.MainDesc,
		},
		"card_image": map[string]interface{}{
			"url":          req.CardImageURL,
			"aspect_ratio": req.CardImageRatio,
		},
		"card_action": map[string]interface{}{
			"type": 1,
			"url":  req.CardActionURL,
		},
	}
	addWeComGroupRobotCardSource(card, req.SourceDesc, req.SourceIconURL, req.SourceColor)
	if hasAnyText(req.ImageTextTitle, req.ImageTextDesc, req.ImageTextImageURL, req.ImageTextURL) {
		if strings.TrimSpace(req.ImageTextImageURL) == "" {
			return nil, fmt.Errorf("左图右文区域启用时必须填写图片 URL")
		}
		imageTextArea := map[string]interface{}{
			"title":     strings.TrimSpace(req.ImageTextTitle),
			"desc":      strings.TrimSpace(req.ImageTextDesc),
			"image_url": strings.TrimSpace(req.ImageTextImageURL),
		}
		if strings.TrimSpace(req.ImageTextURL) != "" {
			imageTextArea["type"] = 1
			imageTextArea["url"] = strings.TrimSpace(req.ImageTextURL)
		}
		card["image_text_area"] = imageTextArea
	}
	addWeComGroupRobotQuoteArea(card, req.QuoteTitle, req.QuoteText, req.QuoteURL)
	if strings.TrimSpace(req.VerticalTitle) != "" {
		card["vertical_content_list"] = []map[string]string{{
			"title": strings.TrimSpace(req.VerticalTitle),
			"desc":  strings.TrimSpace(req.VerticalDesc),
		}}
	}
	addWeComGroupRobotHorizontalContent(card, req.HorizontalKey, req.HorizontalText, req.HorizontalURL)
	addWeComGroupRobotJump(card, req.JumpTitle, req.JumpURL)
	return card, nil
}

func addWeComGroupRobotCardSource(card map[string]interface{}, desc, iconURL string, color int) {
	desc = strings.TrimSpace(desc)
	iconURL = strings.TrimSpace(iconURL)
	if desc == "" && iconURL == "" && color == 0 {
		return
	}
	if color < 0 || color > 3 {
		color = 0
	}
	card["source"] = map[string]interface{}{
		"icon_url":   iconURL,
		"desc":       desc,
		"desc_color": color,
	}
}

func addWeComGroupRobotQuoteArea(card map[string]interface{}, title, text, rawURL string) {
	title = strings.TrimSpace(title)
	text = strings.TrimSpace(text)
	rawURL = strings.TrimSpace(rawURL)
	if title == "" && text == "" && rawURL == "" {
		return
	}
	quoteArea := map[string]interface{}{
		"title":      title,
		"quote_text": text,
	}
	if rawURL != "" {
		quoteArea["type"] = 1
		quoteArea["url"] = rawURL
	}
	card["quote_area"] = quoteArea
}

func addWeComGroupRobotHorizontalContent(card map[string]interface{}, key, text, rawURL string) {
	key = strings.TrimSpace(key)
	text = strings.TrimSpace(text)
	rawURL = strings.TrimSpace(rawURL)
	if key == "" && text == "" && rawURL == "" {
		return
	}
	if key == "" {
		key = "详情"
	}
	item := map[string]interface{}{
		"keyname": key,
		"value":   text,
	}
	if rawURL != "" {
		item["type"] = 1
		item["url"] = rawURL
	}
	card["horizontal_content_list"] = []map[string]interface{}{item}
}

func addWeComGroupRobotJump(card map[string]interface{}, title, rawURL string) {
	title = strings.TrimSpace(title)
	rawURL = strings.TrimSpace(rawURL)
	if title == "" && rawURL == "" {
		return
	}
	if title == "" {
		title = "查看详情"
	}
	item := map[string]interface{}{"title": title}
	if rawURL != "" {
		item["type"] = 1
		item["url"] = rawURL
	}
	card["jump_list"] = []map[string]interface{}{item}
}

func parseWeComGroupRobotRawPayload(value string) (map[string]interface{}, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("请求 JSON 不能为空")
	}
	var payload map[string]interface{}
	dec := json.NewDecoder(strings.NewReader(value))
	dec.UseNumber()
	if err := dec.Decode(&payload); err != nil {
		return nil, fmt.Errorf("请求 JSON 解析失败: %w", err)
	}
	msgType, _ := payload["msgtype"].(string)
	if strings.TrimSpace(msgType) == "" {
		return nil, fmt.Errorf("请求 JSON 必须包含 msgtype")
	}
	return payload, nil
}

func downloadSingleWeComGroupRobotFile(ctx *app.Context, fileRefs, label string) (string, func(), error) {
	fs := ctx.GetFS()
	files := fs.DownloadFiles(fileRefs)
	cleanup := func() {
		fs.RemoveFiles(files)
	}
	if len(files) == 0 {
		cleanup()
		return "", func() {}, fmt.Errorf("%s下载失败，请确认已经上传文件", label)
	}
	return files[0], cleanup, nil
}

func readWeComGroupRobotFileBytes(filePath string, maxBytes int64, label string) ([]byte, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取%s失败: %w", label, err)
	}
	if stat.Size() <= 0 {
		return nil, fmt.Errorf("%s不能为空", label)
	}
	if stat.Size() > maxBytes {
		return nil, fmt.Errorf("%s大小不能超过 %dMB", label, maxBytes/(1024*1024))
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取%s内容失败: %w", label, err)
	}
	return data, nil
}

func validateWeComGroupRobotImage(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("图片文件不能为空")
	}
	contentType := http.DetectContentType(data)
	if contentType != "image/jpeg" && contentType != "image/png" {
		return fmt.Errorf("企业微信群机器人图片消息只支持 JPG 或 PNG，当前检测为 %s", contentType)
	}
	return nil
}

func validateWeComGroupRobotMediaFile(filePath, mediaType string) error {
	stat, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("读取素材文件失败: %w", err)
	}
	if stat.Size() <= 5 {
		return fmt.Errorf("企业微信群机器人上传素材必须大于 5 字节")
	}
	switch mediaType {
	case "voice":
		if stat.Size() > maxRobotVoiceBytes {
			return fmt.Errorf("企业微信群机器人语音素材不能超过 2MB")
		}
		if strings.ToLower(filepath.Ext(filePath)) != ".amr" {
			return fmt.Errorf("企业微信群机器人语音素材仅支持 AMR 文件")
		}
	default:
		if stat.Size() > maxRobotFileBytes {
			return fmt.Errorf("企业微信群机器人普通文件不能超过 20MB")
		}
	}
	return nil
}

func uploadWeComGroupRobotMedia(webhookURL, mediaType, filePath string) (*wecomGroupRobotUploadMediaAPIResp, error) {
	if err := validateWeComGroupRobotWebhook(webhookURL); err != nil {
		return nil, err
	}
	mediaType = normalizeWeComGroupRobotMediaType(mediaType)
	key, err := weComGroupRobotWebhookKey(webhookURL)
	if err != nil {
		return nil, err
	}
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取素材文件失败: %w", err)
	}
	values := url.Values{}
	values.Set("key", key)
	values.Set("type", mediaType)
	endpoint := "https://qyapi.weixin.qq.com" + wecomGroupRobotUploadMediaPath + "?" + values.Encode()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开素材文件失败: %w", err)
	}
	defer file.Close()
	header := textproto.MIMEHeader{}
	filename := safeWeComGroupRobotMultipartFilename(filepath.Base(filePath))
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="media";filename="%s"; filelength=%d`, filename, stat.Size()))
	header.Set("Content-Type", weComGroupRobotMediaContentType(mediaType))
	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, fmt.Errorf("创建企业微信群机器人上传表单失败: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("写入企业微信群机器人上传表单失败: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("关闭企业微信群机器人上传表单失败: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, endpoint, &body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.ContentLength = int64(body.Len())
	httpResp, err := wecomGroupRobotHTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求企业微信群机器人上传素材失败: %w", err)
	}
	defer httpResp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(httpResp.Body, 2<<20))
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("企业微信群机器人上传素材 HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}
	var parsed wecomGroupRobotUploadMediaAPIResp
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("解析企业微信群机器人上传素材响应失败: %w", err)
	}
	if parsed.ErrCode != 0 {
		return nil, fmt.Errorf("企业微信群机器人上传素材失败 [%d]: %s", parsed.ErrCode, humanWeComRobotError(parsed.ErrCode, parsed.ErrMsg))
	}
	if strings.TrimSpace(parsed.MediaID) == "" {
		return nil, fmt.Errorf("企业微信群机器人上传素材未返回 media_id")
	}
	return &parsed, nil
}

func normalizeWeComGroupRobotMediaType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "voice", "语音":
		return "voice"
	default:
		return "file"
	}
}

func weComGroupRobotWebhookKey(webhookURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(webhookURL))
	if err != nil {
		return "", fmt.Errorf("企业微信群机器人 Webhook 格式不正确: %w", err)
	}
	key := strings.TrimSpace(parsed.Query().Get("key"))
	if key == "" {
		return "", fmt.Errorf("企业微信群机器人 Webhook 缺少 key 参数")
	}
	return key, nil
}

func weComGroupRobotMediaContentType(mediaType string) string {
	if mediaType == "voice" {
		return "audio/amr"
	}
	return "application/octet-stream"
}

func safeWeComGroupRobotMultipartFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = "upload"
	}
	ext := filepath.Ext(name)
	cleaned := strings.Map(func(r rune) rune {
		if r == '\\' || r == '"' || r == '\r' || r == '\n' {
			return -1
		}
		if r < 0x20 || r > 0x7e {
			return -1
		}
		return r
	}, name)
	if strings.Trim(cleaned, ". ") == "" {
		cleaned = "upload" + ext
	}
	if filepath.Ext(cleaned) == "" && ext != "" {
		cleaned += ext
	}
	return cleaned
}

func hasAnyText(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func wecomGroupRobotDB(ctx *app.Context) (*gorm.DB, error) {
	db := ctx.GetGormDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接失败")
	}
	if err := db.AutoMigrate(&WeComGroupRobotConfig{}); err != nil {
		return nil, fmt.Errorf("初始化企业微信群机器人配置表失败: %w", err)
	}
	return db, nil
}

func loadWeComGroupRobotConfig(ctx *app.Context, configID int) (*WeComGroupRobotConfig, string, error) {
	db, err := wecomGroupRobotDB(ctx)
	if err != nil {
		return nil, "", err
	}
	var cfg WeComGroupRobotConfig
	query := db.Model(&WeComGroupRobotConfig{})
	if configID > 0 {
		query = query.Where("id = ?", configID)
	} else {
		query = query.Where("enabled = ?", true).Order("id ASC")
	}
	if err := query.First(&cfg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if configID > 0 {
				return nil, "", fmt.Errorf("未找到企业微信群机器人配置 ID=%d", configID)
			}
			return nil, "", fmt.Errorf("还没有可用的企业微信群机器人配置，请先打开“群机器人配置”填写 webhook 地址")
		}
		return nil, "", err
	}
	if !cfg.Enabled {
		return nil, "", fmt.Errorf("企业微信群机器人“%s”已停用", cfg.Name)
	}
	webhookURL, err := decryptGroupRobotSecret(cfg.WebhookURLCipher)
	if err != nil {
		return nil, "", err
	}
	if err := validateWeComGroupRobotWebhook(webhookURL); err != nil {
		return nil, "", err
	}
	return &cfg, webhookURL, nil
}

func validateWeComGroupRobotWebhook(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("企业微信群机器人 Webhook 不能为空")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("企业微信群机器人 Webhook 格式不正确: %w", err)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("企业微信群机器人 Webhook 必须使用 https")
	}
	if parsed.Host != "qyapi.weixin.qq.com" {
		return fmt.Errorf("企业微信群机器人 Webhook 域名应为 qyapi.weixin.qq.com")
	}
	if parsed.Path != wecomGroupRobotSendPath {
		return fmt.Errorf("企业微信群机器人 Webhook 路径应为 %s", wecomGroupRobotSendPath)
	}
	if strings.TrimSpace(parsed.Query().Get("key")) == "" {
		return fmt.Errorf("企业微信群机器人 Webhook 缺少 key 参数")
	}
	return nil
}

func postWeComGroupRobot(webhookURL string, payload map[string]interface{}) error {
	if err := validateWeComGroupRobotWebhook(webhookURL); err != nil {
		return err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化企业微信群机器人请求失败: %w", err)
	}
	httpReq, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json; charset=utf-8")
	httpResp, err := wecomGroupRobotHTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("请求企业微信群机器人失败: %w", err)
	}
	defer httpResp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return fmt.Errorf("企业微信群机器人 HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}
	var parsed groupRobotBaseResp
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return fmt.Errorf("解析企业微信群机器人响应失败: %w", err)
	}
	if parsed.ErrCode != 0 {
		return fmt.Errorf("企业微信群机器人发送失败 [%d]: %s", parsed.ErrCode, humanWeComRobotError(parsed.ErrCode, parsed.ErrMsg))
	}
	return nil
}

func updateWeComGroupRobotStatus(ctx *app.Context, id int, status, message string) {
	db, err := wecomGroupRobotDB(ctx)
	if err != nil {
		logger.Errorf(ctx, "更新企业微信群机器人状态失败: %v", err)
		return
	}
	if err := db.Model(&WeComGroupRobotConfig{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_status":  status,
		"last_message": message,
	}).Error; err != nil {
		logger.Errorf(ctx, "更新企业微信群机器人状态失败 id=%d err=%v", id, err)
	}
}

func wecomGroupRobotOnSelectFuzzy(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db, err := wecomGroupRobotDB(ctx)
	if err != nil {
		return nil, err
	}
	queryDB := db.Model(&WeComGroupRobotConfig{})
	if req != nil && !req.IsByKeyword() {
		switch value := req.GetValue().(type) {
		case int:
			if value > 0 {
				queryDB = queryDB.Where("id = ?", value)
			}
		case float64:
			if value > 0 {
				queryDB = queryDB.Where("id = ?", int(value))
			}
		case string:
			if strings.TrimSpace(value) != "" {
				queryDB = queryDB.Where("id = ? OR name LIKE ? OR group_name LIKE ?", strings.TrimSpace(value), "%"+strings.TrimSpace(value)+"%", "%"+strings.TrimSpace(value)+"%")
			}
		}
	} else {
		keyword := ""
		if req != nil {
			keyword = strings.TrimSpace(req.Keyword())
		}
		if keyword != "" {
			queryDB = queryDB.Where("name LIKE ? OR group_name LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
		}
	}
	var rows []WeComGroupRobotConfig
	if err := queryDB.Order("id ASC").Limit(20).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]*callback.SelectFuzzyItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &callback.SelectFuzzyItem{
			Value: row.ID,
			Label: fmt.Sprintf("%s (#%d)", row.Name, row.ID),
			DisplayInfo: map[string]interface{}{
				"群名称": row.GroupName,
				"启用":  row.Enabled,
				"状态":  firstNonEmptyRobot(row.LastStatus, "未测试"),
			},
		})
	}
	return &callback.OnSelectFuzzyResp{Items: items}, nil
}

func maskWebhookURL(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return maskGroupRobotSecret(rawURL)
	}
	key := parsed.Query().Get("key")
	if key == "" {
		return parsed.Scheme + "://" + parsed.Host + parsed.Path
	}
	values := parsed.Query()
	values.Set("key", maskGroupRobotSecret(key))
	parsed.RawQuery = values.Encode()
	return parsed.String()
}

func splitRobotMentionList(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == '|' || r == ',' || r == '，' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			result = append(result, field)
		}
	}
	return result
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func humanWeComRobotError(code int, msg string) string {
	msg = strings.TrimSpace(msg)
	hint := ""
	switch code {
	case 93000:
		hint = "Webhook key 不存在或已失效，请重新复制群机器人 Webhook"
	case 93004:
		hint = "机器人发送频率过高，请稍后重试"
	case 301019:
		hint = "消息内容可能命中了企业微信安全策略"
	default:
		hint = "请检查 Webhook 地址、群机器人是否仍存在、消息内容是否符合企业微信限制"
	}
	if msg == "" {
		return hint
	}
	return hint + "；原始错误: " + msg
}

func encryptGroupRobotSecret(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	gcm, err := newGroupRobotSecretCipher()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成企业微信群机器人密钥随机数失败: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, []byte(value), nil)
	return groupRobotSecretCipherPrefix + base64.RawURLEncoding.EncodeToString(sealed), nil
}

func decryptGroupRobotSecret(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if !strings.HasPrefix(value, groupRobotSecretCipherPrefix) {
		return value, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, groupRobotSecretCipherPrefix))
	if err != nil {
		return "", fmt.Errorf("解析企业微信群机器人密文失败: %w", err)
	}
	gcm, err := newGroupRobotSecretCipher()
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("企业微信群机器人密文格式不完整")
	}
	nonce := raw[:gcm.NonceSize()]
	cipherText := raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", fmt.Errorf("解密企业微信群机器人密文失败: %w", err)
	}
	return string(plain), nil
}

func newGroupRobotSecretCipher() (cipher.AEAD, error) {
	seed := strings.TrimSpace(os.Getenv("KAGEOS_WECOM_GROUP_ROBOT_SECRET_KEY"))
	if seed == "" {
		seed = strings.TrimSpace(env.User) + ":" + strings.TrimSpace(env.App)
	}
	if strings.Trim(seed, ":") == "" {
		seed = "kageos-system-connector-wecom-group-robot"
	}
	sum := sha256.Sum256([]byte("kageos-wecom-group-robot-secret-v1:" + seed))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, fmt.Errorf("初始化企业微信群机器人密钥加密器失败: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("初始化企业微信群机器人密钥 GCM 失败: %w", err)
	}
	return gcm, nil
}

func maskGroupRobotSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
}

func firstNonEmptyRobot(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

var WeComGroupRobotConfigListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "企业微信群机器人列表",
		Desc:         "查看已保存的企业微信群机器人 Webhook 配置。Webhook 地址按密文保存，不在列表展示。",
		Tags:         []string{"企业微信", "群机器人", "Webhook"},
		Request:      &WeComGroupRobotConfigListReq{},
		CreateTables: []interface{}{&WeComGroupRobotConfig{}},
	},
	AutoCrudTable: &WeComGroupRobotConfigListItem{},
}

var WeComGroupRobotConfigSaveTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "企业微信群机器人配置",
		Desc:         "保存企业微信群机器人的 Webhook 地址，用于轻量通知，不需要 corp_id、agent_id、可信 IP 或公网回调。",
		Tags:         []string{"企业微信", "群机器人", "Webhook"},
		Request:      &WeComGroupRobotConfigSaveReq{},
		Response:     &WeComGroupRobotConfigSaveResp{},
		CreateTables: []interface{}{&WeComGroupRobotConfig{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"config_id": wecomGroupRobotOnSelectFuzzy,
		},
	},
}

var WeComGroupRobotStatusTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "企业微信群机器人状态",
		Desc:         "检查企业微信群机器人 Webhook 配置格式。这个检查不会主动向群里发送消息。",
		Tags:         []string{"企业微信", "群机器人", "状态"},
		Request:      &WeComGroupRobotStatusReq{},
		Response:     &WeComGroupRobotStatusResp{},
		CreateTables: []interface{}{&WeComGroupRobotConfig{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"config_id": wecomGroupRobotOnSelectFuzzy,
		},
	},
}

var WeComGroupRobotSendTextTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "企业微信群机器人发送文本",
		Desc:         "通过企业微信群机器人 Webhook 发送文本消息，支持按 UserID 或手机号 @ 群成员。",
		Tags:         []string{"企业微信", "群机器人", "文本通知"},
		Request:      &WeComGroupRobotSendTextReq{},
		Response:     &WeComGroupRobotSendResp{},
		CreateTables: []interface{}{&WeComGroupRobotConfig{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"config_id": wecomGroupRobotOnSelectFuzzy,
		},
	},
}

var WeComGroupRobotSendMarkdownTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "企业微信群机器人发送 Markdown",
		Desc:         "通过企业微信群机器人 Webhook 发送 Markdown 消息，适合任务结果、告警和报告链接通知。",
		Tags:         []string{"企业微信", "群机器人", "Markdown通知"},
		Request:      &WeComGroupRobotSendMarkdownReq{},
		Response:     &WeComGroupRobotSendResp{},
		CreateTables: []interface{}{&WeComGroupRobotConfig{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"config_id": wecomGroupRobotOnSelectFuzzy,
		},
	},
}

var WeComGroupRobotSendMarkdownV2Template = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "企业微信群机器人发送 Markdown V2",
		Desc:         "通过企业微信群机器人 Webhook 发送 markdown_v2 消息，支持表格、图片链接、代码块等新版语法。",
		Tags:         []string{"企业微信", "群机器人", "Markdown V2"},
		Request:      &WeComGroupRobotSendMarkdownV2Req{},
		Response:     &WeComGroupRobotSendResp{},
		CreateTables: []interface{}{&WeComGroupRobotConfig{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"config_id": wecomGroupRobotOnSelectFuzzy,
		},
	},
}

var WeComGroupRobotSendImageTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "企业微信群机器人发送图片",
		Desc:         "上传 JPG 或 PNG 图片后，通过企业微信群机器人 Webhook 发送 image 消息。",
		Tags:         []string{"企业微信", "群机器人", "图片通知"},
		Request:      &WeComGroupRobotSendImageReq{},
		Response:     &WeComGroupRobotSendResp{},
		CreateTables: []interface{}{&WeComGroupRobotConfig{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"config_id": wecomGroupRobotOnSelectFuzzy,
		},
	},
}

var WeComGroupRobotSendNewsTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "企业微信群机器人发送图文",
		Desc:         "通过企业微信群机器人 Webhook 发送 news 图文消息，支持 1 到 8 条图文。",
		Tags:         []string{"企业微信", "群机器人", "图文通知"},
		Request:      &WeComGroupRobotSendNewsReq{},
		Response:     &WeComGroupRobotSendResp{},
		CreateTables: []interface{}{&WeComGroupRobotConfig{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"config_id": wecomGroupRobotOnSelectFuzzy,
		},
	},
}

var WeComGroupRobotUploadMediaTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "企业微信群机器人上传素材",
		Desc:         "通过企业微信群机器人 Webhook 上传普通文件或 AMR 语音，返回三天内有效的 media_id。",
		Tags:         []string{"企业微信", "群机器人", "素材上传"},
		Request:      &WeComGroupRobotUploadMediaReq{},
		Response:     &WeComGroupRobotUploadMediaResp{},
		CreateTables: []interface{}{&WeComGroupRobotConfig{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"config_id": wecomGroupRobotOnSelectFuzzy,
		},
	},
}

var WeComGroupRobotSendFileTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "企业微信群机器人发送文件",
		Desc:         "通过企业微信群机器人 Webhook 发送 file 消息，可填写已有 media_id，也可上传文件后自动发送。",
		Tags:         []string{"企业微信", "群机器人", "文件通知"},
		Request:      &WeComGroupRobotSendFileReq{},
		Response:     &WeComGroupRobotSendMediaResp{},
		CreateTables: []interface{}{&WeComGroupRobotConfig{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"config_id": wecomGroupRobotOnSelectFuzzy,
		},
	},
}

var WeComGroupRobotSendVoiceTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "企业微信群机器人发送语音",
		Desc:         "通过企业微信群机器人 Webhook 发送 voice 消息，可填写已有 media_id，也可上传 AMR 文件后自动发送。",
		Tags:         []string{"企业微信", "群机器人", "语音通知"},
		Request:      &WeComGroupRobotSendVoiceReq{},
		Response:     &WeComGroupRobotSendMediaResp{},
		CreateTables: []interface{}{&WeComGroupRobotConfig{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"config_id": wecomGroupRobotOnSelectFuzzy,
		},
	},
}

var WeComGroupRobotSendTextNoticeCardTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "企业微信群机器人发送文本卡片",
		Desc:         "通过企业微信群机器人 Webhook 发送 text_notice 模板卡片，适合状态通知和审批入口。",
		Tags:         []string{"企业微信", "群机器人", "模板卡片"},
		Request:      &WeComGroupRobotSendTextNoticeCardReq{},
		Response:     &WeComGroupRobotSendResp{},
		CreateTables: []interface{}{&WeComGroupRobotConfig{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"config_id": wecomGroupRobotOnSelectFuzzy,
		},
	},
}

var WeComGroupRobotSendNewsNoticeCardTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "企业微信群机器人发送图文卡片",
		Desc:         "通过企业微信群机器人 Webhook 发送 news_notice 模板卡片，适合带图片的报告和公告。",
		Tags:         []string{"企业微信", "群机器人", "模板卡片"},
		Request:      &WeComGroupRobotSendNewsNoticeCardReq{},
		Response:     &WeComGroupRobotSendResp{},
		CreateTables: []interface{}{&WeComGroupRobotConfig{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"config_id": wecomGroupRobotOnSelectFuzzy,
		},
	},
}

var WeComGroupRobotSendRawJSONTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "企业微信群机器人发送原始 JSON",
		Desc:         "通过企业微信群机器人 Webhook 透传原始 JSON 请求体，用于快速试验复杂模板卡片或未来新增消息类型。",
		Tags:         []string{"企业微信", "群机器人", "高级发送"},
		Request:      &WeComGroupRobotSendRawJSONReq{},
		Response:     &WeComGroupRobotSendResp{},
		CreateTables: []interface{}{&WeComGroupRobotConfig{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"config_id": wecomGroupRobotOnSelectFuzzy,
		},
	},
}

func init() {
	packageContext.GET("configs.table", WeComGroupRobotConfigList, WeComGroupRobotConfigListTemplate)
	packageContext.POST("config.form", WeComGroupRobotConfigSave, WeComGroupRobotConfigSaveTemplate)
	packageContext.POST("connection_status.form", WeComGroupRobotStatus, WeComGroupRobotStatusTemplate)
	packageContext.POST("send_text.form", WeComGroupRobotSendText, WeComGroupRobotSendTextTemplate)
	packageContext.POST("send_markdown.form", WeComGroupRobotSendMarkdown, WeComGroupRobotSendMarkdownTemplate)
	packageContext.POST("send_markdown_v2.form", WeComGroupRobotSendMarkdownV2, WeComGroupRobotSendMarkdownV2Template)
	packageContext.POST("send_image.form", WeComGroupRobotSendImage, WeComGroupRobotSendImageTemplate)
	packageContext.POST("send_news.form", WeComGroupRobotSendNews, WeComGroupRobotSendNewsTemplate)
	packageContext.POST("upload_media.form", WeComGroupRobotUploadMedia, WeComGroupRobotUploadMediaTemplate)
	packageContext.POST("send_file.form", WeComGroupRobotSendFile, WeComGroupRobotSendFileTemplate)
	packageContext.POST("send_voice.form", WeComGroupRobotSendVoice, WeComGroupRobotSendVoiceTemplate)
	packageContext.POST("send_text_notice_card.form", WeComGroupRobotSendTextNoticeCard, WeComGroupRobotSendTextNoticeCardTemplate)
	packageContext.POST("send_news_notice_card.form", WeComGroupRobotSendNewsNoticeCard, WeComGroupRobotSendNewsNoticeCardTemplate)
	packageContext.POST("send_raw_json.form", WeComGroupRobotSendRawJSON, WeComGroupRobotSendRawJSONTemplate)
}
