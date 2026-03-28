package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/hr-server/repository"
	appconfig "github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/emailx"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// EmailService 邮箱服务
type EmailService struct {
	config        *appconfig.EmailConfig
	emailCodeRepo *repository.EmailCodeRepository
	sender        *emailx.Sender
}

// NewEmailService 创建邮箱服务（依赖注入）
func NewEmailService(emailCodeRepo *repository.EmailCodeRepository) *EmailService {
	hrConfig := appconfig.GetHRServerConfig()
	return &EmailService{
		config:        &hrConfig.Email,
		emailCodeRepo: emailCodeRepo,
		sender:        emailx.NewSender(hrConfig.Email.SMTP),
	}
}

// SendVerificationCode 发送验证码邮件
func (s *EmailService) SendVerificationCode(email, codeType, ipAddress, userAgent string) error {
	// 生成验证码
	code := s.generateCode()

	// 计算过期时间
	expiresAt := models.Time(time.Now().Add(time.Duration(s.config.Verification.CodeExpire) * time.Second))

	// 检查发送频率（防刷）
	count, err := s.emailCodeRepo.GetEmailCodeCount(email, 5) // 5分钟内
	if err != nil {
		logger.Errorf(nil, "[EmailService] Failed to get email code count: %v", err)
		return err
	}
	if count >= 3 { // 5分钟内最多发送3次
		return fmt.Errorf("验证码发送过于频繁，请稍后再试")
	}

	// 保存验证码到数据库
	err = s.emailCodeRepo.CreateEmailCode(email, code, expiresAt, codeType, ipAddress, userAgent)
	if err != nil {
		logger.Errorf(nil, "[EmailService] Failed to create email code: %v", err)
		return err
	}

	// 发送邮件
	subject := s.getSubject(codeType)
	body := s.getBody(code, codeType)

	err = s.sender.SendHTML(email, subject, body)
	if err != nil {
		logger.Errorf(nil, "[EmailService] Failed to send email: %v", err)
		return err
	}

	logger.Infof(nil, "[EmailService] Verification code sent to %s", email)
	return nil
}

// VerifyCode 验证验证码
func (s *EmailService) VerifyCode(email, code, codeType string) error {
	_, err := s.emailCodeRepo.GetValidEmailCode(email, code, codeType)
	if err != nil {
		return fmt.Errorf("验证码无效或已过期")
	}

	// 标记为已使用
	err = s.emailCodeRepo.MarkEmailCodeAsUsed(email, code, codeType)
	if err != nil {
		logger.Errorf(nil, "[EmailService] Failed to mark email code as used: %v", err)
		return err
	}

	logger.Infof(nil, "[EmailService] Email code verified for %s", email)
	return nil
}

// generateCode 生成验证码
func (s *EmailService) generateCode() string {
	length := s.config.Verification.CodeLength
	code := ""
	for i := 0; i < length; i++ {
		num, _ := rand.Int(rand.Reader, big.NewInt(10))
		code += fmt.Sprintf("%d", num.Int64())
	}
	return code
}

// getSubject 获取邮件主题
func (s *EmailService) getSubject(codeType string) string {
	switch codeType {
	case "register":
		return "AI Agent OS 注册验证码"
	case "forgot_password":
		return "AI Agent OS 重置密码"
	default:
		return "AI Agent OS 验证码"
	}
}

// getBody 获取邮件内容
func (s *EmailService) getBody(code, codeType string) string {
	switch codeType {
	case "register":
		return fmt.Sprintf(`
			<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
				<h2 style="color: #333;">AI Agent OS 注册验证码</h2>
				<p>您好！</p>
				<p>您正在注册 AI Agent OS 账户，验证码为：</p>
				<div style="background-color: #f5f5f5; padding: 20px; text-align: center; margin: 20px 0;">
					<h1 style="color: #007bff; font-size: 32px; margin: 0; letter-spacing: 5px;">%s</h1>
				</div>
				<p>验证码有效期为 %d 分钟，请及时使用。</p>
				<p>如果这不是您的操作，请忽略此邮件。</p>
				<hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
				<p style="color: #666; font-size: 12px;">此邮件由系统自动发送，请勿回复。</p>
			</div>
		`, code, s.config.Verification.CodeExpire/60)
	case "forgot_password":
		// code 在这里是重置密码的链接
		return fmt.Sprintf(`
			<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
				<h2 style="color: #333;">AI Agent OS 重置密码</h2>
				<p>您好！</p>
				<p>您正在重置 AI Agent OS 账户密码，请点击以下链接重置密码：</p>
				<div style="text-align: center; margin: 30px 0;">
					<a href="%s" style="display: inline-block; padding: 12px 30px; background-color: #007bff; color: #ffffff; text-decoration: none; border-radius: 5px; font-weight: bold;">重置密码</a>
				</div>
				<p>如果按钮无法点击，请复制以下链接到浏览器打开：</p>
				<p style="word-break: break-all; color: #666; font-size: 12px;">%s</p>
				<p>链接有效期为 1 小时，请及时使用。</p>
				<p>如果这不是您的操作，请忽略此邮件。</p>
				<hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
				<p style="color: #666; font-size: 12px;">此邮件由系统自动发送，请勿回复。</p>
			</div>
		`, code, code)
	default:
		return fmt.Sprintf("您的验证码是：%s", code)
	}
}

// SendNotificationEmail 发送通知类邮件（通用，供消息服务等调用）
func (s *EmailService) SendNotificationEmail(to, subject, body string) error {
	return s.sender.SendHTML(to, subject, body)
}

// SendPasswordResetEmail 发送密码重置邮件
func (s *EmailService) SendPasswordResetEmail(email, resetToken string) error {
	// 构建重置密码链接
	// 从环境变量获取前端URL，如果没有则使用相对路径（前端会自动补全）
	frontendURL := os.Getenv("FRONTEND_URL")
	resetLink := fmt.Sprintf("/reset-password?token=%s", resetToken)

	// 如果有配置前端URL，使用完整URL
	if frontendURL != "" {
		// 确保URL不以/结尾
		frontendURL = strings.TrimSuffix(frontendURL, "/")
		resetLink = fmt.Sprintf("%s/reset-password?token=%s", frontendURL, resetToken)
	}

	subject := s.getSubject("forgot_password")
	body := s.getBody(resetLink, "forgot_password")

	err := s.sender.SendHTML(email, subject, body)
	if err != nil {
		logger.Errorf(nil, "[EmailService] Failed to send password reset email: %v", err)
		return err
	}

	logger.Infof(nil, "[EmailService] Password reset email sent to %s", email)
	return nil
}
