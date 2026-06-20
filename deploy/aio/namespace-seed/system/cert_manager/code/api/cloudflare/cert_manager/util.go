package cert_manager

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
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
		base := strings.TrimPrefix(value, "*.")
		if base == "" || strings.Contains(base, "*") || !strings.Contains(base, ".") {
			return "", fmt.Errorf("通配符域名只能使用 *.example.com 形式")
		}
		return value, nil
	}
	if strings.Contains(value, "*") {
		return "", fmt.Errorf("通配符域名只能使用 *.example.com 形式")
	}
	if !strings.Contains(value, ".") {
		return "", fmt.Errorf("请输入完整域名，例如 example.com")
	}
	return value, nil
}

func normalizeDomainNames(primary string, sans string) ([]string, error) {
	first, err := normalizeDomainName(primary)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{first: {}}
	out := []string{first}
	for _, part := range splitList(sans) {
		name, err := normalizeDomainName(part)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out, nil
}

func splitList(value string) []string {
	value = strings.NewReplacer("，", ",", "\n", ",", "|", ",", ";", ",").Replace(value)
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func joinCertificateNotifyUsers(domain CertCFDomain) string {
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

func certMetadataFromX509(cert *x509.Certificate, domain string, warnDays int) certMetadata {
	if warnDays <= 0 {
		warnDays = defaultRenewBeforeDays
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

func updateDomainFromCertMeta(db *gorm.DB, domain *CertCFDomain, meta certMetadata, checkedAt time.Time) {
	_ = db.Model(&CertCFDomain{}).Where("id = ?", domain.ID).Updates(map[string]interface{}{
		"current_status":    meta.Status,
		"current_issuer":    meta.Issuer,
		"current_not_after": types.Time(meta.NotAfter),
		"current_days_left": meta.DaysLeft,
		"last_checked_at":   types.Time(checkedAt),
		"last_renewed_at":   types.Time(checkedAt),
	}).Error
}

func updateDomainFailure(db *gorm.DB, domainID int, checkedAt time.Time, message string) {
	_ = db.Model(&CertCFDomain{}).Where("id = ?", domainID).Updates(map[string]interface{}{
		"current_status":  statusFailed,
		"last_checked_at": types.Time(checkedAt),
		"last_error":      message,
	}).Error
}

func parseFirstCertificatePEM(pemText string) (*x509.Certificate, error) {
	rest := []byte(pemText)
	for {
		block, remaining := pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			return x509.ParseCertificate(block.Bytes)
		}
		rest = remaining
	}
	return nil, fmt.Errorf("未找到证书 PEM")
}

func sanitizeFileBase(domain string) string {
	domain = strings.TrimPrefix(domain, "*.")
	re := regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	base := re.ReplaceAllString(domain, "_")
	if base == "" {
		return "certificate"
	}
	return base
}

func writeTextFile(path string, value string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(value), mode)
}

func createZipBundle(zipPath string, files map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(zipPath), 0755); err != nil {
		return err
	}
	out, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer out.Close()
	zw := zip.NewWriter(out)
	defer zw.Close()
	for name, path := range files {
		if err := addFileToZip(zw, name, path); err != nil {
			return err
		}
	}
	return nil
}

func addFileToZip(zw *zip.Writer, name string, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, file)
	return err
}

func txtRecordVisible(records []string, expected string) bool {
	expected = strings.Trim(expected, `"`)
	for _, record := range records {
		if strings.Trim(record, `"`) == expected {
			return true
		}
	}
	return false
}

func lookupTXT(name string) ([]string, error) {
	resolver := net.DefaultResolver
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return resolver.LookupTXT(ctx, strings.TrimSuffix(name, "."))
}
