package service

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"math/big"
	"mime/multipart"
	"net/smtp"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/hr-server/repository"
	appconfig "github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// EmailService 邮箱服务
type EmailService struct {
	config        *appconfig.EmailConfig
	emailCodeRepo *repository.EmailCodeRepository
}

// NewEmailService 创建邮箱服务（依赖注入）
func NewEmailService(emailCodeRepo *repository.EmailCodeRepository) *EmailService {
	hrConfig := appconfig.GetHRServerConfig()
	return &EmailService{
		config:        &hrConfig.Email,
		emailCodeRepo: emailCodeRepo,
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

	err = s.sendEmail(email, subject, body)
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
	default:
		return fmt.Sprintf("您的验证码是：%s", code)
	}
}

// sendEmail 发送邮件
func (s *EmailService) sendEmail(to, subject, body string) error {
	// 构建MIME消息
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	boundary := writer.Boundary()

	// 对主题进行编码，支持中文
	encodedSubject := subject
	if len([]byte(subject)) != len(subject) {
		encodedSubject = fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))
	}

	headers := map[string]string{
		"From":         s.config.SMTP.From,
		"To":           to,
		"Subject":      encodedSubject,
		"MIME-Version": "1.0",
		"Content-Type": fmt.Sprintf("multipart/mixed; boundary=%s", boundary),
		"X-Mailer":     "AI Agent OS Email System v1.0",
		"X-Priority":   "3",
		"Message-ID":   fmt.Sprintf("<%d@%s>", time.Now().UnixNano(), s.config.SMTP.Host),
	}

	for k, v := range headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	buf.WriteString("\r\n")

	// 邮件正文
	part, err := writer.CreatePart(map[string][]string{
		"Content-Type":              {"text/html; charset=UTF-8"},
		"Content-Transfer-Encoding": {"quoted-printable"},
	})
	if err != nil {
		return fmt.Errorf("创建邮件正文失败: %v", err)
	}

	part.Write([]byte(body))
	writer.Close()

	// 连接SMTP服务器
	addr := fmt.Sprintf("%s:%d", s.config.SMTP.Host, s.config.SMTP.Port)
	auth := smtp.PlainAuth("", s.config.SMTP.Username, s.config.SMTP.Password, s.config.SMTP.Host)

	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("连接SMTP服务器失败: %v", err)
	}
	defer client.Quit()

	// 使用STARTTLS
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         s.config.SMTP.Host,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS失败: %v", err)
	}

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP认证失败: %v", err)
	}

	if err := client.Mail(s.config.SMTP.From); err != nil {
		return fmt.Errorf("设置发件人失败: %v", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("设置收件人失败: %v", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("准备发送数据失败: %v", err)
	}
	defer w.Close()

	_, err = w.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("发送邮件内容失败: %v", err)
	}

	return nil
}
