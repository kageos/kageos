package wecom

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

type WeComConfig struct {
	ID                int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:配置ID;type:ID" hide:"create,update"`
	CreatedAt         types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt         types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	CreatedBy         string         `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
	Name              string         `json:"name" gorm:"column:name;comment:配置名称;index" widget:"name:配置名称;type:input" validate:"required"`
	CorpID            string         `json:"corp_id" gorm:"column:corp_id;comment:企业ID;index" widget:"name:企业ID;type:input" validate:"required"`
	AgentID           int            `json:"agent_id" gorm:"column:agent_id;comment:应用AgentId" widget:"name:应用AgentId;type:integer;min:1;step:1" validate:"required,min=1"`
	CorpSecretCipher  string         `json:"-" gorm:"column:corp_secret_cipher;type:text" widget:"-"`
	AccessTokenCipher string         `json:"-" gorm:"column:access_token_cipher;type:text" widget:"-"`
	TokenExpiresAt    types.Time     `json:"token_expires_at" gorm:"column:token_expires_at;type:datetime" widget:"name:Token过期时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	Enabled           bool           `json:"enabled" gorm:"column:enabled;comment:是否启用" widget:"name:启用;type:switch;render_default:true"`
	LastStatus        string         `json:"last_status" gorm:"column:last_status;comment:最近状态" widget:"name:最近状态;type:select;options:未测试,正常,失败;options_colors:909399,67C23A,F56C6C" hide:"create,update"`
	LastMessage       string         `json:"last_message" gorm:"column:last_message;type:text;comment:最近消息" widget:"name:最近消息;type:text_area" hide:"create,update"`
}

func (WeComConfig) TableName() string {
	return "wecom_config"
}

type WeComConfigListReq struct {
	Name              string `json:"name" form:"name" widget:"name:配置名称;type:input"`
	CorpID            string `json:"corp_id" form:"corp_id" widget:"name:企业ID;type:input"`
	LastStatus        string `json:"last_status" form:"last_status" widget:"name:状态;type:select;options:未测试,正常,失败;options_colors:909399,67C23A,F56C6C"`
	query.PageSortReq `widget:"-"`
}

