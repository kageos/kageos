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

type CertAsset struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:证书ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	DomainID        int                `json:"domain_id" gorm:"column:domain_id;index;comment:域名ID" widget:"name:域名;type:select" callback:"OnSelectFuzzy"`
	Domain          *CertManagedDomain `json:"-" widget:"-" gorm:"foreignKey:DomainID;references:ID"`
	DomainName      string             `json:"domain_name" gorm:"column:domain_name;type:varchar(255);index;comment:域名" widget:"name:域名;type:text" hide:"create,update"`
	Source          string             `json:"source" gorm:"column:source;comment:来源" widget:"name:来源;type:select;options:上传,续期结果,公网扫描,手动导入;options_colors:409EFF,67C23A,909399,E6A23C" hide:"create,update"`
	Status          string             `json:"status" gorm:"column:status;comment:状态" widget:"name:状态;type:select;options:正常,即将过期,已过期,检查失败,待部署;options_colors:67C23A,E6A23C,F56C6C,F56C6C,409EFF" hide:"create,update"`
	CertificateFile string             `json:"certificate_file" gorm:"column:certificate_file;type:text;comment:证书文件" widget:"name:证书文件;type:files;accept:.pem,.crt,.cer;max_count:1" hide:"create,update"`
	PrivateKeyFile  string             `json:"private_key_file" gorm:"column:private_key_file;type:text;comment:私钥文件" widget:"name:私钥文件;type:files;accept:.key,.pem;max_count:1" hide:"create,update"`
	BundleFile      string             `json:"bundle_file" gorm:"column:bundle_file;type:text;comment:证书包" widget:"name:证书包;type:files;accept:.zip,.tar,.gz,.tgz,.pem,.crt,.cer,.key;max_count:5" hide:"create,update"`
	Issuer          string             `json:"issuer" gorm:"column:issuer;type:text;comment:签发者" widget:"name:签发者;type:text_area" hide:"create,update"`
	Subject         string             `json:"subject" gorm:"column:subject;type:text;comment:证书主题" widget:"name:证书主题;type:text_area" hide:"create,update"`
	SANs            string             `json:"sans" gorm:"column:sans;type:text;comment:SAN域名" widget:"name:SAN域名;type:text_area" hide:"create,update"`
	SerialNumber    string             `json:"serial_number" gorm:"column:serial_number;type:text;comment:序列号" widget:"name:序列号;type:text" hide:"create,update"`
	FingerprintSHA  string             `json:"fingerprint_sha256" gorm:"column:fingerprint_sha256;type:text;comment:SHA256指纹" widget:"name:SHA256指纹;type:text" hide:"create,update"`
	NotBefore       types.Time         `json:"not_before" gorm:"column:not_before;type:datetime;comment:生效时间" widget:"name:生效时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	NotAfter        types.Time         `json:"not_after" gorm:"column:not_after;type:datetime;comment:过期时间" widget:"name:过期时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DaysLeft        int                `json:"days_left" gorm:"column:days_left;comment:剩余天数" widget:"name:剩余天数;type:integer" hide:"create,update"`
	HostnameMatched bool               `json:"hostname_matched" gorm:"column:hostname_matched;comment:域名匹配" widget:"name:域名匹配;type:switch" hide:"create,update"`
	ImportedBy      string             `json:"imported_by" gorm:"column:imported_by;comment:导入人" widget:"name:导入人;type:user" hide:"create,update"`
	Remark          string             `json:"remark" gorm:"column:remark;type:text;comment:备注" widget:"name:备注;type:text_area" hide:"create,update"`
}

func (CertAsset) TableName() string {
	return "ops_cert_asset"
}

type CertAssetListReq struct {
	DomainName string `json:"domain_name" form:"domain_name" widget:"name:域名;type:input"`
	Status     string `json:"status" form:"status" widget:"name:状态;type:select;options:正常,即将过期,已过期,检查失败,待部署;options_colors:67C23A,E6A23C,F56C6C,F56C6C,409EFF"`
	Source     string `json:"source" form:"source" widget:"name:来源;type:select;options:上传,续期结果,公网扫描,手动导入;options_colors:409EFF,67C23A,909399,E6A23C"`

	query.PageSortReq `widget:"-"`
}

