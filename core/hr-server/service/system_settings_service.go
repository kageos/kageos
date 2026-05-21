package service

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kageos/kageos/core/hr-server/repository"
	"github.com/kageos/kageos/dto"
	appconfig "github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/emailx"
)

const (
	RegistrationModeAdminOnly = "admin_only"
	RegistrationModeEmailCode = "email_code"
	RegistrationModeDebugCode = "debug_code"

	EmailModeSMTP = "smtp"
	EmailModeLog  = "log"

	settingRegistrationMode = "auth.registration_mode"
	settingEmailMode        = "email.mode"
	settingSMTPHost         = "email.smtp.host"
	settingSMTPPort         = "email.smtp.port"
	settingSMTPUsername     = "email.smtp.username"
	settingSMTPPassword     = "email.smtp.password"
	settingSMTPFrom         = "email.smtp.from"
	settingSMTPFromName     = "email.smtp.from_name"
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
		values[settingSMTPFromName] = "Kageos"
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
	return emailx.NewSender(cfg.SMTP).SendHTML(to, "Kageos test email", "<p>Kageos email service is configured correctly.</p>")
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
		FromName: firstNonEmpty(values[settingSMTPFromName], s.cfg.Email.SMTP.FromName, "Kageos"),
	}
	email.Mode = normalizeEmailMode(email.Mode)
	password := firstNonEmpty(values[settingSMTPPassword], s.cfg.Email.SMTP.Password)
	email.PasswordSet = strings.TrimSpace(password) != ""
	if includePassword {
		email.Password = password
	}
	return email
}

func (s *SystemSettingsService) registrationModeFrom(values map[string]string, emailMode string) string {
	if mode := normalizeRegistrationMode(values[settingRegistrationMode]); mode != "" {
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
