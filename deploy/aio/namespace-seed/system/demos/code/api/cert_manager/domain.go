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

type CertManagedDomain struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:域名ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	Domain           string     `json:"domain" gorm:"column:domain;type:varchar(255);uniqueIndex;comment:域名" widget:"name:域名;type:input;placeholder:example.com 或 *.example.com" validate:"required"`
	DisplayName      string     `json:"display_name" gorm:"column:display_name;comment:显示名称" widget:"name:显示名称;type:input"`
	Environment      string     `json:"environment" gorm:"column:environment;comment:环境;default:生产" widget:"name:环境;type:select;options:生产,预发,测试,开发;options_colors:F56C6C,E6A23C,409EFF,909399;render_default:生产" validate:"required"`
	Owner            string     `json:"owner" gorm:"column:owner;comment:负责人" widget:"name:负责人;type:user;render_default:Me()"`
	NotifyUsers      string     `json:"notify_users" gorm:"column:notify_users;type:text;comment:通知人" widget:"name:通知人;type:users;desc:多个用户用逗号分隔，默认会包含负责人"`
	ValidationMethod string     `json:"validation_method" gorm:"column:validation_method;comment:验证方式;default:只监控" widget:"name:验证方式;type:select;options:只监控,HTTP-01,DNS-01,手动;options_colors:909399,409EFF,67C23A,E6A23C;render_default:只监控"`
	DNSProvider      string     `json:"dns_provider" gorm:"column:dns_provider;comment:DNS服务商" widget:"name:DNS服务商;type:input;placeholder:Cloudflare / 阿里云 / 腾讯云 / 手动"`
	DeployTarget     string     `json:"deploy_target" gorm:"column:deploy_target;comment:部署目标" widget:"name:部署目标;type:select;options:KageOS,Nginx,CDN,负载均衡,其他;options_colors:409EFF,67C23A,E6A23C,F56C6C,909399;render_default:KageOS"`
	AutoRenew        bool       `json:"auto_renew" gorm:"column:auto_renew;default:false;comment:是否自动创建续期任务" widget:"name:自动续期任务;type:switch;desc:只自动创建/跟踪续期任务，不自动部署证书"`
	Enabled          bool       `json:"enabled" gorm:"column:enabled;default:true;comment:是否启用巡检" widget:"name:启用巡检;type:switch;render_default:true"`
	WarnDays         int        `json:"warn_days" gorm:"column:warn_days;default:30;comment:提前提醒天数" widget:"name:提前提醒天数;type:integer;min:1;max:365;render_default:30"`
	CurrentStatus    string     `json:"current_status" gorm:"column:current_status;default:未检查;comment:当前状态" widget:"name:当前状态;type:select;options:未检查,正常,即将过期,已过期,检查失败;options_colors:909399,67C23A,E6A23C,F56C6C,F56C6C" hide:"create,update"`
	CurrentIssuer    string     `json:"current_issuer" gorm:"column:current_issuer;type:text;comment:当前签发者" widget:"name:当前签发者;type:text" hide:"create,update"`
	CurrentNotAfter  types.Time `json:"current_not_after" gorm:"column:current_not_after;type:datetime;comment:当前证书过期时间" widget:"name:当前过期时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	CurrentDaysLeft  int        `json:"current_days_left" gorm:"column:current_days_left;comment:剩余天数" widget:"name:剩余天数;type:integer" hide:"create,update"`
	LastCheckedAt    types.Time `json:"last_checked_at" gorm:"column:last_checked_at;type:datetime;comment:最近巡检时间" widget:"name:最近巡检时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	LastNotifiedAt   types.Time `json:"last_notified_at" gorm:"column:last_notified_at;type:datetime;comment:最近提醒时间" widget:"name:最近提醒时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	Remark           string     `json:"remark" gorm:"column:remark;type:text;comment:备注" widget:"name:备注;type:text_area"`
}

func (CertManagedDomain) TableName() string {
	return "ops_cert_managed_domain"
}

type CertManagedDomainListReq struct {
	Domain      string `json:"domain" form:"domain" widget:"name:域名;type:input"`
	Environment string `json:"environment" form:"environment" widget:"name:环境;type:select;options:生产,预发,测试,开发;options_colors:F56C6C,E6A23C,409EFF,909399"`
	Owner       string `json:"owner" form:"owner" widget:"name:负责人;type:user"`
	Status      string `json:"status" form:"status" widget:"name:当前状态;type:select;options:未检查,正常,即将过期,已过期,检查失败;options_colors:909399,67C23A,E6A23C,F56C6C,F56C6C"`
	Enabled     string `json:"enabled" form:"enabled" widget:"name:启用巡检;type:select;options:全部,启用,停用;options_colors:909399,67C23A,F56C6C;render_default:全部"`

	query.PageSortReq `widget:"-"`
}

