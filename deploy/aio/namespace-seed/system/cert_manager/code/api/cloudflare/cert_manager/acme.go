package cert_manager

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"golang.org/x/crypto/acme"
	"gorm.io/gorm"
)

type certificateIssueOptions struct {
	WaitSeconds         int
	PollIntervalSeconds int
	CleanupChallenge    bool
}

type issuedCertificate struct {
	Names              []string
	CertificatePEM     string
	ChainPEM           string
	FullChainPEM       string
	PrivateKeyPEM      string
	CertURL            string
	CertificateFileRef string
	ChainFileRef       string
	FullChainFileRef   string
	PrivateKeyFileRef  string
	BundleFileRef      string
}

type acmeDNSChallenge struct {
	AuthzURI   string
	Identifier string
	Name       string
	Value      string
	Challenge  *acme.Challenge
	Zone       *cloudflareZone
	Record     *cloudflareDNSRecord
}

// issueCertificateForRequest runs the ACME DNS-01 state machine. Logs here use
// request/domain identifiers only; credentials, private keys and TXT values stay out.
func issueCertificateForRequest(ctx *app.Context, db *gorm.DB, domain *CertCFDomain, reqRecord *CertCFRequest, opts certificateIssueOptions) (*issuedCertificate, error) {
	if opts.WaitSeconds <= 0 {
		opts.WaitSeconds = defaultDNSWaitSeconds
	}
	if opts.PollIntervalSeconds <= 0 {
		opts.PollIntervalSeconds = defaultDNSPollSeconds
	}
	startedAt := time.Now()
	logger.Infof(ctx, "[CertManager][Cloudflare] issue start request_id=%d domain_id=%d domain=%s config_id=%d wait=%ds poll=%ds cleanup=%v",
		reqRecord.ID, domain.ID, domain.Domain, domain.ConfigID, opts.WaitSeconds, opts.PollIntervalSeconds, opts.CleanupChallenge)
	cfg, token, err := loadCertCFConfig(ctx, domain.ConfigID)
	if err != nil {
		return nil, err
	}
	logger.Infof(ctx, "[CertManager][Cloudflare] config loaded request_id=%d config_id=%d config=%s env=%s",
		reqRecord.ID, cfg.ID, cfg.Name, directoryLabel(cfg.DirectoryURL))
	client, _, err := newACMEClient(cfg)
	if err != nil {
		return nil, err
	}
	if err := ensureACMEAccount(ctx, client, cfg); err != nil {
		return nil, err
	}

	names, err := normalizeDomainNames(domain.Domain, domain.SANs)
	if err != nil {
		return nil, err
	}
	logger.Infof(ctx, "[CertManager][Cloudflare] creating ACME order request_id=%d names=%s", reqRecord.ID, strings.Join(names, ","))
	updateRequestStatus(db, reqRecord.ID, requestStatusRunning, "创建 ACME Order", nil)
	order, err := client.AuthorizeOrder(ctx, acme.DomainIDs(names...))
	if err != nil {
		return nil, fmt.Errorf("创建 ACME Order 失败: %w", err)
	}
	logger.Infof(ctx, "[CertManager][Cloudflare] ACME order created request_id=%d status=%s authz_count=%d",
		reqRecord.ID, order.Status, len(order.AuthzURLs))
	if order.Status == acme.StatusValid && order.CertURL != "" {
		return nil, fmt.Errorf("ACME Order 已有效但当前流程未保存证书 URL，请重新提交")
	}

	challenges, err := prepareDNSChallenges(ctx, db, client, token, reqRecord.ID, order)
	if err != nil {
		return nil, err
	}
	logger.Infof(ctx, "[CertManager][Cloudflare] DNS challenges prepared request_id=%d count=%d", reqRecord.ID, len(challenges))
	if opts.CleanupChallenge {
		defer cleanupDNSChallenges(ctx, token, challenges)
	}
	if len(challenges) > 0 {
		updateRequestStatus(db, reqRecord.ID, requestStatusWaitDNS, "等待 Cloudflare TXT 记录生效", map[string]interface{}{
			"challenge_name":  challenges[0].Name,
			"challenge_value": challenges[0].Value,
		})
		logger.Infof(ctx, "[CertManager][Cloudflare] waiting DNS propagation request_id=%d count=%d timeout=%ds interval=%ds first_name=%s",
			reqRecord.ID, len(challenges), opts.WaitSeconds, opts.PollIntervalSeconds, challenges[0].Name)
		if err := waitForDNSChallenges(ctx, challenges, time.Duration(opts.WaitSeconds)*time.Second, time.Duration(opts.PollIntervalSeconds)*time.Second); err != nil {
			return nil, err
		}
		logger.Infof(ctx, "[CertManager][Cloudflare] DNS propagation visible request_id=%d count=%d", reqRecord.ID, len(challenges))
		updateRequestStatus(db, reqRecord.ID, requestStatusVerify, "DNS 已生效，提交 ACME 验证", nil)
		if err := acceptDNSChallenges(ctx, client, challenges); err != nil {
			return nil, err
		}
		logger.Infof(ctx, "[CertManager][Cloudflare] ACME challenges accepted request_id=%d count=%d", reqRecord.ID, len(challenges))
	}

	order, err = client.WaitOrder(ctx, order.URI)
	if err != nil {
		return nil, fmt.Errorf("等待 ACME Order 进入可签发状态失败: %w", err)
	}
	logger.Infof(ctx, "[CertManager][Cloudflare] ACME order ready request_id=%d status=%s", reqRecord.ID, order.Status)
	if order.Status != acme.StatusReady && order.Status != acme.StatusValid {
		return nil, fmt.Errorf("ACME Order 状态不是 ready/valid: %s", order.Status)
	}
	certKey, certKeyPEM, err := generateCertificateKeyPEM(domain.KeyAlgorithm)
	if err != nil {
		return nil, err
	}
	csrDER, err := createCSR(names, certKey)
	if err != nil {
		return nil, err
	}
	logger.Infof(ctx, "[CertManager][Cloudflare] finalizing certificate request_id=%d names=%s", reqRecord.ID, strings.Join(names, ","))
	derChain, certURL, err := client.CreateOrderCert(ctx, order.FinalizeURL, csrDER, true)
	if err != nil {
		return nil, fmt.Errorf("签发证书失败: %w", err)
	}
	if len(derChain) == 0 {
		return nil, fmt.Errorf("ACME 未返回证书链")
	}
	issued := buildIssuedCertificate(names, certKeyPEM, derChain, certURL)
	if err := writeIssuedCertificateFiles(ctx, domain.Domain, issued); err != nil {
		return nil, err
	}
	logger.Infof(ctx, "[CertManager][Cloudflare] issue finished request_id=%d domain=%s cert_url_present=%v elapsed=%s",
		reqRecord.ID, domain.Domain, certURL != "", time.Since(startedAt).Round(time.Millisecond))
	return issued, nil
}

