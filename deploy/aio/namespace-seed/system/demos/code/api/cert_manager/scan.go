package cert_manager

import (
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

type CertScanRecord struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:巡检ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	DomainID       int        `json:"domain_id" gorm:"column:domain_id;index;comment:域名ID" widget:"name:域名ID;type:integer" hide:"create,update"`
	DomainName     string     `json:"domain_name" gorm:"column:domain_name;type:varchar(255);index;comment:域名" widget:"name:域名;type:text" hide:"create,update"`
	CheckType      string     `json:"check_type" gorm:"column:check_type;comment:巡检类型" widget:"name:巡检类型;type:select;options:公网TLS,文件解析;options_colors:409EFF,67C23A" hide:"create,update"`
	Status         string     `json:"status" gorm:"column:status;comment:状态" widget:"name:状态;type:select;options:正常,即将过期,已过期,检查失败;options_colors:67C23A,E6A23C,F56C6C,F56C6C" hide:"create,update"`
	CheckedAt      types.Time `json:"checked_at" gorm:"column:checked_at;type:datetime;comment:巡检时间" widget:"name:巡检时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	NotAfter       types.Time `json:"not_after" gorm:"column:not_after;type:datetime;comment:过期时间" widget:"name:过期时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DaysLeft       int        `json:"days_left" gorm:"column:days_left;comment:剩余天数" widget:"name:剩余天数;type:integer" hide:"create,update"`
	Issuer         string     `json:"issuer" gorm:"column:issuer;type:text;comment:签发者" widget:"name:签发者;type:text_area" hide:"create,update"`
	FingerprintSHA string     `json:"fingerprint_sha256" gorm:"column:fingerprint_sha256;type:text;comment:SHA256指纹" widget:"name:SHA256指纹;type:text" hide:"create,update"`
	ErrorMessage   string     `json:"error_message" gorm:"column:error_message;type:text;comment:错误信息" widget:"name:错误信息;type:text_area" hide:"create,update"`
	Notified       bool       `json:"notified" gorm:"column:notified;default:false;comment:是否已提醒" widget:"name:是否已提醒;type:switch" hide:"create,update"`
	NotifiedUsers  string     `json:"notified_users" gorm:"column:notified_users;type:text;comment:提醒用户" widget:"name:提醒用户;type:users" hide:"create,update"`
}

func (CertScanRecord) TableName() string {
	return "ops_cert_scan_record"
}

type CertScanRecordListReq struct {
	DomainName string `json:"domain_name" form:"domain_name" widget:"name:域名;type:input"`
	Status     string `json:"status" form:"status" widget:"name:状态;type:select;options:正常,即将过期,已过期,检查失败;options_colors:67C23A,E6A23C,F56C6C,F56C6C"`
	CheckType  string `json:"check_type" form:"check_type" widget:"name:巡检类型;type:select;options:公网TLS,文件解析;options_colors:409EFF,67C23A"`

	query.PageSortReq `widget:"-"`
}

func CertScanRecordList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}
	var req CertScanRecordListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	queryDB := db.Model(&CertScanRecord{})
	if req.DomainName != "" {
		queryDB = queryDB.Where("domain_name LIKE ?", "%"+strings.TrimSpace(req.DomainName)+"%")
	}
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}
	if req.CheckType != "" {
		queryDB = queryDB.Where("check_type = ?", req.CheckType)
	}
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	} else {
		queryDB = queryDB.Order("checked_at DESC")
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}
	var records []CertScanRecord
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&records).Error; err != nil {
		return err
	}
	return resp.Table(response.TableResult{Items: records, TotalCount: total, PageInfo: &req.PageSortReq}).Build()
}

var CertScanRecordListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "证书巡检记录",
		Desc:         "记录公网 TLS 巡检、文件解析和到期提醒结果，用于审计证书风险和通知历史。",
		Tags:         []string{"证书管理", "巡检记录", "到期提醒"},
		Request:      &CertScanRecordListReq{},
		Response:     query.PaginatedTable[[]CertScanRecord]{},
		CreateTables: []interface{}{&CertScanRecord{}},
	},
	AutoCrudTable: &CertScanRecord{},
}

type CertPublicScanReq struct {
	DomainID int  `json:"domain_id" widget:"name:域名;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Notify   bool `json:"notify" widget:"name:发现风险时提醒;type:switch;render_default:true"`
}

type CertPublicScanResp struct {
	DomainName     string     `json:"domain_name" widget:"name:域名;type:text"`
	Status         string     `json:"status" widget:"name:状态;type:text"`
	NotAfter       types.Time `json:"not_after" widget:"name:过期时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	DaysLeft       int        `json:"days_left" widget:"name:剩余天数;type:integer"`
	Issuer         string     `json:"issuer" widget:"name:签发者;type:text_area"`
	FingerprintSHA string     `json:"fingerprint_sha256" widget:"name:SHA256指纹;type:text"`
	Notified       bool       `json:"notified" widget:"name:是否已提醒;type:switch"`
	ScanLink       string     `json:"scan_link" widget:"name:查看巡检记录;type:link;link_type:primary"`
}