func CertManagedDomainList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req CertManagedDomainListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&CertManagedDomain{})
	if req.Domain != "" {
		queryDB = queryDB.Where("domain LIKE ?", "%"+strings.TrimSpace(req.Domain)+"%")
	}
	if req.Environment != "" {
		queryDB = queryDB.Where("environment = ?", req.Environment)
	}
	if req.Owner != "" {
		queryDB = queryDB.Where("owner = ?", req.Owner)
	}
	if req.Status != "" {
		queryDB = queryDB.Where("current_status = ?", req.Status)
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
	var domains []CertManagedDomain
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&domains).Error; err != nil {
		return err
	}

	return resp.Table(response.TableResult{
		Items:      domains,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

var CertManagedDomainListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "证书域名管理",
		Desc:         "维护需要巡检和管理证书的域名资产，包括负责人、提醒策略、验证方式和当前公网证书状态。本系统只管理证书资产，不自动部署证书。",
		Tags:         []string{"证书管理", "域名资产", "到期提醒"},
		Request:      &CertManagedDomainListReq{},
		Response:     query.PaginatedTable[[]CertManagedDomain]{},
		CreateTables: []interface{}{&CertManagedDomain{}},
	},
	AutoCrudTable: &CertManagedDomain{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row CertManagedDomain
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		if err := normalizeDomainRow(ctx, &row); err != nil {
			return nil, err
		}
		if err := db.Create(&row).Error; err != nil {
			logger.Errorf(ctx, "Create cert managed domain err: %v", err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()
		var updateFields CertManagedDomain
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
		if req.IsFieldUpdated("warn_days") && updateFields.WarnDays <= 0 {
			updates["warn_days"] = defaultCertWarnDays
		}
		if err := db.Model(&CertManagedDomain{}).Where("id = ?", req.GetId()).Updates(updates).Error; err != nil {
			logger.Errorf(ctx, "Update cert managed domain err: %v", err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		var assetCount int64
		if err := db.Model(&CertAsset{}).Where("domain_id in ?", req.GetIds()).Count(&assetCount).Error; err != nil {
			return nil, fmt.Errorf("检查证书资产失败: %w", err)
		}
		if assetCount > 0 {
			return nil, fmt.Errorf("域名下存在证书资产，请先确认是否需要保留资产记录")
		}
		if err := db.Delete(&CertManagedDomain{}, "id in ?", req.GetIds()).Error; err != nil {
			logger.Errorf(ctx, "Delete cert managed domain err: %v", err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

func normalizeDomainRow(ctx *app.Context, row *CertManagedDomain) error {
	domain, err := normalizeDomainName(row.Domain)
	if err != nil {
		return err
	}
	row.Domain = domain
	if strings.TrimSpace(row.Environment) == "" {
		row.Environment = "生产"
	}
	if strings.TrimSpace(row.Owner) == "" {
		row.Owner = ctx.GetRequestUser()
	}
	if strings.TrimSpace(row.ValidationMethod) == "" {
		row.ValidationMethod = "只监控"
	}
	if strings.TrimSpace(row.DeployTarget) == "" {
		row.DeployTarget = "KageOS"
	}
	if row.WarnDays <= 0 {
		row.WarnDays = defaultCertWarnDays
	}
	if strings.TrimSpace(row.CurrentStatus) == "" {
		row.CurrentStatus = statusUnchecked
	}
	row.Enabled = true
	return nil
}

func onSelectFuzzyManagedDomain(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接失败")
	}
	queryDB := db.Model(&CertManagedDomain{})
	if req.IsByValue() {
		queryDB = queryDB.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		queryDB = queryDB.Where("id in ?", req.GetValues())
	} else {
		keyword := strings.TrimSpace(req.Keyword())
		queryDB = queryDB.Where("domain LIKE ? OR display_name LIKE ? OR owner LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").Limit(20)
	}

	var domains []CertManagedDomain
	if err := queryDB.Order("domain ASC").Find(&domains).Error; err != nil {
		return nil, err
	}
	items := make([]*callback.SelectFuzzyItem, 0, len(domains))
	for _, d := range domains {
		label := d.Domain
		if d.DisplayName != "" {
			label = fmt.Sprintf("%s（%s）", d.Domain, d.DisplayName)
		}
		items = append(items, &callback.SelectFuzzyItem{
			Value: d.ID,
			Label: label,
			DisplayInfo: map[string]interface{}{
				"域名":     d.Domain,
				"负责人":    d.Owner,
				"当前状态":   d.CurrentStatus,
				"剩余天数":   d.CurrentDaysLeft,
				"提前提醒天数": d.WarnDays,
			},
		})
	}
	return &callback.OnSelectFuzzyResp{
		MaxSelections: 1,
		Items:         items,
		Statistics: map[string]interface{}{
			"域名":     statistics.Value("域名"),
			"负责人":    statistics.Value("负责人"),
			"当前状态":   statistics.Value("当前状态"),
			"剩余天数":   statistics.Value("剩余天数"),
			"提前提醒天数": statistics.Value("提前提醒天数"),
		},
	}, nil
}

func init() {
	packageContext.GET("cert_domain_list.table", CertManagedDomainList, CertManagedDomainListTemplate)
}