// prepareDNSChallenges creates one provider TXT record for each pending ACME authorization.
func prepareDNSChallenges(ctx *app.Context, db *gorm.DB, client *acme.Client, token string, requestID int, order *acme.Order) ([]*acmeDNSChallenge, error) {
	challenges := make([]*acmeDNSChallenge, 0)
	for _, authzURL := range order.AuthzURLs {
		authz, err := client.GetAuthorization(ctx, authzURL)
		if err != nil {
			return nil, fmt.Errorf("获取 ACME Authorization 失败: %w", err)
		}
		logger.Infof(ctx, "[CertManager][Cloudflare] authorization loaded request_id=%d identifier=%s status=%s",
			requestID, authz.Identifier.Value, authz.Status)
		if authz.Status == acme.StatusValid {
			logger.Infof(ctx, "[CertManager][Cloudflare] authorization already valid request_id=%d identifier=%s",
				requestID, authz.Identifier.Value)
			continue
		}
		chal := findDNS01Challenge(authz.Challenges)
		if chal == nil {
			return nil, fmt.Errorf("域名 %s 没有可用的 dns-01 challenge", authz.Identifier.Value)
		}
		txtValue, err := client.DNS01ChallengeRecord(chal.Token)
		if err != nil {
			return nil, fmt.Errorf("生成 DNS-01 TXT 值失败: %w", err)
		}
		txtName := "_acme-challenge." + strings.TrimPrefix(authz.Identifier.Value, "*.")
		logger.Infof(ctx, "[CertManager][Cloudflare] preparing TXT record request_id=%d identifier=%s name=%s",
			requestID, authz.Identifier.Value, txtName)
		zone, err := findCloudflareZone(token, authz.Identifier.Value)
		if err != nil {
			return nil, err
		}
		logger.Infof(ctx, "[CertManager][Cloudflare] zone matched request_id=%d identifier=%s zone_id=%s zone_name=%s",
			requestID, authz.Identifier.Value, zone.ID, zone.Name)
		record, err := createCloudflareTXTRecord(token, zone.ID, txtName, txtValue)
		if err != nil {
			return nil, fmt.Errorf("创建 Cloudflare TXT 记录失败: %w", err)
		}
		logger.Infof(ctx, "[CertManager][Cloudflare] TXT record created request_id=%d zone_id=%s record_id=%s name=%s",
			requestID, zone.ID, record.ID, txtName)
		challenges = append(challenges, &acmeDNSChallenge{
			AuthzURI:   authz.URI,
			Identifier: authz.Identifier.Value,
			Name:       txtName,
			Value:      txtValue,
			Challenge:  chal,
			Zone:       zone,
			Record:     record,
		})
		appendChallengeRecord(db, requestID, zone, record, txtName, txtValue)
	}
	return challenges, nil
}

