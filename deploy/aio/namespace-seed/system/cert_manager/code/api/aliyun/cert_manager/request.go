package cert_manager

import (
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/statistics"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

type CertAliyunRequest struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:申请ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	DomainID        int        `json:"domain_id" gorm:"column:domain_id;index;comment:域名ID" widget:"name:域名ID;type:integer" hide:"create,update"`
	ConfigID        int        `json:"config_id" gorm:"column:config_id;index;comment:配置ID" widget:"name:配置ID;type:integer" hide:"create,update"`
	DomainName      string     `json:"domain_name" gorm:"column:domain_name;type:varchar(255);index;comment:主域名" widget:"name:主域名;type:text" hide:"create,update"`
	SANs            string     `json:"sans" gorm:"column:sans;type:text;comment:SAN域名" widget:"name:SAN域名;type:text_area" hide:"create,update"`
	RequestType     string     `json:"request_type" gorm:"column:request_type;type:varchar(30);comment:申请类型" widget:"name:申请类型;type:select;options:首次签发,手动续期,自动续期;options_colors:409EFF,E6A23C,67C23A" hide:"create,update"`
	Status          string     `json:"status" gorm:"column:status;type:varchar(30);index;comment:状态" widget:"name:状态;type:select;options:待执行,执行中,等待DNS生效,验证中,签发成功,失败,已取消;options_colors:909399,409EFF,E6A23C,409EFF,67C23A,F56C6C,909399" hide:"create,update"`
	RequestedBy     string     `json:"requested_by" gorm:"column:requested_by;type:varchar(120);comment:发起人" widget:"name:发起人;type:user" hide:"create,update"`
	StartedAt       types.Time `json:"started_at" gorm:"column:started_at;type:datetime;comment:开始时间" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	CompletedAt     types.Time `json:"completed_at" gorm:"column:completed_at;type:datetime;comment:完成时间" widget:"name:完成时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	ChallengeName   string     `json:"challenge_name" gorm:"column:challenge_name;type:text;comment:DNS验证名称" widget:"name:DNS验证名称;type:text" hide:"create,update"`
	ChallengeValue  string     `json:"challenge_value" gorm:"column:challenge_value;type:text;comment:DNS验证值" widget:"name:DNS验证值;type:text_area" hide:"create,update"`
	AliyunZoneID    string     `json:"aliyun_zone_id" gorm:"column:aliyun_zone_id;type:varchar(80);comment:Aliyun Zone ID" widget:"name:Zone ID;type:text" hide:"create,update"`
	AliyunZoneName  string     `json:"aliyun_zone_name" gorm:"column:aliyun_zone_name;type:varchar(255);comment:Aliyun Zone" widget:"name:Zone;type:text" hide:"create,update"`
	AliyunRecordIDs string     `json:"aliyun_record_ids" gorm:"column:aliyun_record_ids;type:text;comment:DNS记录ID" widget:"name:DNS记录ID;type:text_area" hide:"create,update"`
	AssetID         int        `json:"asset_id" gorm:"column:asset_id;index;comment:证书资产ID" widget:"name:证书资产ID;type:integer" hide:"create,update"`
	AssetLink       string     `json:"asset_link" gorm:"-" widget:"name:证书资产;type:link;link_type:primary" hide:"create,update"`
	ErrorMessage    string     `json:"error_message" gorm:"column:error_message;type:text;comment:错误信息" widget:"name:错误信息;type:text_area" hide:"create,update"`
	LastMessage     string     `json:"last_message" gorm:"column:last_message;type:text;comment:最近消息" widget:"name:最近消息;type:text_area" hide:"create,update"`
	Remark          string     `json:"remark" gorm:"column:remark;type:text;comment:备注" widget:"name:备注;type:text_area" hide:"create,update"`
}

func (CertAliyunRequest) TableName() string {
	return "aliyun_cert_request"
}

type CertAliyunRequestListReq struct {
	DomainName        string `json:"domain_name" form:"domain_name" widget:"name:域名;type:input"`
	Status            string `json:"status" form:"status" widget:"name:状态;type:select;options:待执行,执行中,等待DNS生效,验证中,签发成功,失败,已取消;options_colors:909399,409EFF,E6A23C,409EFF,67C23A,F56C6C,909399"`
	RequestType       string `json:"request_type" form:"request_type" widget:"name:申请类型;type:select;options:首次签发,手动续期,自动续期;options_colors:409EFF,E6A23C,67C23A"`
	query.PageSortReq `widget:"-"`
}

