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

type CertRenewalTask struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:任务ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	DomainID         int                `json:"domain_id" gorm:"column:domain_id;index;comment:域名ID" widget:"name:域名;type:select" callback:"OnSelectFuzzy"`
	Domain           *CertManagedDomain `json:"-" widget:"-" gorm:"foreignKey:DomainID;references:ID"`
	DomainName       string             `json:"domain_name" gorm:"column:domain_name;type:varchar(255);index;comment:域名" widget:"name:域名;type:text" hide:"create,update"`
	Mode             string             `json:"mode" gorm:"column:mode;comment:任务模式" widget:"name:任务模式;type:select;options:手动续期,自动续期;options_colors:409EFF,67C23A" hide:"create,update"`
	Status           string             `json:"status" gorm:"column:status;comment:状态" widget:"name:状态;type:select;options:待执行,执行中,已续期待部署,失败,已取消;options_colors:E6A23C,409EFF,67C23A,F56C6C,909399"`
	ValidationMethod string             `json:"validation_method" gorm:"column:validation_method;comment:验证方式" widget:"name:验证方式;type:select;options:HTTP-01,DNS-01,手动;options_colors:409EFF,67C23A,E6A23C" hide:"create,update"`
	RequestedBy      string             `json:"requested_by" gorm:"column:requested_by;comment:发起人" widget:"name:发起人;type:user" hide:"create,update"`
	RequestedAt      types.Time         `json:"requested_at" gorm:"column:requested_at;type:datetime;comment:发起时间" widget:"name:发起时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	CompletedAt      types.Time         `json:"completed_at" gorm:"column:completed_at;type:datetime;comment:完成时间" widget:"name:完成时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	CertificateFile  string             `json:"certificate_file" gorm:"column:certificate_file;type:text;comment:续期证书文件" widget:"name:续期证书文件;type:files;accept:.pem,.crt,.cer;max_count:1" hide:"create,update"`
	PrivateKeyFile   string             `json:"private_key_file" gorm:"column:private_key_file;type:text;comment:续期私钥文件" widget:"name:续期私钥文件;type:files;accept:.key,.pem;max_count:1" hide:"create,update"`
	BundleFile       string             `json:"bundle_file" gorm:"column:bundle_file;type:text;comment:续期证书包" widget:"name:续期证书包;type:files;accept:.zip,.tar,.gz,.tgz,.pem,.crt,.cer,.key;max_count:5" hide:"create,update"`
	AssetID          int                `json:"asset_id" gorm:"column:asset_id;comment:证书资产ID" widget:"name:证书资产ID;type:integer" hide:"create,update"`
	AssetLink        string             `json:"asset_link" gorm:"-" widget:"name:证书资产;type:link;link_type:primary" hide:"create,update"`
	ErrorMessage     string             `json:"error_message" gorm:"column:error_message;type:text;comment:错误信息" widget:"name:错误信息;type:text_area"`
	Remark           string             `json:"remark" gorm:"column:remark;type:text;comment:备注" widget:"name:备注;type:text_area"`
}

func (CertRenewalTask) TableName() string {
	return "ops_cert_renewal_task"
}

type CertRenewalTaskListReq struct {
	DomainName string `json:"domain_name" form:"domain_name" widget:"name:域名;type:input"`
	Status     string `json:"status" form:"status" widget:"name:状态;type:select;options:待执行,执行中,已续期待部署,失败,已取消;options_colors:E6A23C,409EFF,67C23A,F56C6C,909399"`
	Mode       string `json:"mode" form:"mode" widget:"name:任务模式;type:select;options:手动续期,自动续期;options_colors:409EFF,67C23A"`

	query.PageSortReq `widget:"-"`
}

func CertRenewalTaskList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}
	var req CertRenewalTaskListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	queryDB := db.Model(&CertRenewalTask{})
	if req.DomainName != "" {
		queryDB = queryDB.Where("domain_name LIKE ?", "%"+strings.TrimSpace(req.DomainName)+"%")
	}
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}
	if req.Mode != "" {
		queryDB = queryDB.Where("mode = ?", req.Mode)
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
	var tasks []CertRenewalTask
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&tasks).Error; err != nil {
		return err
	}
	for i := range tasks {
		if tasks[i].AssetID > 0 {
			tasks[i].AssetLink, _ = ctx.BuildFunctionUrlWithText("cert_asset_list.table", CertAsset{ID: tasks[i].AssetID}, "查看证书资产")
		}
	}
	return resp.Table(response.TableResult{Items: tasks, TotalCount: total, PageInfo: &req.PageSortReq}).Build()
}

var CertRenewalTaskListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "证书续期任务",
		Desc:         "跟踪证书续期申请、人工执行、续期结果上传和待部署状态。系统不会自动部署证书，只记录和保存续期产物。",
		Tags:         []string{"证书管理", "续期任务", "待部署"},
		Request:      &CertRenewalTaskListReq{},
		Response:     query.PaginatedTable[[]CertRenewalTask]{},
		CreateTables: []interface{}{&CertRenewalTask{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"domain_id": onSelectFuzzyManagedDomain,
		},
	},
	AutoCrudTable: &CertRenewalTask{},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()
		updates := req.ChangedFields()
		allowed := map[string]bool{
			"status":        true,
			"error_message": true,
			"remark":        true,
		}
		filtered := make(map[string]interface{})
		for key, value := range updates {
			if allowed[key] {
				filtered[key] = value
			}
		}
		if len(filtered) == 0 {
			return nil, fmt.Errorf("没有可更新的字段")
		}
		if err := db.Model(&CertRenewalTask{}).Where("id = ?", req.GetId()).Updates(filtered).Error; err != nil {
			logger.Errorf(ctx, "Update cert renewal task err: %v", err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
}

type CertRenewalCreateReq struct {
	DomainID         int    `json:"domain_id" widget:"name:域名;type:select" validate:"required" callback:"OnSelectFuzzy"`
	ValidationMethod string `json:"validation_method" widget:"name:验证方式;type:select;options:HTTP-01,DNS-01,手动;options_colors:409EFF,67C23A,E6A23C;render_default:手动"`
	Remark           string `json:"remark" widget:"name:续期说明;type:text_area"`
}

type CertRenewalCreateResp struct {
	TaskID     int    `json:"task_id" widget:"name:任务ID;type:ID"`
	DomainName string `json:"domain_name" widget:"name:域名;type:text"`
	Status     string `json:"status" widget:"name:状态;type:text"`
	TaskLink   string `json:"task_link" widget:"name:查看续期任务;type:link;link_type:primary"`
}

func CertRenewalCreate(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}
	var req CertRenewalCreateReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	var domain CertManagedDomain
	if err := db.Where("id = ?", req.DomainID).First(&domain).Error; err != nil {
		return fmt.Errorf("域名资产不存在")
	}
	task, err := createRenewalTask(ctx, db, &domain, "手动续期", firstNonEmpty(req.ValidationMethod, domain.ValidationMethod, "手动"), req.Remark)
	if err != nil {
		return err
	}
	link, _ := ctx.BuildFunctionUrlWithText("cert_renewal_task_list.table", CertRenewalTask{ID: task.ID}, "查看续期任务")
	return resp.Form(&CertRenewalCreateResp{
		TaskID:     task.ID,
		DomainName: task.DomainName,
		Status:     task.Status,
		TaskLink:   link,
	}).Build()
}

type CertRenewalResultReq struct {
	TaskID          int    `json:"task_id" widget:"name:续期任务;type:select" validate:"required" callback:"OnSelectFuzzy"`
	CertificateFile string `json:"certificate_file" widget:"name:续期证书文件;type:files;accept:.pem,.crt,.cer;max_count:1" validate:"required"`
	PrivateKeyFile  string `json:"private_key_file" widget:"name:续期私钥文件;type:files;accept:.key,.pem;max_count:1"`
	BundleFile      string `json:"bundle_file" widget:"name:续期证书包;type:files;accept:.zip,.tar,.gz,.tgz,.pem,.crt,.cer,.key;max_count:5"`
	Remark          string `json:"remark" widget:"name:备注;type:text_area"`
}