func CertPublicScan(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}
	var req CertPublicScanReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	var domain CertManagedDomain
	if err := db.Where("id = ?", req.DomainID).First(&domain).Error; err != nil {
		return fmt.Errorf("域名资产不存在")
	}
	record, notified, err := scanOneDomain(ctx, db, &domain, req.Notify, time.Now())
	if err != nil {
		return err
	}
	link, _ := ctx.BuildFunctionUrlWithText("cert_scan_record_list.table", CertScanRecord{ID: record.ID}, "查看巡检记录")
	return resp.Form(&CertPublicScanResp{
		DomainName:     record.DomainName,
		Status:         record.Status,
		NotAfter:       record.NotAfter,
		DaysLeft:       record.DaysLeft,
		Issuer:         record.Issuer,
		FingerprintSHA: record.FingerprintSHA,
		Notified:       notified,
		ScanLink:       link,
	}).Build()
}

type CertExpirySweepReq struct {
	WarningDays       int  `json:"warning_days" widget:"name:提醒阈值天数;type:integer;min:1;max:365;render_default:30"`
	Notify            bool `json:"notify" widget:"name:发送提醒;type:switch;render_default:true"`
	AutoCreateRenewal bool `json:"auto_create_renewal" widget:"name:自动创建续期任务;type:switch;render_default:true"`
}

type CertExpirySweepResp struct {
	CheckedCount       int `json:"checked_count" widget:"name:巡检域名数;type:integer"`
	WarningCount       int `json:"warning_count" widget:"name:即将过期数;type:integer"`
	ExpiredCount       int `json:"expired_count" widget:"name:已过期数;type:integer"`
	FailedCount        int `json:"failed_count" widget:"name:检查失败数;type:integer"`
	NotifiedCount      int `json:"notified_count" widget:"name:提醒次数;type:integer"`
	RenewalTaskCreated int `json:"renewal_task_created" widget:"name:创建续期任务数;type:integer"`
}

func CertExpirySweep(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}
	req := CertExpirySweepReq{WarningDays: defaultCertWarnDays, Notify: true, AutoCreateRenewal: true}
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	if req.WarningDays <= 0 {
		req.WarningDays = defaultCertWarnDays
	}

	var domains []CertManagedDomain
	if err := db.Where("enabled = ?", true).Order("domain ASC").Find(&domains).Error; err != nil {
		return err
	}
	now := time.Now()
	result := CertExpirySweepResp{CheckedCount: len(domains)}
	for _, domain := range domains {
		if domain.WarnDays <= 0 {
			domain.WarnDays = req.WarningDays
		}
		record, notified, err := scanOneDomain(ctx, db, &domain, req.Notify, now)
		if err != nil {
			logger.Warnf(ctx, "[CertExpirySweep] scan domain failed domain=%s err=%v", domain.Domain, err)
			result.FailedCount++
			continue
		}
		switch record.Status {
		case statusWarning:
			result.WarningCount++
		case statusExpired:
			result.ExpiredCount++
		case statusFailed:
			result.FailedCount++
		}
		if notified {
			result.NotifiedCount++
		}
		if req.AutoCreateRenewal && domain.AutoRenew && (record.Status == statusWarning || record.Status == statusExpired) {
			created, err := ensureRenewalTask(ctx, db, &domain, "自动续期", "到期巡检自动创建")
			if err != nil {
				logger.Warnf(ctx, "[CertExpirySweep] create renewal task failed domain=%s err=%v", domain.Domain, err)
				continue
			}
			if created {
				result.RenewalTaskCreated++
			}
		}
	}
	return resp.Form(&result).Build()
}

