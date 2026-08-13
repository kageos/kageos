package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/kageos/kageos/core/hr-server/repository"
	appconfig "github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/emailx"
	"github.com/kageos/kageos/pkg/gormx/models"
	"github.com/kageos/kageos/pkg/logger"
)

// EmailService 邮箱服务
type EmailService struct {
	config          *appconfig.EmailConfig
	settingsService *SystemSettingsService
	emailCodeRepo   *repository.EmailCodeRepository
}

// NewEmailService 创建邮箱服务（依赖注入）
func NewEmailService(emailCodeRepo *repository.EmailCodeRepository, settingsService *SystemSettingsService) *EmailService {
	hrConfig := appconfig.GetHRServerConfig()
	return &EmailService{
		config:          &hrConfig.Email,
		settingsService: settingsService,
		emailCodeRepo:   emailCodeRepo,
	}
}

// SendVerificationCode 发送验证码。log 模式用于本地开发：验证码写入日志并返回给调用方，不依赖真实 SMTP。
func (s *EmailService) SendVerificationCode(email, codeType, ipAddress, userAgent string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	codeType = strings.TrimSpace(codeType)
	if email == "" {
		return "", fmt.Errorf("邮箱不能为空")
	}
	if codeType != "register" && codeType != "forgot_password" {
		return "", fmt.Errorf("不支持的验证码类型")
	}
	// 生成验证码
	code, err := s.generateCode()
	if err != nil {
		return "", fmt.Errorf("生成验证码失败: %w", err)
	}
	emailCfg := s.runtimeEmailConfig()

	// 计算过期时间
	expiresAt := models.Time(time.Now().Add(time.Duration(s.verificationCodeExpireSeconds()) * time.Second))

	// 检查发送频率（防刷）
	count, err := s.emailCodeRepo.GetEmailCodeCount(email, 5) // 5分钟内
	if err != nil {
		logger.Errorf(nil, "[EmailService] Failed to get email code count: %v", err)
		return "", err
	}
	if count >= 3 { // 5分钟内最多发送3次
		return "", fmt.Errorf("验证码发送过于频繁，请稍后再试")
	}
	if strings.TrimSpace(ipAddress) != "" {
		ipCount, err := s.emailCodeRepo.GetEmailCodeCountByIP(ipAddress, 5)
		if err != nil {
			logger.Errorf(nil, "[EmailService] Failed to get email code IP count: %v", err)
			return "", err
		}
		if ipCount >= 10 {
			return "", fmt.Errorf("验证码请求过于频繁，请稍后再试")
		}
	}

	// 保存验证码到数据库
	err = s.emailCodeRepo.CreateEmailCode(email, code, expiresAt, codeType, ipAddress, userAgent)
	if err != nil {
		logger.Errorf(nil, "[EmailService] Failed to create email code: %v", err)
		return "", err
	}

	if s.emailMode(emailCfg) == "log" {
		logger.Infof(nil, "[EmailService] Verification code for %s (%s): %s", email, codeType, code)
		return code, nil
	}

	// 发送邮件
	subject := s.getSubject(codeType)
	body := s.getBody(code, codeType)

	err = emailx.NewSender(emailCfg.SMTP).SendHTML(email, subject, body)
	if err != nil {
		if invalidateErr := s.emailCodeRepo.InvalidateEmailCode(email, code, codeType); invalidateErr != nil {
			logger.Errorf(nil, "[EmailService] Failed to invalidate undelivered email code: %v", invalidateErr)
		}
		logger.Errorf(nil, "[EmailService] Failed to send email: %v", err)
		return "", err
	}

	logger.Infof(nil, "[EmailService] Verification code sent to %s", email)
	return "", nil
}

func (s *EmailService) runtimeEmailConfig() appconfig.EmailConfig {
	if s.settingsService != nil {
		cfg, err := s.settingsService.GetRuntimeEmailConfig()
		if err == nil {
			return cfg
		}
		logger.Errorf(nil, "[EmailService] Failed to load runtime email settings: %v", err)
	}
	return *s.config
}