func findDNS01Challenge(challenges []*acme.Challenge) *acme.Challenge {
	for _, challenge := range challenges {
		if challenge != nil && challenge.Type == "dns-01" {
			return challenge
		}
	}
	return nil
}

func waitForDNSChallenges(ctx context.Context, challenges []*acmeDNSChallenge, timeout time.Duration, interval time.Duration) error {
	if timeout <= 0 {
		timeout = time.Duration(defaultDNSWaitSeconds) * time.Second
	}
	if interval <= 0 {
		interval = time.Duration(defaultDNSPollSeconds) * time.Second
	}
	deadline := time.Now().Add(timeout)
	for {
		allVisible := true
		var lastErr error
		for _, challenge := range challenges {
			txts, err := lookupTXTWithContext(ctx, challenge.Name)
			if err != nil {
				lastErr = err
				allVisible = false
				continue
			}
			if !txtRecordVisible(txts, challenge.Value) {
				allVisible = false
			}
		}
		if allVisible {
			return nil
		}
		if time.Now().After(deadline) {
			if lastErr != nil {
				return fmt.Errorf("等待 DNS TXT 生效超时，最近一次查询错误: %w", lastErr)
			}
			return fmt.Errorf("等待 DNS TXT 生效超时")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

func lookupTXTWithContext(ctx context.Context, name string) ([]string, error) {
	lookupCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	return net.DefaultResolver.LookupTXT(lookupCtx, strings.TrimSuffix(name, "."))
}

func acceptDNSChallenges(ctx context.Context, client *acme.Client, challenges []*acmeDNSChallenge) error {
	for _, challenge := range challenges {
		if _, err := client.Accept(ctx, challenge.Challenge); err != nil {
			return fmt.Errorf("提交 ACME DNS-01 验证失败 %s: %w", challenge.Name, err)
		}
		authz, err := client.WaitAuthorization(ctx, challenge.AuthzURI)
		if err != nil {
			return fmt.Errorf("等待 ACME 验证失败 %s: %w", challenge.Identifier, err)
		}
		if authz.Status != acme.StatusValid {
			return fmt.Errorf("ACME Authorization 状态不是 valid: %s", authz.Status)
		}
	}
	return nil
}

func cleanupDNSChallenges(ctx *app.Context, token string, challenges []*acmeDNSChallenge) {
	for _, challenge := range challenges {
		if challenge == nil || challenge.Zone == nil || challenge.Record == nil {
			continue
		}
		if err := deleteCloudflareDNSRecord(token, challenge.Zone.ID, challenge.Record.ID); err != nil {
			logger.Warnf(ctx, "delete cloudflare dns record failed zone=%s record=%s err=%v", challenge.Zone.ID, challenge.Record.ID, err)
			continue
		}
		logger.Infof(ctx, "[CertManager][Cloudflare] TXT record cleaned zone_id=%s record_id=%s name=%s",
			challenge.Zone.ID, challenge.Record.ID, challenge.Name)
	}
}

func newACMEClient(cfg *CertCFConfig) (*acme.Client, crypto.Signer, error) {
	keyPEM, err := decryptSecret(cfg.AccountKeyCipher)
	if err != nil {
		return nil, nil, err
	}
	key, err := parseECPrivateKeyPEM(keyPEM)
	if err != nil {
		return nil, nil, err
	}
	directoryURL := strings.TrimSpace(cfg.DirectoryURL)
	if directoryURL == "" {
		directoryURL = acme.LetsEncryptURL
	}
	client := &acme.Client{
		Key:          key,
		DirectoryURL: directoryURL,
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
		UserAgent:    "KageOS Cloudflare Cert Manager",
	}
	return client, key, nil
}

func ensureACMEAccount(ctx context.Context, client *acme.Client, cfg *CertCFConfig) error {
	account := &acme.Account{Contact: []string{"mailto:" + strings.TrimSpace(cfg.Email)}}
	if _, err := client.Register(ctx, account, acme.AcceptTOS); err != nil && err != acme.ErrAccountAlreadyExists {
		return fmt.Errorf("注册/读取 ACME 账号失败: %w", err)
	}
	return nil
}

func generateAccountKeyPEM() (string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", fmt.Errorf("生成 ACME Account Key 失败: %w", err)
	}
	return encodeECPrivateKeyPEM(key)
}

func generateCertificateKeyPEM(algorithm string) (crypto.Signer, string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, "", fmt.Errorf("生成证书私钥失败: %w", err)
	}
	keyPEM, err := encodeECPrivateKeyPEM(key)
	if err != nil {
		return nil, "", err
	}
	return key, keyPEM, nil
}

func encodeECPrivateKeyPEM(key *ecdsa.PrivateKey) (string, error) {
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", fmt.Errorf("编码 EC 私钥失败: %w", err)
	}
	block := &pem.Block{Type: "EC PRIVATE KEY", Bytes: der}
	return string(pem.EncodeToMemory(block)), nil
}