func scanOneDomain(ctx *app.Context, db *gorm.DB, domain *CertManagedDomain, notify bool, now time.Time) (*CertScanRecord, bool, error) {
	checkedAt := now
	record := CertScanRecord{
		DomainID:   domain.ID,
		DomainName: domain.Domain,
		CheckType:  scanTypePublicTLS,
		CheckedAt:  types.Time(checkedAt),
	}
	cert, err := fetchPublicTLSCertificate(domain.Domain)
	if err != nil {
		record.Status = statusFailed
		record.ErrorMessage = err.Error()
		if createErr := db.Create(&record).Error; createErr != nil {
			return nil, false, createErr
		}
		updateDomainCheckFailure(db, domain, checkedAt)
		return &record, false, nil
	}
	meta := certMetadataFromX509(cert, domain.Domain, domain.WarnDays)
	record.Status = meta.Status
	record.NotAfter = types.Time(meta.NotAfter)
	record.DaysLeft = meta.DaysLeft
	record.Issuer = meta.Issuer
	record.FingerprintSHA = meta.FingerprintSHA
	if !meta.HostnameMatched {
		record.Status = statusFailed
		record.ErrorMessage = "证书 SAN/CN 与域名不匹配"
	}
	notified := false
	if notify && (record.Status == statusWarning || record.Status == statusExpired || record.Status == statusFailed) && shouldNotifyDomain(*domain, now) {
		toUsers := joinCertificateNotifyUsers(*domain)
		if toUsers != "" {
			content := fmt.Sprintf("域名 `%s` 证书状态：%s\n\n剩余天数：%d\n过期时间：%s\n签发者：%s\n\n本系统只做证书管理和提醒，不会自动部署证书。",
				domain.Domain, record.Status, record.DaysLeft, record.NotAfter.String(), record.Issuer)
			if record.ErrorMessage != "" {
				content += "\n\n错误信息：" + record.ErrorMessage
			}
			if err := ctx.SendMessage(&app.SendMessageOpts{
				ToUsers: toUsers,
				Title:   "证书到期/风险提醒",
				Content: content,
			}); err != nil {
				logger.Warnf(ctx, "[CertScan] send message failed domain=%s err=%v", domain.Domain, err)
			} else {
				record.Notified = true
				record.NotifiedUsers = toUsers
				updateDomainNotifiedAt(db, domain.ID, now)
				notified = true
			}
		}
	}
	if err := db.Create(&record).Error; err != nil {
		return nil, notified, err
	}
	if record.Status == statusFailed {
		updateDomainCheckFailure(db, domain, checkedAt)
	} else {
		updateDomainFromCertMeta(db, domain, meta, checkedAt)
	}
	return &record, notified, nil
}

func createFileParseRecord(db *gorm.DB, domain *CertManagedDomain, meta certMetadata, checkedAt time.Time) error {
	status, errorMessage := certStatusAndError(meta)
	record := CertScanRecord{
		DomainID:       domain.ID,
		DomainName:     domain.Domain,
		CheckType:      scanTypeFileParse,
		Status:         status,
		CheckedAt:      types.Time(checkedAt),
		NotAfter:       types.Time(meta.NotAfter),
		DaysLeft:       meta.DaysLeft,
		Issuer:         meta.Issuer,
		FingerprintSHA: meta.FingerprintSHA,
		ErrorMessage:   errorMessage,
	}
	return db.Create(&record).Error
}

func init() {
	packageContext.GET("cert_scan_record_list.table", CertScanRecordList, CertScanRecordListTemplate)
	packageContext.POST("cert_public_scan.form", CertPublicScan, &app.FormTemplate{
		BaseConfig: app.BaseConfig{
			Name:     "扫描公网证书",
			Desc:     "连接域名公网 443 端口读取当前服务端证书，解析有效期、签发者、指纹和域名匹配结果。只做巡检和提醒，不部署证书。",
			Tags:     []string{"证书管理", "公网巡检", "到期提醒"},
			Request:  &CertPublicScanReq{},
			Response: &CertPublicScanResp{},
			CreateTables: []interface{}{
				&CertManagedDomain{},
				&CertScanRecord{},
			},
			OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
				"domain_id": onSelectFuzzyManagedDomain,
			},
		},
	})
	packageContext.POST("cert_expiry_sweep.form", CertExpirySweep, &app.FormTemplate{
		BaseConfig: app.BaseConfig{
			Name:     "证书到期巡检（定时任务）",
			Desc:     "每天自动巡检启用的域名证书，发现即将过期、已过期或检查失败时提醒负责人；开启自动续期任务的域名会创建续期任务，但不会自动部署证书。",
			Tags:     []string{"证书管理", "定时任务", "到期提醒"},
			Request:  &CertExpirySweepReq{},
			Response: &CertExpirySweepResp{},
			CreateTables: []interface{}{
				&CertManagedDomain{},
				&CertScanRecord{},
				&CertRenewalTask{},
			},
		},
		Schedules: []app.FormSchedule{
			{
				Code:        "cert_expiry_daily_sweep",
				Title:       "证书每日到期巡检",
				Description: "每天 09:00 扫描启用域名的公网证书，提醒即将过期、已过期或检查失败的证书。",
				CronExpr:    "0 9 * * *",
				Body:        CertExpirySweepReq{WarningDays: defaultCertWarnDays, Notify: true, AutoCreateRenewal: true},
			},
		},
	})
}