type WeComConfigListItem struct {
	ID              int        `json:"id" widget:"name:配置ID;type:integer"`
	Name            string     `json:"name" widget:"name:配置名称;type:input"`
	CorpID          string     `json:"corp_id" widget:"name:企业ID;type:input"`
	AgentID         int        `json:"agent_id" widget:"name:应用AgentId;type:integer"`
	HasSecret       bool       `json:"has_secret" widget:"name:已配置Secret;type:switch"`
	Enabled         bool       `json:"enabled" widget:"name:启用;type:switch"`
	LastStatus      string     `json:"last_status" widget:"name:最近状态;type:select;options:未测试,正常,失败;options_colors:909399,67C23A,F56C6C"`
	LastMessage     string     `json:"last_message" widget:"name:最近消息;type:text_area"`
	TokenExpiresAt  types.Time `json:"token_expires_at" widget:"name:Token过期时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	ConfigUpdatedAt types.Time `json:"config_updated_at" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

func WeComConfigList(ctx *app.Context, resp response.Response) error {
	db, err := wecomDB(ctx)
	if err != nil {
		return err
	}
	var req WeComConfigListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	queryDB := db.Model(&WeComConfig{})
	if strings.TrimSpace(req.Name) != "" {
		queryDB = queryDB.Where("name LIKE ?", "%"+strings.TrimSpace(req.Name)+"%")
	}
	if strings.TrimSpace(req.CorpID) != "" {
		queryDB = queryDB.Where("corp_id LIKE ?", "%"+strings.TrimSpace(req.CorpID)+"%")
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
	var rows []WeComConfig
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&rows).Error; err != nil {
		return err
	}
	items := make([]WeComConfigListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, wecomConfigListItem(row))
	}
	return resp.Table(response.TableResult{
		Items:      items,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

func wecomConfigListItem(row WeComConfig) WeComConfigListItem {
	return WeComConfigListItem{
		ID:              row.ID,
		Name:            row.Name,
		CorpID:          row.CorpID,
		AgentID:         row.AgentID,
		HasSecret:       strings.TrimSpace(row.CorpSecretCipher) != "",
		Enabled:         row.Enabled,
		LastStatus:      firstNonEmpty(row.LastStatus, "未测试"),
		LastMessage:     row.LastMessage,
		TokenExpiresAt:  row.TokenExpiresAt,
		ConfigUpdatedAt: row.UpdatedAt,
	}
}

type WeComConfigSaveReq struct {
	ConfigID   int    `json:"config_id" widget:"name:更新已有配置;type:select;placeholder:留空则创建新配置" callback:"OnSelectFuzzy"`
	Name       string `json:"name" widget:"name:配置名称;type:input;placeholder:如 默认企业微信应用" validate:"required"`
	CorpID     string `json:"corp_id" widget:"name:企业ID;type:input;placeholder:企业微信后台“我的企业”里的企业ID" validate:"required"`
	AgentID    int    `json:"agent_id" widget:"name:应用AgentId;type:integer;min:1;step:1" validate:"required,min=1"`
	CorpSecret string `json:"corp_secret" widget:"name:应用Secret;type:input;placeholder:编辑时留空表示沿用原Secret" sensitive:"true"`
	Enabled    bool   `json:"enabled" widget:"name:启用;type:switch;render_default:true"`
}

type WeComConfigSaveResp struct {
	ConfigID       int        `json:"config_id" widget:"name:配置ID;type:integer"`
	Status         string     `json:"status" widget:"name:状态;type:select;options:正常,失败;options_colors:67C23A,F56C6C"`
	Name           string     `json:"name" widget:"name:配置名称;type:input"`
	CorpID         string     `json:"corp_id" widget:"name:企业ID;type:input"`
	AgentID        int        `json:"agent_id" widget:"name:应用AgentId;type:integer"`
	SecretPreview  string     `json:"secret_preview" widget:"name:Secret预览;type:input"`
	TokenExpiresAt types.Time `json:"token_expires_at" widget:"name:Token过期时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	Message        string     `json:"message" widget:"name:说明;type:text_area"`
}

func WeComConfigSave(ctx *app.Context, resp response.Response) error {
	db, err := wecomDB(ctx)
	if err != nil {
		return err
	}
	var req WeComConfigSaveReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	req.Name = strings.TrimSpace(req.Name)
	req.CorpID = strings.TrimSpace(req.CorpID)
	req.CorpSecret = strings.TrimSpace(req.CorpSecret)
	if req.Name == "" || req.CorpID == "" || req.AgentID <= 0 {
		return fmt.Errorf("配置名称、企业ID、应用AgentId不能为空")
	}

	var cfg WeComConfig
	if req.ConfigID > 0 {
		if err := db.First(&cfg, req.ConfigID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("未找到要更新的企业微信配置 ID=%d", req.ConfigID)
			}
			return err
		}
	} else {
		cfg.Enabled = true
		cfg.LastStatus = "未测试"
		cfg.CreatedBy = ctx.GetRequestUser()
	}
	if req.CorpSecret == "" && strings.TrimSpace(cfg.CorpSecretCipher) == "" {
		return fmt.Errorf("首次创建企业微信配置时必须填写应用 Secret")
	}
	if req.CorpSecret != "" {
		cipherText, err := encryptWeComSecret(req.CorpSecret)
		if err != nil {
			return err
		}
		cfg.CorpSecretCipher = cipherText
		cfg.AccessTokenCipher = ""
		cfg.TokenExpiresAt = types.Time{}
	}
	cfg.Name = req.Name
	cfg.CorpID = req.CorpID
	cfg.AgentID = req.AgentID
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

	status := "正常"
	message := "企业微信配置已保存，access_token 获取成功。"
	if _, err := ensureWeComAccessToken(ctx, &cfg, true); err != nil {
		status = "失败"
		message = "企业微信配置已保存，但测试连接失败：" + err.Error()
		logger.Errorf(ctx, "企业微信配置测试失败 config_id=%d err=%v", cfg.ID, err)
	}
	secretPreview := ""
	if req.CorpSecret != "" {
		secretPreview = maskWeComSecret(req.CorpSecret)
	} else {
		secretPreview = "沿用已保存的 Secret"
	}
	return resp.Form(&WeComConfigSaveResp{
		ConfigID:       cfg.ID,
		Status:         status,
		Name:           cfg.Name,
		CorpID:         cfg.CorpID,
		AgentID:        cfg.AgentID,
		SecretPreview:  secretPreview,
		TokenExpiresAt: cfg.TokenExpiresAt,
		Message:        message,
	}).Build()
}

func wecomConfigOnSelectFuzzy(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db, err := wecomDB(ctx)
	if err != nil {
		return nil, err
	}
	queryDB := db.Model(&WeComConfig{})
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
				queryDB = queryDB.Where("id = ? OR name LIKE ?", strings.TrimSpace(value), "%"+strings.TrimSpace(value)+"%")
			}
		}
	} else {
		keyword := ""
		if req != nil {
			keyword = strings.TrimSpace(req.Keyword())
		}
		if keyword != "" {
			queryDB = queryDB.Where("name LIKE ? OR corp_id LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
		}
	}
	var rows []WeComConfig
	if err := queryDB.Order("id ASC").Limit(20).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]*callback.SelectFuzzyItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &callback.SelectFuzzyItem{
			Value: row.ID,
			Label: fmt.Sprintf("%s (#%d)", row.Name, row.ID),
			DisplayInfo: map[string]interface{}{
				"企业ID":    row.CorpID,
				"AgentId": row.AgentID,
				"启用":      row.Enabled,
				"状态":      firstNonEmpty(row.LastStatus, "未测试"),
			},
		})
	}
	return &callback.OnSelectFuzzyResp{Items: items}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

var WeComConfigListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "企业微信配置列表",
		Desc:         "查看已保存的企业微信自建应用配置。Secret 和 access_token 只保存密文列，不在列表中展示。",
		Tags:         []string{"企业微信", "配置", "自建应用"},
		Request:      &WeComConfigListReq{},
		CreateTables: []interface{}{&WeComConfig{}},
	},
	AutoCrudTable: &WeComConfigListItem{},
}

var WeComConfigSaveTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "企业微信配置",
		Desc:         "保存企业微信自建应用的 corp_id、agent_id、corp_secret，并立即测试 access_token 获取是否成功。",
		Tags:         []string{"企业微信", "配置", "Secret"},
		Request:      &WeComConfigSaveReq{},
		Response:     &WeComConfigSaveResp{},
		CreateTables: []interface{}{&WeComConfig{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"config_id": wecomConfigOnSelectFuzzy,
		},
	},
}

func init() {
	packageContext.GET("configs.table", WeComConfigList, WeComConfigListTemplate)
	packageContext.POST("config.form", WeComConfigSave, WeComConfigSaveTemplate)
}