func parseECPrivateKeyPEM(value string) (crypto.Signer, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(value)))
	if block == nil {
		return nil, fmt.Errorf("ACME Account Key PEM 格式错误")
	}
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析 ACME Account Key 失败: %w", err)
	}
	return key, nil
}

func createCSR(names []string, key crypto.Signer) ([]byte, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("证书域名不能为空")
	}
	template := &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: names[0]},
		DNSNames: names,
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, key)
	if err != nil {
		return nil, fmt.Errorf("生成 CSR 失败: %w", err)
	}
	return csrDER, nil
}

func buildIssuedCertificate(names []string, keyPEM string, derChain [][]byte, certURL string) *issuedCertificate {
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derChain[0]})
	chainBuilder := strings.Builder{}
	fullChainBuilder := strings.Builder{}
	for i, der := range derChain {
		pemText := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
		fullChainBuilder.WriteString(pemText)
		if i > 0 {
			chainBuilder.WriteString(pemText)
		}
	}
	return &issuedCertificate{
		Names:          names,
		CertificatePEM: string(certPEM),
		ChainPEM:       chainBuilder.String(),
		FullChainPEM:   fullChainBuilder.String(),
		PrivateKeyPEM:  keyPEM,
		CertURL:        certURL,
	}
}

func writeIssuedCertificateFiles(ctx *app.Context, domain string, issued *issuedCertificate) error {
	outputDir := filepath.Join(ctx.GetFS().GetTraceOutputDir(), "cloudflare-cert")
	base := sanitizeFileBase(domain)
	certPath := filepath.Join(outputDir, base+".cert.pem")
	chainPath := filepath.Join(outputDir, base+".chain.pem")
	fullChainPath := filepath.Join(outputDir, base+".fullchain.pem")
	keyPath := filepath.Join(outputDir, base+".private.key")
	zipPath := filepath.Join(outputDir, base+".bundle.zip")
	if err := writeTextFile(certPath, issued.CertificatePEM, 0644); err != nil {
		return err
	}
	if err := writeTextFile(chainPath, issued.ChainPEM, 0644); err != nil {
		return err
	}
	if err := writeTextFile(fullChainPath, issued.FullChainPEM, 0644); err != nil {
		return err
	}
	if err := writeTextFile(keyPath, issued.PrivateKeyPEM, 0600); err != nil {
		return err
	}
	if err := createZipBundle(zipPath, map[string]string{
		"cert.pem":      certPath,
		"chain.pem":     chainPath,
		"fullchain.pem": fullChainPath,
		"private.key":   keyPath,
	}); err != nil {
		return err
	}
	issued.CertificateFileRef = ctx.GetFS().ResponseFiles([]string{certPath})
	issued.ChainFileRef = ctx.GetFS().ResponseFiles([]string{chainPath})
	issued.FullChainFileRef = ctx.GetFS().ResponseFiles([]string{fullChainPath})
	issued.PrivateKeyFileRef = ctx.GetFS().ResponseFiles([]string{keyPath})
	issued.BundleFileRef = ctx.GetFS().ResponseFiles([]string{zipPath})
	if issued.CertificateFileRef == "" || issued.PrivateKeyFileRef == "" || issued.FullChainFileRef == "" {
		return fmt.Errorf("证书文件上传失败")
	}
	_ = os.Remove(certPath)
	_ = os.Remove(chainPath)
	_ = os.Remove(fullChainPath)
	_ = os.Remove(keyPath)
	_ = os.Remove(zipPath)
	return nil
}
