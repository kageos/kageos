package service

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/core/hr-server/repository"
	"github.com/kageos/kageos/dto"
	appconfig "github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/emailx"
	"github.com/kageos/kageos/pkg/systembackup"
)

const (
	RegistrationModeAdminOnly = "admin_only"
	RegistrationModeEmailCode = "email_code"
	RegistrationModeDebugCode = "debug_code"

	EmailModeSMTP = "smtp"
	EmailModeLog  = "log"

	settingRegistrationMode         = "auth.registration_mode"
	settingEmailMode                = "email.mode"
	settingSMTPHost                 = "email.smtp.host"
	settingSMTPPort                 = "email.smtp.port"
	settingSMTPUsername             = "email.smtp.username"
	settingSMTPPassword             = "email.smtp.password"
	settingSMTPFrom                 = "email.smtp.from"
	settingSMTPFromName             = "email.smtp.from_name"
	settingLoginAnnouncementEnabled = "auth.login_announcement.enabled"
	settingLoginAnnouncementContent = "auth.login_announcement.content"
)

const (
	defaultTLSCertFile = "/app/tls/fullchain.pem"
	defaultTLSKeyFile  = "/app/tls/privkey.pem"
)

type SystemSettingsService struct {
	repo *repository.SystemSettingRepository
	cfg  *appconfig.HRServerConfig
}

func NewSystemSettingsService(repo *repository.SystemSettingRepository) *SystemSettingsService {
	return &SystemSettingsService{
		repo: repo,
		cfg:  appconfig.GetHRServerConfig(),
	}
}

func (s *SystemSettingsService) GetSettings() (*dto.SystemSettingsResp, error) {
	values, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}
	email := s.emailSettingsFrom(values, false)
	return &dto.SystemSettingsResp{
		RegistrationMode: s.registrationModeFrom(values, email.Mode),
		Email:            email,
	}, nil
}

func (s *SystemSettingsService) GetLoginAnnouncement() (*dto.LoginAnnouncement, error) {
	values, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}
	return &dto.LoginAnnouncement{
		Enabled:  values[settingLoginAnnouncementEnabled] == "true",
		Markdown: values[settingLoginAnnouncementContent],
	}, nil
}

func (s *SystemSettingsService) UpdateLoginAnnouncement(req dto.UpdateLoginAnnouncementReq, updatedBy string) (*dto.LoginAnnouncement, error) {
	markdown := strings.TrimSpace(req.Markdown)
	if len([]rune(markdown)) > 10000 {
		return nil, fmt.Errorf("announcement markdown must not exceed 10000 characters")
	}
	if req.Enabled && markdown == "" {
		return nil, fmt.Errorf("announcement markdown is required when enabled")
	}
	if err := s.repo.UpsertMany(map[string]string{
		settingLoginAnnouncementEnabled: strconv.FormatBool(req.Enabled),
		settingLoginAnnouncementContent: markdown,
	}, updatedBy); err != nil {
		return nil, err
	}
	return s.GetLoginAnnouncement()
}

func (s *SystemSettingsService) UpdateSettings(req dto.UpdateSystemSettingsReq, updatedBy string) (*dto.SystemSettingsResp, error) {
	mode := normalizeRegistrationMode(req.RegistrationMode)
	emailMode := normalizeEmailMode(req.Email.Mode)
	if mode == RegistrationModeEmailCode && emailMode != EmailModeSMTP {
		return nil, fmt.Errorf("email verification registration requires email.mode=smtp")
	}

	current, err := s.GetRuntimeEmailConfig()
	if err != nil {
		return nil, err
	}
	password := strings.TrimSpace(req.Email.Password)
	if password == "" {
		password = current.SMTP.Password
	}

	values := map[string]string{
		settingRegistrationMode: mode,
		settingEmailMode:        emailMode,
		settingSMTPHost:         strings.TrimSpace(req.Email.Host),
		settingSMTPPort:         strconv.Itoa(req.Email.Port),
		settingSMTPUsername:     strings.TrimSpace(req.Email.Username),
		settingSMTPPassword:     password,
		settingSMTPFrom:         strings.TrimSpace(req.Email.From),
		settingSMTPFromName:     strings.TrimSpace(req.Email.FromName),
	}
	if values[settingSMTPPort] == "0" {
		values[settingSMTPPort] = "587"
	}
	if values[settingSMTPFromName] == "" {
		values[settingSMTPFromName] = "kageos"
	}
	if mode == RegistrationModeEmailCode {
		if err := validateSMTPConfig(appconfig.EmailSMTPConfig{
			Host:     values[settingSMTPHost],
			Port:     firstNonZeroInt(values[settingSMTPPort]),
			Username: values[settingSMTPUsername],
			Password: values[settingSMTPPassword],
			From:     values[settingSMTPFrom],
			FromName: values[settingSMTPFromName],
		}); err != nil {
			return nil, err
		}
	}
	if err := s.repo.UpsertMany(values, updatedBy); err != nil {
		return nil, err
	}
	return s.GetSettings()
}

