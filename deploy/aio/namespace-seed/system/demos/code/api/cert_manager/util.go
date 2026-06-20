package cert_manager

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

const (
	defaultCertWarnDays = 30

	statusUnchecked = "未检查"
	statusOK        = "正常"
	statusWarning   = "即将过期"
	statusExpired   = "已过期"
	statusFailed    = "检查失败"
	statusPending   = "待部署"

	scanTypePublicTLS = "公网TLS"
	scanTypeFileParse = "文件解析"
)

type certMetadata struct {
	Issuer          string
	Subject         string
	SANs            string
	SerialNumber    string
	FingerprintSHA  string
	NotBefore       time.Time
	NotAfter        time.Time
	DaysLeft        int
	Status          string
	HostnameMatched bool
}

func normalizeDomainName(value string) (string, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "", fmt.Errorf("域名不能为空")
	}
	if strings.Contains(value, "://") {
		parsed, err := url.Parse(value)
		if err != nil {
			return "", fmt.Errorf("域名格式错误")
		}
		value = parsed.Hostname()
	}
	value = strings.TrimSuffix(strings.TrimSpace(value), ".")
	value = strings.Trim(value, "/")
	if strings.Contains(value, "/") || strings.Contains(value, " ") {
		return "", fmt.Errorf("域名不能包含路径或空格")
	}
	if strings.HasPrefix(value, "*.") {
		if strings.Count(value, "*") > 1 {
			return "", fmt.Errorf("通配符域名只能使用 *.example.com 形式")
		}
		base := strings.TrimPrefix(value, "*.")
		if base == "" || !strings.Contains(base, ".") {
			return "", fmt.Errorf("通配符域名格式错误")
		}
		return value, nil
	}
	if !strings.Contains(value, ".") {
		return "", fmt.Errorf("请输入完整域名，例如 example.com")
	}
	return value, nil
}

func parseUploadedCertificate(ctx *app.Context, fileRef string) (*x509.Certificate, []string, error) {
	fs := ctx.GetFS()
	downloaded := fs.DownloadFiles(fileRef)
	if len(downloaded) == 0 {
		return nil, downloaded, fmt.Errorf("请上传证书文件")
	}
	cert, err := parseCertificateFile(downloaded[0])
	if err != nil {
		return nil, downloaded, err
	}
	return cert, downloaded, nil
}

func parseCertificateFile(path string) (*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取证书文件失败: %w", err)
	}
	var firstDER []byte
	rest := data
	for {
		block, remaining := pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			firstDER = block.Bytes
			break
		}
		rest = remaining
	}
	if len(firstDER) == 0 {
		firstDER = data
	}
	cert, err := x509.ParseCertificate(firstDER)
	if err != nil {
		return nil, fmt.Errorf("解析证书失败: %w", err)
	}
	return cert, nil
}

func fetchPublicTLSCertificate(domain string) (*x509.Certificate, error) {
	normalized, err := normalizeDomainName(domain)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(normalized, "*.") {
		return nil, fmt.Errorf("通配符域名无法直接做公网 TLS 探测，请上传证书文件或配置具体子域名")
	}
	dialer := &net.Dialer{Timeout: 8 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(normalized, "443"), &tls.Config{
		ServerName:         normalized,
		InsecureSkipVerify: true, // 只做资产巡检，手动验证证书元数据和域名匹配。
		MinVersion:         tls.VersionTLS12,
	})
	if err != nil {
		return nil, fmt.Errorf("连接公网 TLS 失败: %w", err)
	}
	defer conn.Close()
	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return nil, fmt.Errorf("未获取到服务端证书")
	}
	return state.PeerCertificates[0], nil
}

