package cert_manager

import (
	"fmt"
	"strings"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

type CertCFAsset struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:证书ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	DomainID        int        `json:"domain_id" gorm:"column:domain_id;index;comment:域名ID" widget:"name:域名ID;type:integer" hide:"create,update"`
	RequestID       int        `json:"request_id" gorm:"column:request_id;index;comment:申请ID" widget:"name:申请ID;type:integer" hide:"create,update"`
	ConfigID        int        `json:"config_id" gorm:"column:config_id;index;comment:配置ID" widget:"name:配置ID;type:integer" hide:"create,update"`
	DomainName      string     `json:"domain_name" gorm:"column:domain_name;type:varchar(255);index;comment:主域名" widget:"name:主域名;type:text" hide:"create,update"`
	SANs            string     `json:"sans" gorm:"column:sans;type:text;comment:SAN域名" widget:"name:SAN域名;type:text_area" hide:"create,update"`
	Source          string     `json:"source" gorm:"column:source;type:varchar(30);comment:来源" widget:"name:来源;type:select;options:首次签发,手动续期,自动续期;options_colors:409EFF,E6A23C,67C23A" hide:"create,update"`
	Status          string     `json:"status" gorm:"column:status;type:varchar(30);comment:状态" widget:"name:状态;type:select;options:正常,即将过期,已过期,失败,待部署;options_colors:67C23A,E6A23C,F56C6C,F56C6C,409EFF" hide:"create,update"`
	CertificateFile string     `json:"certificate_file" gorm:"column:certificate_file;type:text;comment:证书文件" widget:"name:cert.pem;type:files" hide:"create,update"`
	ChainFile       string     `json:"chain_file" gorm:"column:chain_file;type:text;comment:证书链" widget:"name:chain.pem;type:files" hide:"create,update"`
	FullChainFile   string     `json:"fullchain_file" gorm:"column:fullchain_file;type:text;comment:完整证书链" widget:"name:fullchain.pem;type:files" hide:"create,update"`
	PrivateKeyFile  string     `json:"private_key_file" gorm:"column:private_key_file;type:text;comment:私钥文件" widget:"name:private.key;type:files" hide:"create,update"`
	BundleFile      string     `json:"bundle_file" gorm:"column:bundle_file;type:text;comment:证书包" widget:"name:证书包ZIP;type:files" hide:"create,update"`
	Issuer          string     `json:"issuer" gorm:"column:issuer;type:text;comment:签发者" widget:"name:签发者;type:text_area" hide:"create,update"`
	Subject         string     `json:"subject" gorm:"column:subject;type:text;comment:证书主题" widget:"name:证书主题;type:text_area" hide:"create,update"`
	SerialNumber    string     `json:"serial_number" gorm:"column:serial_number;type:text;comment:序列号" widget:"name:序列号;type:text" hide:"create,update"`
	FingerprintSHA  string     `json:"fingerprint_sha256" gorm:"column:fingerprint_sha256;type:text;comment:SHA256指纹" widget:"name:SHA256指纹;type:text" hide:"create,update"`
	NotBefore       types.Time `json:"not_before" gorm:"column:not_before;type:datetime;comment:生效时间" widget:"name:生效时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	NotAfter        types.Time `json:"not_after" gorm:"column:not_after;type:datetime;comment:过期时间" widget:"name:过期时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DaysLeft        int        `json:"days_left" gorm:"column:days_left;comment:剩余天数" widget:"name:剩余天数;type:integer" hide:"create,update"`
	HostnameMatched bool       `json:"hostname_matched" gorm:"column:hostname_matched;comment:域名匹配" widget:"name:域名匹配;type:switch" hide:"create,update"`
	CertURL         string     `json:"cert_url" gorm:"column:cert_url;type:text;comment:ACME证书URL" widget:"name:ACME证书URL;type:text" hide:"create,update"`
	ImportedBy      string     `json:"imported_by" gorm:"column:imported_by;type:varchar(120);comment:入库人" widget:"name:入库人;type:user" hide:"create,update"`
	Remark          string     `json:"remark" gorm:"column:remark;type:text;comment:备注" widget:"name:备注;type:text_area" hide:"create,update"`
}

func (CertCFAsset) TableName() string {
	return "cf_cert_asset"
}