func (s *SystemSettingsService) TestEmail(to string) error {
	cfg, err := s.GetRuntimeEmailConfig()
	if err != nil {
		return err
	}
	if normalizeEmailMode(cfg.Mode) != EmailModeSMTP {
		return fmt.Errorf("email.mode must be smtp before sending a test email")
	}
	if err := validateSMTPConfig(cfg.SMTP); err != nil {
		return err
	}
	return emailx.NewSender(cfg.SMTP).SendHTML(to, "kageos test email", "<p>kageos email service is configured correctly.</p>")
}

func (s *SystemSettingsService) GetBackupOverview() (*dto.SystemBackupOverview, error) {
	cfg, err := systembackup.LoadConfig(systembackup.StateDir(), backupEncryptionSecret())
	if err != nil {
		return nil, err
	}
	state, err := systembackup.LoadState(systembackup.StateDir())
	if err != nil {
		return nil, err
	}
	return backupOverview(systembackup.PublicConfig(cfg), state), nil
}

func (s *SystemSettingsService) UpdateBackupConfig(req dto.SystemBackupConfig) (*dto.SystemBackupOverview, error) {
	current, err := systembackup.LoadConfig(systembackup.StateDir(), backupEncryptionSecret())
	if err != nil {
		return nil, err
	}
	cfg := backupConfigFromDTO(req)
	if strings.TrimSpace(cfg.SecretAccessKey) == "" {
		cfg.SecretAccessKey = current.SecretAccessKey
	}
	cfg.RunNowRequestedAt = current.RunNowRequestedAt
	cfg.LastRunNowProcessedAt = current.LastRunNowProcessedAt
	if err := systembackup.SaveConfig(systembackup.StateDir(), backupEncryptionSecret(), cfg); err != nil {
		return nil, err
	}
	return s.GetBackupOverview()
}

func (s *SystemSettingsService) TestBackupS3(ctx context.Context, req dto.SystemBackupConfig) error {
	current, err := systembackup.LoadConfig(systembackup.StateDir(), backupEncryptionSecret())
	if err != nil {
		return err
	}
	cfg := backupConfigFromDTO(req)
	if strings.TrimSpace(cfg.SecretAccessKey) == "" {
		cfg.SecretAccessKey = current.SecretAccessKey
	}
	testCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return systembackup.TestS3(testCtx, cfg)
}

func (s *SystemSettingsService) RequestBackupRunNow() (*dto.SystemBackupOverview, error) {
	cfg, err := systembackup.LoadConfig(systembackup.StateDir(), backupEncryptionSecret())
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, fmt.Errorf("请先保存并启用自动备份")
	}
	cfg.RunNowRequestedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := systembackup.SaveConfig(systembackup.StateDir(), backupEncryptionSecret(), cfg); err != nil {
		return nil, err
	}
	return s.GetBackupOverview()
}

func backupEncryptionSecret() string {
	return appconfig.GetGlobalSharedConfig().JWT.Secret
}

func backupConfigFromDTO(req dto.SystemBackupConfig) systembackup.Config {
	return systembackup.Config{
		Enabled: req.Enabled, ScheduleTime: req.ScheduleTime, Endpoint: req.Endpoint, Region: req.Region,
		Bucket: req.Bucket, Prefix: req.Prefix, AccessKeyID: req.AccessKeyID, SecretAccessKey: req.SecretAccessKey,
		SecretAccessKeySet: req.SecretAccessKeySet, UseSSL: req.UseSSL, ForcePathStyle: req.ForcePathStyle, KeepLocal: req.KeepLocal,
		RetentionDays: req.RetentionDays,
	}
}

