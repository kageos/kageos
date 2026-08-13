package emailx

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/smtp"
	"strings"
	"time"

	appconfig "github.com/kageos/kageos/pkg/config"
)

// Sender 封装 SMTP 发送能力，供各服务复用。
type Sender struct {
	cfg appconfig.EmailSMTPConfig
}

func NewSender(cfg appconfig.EmailSMTPConfig) *Sender {
	return &Sender{cfg: cfg}
}

// SendHTML 发送 HTML 邮件。
func (s *Sender) SendHTML(to, subject, body string) error {
	for name, value := range map[string]string{
		"From":    s.cfg.From,
		"To":      to,
		"Subject": subject,
	} {
		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("邮件头 %s 包含非法换行符", name)
		}
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	boundary := writer.Boundary()

	encodedSubject := subject
	if len([]byte(subject)) != len(subject) {
		encodedSubject = fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))
	}

	headers := map[string]string{
		"From":         s.cfg.From,
		"To":           to,
		"Subject":      encodedSubject,
		"MIME-Version": "1.0",
		"Content-Type": fmt.Sprintf("multipart/mixed; boundary=%s", boundary),
		"X-Mailer":     "kageos Email System v1.0",
		"X-Priority":   "3",
		"Message-ID":   fmt.Sprintf("<%d@%s>", time.Now().UnixNano(), s.cfg.Host),
	}

	for k, v := range headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	buf.WriteString("\r\n")

	part, err := writer.CreatePart(map[string][]string{
		"Content-Type":              {"text/html; charset=UTF-8"},
		"Content-Transfer-Encoding": {"quoted-printable"},
	})
	if err != nil {
		return fmt.Errorf("创建邮件正文失败: %v", err)
	}
	if _, err := part.Write([]byte(body)); err != nil {
		return fmt.Errorf("写入邮件正文失败: %v", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("关闭邮件正文失败: %v", err)
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)

	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("连接SMTP服务器失败: %v", err)
	}
	defer client.Quit()

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: s.cfg.Host,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS失败: %v", err)
	}
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP认证失败: %v", err)
	}
	if err := client.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("设置发件人失败: %v", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("设置收件人失败: %v", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("准备发送数据失败: %v", err)
	}
	return writeSMTPData(w, buf.Bytes())
}

func writeSMTPData(w io.WriteCloser, data []byte) error {
	if _, err := w.Write(data); err != nil {
		_ = w.Close()
		return fmt.Errorf("发送邮件内容失败: %v", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP 服务器拒绝邮件内容: %v", err)
	}
	return nil
}
