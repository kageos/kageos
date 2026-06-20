package cert_manager

import (
	"fmt"
	"strings"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/statistics"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

type CertAliyunDomain struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:域名ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	Domain          string            `json:"domain" gorm:"column:domain;type:varchar(255);uniqueIndex;comment:主域名" widget:"name:主域名;type:input;placeholder:example.com 或 *.example.com" validate:"required"`
	SANs            string            `json:"sans" gorm:"column:sans;type:text;comment:附加SAN域名" widget:"name:附加SAN域名;type:text_area;placeholder:一行一个或逗号分隔，可选"`
	DisplayName     string            `json:"display_name" gorm:"column:display_name;type:varchar(120);comment:显示名称" widget:"name:显示名称;type:input"`
	ConfigID        int               `json:"config_id" gorm:"column:config_id;index;comment:阿里云配置ID" widget:"name:阿里云配置;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Config          *CertAliyunConfig `json:"-" widget:"-" gorm:"foreignKey:ConfigID;references:ID"`
	ConfigName      string            `json:"config_name" gorm:"column:config_name;type:varchar(120);comment:阿里云配置名称" widget:"name:阿里云配置;type:text" hide:"create,update"`
	Owner           string            `json:"owner" gorm:"column:owner;type:varchar(120);comment:负责人" widget:"name:负责人;type:user;render_default:Me()"`
	NotifyUsers     string            `json:"notify_users" gorm:"column:notify_users;type:text;comment:通知人" widget:"name:通知人;type:users;desc:多个用户用逗号分隔，默认会包含负责人"`
	AutoRenew       bool              `json:"auto_renew" gorm:"column:auto_renew;default:true;comment:自动续期" widget:"name:自动续期;type:switch;render_default:true"`
	Enabled         bool              `json:"enabled" gorm:"column:enabled;default:true;comment:启用" widget:"name:启用;type:switch;render_default:true"`
	RenewBeforeDays int               `json:"renew_before_days" gorm:"column:renew_before_days;default:30;comment:提前续期天数" widget:"name:提前续期天数;type:integer;min:1;max:60;render_default:30"`
	KeyAlgorithm    string            `json:"key_algorithm" gorm:"column:key_algorithm;type:varchar(30);comment:证书私钥算法" widget:"name:证书私钥算法;type:select;options:ECDSA P-256;render_default:ECDSA P-256"`
	CurrentStatus   string            `json:"current_status" gorm:"column:current_status;type:varchar(30);default:未检查;comment:当前状态" widget:"name:当前状态;type:select;options:未检查,正常,即将过期,已过期,失败;options_colors:909399,67C23A,E6A23C,F56C6C,F56C6C" hide:"create,update"`
	CurrentIssuer   string            `json:"current_issuer" gorm:"column:current_issuer;type:text;comment:当前签发者" widget:"name:当前签发者;type:text" hide:"create,update"`
	CurrentNotAfter types.Time        `json:"current_not_after" gorm:"column:current_not_after;type:datetime;comment:当前过期时间" widget:"name:当前过期时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	CurrentDaysLeft int               `json:"current_days_left" gorm:"column:current_days_left;comment:剩余天数" widget:"name:剩余天数;type:integer" hide:"create,update"`
	LastCheckedAt   types.Time        `json:"last_checked_at" gorm:"column:last_checked_at;type:datetime;comment:最近检查时间" widget:"name:最近检查时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	LastRenewedAt   types.Time        `json:"last_renewed_at" gorm:"column:last_renewed_at;type:datetime;comment:最近续期时间" widget:"name:最近续期时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	LastNotifiedAt  types.Time        `json:"last_notified_at" gorm:"column:last_notified_at;type:datetime;comment:最近提醒时间" widget:"name:最近提醒时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	LastError       string            `json:"last_error" gorm:"column:last_error;type:text;comment:最近错误" widget:"name:最近错误;type:text_area" hide:"create,update"`
	Remark          string            `json:"remark" gorm:"column:remark;type:text;comment:备注" widget:"name:备注;type:text_area"`
}

func (CertAliyunDomain) TableName() string {
	return "aliyun_cert_domain"
}

type CertAliyunDomainListReq struct {
	Domain            string `json:"domain" form:"domain" widget:"name:域名;type:input"`
	ConfigName        string `json:"config_name" form:"config_name" widget:"name:阿里云配置;type:input"`
	Owner             string `json:"owner" form:"owner" widget:"name:负责人;type:user"`
	Status            string `json:"status" form:"status" widget:"name:当前状态;type:select;options:未检查,正常,即将过期,已过期,失败;options_colors:909399,67C23A,E6A23C,F56C6C,F56C6C"`
	Enabled           string `json:"enabled" form:"enabled" widget:"name:启用;type:select;options:全部,启用,停用;options_colors:909399,67C23A,F56C6C;render_default:全部"`
	query.PageSortReq `widget:"-"`
}

