package cert_manager

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
	"golang.org/x/crypto/acme"
	"gorm.io/gorm"
)

type CertTencentConfig struct {
	ID               int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:配置ID;type:ID" hide:"create,update"`
	CreatedAt        types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt        types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	CreatedBy        string         `json:"created_by" gorm:"column:created_by;type:varchar(120);index" widget:"name:创建人;type:user" hide:"create,update"`
	Name             string         `json:"name" gorm:"column:name;type:varchar(120);index;comment:配置名称" widget:"name:配置名称;type:input" validate:"required"`
	Email            string         `json:"email" gorm:"column:email;type:varchar(255);comment:ACME账号邮箱" widget:"name:ACME账号邮箱;type:input" validate:"required,email"`
	DirectoryURL     string         `json:"directory_url" gorm:"column:directory_url;type:varchar(255);comment:ACME目录地址" widget:"name:ACME环境;type:select;options:生产,测试;options_colors:67C23A,E6A23C;render_default:生产"`
	CredentialCipher string         `json:"-" gorm:"column:credential_cipher;type:text" widget:"-"`
	AccountKeyCipher string         `json:"-" gorm:"column:account_key_cipher;type:text" widget:"-"`
	Enabled          bool           `json:"enabled" gorm:"column:enabled;comment:是否启用" widget:"name:启用;type:switch;render_default:true"`
	LastStatus       string         `json:"last_status" gorm:"column:last_status;type:varchar(30);comment:最近状态" widget:"name:最近状态;type:select;options:未测试,正常,失败;options_colors:909399,67C23A,F56C6C" hide:"create,update"`
	LastMessage      string         `json:"last_message" gorm:"column:last_message;type:text;comment:最近消息" widget:"name:最近消息;type:text_area" hide:"create,update"`
}

func (CertTencentConfig) TableName() string {
	return "tencent_cert_config"
}

type CertTencentConfigListReq struct {
	Name              string `json:"name" form:"name" widget:"name:配置名称;type:input"`
	Email             string `json:"email" form:"email" widget:"name:邮箱;type:input"`
	LastStatus        string `json:"last_status" form:"last_status" widget:"name:状态;type:select;options:未测试,正常,失败;options_colors:909399,67C23A,F56C6C"`
	query.PageSortReq `widget:"-"`
}