func backupOverview(cfg systembackup.Config, state systembackup.State) *dto.SystemBackupOverview {
	records := make([]dto.SystemBackupRecord, 0, len(state.Records))
	for _, item := range state.Records {
		records = append(records, dto.SystemBackupRecord{
			ID: item.ID, TriggeredBy: item.TriggeredBy, Status: item.Status, StartedAt: item.StartedAt, FinishedAt: item.FinishedAt,
			ArchiveName: item.ArchiveName, SizeBytes: item.SizeBytes, SHA256: item.SHA256, Bucket: item.Bucket,
			ObjectKey: item.ObjectKey, ETag: item.ETag, ErrorMessage: item.ErrorMessage,
		})
	}
	lastSeen, _ := time.Parse(time.RFC3339, state.AgentLastSeenAt)
	return &dto.SystemBackupOverview{
		Config: dto.SystemBackupConfig{
			Enabled: cfg.Enabled, ScheduleTime: cfg.ScheduleTime, Endpoint: cfg.Endpoint, Region: cfg.Region, Bucket: cfg.Bucket,
			Prefix: cfg.Prefix, AccessKeyID: cfg.AccessKeyID, SecretAccessKeySet: cfg.SecretAccessKeySet,
			UseSSL: cfg.UseSSL, ForcePathStyle: cfg.ForcePathStyle, KeepLocal: cfg.KeepLocal,
			RetentionDays: cfg.RetentionDays,
		},
		AgentAvailable:  state.Running || (!lastSeen.IsZero() && time.Since(lastSeen) < 15*time.Minute),
		AgentLastSeenAt: state.AgentLastSeenAt, Running: state.Running, Records: records,
	}
}

func (s *SystemSettingsService) GetTLSSettings() (*dto.TLSSettingsResp, error) {
	certFile := tlsCertFile()
	keyFile := tlsKeyFile()
	resp := &dto.TLSSettingsResp{
		Mode:            tlsMode(),
		BaseURL:         strings.TrimSpace(os.Getenv("CANONICAL_BASE_URL")),
		CertFile:        certFile,
		KeyFile:         keyFile,
		CertExists:      fileExists(certFile),
		KeyExists:       fileExists(keyFile),
		Writable:        dirWritable(filepath.Dir(certFile)) && dirWritable(filepath.Dir(keyFile)),
		ReloadSupported: tlsReloadSupported(),
	}
	resp.Ready = resp.CertExists && resp.KeyExists
	if resp.CertExists {
		certPEM, err := os.ReadFile(certFile)
		if err != nil {
			resp.Message = "读取证书失败: " + err.Error()
		} else if cert, err := parseCertificatePEM(string(certPEM)); err != nil {
			resp.Message = "解析证书失败: " + err.Error()
		} else {
			resp.Certificate = certificateInfo(cert)
		}
	}
	return resp, nil
}

func (s *SystemSettingsService) UpdateTLSCertificate(req dto.UpdateTLSCertificateReq, _ string) (*dto.TLSSettingsResp, error) {
	certPEM := normalizePEM(req.CertificatePEM)
	keyPEM := normalizePEM(req.PrivateKeyPEM)
	cert, err := parseCertificatePEM(certPEM)
	if err != nil {
		return nil, err
	}
	keyPublic, err := parsePrivateKeyPublicKey(keyPEM)
	if err != nil {
		return nil, err
	}
	if !publicKeysEqual(cert.PublicKey, keyPublic) {
		return nil, fmt.Errorf("certificate and private key do not match")
	}

	certFile := tlsCertFile()
	keyFile := tlsKeyFile()
	oldCert, oldCertErr := os.ReadFile(certFile)
	oldKey, oldKeyErr := os.ReadFile(keyFile)
	if err := writeTLSPEMFiles(certFile, keyFile, []byte(certPEM), []byte(keyPEM)); err != nil {
		return nil, err
	}
	if req.Reload {
		if err := s.ReloadTLS(); err != nil {
			_ = restoreTLSPEMFiles(certFile, keyFile, oldCert, oldCertErr == nil, oldKey, oldKeyErr == nil)
			return nil, err
		}
	}

	return s.GetTLSSettings()
}