type CertCFAssetListReq struct {
	DomainName        string `json:"domain_name" form:"domain_name" widget:"name:域名;type:input"`
	Status            string `json:"status" form:"status" widget:"name:状态;type:select;options:正常,即将过期,已过期,失败,待部署;options_colors:67C23A,E6A23C,F56C6C,F56C6C,409EFF"`
	Source            string `json:"source" form:"source" widget:"name:来源;type:select;options:首次签发,手动续期,自动续期;options_colors:409EFF,E6A23C,67C23A"`
	query.PageSortReq `widget:"-"`
}

func CertCFAssetList(ctx *app.Context, resp response.Response) error {
	db, err := certManagerDB(ctx)
	if err != nil {
		return err
	}
	var req CertCFAssetListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	queryDB := db.Model(&CertCFAsset{})
	if strings.TrimSpace(req.DomainName) != "" {
		queryDB = queryDB.Where("domain_name LIKE ? OR sans LIKE ?", "%"+strings.TrimSpace(req.DomainName)+"%", "%"+strings.TrimSpace(req.DomainName)+"%")
	}
	if strings.TrimSpace(req.Status) != "" {
		queryDB = queryDB.Where("status = ?", strings.TrimSpace(req.Status))
	}
	if strings.TrimSpace(req.Source) != "" {
		queryDB = queryDB.Where("source = ?", strings.TrimSpace(req.Source))
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
	var rows []CertCFAsset
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&rows).Error; err != nil {
		return err
	}
	return resp.Table(response.TableResult{Items: rows, TotalCount: total, PageInfo: &req.PageSortReq}).Build()
}

var CertCFAssetListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "证书资产库",
		Desc:         "保存 Cloudflare DNS-01 自动签发得到的证书文件、私钥、证书链和 ZIP 包。证书只保存和下载，不自动部署。",
		Tags:         []string{"Cloudflare", "证书资产", "files"},
		Request:      &CertCFAssetListReq{},
		Response:     query.PaginatedTable[[]CertCFAsset]{},
		CreateTables: certManagerTables(),
	},
}

func createAssetFromIssuedCertificate(ctx *app.Context, db *gorm.DB, domain *CertCFDomain, reqRecord *CertCFRequest, issued *issuedCertificate) (*CertCFAsset, error) {
	cert, err := parseFirstCertificatePEM(issued.CertificatePEM)
	if err != nil {
		return nil, err
	}
	meta := certMetadataFromX509(cert, domain.Domain, domain.RenewBeforeDays)
	status := meta.Status
	if !meta.HostnameMatched {
		status = statusFailed
	}
	asset := &CertCFAsset{
		DomainID:        domain.ID,
		RequestID:       reqRecord.ID,
		ConfigID:        domain.ConfigID,
		DomainName:      domain.Domain,
		SANs:            strings.Join(issued.Names, ","),
		Source:          reqRecord.RequestType,
		Status:          status,
		CertificateFile: issued.CertificateFileRef,
		ChainFile:       issued.ChainFileRef,
		FullChainFile:   issued.FullChainFileRef,
		PrivateKeyFile:  issued.PrivateKeyFileRef,
		BundleFile:      issued.BundleFileRef,
		Issuer:          meta.Issuer,
		Subject:         meta.Subject,
		SerialNumber:    meta.SerialNumber,
		FingerprintSHA:  meta.FingerprintSHA,
		NotBefore:       types.Time(meta.NotBefore),
		NotAfter:        types.Time(meta.NotAfter),
		DaysLeft:        meta.DaysLeft,
		HostnameMatched: meta.HostnameMatched,
		CertURL:         issued.CertURL,
		ImportedBy:      ctx.GetRequestUser(),
		Remark:          "Cloudflare DNS-01 自动签发入库",
	}
	if asset.ImportedBy == "" {
		asset.ImportedBy = "system"
	}
	if err := db.Create(asset).Error; err != nil {
		return nil, fmt.Errorf("保存证书资产失败: %w", err)
	}
	updateDomainFromCertMeta(db, domain, meta, asset.CreatedAt.Time())
	logger.Infof(ctx, "[CertManager][Cloudflare] asset created asset_id=%d request_id=%d domain_id=%d domain=%s status=%s days_left=%d not_after=%s",
		asset.ID, reqRecord.ID, domain.ID, domain.Domain, asset.Status, asset.DaysLeft, asset.NotAfter.String())
	return asset, nil
}

func init() {
	packageContext.GET("assets.table", CertCFAssetList, CertCFAssetListTemplate)
}