type CertTencentConfigListItem struct {
	ID              int        `json:"id" widget:"name:配置ID;type:integer"`
	Name            string     `json:"name" widget:"name:配置名称;type:input"`
	Email           string     `json:"email" widget:"name:ACME账号邮箱;type:input"`
	Environment     string     `json:"environment" widget:"name:ACME环境;type:select;options:生产,测试;options_colors:67C23A,E6A23C"`
	HasCredential   bool       `json:"has_credential" widget:"name:已配置Secret;type:switch"`
	HasAccountKey   bool       `json:"has_account_key" widget:"name:已生成账号密钥;type:switch"`
	Enabled         bool       `json:"enabled" widget:"name:启用;type:switch"`
	LastStatus      string     `json:"last_status" widget:"name:最近状态;type:select;options:未测试,正常,失败;options_colors:909399,67C23A,F56C6C"`
	LastMessage     string     `json:"last_message" widget:"name:最近消息;type:text_area"`
	ConfigUpdatedAt types.Time `json:"config_updated_at" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

func CertTencentConfigList(ctx *app.Context, resp response.Response) error {
	db, err := certManagerDB(ctx)
	if err != nil {
		return err
	}
	var req CertTencentConfigListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	queryDB := db.Model(&CertTencentConfig{})
	if strings.TrimSpace(req.Name) != "" {
		queryDB = queryDB.Where("name LIKE ?", "%"+strings.TrimSpace(req.Name)+"%")
	}
	if strings.TrimSpace(req.Email) != "" {
		queryDB = queryDB.Where("email LIKE ?", "%"+strings.TrimSpace(req.Email)+"%")
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
	var rows []CertTencentConfig
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&rows).Error; err != nil {
		return err
	}
	items := make([]CertTencentConfigListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, certTencentConfigListItem(row))
	}
	return resp.Table(response.TableResult{Items: items, TotalCount: total, PageInfo: &req.PageSortReq}).Build()
}

func certTencentConfigListItem(row CertTencentConfig) CertTencentConfigListItem {
	return CertTencentConfigListItem{
		ID:              row.ID,
		Name:            row.Name,
		Email:           row.Email,
		Environment:     directoryLabel(row.DirectoryURL),
		HasCredential:   strings.TrimSpace(row.CredentialCipher) != "",
		HasAccountKey:   strings.TrimSpace(row.AccountKeyCipher) != "",
		Enabled:         row.Enabled,
		LastStatus:      firstNonEmpty(row.LastStatus, "未测试"),
		LastMessage:     row.LastMessage,
		ConfigUpdatedAt: row.UpdatedAt,
	}
}

type CertTencentConfigSaveReq struct {
	ConfigID    int    `json:"config_id" widget:"name:更新已有配置;type:select;placeholder:留空则创建新配置" callback:"OnSelectFuzzy"`
	Name        string `json:"name" widget:"name:配置名称;type:input;placeholder:如 默认腾讯云" validate:"required"`
	Email       string `json:"email" widget:"name:ACME账号邮箱;type:input;placeholder:admin@example.com" validate:"required,email"`
	Environment string `json:"environment" widget:"name:ACME环境;type:select;options:生产,测试;options_colors:67C23A,E6A23C;render_default:生产"`
	SecretID    string `json:"secret_id" widget:"name:SecretId;type:input;placeholder:编辑时留空表示沿用原 Secret" sensitive:"true"`
	SecretKey   string `json:"secret_key" widget:"name:SecretKey;type:input;placeholder:编辑时留空表示沿用原 Secret" sensitive:"true"`
	Enabled     bool   `json:"enabled" widget:"name:启用;type:switch;render_default:true"`
}

type CertTencentConfigSaveResp struct {
	ConfigID      int    `json:"config_id" widget:"name:配置ID;type:integer"`
	Status        string `json:"status" widget:"name:状态;type:select;options:正常,失败;options_colors:67C23A,F56C6C"`
	Name          string `json:"name" widget:"name:配置名称;type:input"`
	Email         string `json:"email" widget:"name:ACME账号邮箱;type:input"`
	Environment   string `json:"environment" widget:"name:ACME环境;type:input"`
	TokenPreview  string `json:"token_preview" widget:"name:Secret预览;type:input"`
	HasAccountKey bool   `json:"has_account_key" widget:"name:已生成账号密钥;type:switch"`
	Message       string `json:"message" widget:"name:说明;type:text_area"`
}

func CertTencentConfigSave(ctx *app.Context, resp response.Response) error {
	db, err := certManagerDB(ctx)
	if err != nil {
		return err
	}
	req := CertTencentConfigSaveReq{Enabled: true}
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.SecretID = strings.TrimSpace(req.SecretID)
	req.SecretKey = strings.TrimSpace(req.SecretKey)
	if req.Name == "" || req.Email == "" {
		return fmt.Errorf("配置名称和 ACME 账号邮箱不能为空")
	}
	hasNewCredential := req.SecretID != "" || req.SecretKey != ""
	if hasNewCredential && (req.SecretID == "" || req.SecretKey == "") {
		return fmt.Errorf("SecretId 和 SecretKey 必须同时填写")
	}
	logger.Infof(ctx, "[CertManager][Tencent] config save start config_id=%d name=%s env=%s enabled=%v credential_updated=%v user=%s",
		req.ConfigID, req.Name, req.Environment, req.Enabled, hasNewCredential, ctx.GetRequestUser())

	var cfg CertTencentConfig
	if req.ConfigID > 0 {
		if err := db.First(&cfg, req.ConfigID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("未找到要更新的腾讯云证书配置 ID=%d", req.ConfigID)
			}
			return err
		}
	} else {
		cfg.Enabled = true
		cfg.LastStatus = "未测试"
		cfg.CreatedBy = ctx.GetRequestUser()
	}
	if !hasNewCredential && strings.TrimSpace(cfg.CredentialCipher) == "" {
		return fmt.Errorf("首次创建腾讯云配置时必须填写 SecretId 和 SecretKey")
	}
	if hasNewCredential {
		cipherText, err := encryptSecret(encodeTencentCredentials(req.SecretID, req.SecretKey))
		if err != nil {
			return err
		}
		cfg.CredentialCipher = cipherText
	}
	if strings.TrimSpace(cfg.AccountKeyCipher) == "" {
		keyPEM, err := generateAccountKeyPEM()
		if err != nil {
			return err
		}
		cipherText, err := encryptSecret(keyPEM)
		if err != nil {
			return err
		}
		cfg.AccountKeyCipher = cipherText
	}
	cfg.Name = req.Name
	cfg.Email = req.Email
	cfg.DirectoryURL = directoryURL(req.Environment)
	cfg.Enabled = req.Enabled
	if cfg.LastStatus == "" {
		cfg.LastStatus = "未测试"
	}

	status := "正常"
	message := "腾讯云配置已保存，Secret 验证成功。"
	tokenPreview := "沿用已保存的 Secret"
	token, err := decryptSecret(cfg.CredentialCipher)
	if err != nil {
		return err
	}
	if hasNewCredential {
		tokenPreview = maskSecret(req.SecretID) + " / " + maskSecret(req.SecretKey)
	}
	if err := verifyTencentToken(token); err != nil {
		status = "失败"
		message = "腾讯云配置已保存，但 Secret 验证失败：" + err.Error()
		logger.Warnf(ctx, "Tencent secret verify failed config=%s err=%v", cfg.Name, err)
	}
	cfg.LastStatus = status
	cfg.LastMessage = message

	if cfg.ID == 0 {
		if err := db.Create(&cfg).Error; err != nil {
			return err
		}
	} else if err := db.Save(&cfg).Error; err != nil {
		return err
	}
	logger.Infof(ctx, "[CertManager][Tencent] config saved config_id=%d name=%s env=%s enabled=%v credential_updated=%v status=%s",
		cfg.ID, cfg.Name, directoryLabel(cfg.DirectoryURL), cfg.Enabled, hasNewCredential, status)

	return resp.Form(&CertTencentConfigSaveResp{
		ConfigID:      cfg.ID,
		Status:        status,
		Name:          cfg.Name,
		Email:         cfg.Email,
		Environment:   directoryLabel(cfg.DirectoryURL),
		TokenPreview:  tokenPreview,
		HasAccountKey: strings.TrimSpace(cfg.AccountKeyCipher) != "",
		Message:       message,
	}).Build()
}

type CertTencentConfigStatusReq struct {
	ConfigID int `json:"config_id" widget:"name:腾讯云配置;type:select;placeholder:留空使用第一个启用配置" callback:"OnSelectFuzzy"`
}

type CertTencentConfigStatusResp struct {
	Status      string `json:"status" widget:"name:配置状态;type:select;options:可用,不可用;options_colors:67C23A,F56C6C"`
	ConfigID    int    `json:"config_id" widget:"name:配置ID;type:integer"`
	ConfigName  string `json:"config_name" widget:"name:配置名称;type:input"`
	Environment string `json:"environment" widget:"name:ACME环境;type:input"`
	Summary     string `json:"summary" widget:"name:说明;type:text_area"`
}

func CertTencentConfigStatus(ctx *app.Context, resp response.Response) error {
	var req CertTencentConfigStatusReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	logger.Infof(ctx, "[CertManager][Tencent] config status check start config_id=%d", req.ConfigID)
	cfg, token, err := loadCertTencentConfig(ctx, req.ConfigID)
	if err != nil {
		return err
	}
	if err := verifyTencentToken(token); err != nil {
		logger.Warnf(ctx, "[CertManager][Tencent] secret verify failed config_id=%d config=%s err=%v", cfg.ID, cfg.Name, err)
		updateConfigStatus(ctx, cfg.ID, "失败", err.Error())
		return resp.Form(&CertTencentConfigStatusResp{
			Status:      "不可用",
			ConfigID:    cfg.ID,
			ConfigName:  cfg.Name,
			Environment: directoryLabel(cfg.DirectoryURL),
			Summary:     err.Error(),
		}).Build()
	}
	client, _, err := newACMEClient(cfg)
	if err != nil {
		logger.Warnf(ctx, "[CertManager][Tencent] ACME client init failed config_id=%d config=%s err=%v", cfg.ID, cfg.Name, err)
		updateConfigStatus(ctx, cfg.ID, "失败", err.Error())
		return err
	}
	if _, err := client.Discover(ctx); err != nil {
		logger.Warnf(ctx, "[CertManager][Tencent] ACME directory discover failed config_id=%d config=%s err=%v", cfg.ID, cfg.Name, err)
		updateConfigStatus(ctx, cfg.ID, "失败", err.Error())
		return err
	}
	updateConfigStatus(ctx, cfg.ID, "正常", "腾讯云 Secret 和 ACME Directory 均可访问")
	logger.Infof(ctx, "[CertManager][Tencent] config status check success config_id=%d config=%s env=%s",
		cfg.ID, cfg.Name, directoryLabel(cfg.DirectoryURL))
	return resp.Form(&CertTencentConfigStatusResp{
		Status:      "可用",
		ConfigID:    cfg.ID,
		ConfigName:  cfg.Name,
		Environment: directoryLabel(cfg.DirectoryURL),
		Summary:     "腾讯云 Secret 和 ACME Directory 均可访问，可以自动写入 DNS-01 TXT 并申请证书。",
	}).Build()
}

func loadCertTencentConfig(ctx *app.Context, configID int) (*CertTencentConfig, string, error) {
	db, err := certManagerDB(ctx)
	if err != nil {
		return nil, "", err
	}
	var cfg CertTencentConfig
	queryDB := db.Model(&CertTencentConfig{})
	if configID > 0 {
		queryDB = queryDB.Where("id = ?", configID)
	} else {
		queryDB = queryDB.Where("enabled = ?", true).Order("id ASC")
	}
	if err := queryDB.First(&cfg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if configID > 0 {
				return nil, "", fmt.Errorf("未找到腾讯云配置 ID=%d", configID)
			}
			return nil, "", fmt.Errorf("还没有可用的腾讯云配置，请先打开“腾讯云配置”填写 Secret")
		}
		return nil, "", err
	}
	if !cfg.Enabled {
		return nil, "", fmt.Errorf("腾讯云配置“%s”已停用", cfg.Name)
	}
	token, err := decryptSecret(cfg.CredentialCipher)
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(token) == "" {
		return nil, "", fmt.Errorf("腾讯云 Secret 未配置")
	}
	return &cfg, token, nil
}

func updateConfigStatus(ctx *app.Context, id int, status string, message string) {
	db, err := certManagerDB(ctx)
	if err != nil {
		return
	}
	_ = db.Model(&CertTencentConfig{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_status":  status,
		"last_message": message,
	}).Error
}

func certTencentConfigOnSelectFuzzy(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db, err := certManagerDB(ctx)
	if err != nil {
		return nil, err
	}
	queryDB := db.Model(&CertTencentConfig{})
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
			queryDB = queryDB.Where("name LIKE ? OR email LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
		}
	}
	var rows []CertTencentConfig
	if err := queryDB.Order("id ASC").Limit(20).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]*callback.SelectFuzzyItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &callback.SelectFuzzyItem{
			Value: row.ID,
			Label: fmt.Sprintf("%s (#%d)", row.Name, row.ID),
			DisplayInfo: map[string]interface{}{
				"邮箱":   row.Email,
				"环境":   directoryLabel(row.DirectoryURL),
				"启用":   row.Enabled,
				"最近状态": firstNonEmpty(row.LastStatus, "未测试"),
			},
		})
	}
	return &callback.OnSelectFuzzyResp{MaxSelections: 1, Items: items}, nil
}

func directoryURL(label string) string {
	label = strings.TrimSpace(label)
	switch label {
	case "测试", "staging":
		return "https://acme-staging-v02.api.letsencrypt.org/directory"
	default:
		return acme.LetsEncryptURL
	}
}

func directoryLabel(value string) string {
	if strings.Contains(value, "staging") {
		return "测试"
	}
	return "生产"
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

var CertTencentConfigListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "腾讯云配置列表",
		Desc:         "查看已保存的腾讯云 DNSPod 配置。Secret 和 ACME Account Key 只保存密文列，不在列表中展示。",
		Tags:         []string{"腾讯云", "证书管理", "配置"},
		Request:      &CertTencentConfigListReq{},
		Response:     query.PaginatedTable[[]CertTencentConfigListItem]{},
		CreateTables: certManagerTables(),
	},
}

var CertTencentConfigSaveTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "腾讯云配置",
		Desc:         "保存腾讯云 SecretId/SecretKey，用于自动创建和删除 DNSPod ACME DNS-01 TXT 记录。建议只授予 DNSPod 读写所需权限。",
		Tags:         []string{"腾讯云", "DNSPod", "证书管理"},
		Request:      &CertTencentConfigSaveReq{},
		Response:     &CertTencentConfigSaveResp{},
		CreateTables: certManagerTables(),
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"config_id": certTencentConfigOnSelectFuzzy,
		},
	},
}

func init() {
	packageContext.GET("configs.table", CertTencentConfigList, CertTencentConfigListTemplate)
	packageContext.POST("config.form", CertTencentConfigSave, CertTencentConfigSaveTemplate)
	packageContext.POST("config_status.form", CertTencentConfigStatus, &app.FormTemplate{
		BaseConfig: app.BaseConfig{
			Name:         "腾讯云配置状态",
			Desc:         "检查腾讯云 Secret 和 Let's Encrypt ACME Directory 是否可用。",
			Tags:         []string{"腾讯云", "配置检查"},
			Request:      &CertTencentConfigStatusReq{},
			Response:     &CertTencentConfigStatusResp{},
			CreateTables: certManagerTables(),
			OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
				"config_id": certTencentConfigOnSelectFuzzy,
			},
		},
	})
}