func (s *SystemSettingsService) ReloadTLS() error {
	mode := tlsMode()
	if mode != "https" && mode != "redirect" {
		return fmt.Errorf("TLS reload requires TLS_MODE=https or redirect, current mode is %q", mode)
	}
	if !fileExists(tlsCertFile()) || !fileExists(tlsKeyFile()) {
		return fmt.Errorf("TLS certificate and private key must both exist before reload")
	}
	if err := runNginxCommand("-t"); err != nil {
		return err
	}
	return runNginxCommand("-s", "reload")
}

func (s *SystemSettingsService) GetRuntimeEmailConfig() (appconfig.EmailConfig, error) {
	values, err := s.repo.GetAll()
	if err != nil {
		return appconfig.EmailConfig{}, err
	}
	email := s.emailSettingsFrom(values, true)
	return appconfig.EmailConfig{
		Mode: normalizeEmailMode(email.Mode),
		SMTP: appconfig.EmailSMTPConfig{
			Host:     email.Host,
			Port:     email.Port,
			Username: email.Username,
			Password: email.Password,
			From:     email.From,
			FromName: email.FromName,
		},
		Verification: s.cfg.Email.Verification,
	}, nil
}

func (s *SystemSettingsService) GetRegistrationMode() (string, error) {
	values, err := s.repo.GetAll()
	if err != nil {
		return "", err
	}
	emailMode := s.emailSettingsFrom(values, true).Mode
	return s.registrationModeFrom(values, emailMode), nil
}

func (s *SystemSettingsService) emailSettingsFrom(values map[string]string, includePassword bool) dto.EmailSettings {
	email := dto.EmailSettings{
		Mode:     firstNonEmpty(values[settingEmailMode], s.cfg.Email.Mode),
		Host:     firstNonEmpty(values[settingSMTPHost], s.cfg.Email.SMTP.Host),
		Port:     firstNonZeroInt(values[settingSMTPPort], s.cfg.Email.SMTP.Port, 587),
		Username: firstNonEmpty(values[settingSMTPUsername], s.cfg.Email.SMTP.Username),
		From:     firstNonEmpty(values[settingSMTPFrom], s.cfg.Email.SMTP.From),
		FromName: firstNonEmpty(values[settingSMTPFromName], s.cfg.Email.SMTP.FromName, "kageos"),
	}
	email.Mode = normalizeEmailMode(email.Mode)
	password := firstNonEmpty(values[settingSMTPPassword], s.cfg.Email.SMTP.Password)
	email.PasswordSet = strings.TrimSpace(password) != ""
	if includePassword {
		email.Password = password
	}
	return email
}

func tlsMode() string {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("TLS_MODE")))
	if mode == "" {
		return "http"
	}
	return mode
}

func tlsCertFile() string {
	return firstNonEmpty(os.Getenv("TLS_CERT_FILE"), defaultTLSCertFile)
}

func tlsKeyFile() string {
	return firstNonEmpty(os.Getenv("TLS_KEY_FILE"), defaultTLSKeyFile)
}

func tlsReloadSupported() bool {
	mode := tlsMode()
	if mode != "https" && mode != "redirect" {
		return false
	}
	if _, err := exec.LookPath("nginx"); err != nil {
		return false
	}
	return fileExists(tlsCertFile()) && fileExists(tlsKeyFile())
}

func normalizePEM(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\r\n", "\n"))
	if value == "" {
		return ""
	}
	return value + "\n"
}

func parseCertificatePEM(value string) (*x509.Certificate, error) {
	rest := []byte(value)
	for {
		block, next := pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("parse certificate PEM: %w", err)
			}
			return cert, nil
		}
		rest = next
	}
	return nil, fmt.Errorf("certificate PEM is required")
}

func parsePrivateKeyPublicKey(value string) (crypto.PublicKey, error) {
	block, _ := pem.Decode([]byte(value))
	if block == nil {
		return nil, fmt.Errorf("private key PEM is required")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return publicKeyFromPrivateKey(key)
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return &key.PublicKey, nil
	}
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return &key.PublicKey, nil
	}
	return nil, fmt.Errorf("unsupported private key PEM")
}

func publicKeyFromPrivateKey(key any) (crypto.PublicKey, error) {
	switch typed := key.(type) {
	case *rsa.PrivateKey:
		return &typed.PublicKey, nil
	case *ecdsa.PrivateKey:
		return &typed.PublicKey, nil
	case ed25519.PrivateKey:
		return typed.Public(), nil
	default:
		return nil, fmt.Errorf("unsupported private key type")
	}
}