func CertAliyunRequestList(ctx *app.Context, resp response.Response) error {
	db, err := certManagerDB(ctx)
	if err != nil {
		return err
	}
	var req CertAliyunRequestListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	queryDB := db.Model(&CertAliyunRequest{})
	if strings.TrimSpace(req.DomainName) != "" {
		queryDB = queryDB.Where("domain_name LIKE ? OR sans LIKE ?", "%"+strings.TrimSpace(req.DomainName)+"%", "%"+strings.TrimSpace(req.DomainName)+"%")
	}
	if strings.TrimSpace(req.Status) != "" {
		queryDB = queryDB.Where("status = ?", strings.TrimSpace(req.Status))
	}
	if strings.TrimSpace(req.RequestType) != "" {
		queryDB = queryDB.Where("request_type = ?", strings.TrimSpace(req.RequestType))
	}
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	} else {
		queryDB = queryDB.Order("created_at DESC")
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}
	var rows []CertAliyunRequest
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&rows).Error; err != nil {
		return err
	}
	for i := range rows {
		if rows[i].AssetID > 0 {
			rows[i].AssetLink, _ = ctx.BuildFunctionUrlWithText("assets.table", CertAliyunAsset{ID: rows[i].AssetID}, "查看证书资产")
		}
	}
	return resp.Table(response.TableResult{Items: rows, TotalCount: total, PageInfo: &req.PageSortReq}).Build()
}

type CertAliyunIssueReq struct {
	DomainID            int    `json:"domain_id" widget:"name:证书域名;type:select" validate:"required" callback:"OnSelectFuzzy"`
	RequestType         string `json:"request_type" widget:"name:申请类型;type:select;options:首次签发,手动续期;options_colors:409EFF,E6A23C;render_default:首次签发"`
	WaitSeconds         int    `json:"wait_seconds" widget:"name:DNS等待秒数;type:integer;min:30;max:900;render_default:300"`
	PollIntervalSeconds int    `json:"poll_interval_seconds" widget:"name:DNS轮询间隔;type:integer;min:5;max:120;render_default:15"`
	CleanupChallenge    bool   `json:"cleanup_challenge" widget:"name:签发后删除TXT;type:switch;render_default:true"`
	Remark              string `json:"remark" widget:"name:备注;type:text_area"`
}