func certMetadataFromX509(cert *x509.Certificate, domain string, warnDays int) certMetadata {
	if warnDays <= 0 {
		warnDays = defaultCertWarnDays
	}
	now := time.Now()
	daysLeft := int(cert.NotAfter.Sub(now).Hours() / 24)
	status := statusOK
	if cert.NotAfter.Before(now) {
		status = statusExpired
	} else if daysLeft <= warnDays {
		status = statusWarning
	}
	sum := sha256.Sum256(cert.Raw)
	sans := strings.Join(cert.DNSNames, ",")
	if sans == "" && cert.Subject.CommonName != "" {
		sans = cert.Subject.CommonName
	}
	return certMetadata{
		Issuer:          cert.Issuer.String(),
		Subject:         cert.Subject.String(),
		SANs:            sans,
		SerialNumber:    cert.SerialNumber.String(),
		FingerprintSHA:  strings.ToUpper(hex.EncodeToString(sum[:])),
		NotBefore:       cert.NotBefore,
		NotAfter:        cert.NotAfter,
		DaysLeft:        daysLeft,
		Status:          status,
		HostnameMatched: verifyCertificateHostname(cert, domain),
	}
}

func verifyCertificateHostname(cert *x509.Certificate, domain string) bool {
	normalized, err := normalizeDomainName(domain)
	if err != nil {
		return false
	}
	if strings.HasPrefix(normalized, "*.") {
		for _, dnsName := range cert.DNSNames {
			if strings.EqualFold(dnsName, normalized) {
				return true
			}
		}
		return strings.EqualFold(cert.Subject.CommonName, normalized)
	}
	return cert.VerifyHostname(normalized) == nil
}

func updateDomainFromCertMeta(db *gorm.DB, domain *CertManagedDomain, meta certMetadata, checkedAt time.Time) {
	_ = db.Model(&CertManagedDomain{}).Where("id = ?", domain.ID).Updates(map[string]interface{}{
		"current_status":    meta.Status,
		"current_issuer":    meta.Issuer,
		"current_not_after": types.Time(meta.NotAfter),
		"current_days_left": meta.DaysLeft,
		"last_checked_at":   types.Time(checkedAt),
	}).Error
}

func certStatusAndError(meta certMetadata) (string, string) {
	if !meta.HostnameMatched {
		return statusFailed, "证书 SAN/CN 与域名不匹配"
	}
	return meta.Status, ""
}

func updateDomainFromParsedCert(db *gorm.DB, domain *CertManagedDomain, meta certMetadata, checkedAt time.Time) {
	status, _ := certStatusAndError(meta)
	if status == statusFailed {
		updateDomainCheckFailure(db, domain, checkedAt)
		return
	}
	updateDomainFromCertMeta(db, domain, meta, checkedAt)
}

func updateDomainCheckFailure(db *gorm.DB, domain *CertManagedDomain, checkedAt time.Time) {
	_ = db.Model(&CertManagedDomain{}).Where("id = ?", domain.ID).Updates(map[string]interface{}{
		"current_status":  statusFailed,
		"last_checked_at": types.Time(checkedAt),
	}).Error
}

func joinCertificateNotifyUsers(domain CertManagedDomain) string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, value := range []string{domain.Owner, domain.NotifyUsers} {
		for _, part := range strings.Split(value, ",") {
			user := strings.TrimSpace(part)
			if user == "" {
				continue
			}
			if _, ok := seen[user]; ok {
				continue
			}
			seen[user] = struct{}{}
			out = append(out, user)
		}
	}
	return strings.Join(out, ",")
}

func shouldNotifyDomain(domain CertManagedDomain, now time.Time) bool {
	return domain.LastNotifiedAt.IsZero() || now.Sub(domain.LastNotifiedAt.Time()) >= 23*time.Hour
}

func updateDomainNotifiedAt(db *gorm.DB, domainID int, notifiedAt time.Time) {
	_ = db.Model(&CertManagedDomain{}).Where("id = ?", domainID).Update("last_notified_at", types.Time(notifiedAt)).Error
}

func asInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return 0
	}
}