func CertAssetList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}
	var req CertAssetListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	queryDB := db.Model(&CertAsset{})
	if req.DomainName != "" {
		queryDB = queryDB.Where("domain_name LIKE ?", "%"+strings.TrimSpace(req.DomainName)+"%")
	}
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}
	if req.Source != "" {
		queryDB = queryDB.Where("source = ?", req.Source)
	}
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	} else {
		queryDB = queryDB.Order("not_after ASC, created_at DESC")
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}
	var assets []CertAsset
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&assets).Error; err != nil {
		return err
	}
	return resp.Table(response.TableResult{
		Items:      assets,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

var CertAssetListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "证书资产库",
		Desc:         "集中保存证书文件、私钥文件和证书包，并展示证书有效期、签发者、SAN、指纹和域名匹配结果。证书只保存和下载，不自动部署。",
		Tags:         []string{"证书管理", "证书文件", "files"},
		Request:      &CertAssetListReq{},
		Response:     query.PaginatedTable[[]CertAsset]{},
		CreateTables: []interface{}{&CertAsset{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"domain_id": onSelectFuzzyManagedDomain,
		},
	},
	AutoCrudTable: &CertAsset{},
}

type CertImportReq struct {
	DomainID        int    `json:"domain_id" widget:"name:域名;type:select" validate:"required" callback:"OnSelectFuzzy"`
	CertificateFile string `json:"certificate_file" widget:"name:证书文件;type:files;accept:.pem,.crt,.cer;max_count:1" validate:"required"`
	PrivateKeyFile  string `json:"private_key_file" widget:"name:私钥文件;type:files;accept:.key,.pem;max_count:1"`
	BundleFile      string `json:"bundle_file" widget:"name:证书包;type:files;accept:.zip,.tar,.gz,.tgz,.pem,.crt,.cer,.key;max_count:5"`
	Source          string `json:"source" widget:"name:来源;type:select;options:上传,续期结果,手动导入;options_colors:409EFF,67C23A,E6A23C;render_default:上传"`
	Remark          string `json:"remark" widget:"name:备注;type:text_area"`
}

type CertImportResp struct {
	AssetID         int        `json:"asset_id" widget:"name:证书ID;type:ID"`
	DomainName      string     `json:"domain_name" widget:"name:域名;type:text"`
	Status          string     `json:"status" widget:"name:状态;type:text"`
	NotAfter        types.Time `json:"not_after" widget:"name:过期时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	DaysLeft        int        `json:"days_left" widget:"name:剩余天数;type:integer"`
	HostnameMatched bool       `json:"hostname_matched" widget:"name:域名匹配;type:switch"`
	FingerprintSHA  string     `json:"fingerprint_sha256" widget:"name:SHA256指纹;type:text"`
	AssetLink       string     `json:"asset_link" widget:"name:查看证书资产;type:link;link_type:primary"`
}

func CertImport(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}
	var req CertImportReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	var domain CertManagedDomain
	if err := db.Where("id = ?", req.DomainID).First(&domain).Error; err != nil {
		return fmt.Errorf("域名资产不存在")
	}
	cert, downloaded, err := parseUploadedCertificate(ctx, req.CertificateFile)
	if err != nil {
		return err
	}
	defer ctx.GetFS().RemoveFiles(downloaded)

	meta := certMetadataFromX509(cert, domain.Domain, domain.WarnDays)
	assetStatus, _ := certStatusAndError(meta)
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "上传"
	}
	now := time.Now()
	asset := CertAsset{
		DomainID:        domain.ID,
		DomainName:      domain.Domain,
		Source:          source,
		Status:          assetStatus,
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
		logger.Errorf(ctx, "Create cert asset err: %v", err)
		return err
	}
	if err := createFileParseRecord(db, &domain, meta, now); err != nil {
		logger.Warnf(ctx, "Create cert file parse record err: %v", err)
	}
	updateDomainFromParsedCert(db, &domain, meta, now)

	link, _ := ctx.BuildFunctionUrlWithText("cert_asset_list.table", CertAsset{ID: asset.ID}, "查看证书资产")
	return resp.Form(&CertImportResp{
		AssetID:         asset.ID,
		DomainName:      asset.DomainName,
		Status:          asset.Status,
		NotAfter:        asset.NotAfter,
		DaysLeft:        asset.DaysLeft,
		HostnameMatched: asset.HostnameMatched,
		FingerprintSHA:  asset.FingerprintSHA,
		AssetLink:       link,
	}).Build()
}

func init() {
	packageContext.GET("cert_asset_list.table", CertAssetList, CertAssetListTemplate)
	packageContext.POST("cert_import.form", CertImport, &app.FormTemplate{
		BaseConfig: app.BaseConfig{
			Name:     "导入证书文件",
			Desc:     "上传证书文件、私钥文件和证书包，系统解析证书有效期、SAN、签发者和指纹后写入证书资产库。只做保存和下载，不自动部署。",
			Tags:     []string{"证书管理", "证书导入", "files"},
			Request:  &CertImportReq{},
			Response: &CertImportResp{},
			CreateTables: []interface{}{
				&CertManagedDomain{},
				&CertAsset{},
			},
			OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
				"domain_id": onSelectFuzzyManagedDomain,
			},
		},
	})
}