type CertAliyunIssueResp struct {
	RequestID       int        `json:"request_id" widget:"name:申请ID;type:ID"`
	AssetID         int        `json:"asset_id" widget:"name:证书ID;type:ID"`
	DomainName      string     `json:"domain_name" widget:"name:主域名;type:text"`
	Status          string     `json:"status" widget:"name:状态;type:text"`
	NotAfter        types.Time `json:"not_after" widget:"name:过期时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	DaysLeft        int        `json:"days_left" widget:"name:剩余天数;type:integer"`
	CertificateFile string     `json:"certificate_file" widget:"name:cert.pem;type:files"`
	FullChainFile   string     `json:"fullchain_file" widget:"name:fullchain.pem;type:files"`
	PrivateKeyFile  string     `json:"private_key_file" widget:"name:private.key;type:files"`
	BundleFile      string     `json:"bundle_file" widget:"name:证书包ZIP;type:files"`
	AssetLink       string     `json:"asset_link" widget:"name:查看证书资产;type:link;link_type:primary"`
}

func CertAliyunIssue(ctx *app.Context, resp response.Response) error {
	db, err := certManagerDB(ctx)
	if err != nil {
		return err
	}
	req := CertAliyunIssueReq{
		WaitSeconds:         defaultDNSWaitSeconds,
		PollIntervalSeconds: defaultDNSPollSeconds,
		CleanupChallenge:    true,
	}
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	var domain CertAliyunDomain
	if err := db.First(&domain, req.DomainID).Error; err != nil {
		return fmt.Errorf("证书域名不存在")
	}
	logger.Infof(ctx, "[CertManager][Aliyun] manual issue submitted domain_id=%d domain=%s request_type=%s wait=%d poll=%d cleanup=%v user=%s",
		domain.ID, domain.Domain, firstNonEmpty(req.RequestType, requestTypeIssue), req.WaitSeconds, req.PollIntervalSeconds, req.CleanupChallenge, ctx.GetRequestUser())
	if err := ensureNoRunningRequest(db, domain.ID); err != nil {
		logger.Warnf(ctx, "[CertManager][Aliyun] manual issue blocked by running request domain_id=%d domain=%s err=%v", domain.ID, domain.Domain, err)
		return err
	}
	requestType := firstNonEmpty(req.RequestType, requestTypeIssue)
	reqRecord, err := createRequestRecord(ctx, db, &domain, requestType, req.Remark)
	if err != nil {
		return err
	}
	logger.Infof(ctx, "[CertManager][Aliyun] request record created request_id=%d domain_id=%d domain=%s type=%s",
		reqRecord.ID, domain.ID, domain.Domain, requestType)
	asset, issueErr := runIssueRequest(ctx, db, &domain, reqRecord, certificateIssueOptions{
		WaitSeconds:         req.WaitSeconds,
		PollIntervalSeconds: req.PollIntervalSeconds,
		CleanupChallenge:    req.CleanupChallenge,
	})
	if issueErr != nil {
		logger.Errorf(ctx, "[CertManager][Aliyun] manual issue failed request_id=%d domain=%s err=%v", reqRecord.ID, domain.Domain, issueErr)
		markRequestFailed(db, reqRecord.ID, issueErr)
		updateDomainFailure(db, domain.ID, time.Now(), issueErr.Error())
		sendIssueFailureMessage(ctx, &domain, reqRecord, issueErr)
		return issueErr
	}
	logger.Infof(ctx, "[CertManager][Aliyun] manual issue succeeded request_id=%d asset_id=%d domain=%s", reqRecord.ID, asset.ID, domain.Domain)
	link, _ := ctx.BuildFunctionUrlWithText("assets.table", CertAliyunAsset{ID: asset.ID}, "查看证书资产")
	return resp.Form(&CertAliyunIssueResp{
		RequestID:       reqRecord.ID,
		AssetID:         asset.ID,
		DomainName:      asset.DomainName,
		Status:          requestStatusIssued,
		NotAfter:        asset.NotAfter,
		DaysLeft:        asset.DaysLeft,
		CertificateFile: asset.CertificateFile,
		FullChainFile:   asset.FullChainFile,
		PrivateKeyFile:  asset.PrivateKeyFile,
		BundleFile:      asset.BundleFile,
		AssetLink:       link,
	}).Build()
}

type CertAliyunSweepReq struct {
	WaitSeconds         int  `json:"wait_seconds" widget:"name:DNS等待秒数;type:integer;min:30;max:900;render_default:300"`
	PollIntervalSeconds int  `json:"poll_interval_seconds" widget:"name:DNS轮询间隔;type:integer;min:5;max:120;render_default:15"`
	CleanupChallenge    bool `json:"cleanup_challenge" widget:"name:签发后删除TXT;type:switch;render_default:true"`
}

type CertAliyunSweepResp struct {
	CheckedCount int `json:"checked_count" widget:"name:检查域名数;type:integer"`
	SkippedCount int `json:"skipped_count" widget:"name:跳过数;type:integer"`
	IssuedCount  int `json:"issued_count" widget:"name:续期成功数;type:integer"`
	FailedCount  int `json:"failed_count" widget:"name:续期失败数;type:integer"`
}

func CertAliyunSweep(ctx *app.Context, resp response.Response) error {
	db, err := certManagerDB(ctx)
	if err != nil {
		return err
	}
	req := CertAliyunSweepReq{WaitSeconds: defaultDNSWaitSeconds, PollIntervalSeconds: defaultDNSPollSeconds, CleanupChallenge: true}
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	var domains []CertAliyunDomain
	if err := db.Where("enabled = ? AND auto_renew = ?", true, true).Order("domain ASC").Find(&domains).Error; err != nil {
		return err
	}
	result := CertAliyunSweepResp{CheckedCount: len(domains)}
	logger.Infof(ctx, "[CertManager][Aliyun] auto renew sweep start checked=%d wait=%d poll=%d cleanup=%v",
		len(domains), req.WaitSeconds, req.PollIntervalSeconds, req.CleanupChallenge)
	for i := range domains {
		domain := domains[i]
		if !domainNeedsRenewal(domain) {
			logger.Debugf(ctx, "[CertManager][Aliyun] auto renew skipped no renewal needed domain_id=%d domain=%s days_left=%d renew_before=%d",
				domain.ID, domain.Domain, domain.CurrentDaysLeft, domain.RenewBeforeDays)
			result.SkippedCount++
			continue
		}
		if err := ensureNoRunningRequest(db, domain.ID); err != nil {
			logger.Warnf(ctx, "[CertManager][Aliyun] auto renew skipped running request domain_id=%d domain=%s err=%v", domain.ID, domain.Domain, err)
			result.SkippedCount++
			continue
		}
		reqRecord, err := createRequestRecord(ctx, db, &domain, requestTypeAutoRenew, "每日自动续期巡检触发")
		if err != nil {
			logger.Warnf(ctx, "create auto renewal request failed domain=%s err=%v", domain.Domain, err)
			result.FailedCount++
			continue
		}
		logger.Infof(ctx, "[CertManager][Aliyun] auto renew request created request_id=%d domain_id=%d domain=%s",
			reqRecord.ID, domain.ID, domain.Domain)
		if _, err := runIssueRequest(ctx, db, &domain, reqRecord, certificateIssueOptions{
			WaitSeconds:         req.WaitSeconds,
			PollIntervalSeconds: req.PollIntervalSeconds,
			CleanupChallenge:    req.CleanupChallenge,
		}); err != nil {
			logger.Errorf(ctx, "[CertManager][Aliyun] auto renew failed request_id=%d domain=%s err=%v", reqRecord.ID, domain.Domain, err)
			markRequestFailed(db, reqRecord.ID, err)
			updateDomainFailure(db, domain.ID, time.Now(), err.Error())
			sendIssueFailureMessage(ctx, &domain, reqRecord, err)
			result.FailedCount++
			continue
		}
		logger.Infof(ctx, "[CertManager][Aliyun] auto renew succeeded request_id=%d domain=%s", reqRecord.ID, domain.Domain)
		result.IssuedCount++
	}
	logger.Infof(ctx, "[CertManager][Aliyun] auto renew sweep finished checked=%d skipped=%d issued=%d failed=%d",
		result.CheckedCount, result.SkippedCount, result.IssuedCount, result.FailedCount)
	return resp.Form(&result).Build()
}

func runIssueRequest(ctx *app.Context, db *gorm.DB, domain *CertAliyunDomain, reqRecord *CertAliyunRequest, opts certificateIssueOptions) (*CertAliyunAsset, error) {
	now := time.Now()
	logger.Infof(ctx, "[CertManager][Aliyun] run issue request start request_id=%d domain_id=%d domain=%s type=%s",
		reqRecord.ID, domain.ID, domain.Domain, reqRecord.RequestType)
	_ = db.Model(&CertAliyunRequest{}).Where("id = ?", reqRecord.ID).Updates(map[string]interface{}{
		"status":     requestStatusRunning,
		"started_at": types.Time(now),
	}).Error
	issued, err := issueCertificateForRequest(ctx, db, domain, reqRecord, opts)
	if err != nil {
		logger.Errorf(ctx, "[CertManager][Aliyun] issue certificate failed request_id=%d domain=%s err=%v", reqRecord.ID, domain.Domain, err)
		return nil, err
	}
	asset, err := createAssetFromIssuedCertificate(ctx, db, domain, reqRecord, issued)
	if err != nil {
		logger.Errorf(ctx, "[CertManager][Aliyun] create asset failed request_id=%d domain=%s err=%v", reqRecord.ID, domain.Domain, err)
		return nil, err
	}
	_ = db.Model(&CertAliyunRequest{}).Where("id = ?", reqRecord.ID).Updates(map[string]interface{}{
		"status":       requestStatusIssued,
		"completed_at": types.Time(time.Now()),
		"asset_id":     asset.ID,
		"last_message": "证书签发成功，文件已归档到证书资产库。",
	}).Error
	sendIssueSuccessMessage(ctx, domain, reqRecord, asset)
	logger.Infof(ctx, "[CertManager][Aliyun] run issue request finished request_id=%d asset_id=%d domain=%s elapsed=%s",
		reqRecord.ID, asset.ID, domain.Domain, time.Since(now).Round(time.Millisecond))
	return asset, nil
}

func createRequestRecord(ctx *app.Context, db *gorm.DB, domain *CertAliyunDomain, requestType string, remark string) (*CertAliyunRequest, error) {
	if requestType == "" {
		requestType = requestTypeIssue
	}
	row := &CertAliyunRequest{
		DomainID:    domain.ID,
		ConfigID:    domain.ConfigID,
		DomainName:  domain.Domain,
		SANs:        domain.SANs,
		RequestType: requestType,
		Status:      requestStatusPending,
		RequestedBy: ctx.GetRequestUser(),
		Remark:      remark,
	}
	if row.RequestedBy == "" {
		row.RequestedBy = "system"
	}
	if err := db.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func ensureNoRunningRequest(db *gorm.DB, domainID int) error {
	var count int64
	if err := db.Model(&CertAliyunRequest{}).
		Where("domain_id = ? AND status IN ?", domainID, []string{requestStatusPending, requestStatusRunning, requestStatusWaitDNS, requestStatusVerify}).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("当前域名已有执行中的证书申请，请等待完成后再提交")
	}
	return nil
}

func updateRequestStatus(db *gorm.DB, requestID int, status string, message string, extra map[string]interface{}) {
	updates := map[string]interface{}{
		"status":       status,
		"last_message": message,
	}
	for key, value := range extra {
		updates[key] = value
	}
	_ = db.Model(&CertAliyunRequest{}).Where("id = ?", requestID).Updates(updates).Error
}

func appendChallengeRecord(db *gorm.DB, requestID int, zone *aliyunZone, record *aliyunDNSRecord, name string, value string) {
	var row CertAliyunRequest
	if err := db.First(&row, requestID).Error; err != nil {
		return
	}
	recordRef := zone.ID + ":" + record.ID
	records := splitList(row.AliyunRecordIDs)
	records = append(records, recordRef)
	updates := map[string]interface{}{
		"challenge_name":    firstNonEmpty(row.ChallengeName, name),
		"challenge_value":   firstNonEmpty(row.ChallengeValue, value),
		"aliyun_zone_id":    firstNonEmpty(row.AliyunZoneID, zone.ID),
		"aliyun_zone_name":  firstNonEmpty(row.AliyunZoneName, zone.Name),
		"aliyun_record_ids": strings.Join(records, ","),
	}
	_ = db.Model(&CertAliyunRequest{}).Where("id = ?", requestID).Updates(updates).Error
}

func markRequestFailed(db *gorm.DB, requestID int, err error) {
	_ = db.Model(&CertAliyunRequest{}).Where("id = ?", requestID).Updates(map[string]interface{}{
		"status":        requestStatusFailed,
		"completed_at":  types.Time(time.Now()),
		"error_message": err.Error(),
		"last_message":  "证书签发失败：" + err.Error(),
	}).Error
}

func domainNeedsRenewal(domain CertAliyunDomain) bool {
	if domain.CurrentNotAfter.IsZero() {
		return true
	}
	renewBefore := domain.RenewBeforeDays
	if renewBefore <= 0 {
		renewBefore = defaultRenewBeforeDays
	}
	return domain.CurrentDaysLeft <= renewBefore
}

func sendIssueSuccessMessage(ctx *app.Context, domain *CertAliyunDomain, reqRecord *CertAliyunRequest, asset *CertAliyunAsset) {
	toUsers := joinCertificateNotifyUsers(*domain)
	if toUsers == "" {
		return
	}
	content := fmt.Sprintf("域名 `%s` 证书%s成功。\n\n过期时间：%s\n剩余天数：%d\n\n证书文件已保存到证书资产库，系统不会自动部署证书。",
		domain.Domain, reqRecord.RequestType, asset.NotAfter.String(), asset.DaysLeft)
	_ = ctx.SendMessage(&app.SendMessageOpts{
		ToUsers: toUsers,
		Title:   "阿里云证书签发成功",
		Content: content,
	})
}

func sendIssueFailureMessage(ctx *app.Context, domain *CertAliyunDomain, reqRecord *CertAliyunRequest, err error) {
	toUsers := joinCertificateNotifyUsers(*domain)
	if toUsers == "" {
		return
	}
	content := fmt.Sprintf("域名 `%s` 证书%s失败。\n\n错误信息：%s", domain.Domain, reqRecord.RequestType, err.Error())
	_ = ctx.SendMessage(&app.SendMessageOpts{
		ToUsers: toUsers,
		Title:   "阿里云证书签发失败",
		Content: content,
	})
}

func certAliyunRequestOnSelectFuzzy(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db, err := certManagerDB(ctx)
	if err != nil {
		return nil, err
	}
	queryDB := db.Model(&CertAliyunRequest{})
	if req != nil && req.IsByValue() {
		queryDB = queryDB.Where("id = ?", req.GetValue()).Limit(1)
	} else {
		keyword := ""
		if req != nil {
			keyword = strings.TrimSpace(req.Keyword())
		}
		if keyword != "" {
			queryDB = queryDB.Where("domain_name LIKE ? OR status LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
		}
		queryDB = queryDB.Limit(20)
	}
	var rows []CertAliyunRequest
	if err := queryDB.Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]*callback.SelectFuzzyItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &callback.SelectFuzzyItem{
			Value: row.ID,
			Label: fmt.Sprintf("#%d %s - %s", row.ID, row.DomainName, row.Status),
			DisplayInfo: map[string]interface{}{
				"域名":   row.DomainName,
				"状态":   row.Status,
				"申请类型": row.RequestType,
			},
		})
	}
	return &callback.OnSelectFuzzyResp{
		MaxSelections: 1,
		Items:         items,
		Statistics: map[string]interface{}{
			"域名":   statistics.Value("域名"),
			"状态":   statistics.Value("状态"),
			"申请类型": statistics.Value("申请类型"),
		},
	}, nil
}

var CertAliyunRequestListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "证书申请与续期记录",
		Desc:         "记录阿里云 DNS-01 自动签发和自动续期的状态、DNS challenge、错误信息和资产关联。",
		Tags:         []string{"阿里云", "证书申请", "自动续期"},
		Request:      &CertAliyunRequestListReq{},
		Response:     query.PaginatedTable[[]CertAliyunRequest]{},
		CreateTables: certManagerTables(),
	},
}

func init() {
	packageContext.GET("requests.table", CertAliyunRequestList, CertAliyunRequestListTemplate)
	packageContext.POST("issue.form", CertAliyunIssue, &app.FormTemplate{
		BaseConfig: app.BaseConfig{
			Name:         "申请/续期证书",
			Desc:         "为域名自动执行 Let's Encrypt DNS-01 签发：写入阿里云 TXT、等待 DNS 生效、提交 ACME 验证、签发证书并归档文件。不会自动部署证书。",
			Tags:         []string{"阿里云", "自动签发", "DNS-01"},
			Request:      &CertAliyunIssueReq{},
			Response:     &CertAliyunIssueResp{},
			CreateTables: certManagerTables(),
			OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
				"domain_id": certAliyunDomainOnSelectFuzzy,
			},
		},
	})
	packageContext.POST("auto_renew_sweep.form", CertAliyunSweep, &app.FormTemplate{
		BaseConfig: app.BaseConfig{
			Name:         "证书自动续期巡检",
			Desc:         "扫描启用自动续期的域名，发现未签发或即将过期时自动走阿里云 DNS-01 签发流程并归档证书文件。",
			Tags:         []string{"阿里云", "自动续期", "定时任务"},
			Request:      &CertAliyunSweepReq{},
			Response:     &CertAliyunSweepResp{},
			CreateTables: certManagerTables(),
		},
		Schedules: []app.FormSchedule{
			{
				Code:        "aliyun_cert_auto_renew_daily",
				Title:       "阿里云证书每日自动续期巡检",
				Description: "每天 03:00 检查启用自动续期的域名，快过期时自动签发新证书并归档。",
				CronExpr:    "0 3 * * *",
				Body:        CertAliyunSweepReq{WaitSeconds: defaultDNSWaitSeconds, PollIntervalSeconds: defaultDNSPollSeconds, CleanupChallenge: true},
			},
		},
	})
}