func CertAliyunDomainList(ctx *app.Context, resp response.Response) error {
	db, err := certManagerDB(ctx)
	if err != nil {
		return err
	}
	var req CertAliyunDomainListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	queryDB := db.Model(&CertAliyunDomain{})
	if strings.TrimSpace(req.Domain) != "" {
		queryDB = queryDB.Where("domain LIKE ? OR sans LIKE ?", "%"+strings.TrimSpace(req.Domain)+"%", "%"+strings.TrimSpace(req.Domain)+"%")
	}
	if strings.TrimSpace(req.ConfigName) != "" {
		queryDB = queryDB.Where("config_name LIKE ?", "%"+strings.TrimSpace(req.ConfigName)+"%")
	}
	if strings.TrimSpace(req.Owner) != "" {
		queryDB = queryDB.Where("owner = ?", strings.TrimSpace(req.Owner))
	}
	if strings.TrimSpace(req.Status) != "" {
		queryDB = queryDB.Where("current_status = ?", strings.TrimSpace(req.Status))
	}
	switch req.Enabled {
	case "启用":
		queryDB = queryDB.Where("enabled = ?", true)
	case "停用":
		queryDB = queryDB.Where("enabled = ?", false)
	}
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	} else {
		queryDB = queryDB.Order("current_days_left ASC, updated_at DESC")
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}
	var rows []CertAliyunDomain
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&rows).Error; err != nil {
		return err
	}
	return resp.Table(response.TableResult{Items: rows, TotalCount: total, PageInfo: &req.PageSortReq}).Build()
}

var CertAliyunDomainListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "证书域名管理",
		Desc:         "维护由阿里云 DNS 自动验证和续期的证书域名。系统会自动写入 DNS-01 TXT、签发证书并归档文件，但不会自动部署。",
		Tags:         []string{"阿里云", "证书管理", "域名"},
		Request:      &CertAliyunDomainListReq{},
		Response:     query.PaginatedTable[[]CertAliyunDomain]{},
		CreateTables: certManagerTables(),
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"config_id": certAliyunConfigOnSelectFuzzy,
		},
	},
	AutoCrudTable: &CertAliyunDomain{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db, err := certManagerDB(ctx)
		if err != nil {
			return nil, err
		}
		row := CertAliyunDomain{
			AutoRenew:       true,
			Enabled:         true,
			RenewBeforeDays: defaultRenewBeforeDays,
			KeyAlgorithm:    defaultCertKeyAlgorithm,
		}
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		if err := normalizeDomainRow(ctx, db, &row); err != nil {
			return nil, err
		}
		if err := db.Create(&row).Error; err != nil {
			logger.Errorf(ctx, "Create cert aliyun domain err: %v", err)
			return nil, err
		}
		logger.Infof(ctx, "[CertManager][Aliyun] domain created domain_id=%d domain=%s config_id=%d owner=%s auto_renew=%v enabled=%v",
			row.ID, row.Domain, row.ConfigID, row.Owner, row.AutoRenew, row.Enabled)
		return &callback.OnTableAddRowResp{Data: row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db, err := certManagerDB(ctx)
		if err != nil {
			return nil, err
		}
		var updateFields CertAliyunDomain
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, fmt.Errorf("绑定更新字段失败: %w", err)
		}
		updates := req.ChangedFields()
		if req.IsFieldUpdated("domain") {
			domain, err := normalizeDomainName(updateFields.Domain)
			if err != nil {
				return nil, err
			}
			updates["domain"] = domain
		}
		if req.IsFieldUpdated("sans") {
			domain := updateFields.Domain
			if strings.TrimSpace(domain) == "" {
				var current CertAliyunDomain
				if err := db.First(&current, req.GetId()).Error; err != nil {
					return nil, fmt.Errorf("证书域名不存在")
				}
				domain = current.Domain
			}
			if _, err := normalizeDomainNames(domain, updateFields.SANs); err != nil {
				return nil, err
			}
		}
		if req.IsFieldUpdated("config_id") {
			var cfg CertAliyunConfig
			if err := db.First(&cfg, updateFields.ConfigID).Error; err != nil {
				return nil, fmt.Errorf("阿里云配置不存在")
			}
			updates["config_name"] = cfg.Name
		}
		if req.IsFieldUpdated("renew_before_days") && updateFields.RenewBeforeDays <= 0 {
			updates["renew_before_days"] = defaultRenewBeforeDays
		}
		if req.IsFieldUpdated("key_algorithm") && strings.TrimSpace(updateFields.KeyAlgorithm) == "" {
			updates["key_algorithm"] = defaultCertKeyAlgorithm
		}
		if err := db.Model(&CertAliyunDomain{}).Where("id = ?", req.GetId()).Updates(updates).Error; err != nil {
			logger.Errorf(ctx, "Update cert aliyun domain err: %v", err)
			return nil, err
		}
		logger.Infof(ctx, "[CertManager][Aliyun] domain updated domain_id=%d changed_fields=%d", req.GetId(), len(updates))
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db, err := certManagerDB(ctx)
		if err != nil {
			return nil, err
		}
		var assetCount int64
		if err := db.Model(&CertAliyunAsset{}).Where("domain_id in ?", req.GetIds()).Count(&assetCount).Error; err != nil {
			return nil, fmt.Errorf("检查证书资产失败: %w", err)
		}
		if assetCount > 0 {
			return nil, fmt.Errorf("域名下存在证书资产，请先确认是否需要保留资产记录")
		}
		logger.Infof(ctx, "[CertManager][Aliyun] domain delete requested ids=%v", req.GetIds())
		if err := db.Delete(&CertAliyunDomain{}, "id in ?", req.GetIds()).Error; err != nil {
			return nil, err
		}
		logger.Infof(ctx, "[CertManager][Aliyun] domain deleted ids=%v", req.GetIds())
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

func normalizeDomainRow(ctx *app.Context, db *gorm.DB, row *CertAliyunDomain) error {
	domain, err := normalizeDomainName(row.Domain)
	if err != nil {
		return err
	}
	if _, err := normalizeDomainNames(domain, row.SANs); err != nil {
		return err
	}
	row.Domain = domain
	if row.ConfigID <= 0 {
		return fmt.Errorf("请选择阿里云配置")
	}
	var cfg CertAliyunConfig
	if err := db.First(&cfg, row.ConfigID).Error; err != nil {
		return fmt.Errorf("阿里云配置不存在")
	}
	row.ConfigName = cfg.Name
	if strings.TrimSpace(row.Owner) == "" {
		row.Owner = ctx.GetRequestUser()
	}
	if row.RenewBeforeDays <= 0 {
		row.RenewBeforeDays = defaultRenewBeforeDays
	}
	if strings.TrimSpace(row.KeyAlgorithm) == "" {
		row.KeyAlgorithm = defaultCertKeyAlgorithm
	}
	if strings.TrimSpace(row.CurrentStatus) == "" {
		row.CurrentStatus = statusUnchecked
	}
	return nil
}

func certAliyunDomainOnSelectFuzzy(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db, err := certManagerDB(ctx)
	if err != nil {
		return nil, err
	}
	queryDB := db.Model(&CertAliyunDomain{})
	if req != nil && req.IsByValue() {
		queryDB = queryDB.Where("id = ?", req.GetValue()).Limit(1)
	} else if req != nil && req.IsByValues() {
		queryDB = queryDB.Where("id in ?", req.GetValues())
	} else {
		keyword := ""
		if req != nil {
			keyword = strings.TrimSpace(req.Keyword())
		}
		if keyword != "" {
			queryDB = queryDB.Where("domain LIKE ? OR display_name LIKE ? OR owner LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
		}
		queryDB = queryDB.Limit(20)
	}
	var rows []CertAliyunDomain
	if err := queryDB.Order("domain ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]*callback.SelectFuzzyItem, 0, len(rows))
	for _, row := range rows {
		label := row.Domain
		if row.DisplayName != "" {
			label = fmt.Sprintf("%s（%s）", row.Domain, row.DisplayName)
		}
		items = append(items, &callback.SelectFuzzyItem{
			Value: row.ID,
			Label: label,
			DisplayInfo: map[string]interface{}{
				"域名":    row.Domain,
				"阿里云配置": row.ConfigName,
				"负责人":   row.Owner,
				"当前状态":  row.CurrentStatus,
				"剩余天数":  row.CurrentDaysLeft,
			},
		})
	}
	return &callback.OnSelectFuzzyResp{
		MaxSelections: 1,
		Items:         items,
		Statistics: map[string]interface{}{
			"域名":    statistics.Value("域名"),
			"阿里云配置": statistics.Value("阿里云配置"),
			"负责人":   statistics.Value("负责人"),
			"当前状态":  statistics.Value("当前状态"),
			"剩余天数":  statistics.Value("剩余天数"),
		},
	}, nil
}

func init() {
	packageContext.GET("domains.table", CertAliyunDomainList, CertAliyunDomainListTemplate)
}