func publicKeysEqual(left crypto.PublicKey, right crypto.PublicKey) bool {
	leftDER, leftErr := x509.MarshalPKIXPublicKey(left)
	rightDER, rightErr := x509.MarshalPKIXPublicKey(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftDER, rightDER)
}

func certificateInfo(cert *x509.Certificate) *dto.TLSCertificateInfo {
	ips := make([]string, 0, len(cert.IPAddresses))
	for _, ip := range cert.IPAddresses {
		ips = append(ips, ip.String())
	}
	return &dto.TLSCertificateInfo{
		Subject:      cert.Subject.String(),
		Issuer:       cert.Issuer.String(),
		DNSNames:     append([]string{}, cert.DNSNames...),
		IPAddresses:  ips,
		NotBefore:    cert.NotBefore.Format("2006-01-02T15:04:05Z07:00"),
		NotAfter:     cert.NotAfter.Format("2006-01-02T15:04:05Z07:00"),
		IsSelfSigned: isSelfSignedCertificate(cert),
	}
}

func isSelfSignedCertificate(cert *x509.Certificate) bool {
	return bytes.Equal(cert.RawSubject, cert.RawIssuer) && cert.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature) == nil
}

func writeTLSPEMFiles(certFile string, keyFile string, certPEM []byte, keyPEM []byte) error {
	for _, dir := range uniqueStrings([]string{filepath.Dir(certFile), filepath.Dir(keyFile)}) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create TLS directory %s: %w", dir, err)
		}
	}
	if err := writeFileAtomic(certFile, certPEM, 0600); err != nil {
		return err
	}
	if err := writeFileAtomic(keyFile, keyPEM, 0600); err != nil {
		return err
	}
	return nil
}

func restoreTLSPEMFiles(certFile string, keyFile string, certPEM []byte, certExists bool, keyPEM []byte, keyExists bool) error {
	if certExists {
		if err := writeFileAtomic(certFile, certPEM, 0600); err != nil {
			return err
		}
	} else {
		_ = os.Remove(certFile)
	}
	if keyExists {
		if err := writeFileAtomic(keyFile, keyPEM, 0600); err != nil {
			return err
		}
	} else {
		_ = os.Remove(keyFile)
	}
	return nil
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary file for %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary file for %s: %w", path, err)
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temporary file for %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary file for %s: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirWritable(dir string) bool {
	if strings.TrimSpace(dir) == "" {
		return false
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return false
	}
	tmp, err := os.CreateTemp(dir, ".kageos-tls-write-test-*")
	if err != nil {
		return false
	}
	name := tmp.Name()
	_ = tmp.Close()
	_ = os.Remove(name)
	return true
}

func runNginxCommand(args ...string) error {
	cmd := exec.Command("nginx", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nginx %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func (s *SystemSettingsService) registrationModeFrom(values map[string]string, emailMode string) string {
	if mode := normalizeRegistrationMode(values[settingRegistrationMode]); mode != "" {
		return mode
	}
	if mode := normalizeRegistrationMode(s.cfg.Auth.RegistrationMode); mode != "" {
		return mode
	}
	if normalizeEmailMode(emailMode) == EmailModeLog {
		return RegistrationModeDebugCode
	}
	return RegistrationModeAdminOnly
}

func normalizeRegistrationMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case RegistrationModeAdminOnly, RegistrationModeEmailCode, RegistrationModeDebugCode:
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return ""
	}
}

func normalizeEmailMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case EmailModeLog:
		return EmailModeLog
	default:
		return EmailModeSMTP
	}
}

func validateSMTPConfig(cfg appconfig.EmailSMTPConfig) error {
	if strings.TrimSpace(cfg.Host) == "" ||
		cfg.Port == 0 ||
		strings.TrimSpace(cfg.Username) == "" ||
		strings.TrimSpace(cfg.Password) == "" ||
		strings.TrimSpace(cfg.From) == "" {
		return fmt.Errorf("SMTP host, port, username, password, and from are required")
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonZeroInt(raw string, values ...int) int {
	if parsed, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && parsed > 0 {
		return parsed
	}
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