func (s *EmailService) emailMode(cfg appconfig.EmailConfig) string {
	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	switch mode {
	case "log", "smtp":
		return mode
	case "":
		if strings.TrimSpace(cfg.SMTP.Username) == "" ||
			strings.TrimSpace(cfg.SMTP.Password) == "" ||
			strings.TrimSpace(cfg.SMTP.From) == "" {
			return "log"
		}
		return "smtp"
	default:
		return "smtp"
	}
}

// VerifyCode 验证验证码
func (s *EmailService) VerifyCode(email, code, codeType string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	code = strings.TrimSpace(code)
	codeType = strings.TrimSpace(codeType)
	err := s.emailCodeRepo.VerifyAndConsumeLatestEmailCode(email, code, codeType, 5)
	if err != nil {
		return fmt.Errorf("验证码无效或已过期")
	}

	logger.Infof(nil, "[EmailService] Email code verified for %s", email)
	return nil
}

// generateCode 生成验证码
func (s *EmailService) generateCode() (string, error) {
	length := s.config.Verification.CodeLength
	if length < 4 || length > 10 {
		length = 6
	}
	var code strings.Builder
	code.Grow(length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		code.WriteByte(byte('0' + num.Int64()))
	}
	return code.String(), nil
}

func (s *EmailService) verificationCodeExpireSeconds() int {
	expireSeconds := s.config.Verification.CodeExpire
	if expireSeconds <= 0 {
		return 10 * 60
	}
	return expireSeconds
}

// getSubject 获取邮件主题
func (s *EmailService) getSubject(codeType string) string {
	switch codeType {
	case "register":
		return "kageos 注册验证码"
	case "forgot_password":
		return "kageos 重置密码"
	default:
		return "kageos 验证码"
	}
}

// getBody 获取邮件内容
func (s *EmailService) getBody(code, codeType string) string {
	switch codeType {
	case "register":
		return fmt.Sprintf(`
			<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
				<h2 style="color: #333;">kageos 注册验证码</h2>
				<p>您好！</p>
				<p>您正在注册 kageos 账户，验证码为：</p>
				<div style="background-color: #f5f5f5; padding: 20px; text-align: center; margin: 20px 0;">
					<h1 style="color: #007bff; font-size: 32px; margin: 0; letter-spacing: 5px;">%s</h1>
				</div>
				<p>验证码有效期为 %d 分钟，请及时使用。</p>
				<p>如果这不是您的操作，请忽略此邮件。</p>
				<hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
				<p style="color: #666; font-size: 12px;">此邮件由系统自动发送，请勿回复。</p>
			</div>
		`, code, s.verificationCodeExpireSeconds()/60)
	case "forgot_password":
		return fmt.Sprintf(`
			<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
				<h2 style="color: #333;">kageos 重置密码验证码</h2>
				<p>您好！</p>
				<p>您正在重置 kageos 账户密码，验证码为：</p>
				<div style="background-color: #f5f5f5; padding: 20px; text-align: center; margin: 20px 0;">
					<h1 style="color: #007bff; font-size: 32px; margin: 0; letter-spacing: 5px;">%s</h1>
				</div>
				<p>验证码有效期为 %d 分钟，请及时使用。</p>
				<p>如果这不是您的操作，请忽略此邮件。</p>
				<hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
				<p style="color: #666; font-size: 12px;">此邮件由系统自动发送，请勿回复。</p>
			</div>
		`, code, s.verificationCodeExpireSeconds()/60)
	default:
		return fmt.Sprintf("您的验证码是：%s", code)
	}
}

// SendNotificationEmail 发送通知类邮件（通用，供消息服务等调用）
func (s *EmailService) SendNotificationEmail(to, subject, body string) error {
	cfg := s.runtimeEmailConfig()
	return emailx.NewSender(cfg.SMTP).SendHTML(to, subject, body)
}