type CertRenewalResultResp struct {
	TaskID     int        `json:"task_id" widget:"name:任务ID;type:ID"`
	AssetID    int        `json:"asset_id" widget:"name:证书ID;type:ID"`
	DomainName string     `json:"domain_name" widget:"name:域名;type:text"`
	Status     string     `json:"status" widget:"name:任务状态;type:text"`
	NotAfter   types.Time `json:"not_after" widget:"name:证书过期时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	DaysLeft   int        `json:"days_left" widget:"name:剩余天数;type:integer"`
	AssetLink  string     `json:"asset_link" widget:"name:查看证书资产;type:link;link_type:primary"`
}

func CertRenewalResult(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}
	var req CertRenewalResultReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	var task CertRenewalTask
	if err := db.Where("id = ?", req.TaskID).First(&task).Error; err != nil {
		return fmt.Errorf("续期任务不存在")
	}
	var domain CertManagedDomain
	if err := db.Where("id = ?", task.DomainID).First(&domain).Error; err != nil {
		return fmt.Errorf("域名资产不存在")
	}
	cert, downloaded, err := parseUploadedCertificate(ctx, req.CertificateFile)
	if err != nil {
		return err
	}
	defer ctx.GetFS().RemoveFiles(downloaded)

	meta := certMetadataFromX509(cert, domain.Domain, domain.WarnDays)
	assetStatus, statusError := certStatusAndError(meta)
	taskStatus := "已续期待部署"
	if assetStatus == statusFailed || assetStatus == statusExpired {
		taskStatus = "失败"
		if statusError == "" {
			statusError = "续期证书已过期"
		}
	}
	storedAssetStatus := statusPending
	if taskStatus == "失败" {
		storedAssetStatus = assetStatus
	}
	asset := CertAsset{
		DomainID:        domain.ID,
		DomainName:      domain.Domain,
		Source:          "续期结果",
		Status:          storedAssetStatus,
		CertificateFile: req.CertificateFile,
		PrivateKeyFile:  req.PrivateKeyFile,
		BundleFile:      req.BundleFile,
		Issuer:          meta.Issuer,
		Subject:         meta.Subject,
		SANs:            meta.SANs,
		SerialNumber:    meta.SerialNumber,
		FingerprintSHA:  meta.FingerprintSHA,
		NotBefore:       types.Time(meta.NotBefore),
		NotAfter:        types.Time(meta.NotAfter),
		DaysLeft:        meta.DaysLeft,
		HostnameMatched: meta.HostnameMatched,
		ImportedBy:      ctx.GetRequestUser(),
		Remark:          req.Remark,
	}
	if err := db.Create(&asset).Error; err != nil {
		logger.Errorf(ctx, "Create cert renewal result asset err: %v", err)
		return err
	}
	now := time.Now()
	updates := map[string]interface{}{
		"status":           taskStatus,
		"completed_at":     types.Time(now),
		"certificate_file": req.CertificateFile,
		"private_key_file": req.PrivateKeyFile,
		"bundle_file":      req.BundleFile,
		"asset_id":         asset.ID,
		"error_message":    statusError,
		"remark":           strings.TrimSpace(task.Remark + "\n" + req.Remark),
	}
	if err := db.Model(&CertRenewalTask{}).Where("id = ?", task.ID).Updates(updates).Error; err != nil {
		return err
	}
	if err := createFileParseRecord(db, &domain, meta, now); err != nil {
		logger.Warnf(ctx, "Create cert renewal file parse record err: %v", err)
	}
	updateDomainFromParsedCert(db, &domain, meta, now)

	link, _ := ctx.BuildFunctionUrlWithText("cert_asset_list.table", CertAsset{ID: asset.ID}, "查看证书资产")
	return resp.Form(&CertRenewalResultResp{
		TaskID:     task.ID,
		AssetID:    asset.ID,
		DomainName: domain.Domain,
		Status:     taskStatus,
		NotAfter:   asset.NotAfter,
		DaysLeft:   asset.DaysLeft,
		AssetLink:  link,
	}).Build()
}

func ensureRenewalTask(ctx *app.Context, db *gorm.DB, domain *CertManagedDomain, mode string, remark string) (bool, error) {
	var count int64
	if err := db.Model(&CertRenewalTask{}).
		Where("domain_id = ? AND status IN ?", domain.ID, []string{"待执行", "执行中"}).
		Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}
	_, err := createRenewalTask(ctx, db, domain, mode, firstNonEmpty(domain.ValidationMethod, "手动"), remark)
	return err == nil, err
}

func createRenewalTask(ctx *app.Context, db *gorm.DB, domain *CertManagedDomain, mode string, validationMethod string, remark string) (*CertRenewalTask, error) {
	if mode == "" {
		mode = "手动续期"
	}
	if validationMethod == "" || validationMethod == "只监控" {
		validationMethod = "手动"
	}
	now := time.Now()
	task := &CertRenewalTask{
		DomainID:         domain.ID,
		DomainName:       domain.Domain,
		Mode:             mode,
		Status:           "待执行",
		ValidationMethod: validationMethod,
		RequestedBy:      ctx.GetRequestUser(),
		RequestedAt:      types.Time(now),
		Remark:           remark,
	}
	if task.RequestedBy == "" {
		task.RequestedBy = "system"
	}
	if err := db.Create(task).Error; err != nil {
		logger.Errorf(ctx, "Create cert renewal task err: %v", err)
		return nil, err
	}
	toUsers := joinCertificateNotifyUsers(*domain)
	if toUsers != "" {
		_ = ctx.SendMessage(&app.SendMessageOpts{
			ToUsers: toUsers,
			Title:   "证书续期任务已创建",
			Content: fmt.Sprintf("域名 `%s` 已创建证书续期任务，状态：待执行。\n\n本系统只保存续期证书文件，不会自动部署证书。", domain.Domain),
		})
	}
	return task, nil
}

func onSelectFuzzyRenewalTask(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接失败")
	}
	queryDB := db.Model(&CertRenewalTask{}).Where("status IN ?", []string{"待执行", "执行中", "失败"})
	if req.IsByValue() {
		queryDB = db.Model(&CertRenewalTask{}).Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		queryDB = db.Model(&CertRenewalTask{}).Where("id in ?", req.GetValues())
	} else {
		keyword := strings.TrimSpace(req.Keyword())
		queryDB = queryDB.Where("domain_name LIKE ? OR status LIKE ?", "%"+keyword+"%", "%"+keyword+"%").Limit(20)
	}
	var tasks []CertRenewalTask
	if err := queryDB.Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	items := make([]*callback.SelectFuzzyItem, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, &callback.SelectFuzzyItem{
			Value: task.ID,
			Label: fmt.Sprintf("#%d %s - %s", task.ID, task.DomainName, task.Status),
			DisplayInfo: map[string]interface{}{
				"域名":   task.DomainName,
				"状态":   task.Status,
				"任务模式": task.Mode,
				"验证方式": task.ValidationMethod,
			},
		})
	}
	return &callback.OnSelectFuzzyResp{
		MaxSelections: 1,
		Items:         items,
		Statistics: map[string]interface{}{
			"域名":   statistics.Value("域名"),
			"状态":   statistics.Value("状态"),
			"任务模式": statistics.Value("任务模式"),
			"验证方式": statistics.Value("验证方式"),
		},
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func init() {
	packageContext.GET("cert_renewal_task_list.table", CertRenewalTaskList, CertRenewalTaskListTemplate)
	packageContext.POST("cert_renewal_create.form", CertRenewalCreate, &app.FormTemplate{
		BaseConfig: app.BaseConfig{
			Name:     "创建证书续期任务",
			Desc:     "为域名创建续期任务，后续可由管理员或受控执行器完成证书申请，并通过“登记续期结果”上传证书文件。不会自动部署证书。",
			Tags:     []string{"证书管理", "续期任务"},
			Request:  &CertRenewalCreateReq{},
			Response: &CertRenewalCreateResp{},
			CreateTables: []interface{}{
				&CertManagedDomain{},
				&CertRenewalTask{},
			},
			OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
				"domain_id": onSelectFuzzyManagedDomain,
			},
		},
	})
	packageContext.POST("cert_renewal_result.form", CertRenewalResult, &app.FormTemplate{
		BaseConfig: app.BaseConfig{
			Name:     "登记续期结果",
			Desc:     "上传续期后得到的证书文件、私钥文件和证书包，系统解析后归档到证书资产库，并把续期任务标记为“已续期待部署”。",
			Tags:     []string{"证书管理", "证书文件", "续期结果"},
			Request:  &CertRenewalResultReq{},
			Response: &CertRenewalResultResp{},
			CreateTables: []interface{}{
				&CertManagedDomain{},
				&CertRenewalTask{},
				&CertAsset{},
			},
			OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
				"task_id": onSelectFuzzyRenewalTask,
			},
		},
	})
}
