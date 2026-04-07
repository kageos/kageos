package service

import (
	"bytes"
	"context"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/yuin/goldmark"
)

// MessageConsumerService 消费消息载荷，并按渠道执行投递（当前实现：邮件）。
type MessageConsumerService struct {
	emailService *EmailService
	userService  *UserService
}

// NewMessageConsumerService 创建消息消费服务。
func NewMessageConsumerService(emailService *EmailService, userService *UserService) *MessageConsumerService {
	return &MessageConsumerService{
		emailService: emailService,
		userService:  userService,
	}
}

// Consume 处理消息载荷。
func (s *MessageConsumerService) Consume(ctx context.Context, payload *dto.MessageSendPayload) error {
	logger.Infof(ctx, "[MessageConsumer] Received message from=%s to_users=%s to_depts=%s title=%s", payload.From, payload.ToUsers, payload.ToDepartments, payload.Title)

	emails, err := s.resolveToEmails(ctx, payload)
	if err != nil {
		return err
	}
	if len(emails) == 0 {
		logger.Warnf(ctx, "[MessageConsumer] No recipients resolved (to_users=%s to_depts=%s), from=%s", payload.ToUsers, payload.ToDepartments, payload.From)
		return nil
	}

	logger.Infof(ctx, "[MessageConsumer] Resolved %d recipient(s), sending email title=%s", len(emails), payload.Title)

	subject := payload.Title
	if subject == "" {
		subject = "通知"
	}
	body := s.wrapBody(payload.ContentType, payload.Content)
	for _, email := range emails {
		if err := s.emailService.SendNotificationEmail(email, subject, body); err != nil {
			logger.Errorf(ctx, "[MessageConsumer] Send email to %s failed: %v", email, err)
		} else {
			logger.Infof(ctx, "[MessageConsumer] Send email to %s ok, title=%s", email, subject)
		}
	}

	return nil
}

// resolveToEmails 将 to_users、to_departments 解析为邮箱列表（去重）
func (s *MessageConsumerService) resolveToEmails(ctx context.Context, payload *dto.MessageSendPayload) ([]string, error) {
	emailSet := make(map[string]struct{})
	// to_users: 逗号分隔用户名
	if payload.ToUsers != "" {
		usernames := splitAndTrim(payload.ToUsers, ",")
		if len(usernames) > 0 {
			users, err := s.userService.GetUsersByUsernames(usernames)
			if err != nil {
				return nil, err
			}
			for _, u := range users {
				if u.Email != "" {
					emailSet[u.Email] = struct{}{}
				}
			}
		}
	}
	// to_departments: 逗号分隔部门路径
	if payload.ToDepartments != "" {
		deptPaths := splitAndTrim(payload.ToDepartments, ",")
		for _, path := range deptPaths {
			users, err := s.userService.GetUsersByDepartmentFullPath(ctx, path)
			if err != nil {
				logger.Warnf(ctx, "[MessageConsumer] GetUsersByDepartmentFullPath %s: %v", path, err)
				continue
			}
			for _, u := range users {
				if u.Email != "" {
					emailSet[u.Email] = struct{}{}
				}
			}
		}
	}
	emails := make([]string, 0, len(emailSet))
	for e := range emailSet {
		emails = append(emails, e)
	}
	return emails, nil
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// wrapBody 根据 content_type 将正文转为邮件可用的 HTML
//   - "markdown"（SDK 默认）：用 goldmark 解析为 HTML，支持加粗、列表、链接、表格等
//   - "html"：业务方自行控制的原始 HTML，直接透传
//   - "text"：纯文本，HTML 转义后保留换行
func (s *MessageConsumerService) wrapBody(contentType, content string) string {
	switch strings.ToLower(contentType) {
	case "html":
		return content
	case "markdown", "":
		return markdownToHTML(content)
	case "text":
		return "<div style=\"white-space: pre-wrap;\">" + escapeHTML(content) + "</div>"
	default:
		return "<div style=\"white-space: pre-wrap;\">" + escapeHTML(content) + "</div>"
	}
}

// markdownToHTML 将 Markdown 正文转为邮件安全的 HTML
func markdownToHTML(md string) string {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(md), &buf); err != nil {
		return "<div style=\"white-space: pre-wrap;\">" + escapeHTML(md) + "</div>"
	}
	return buf.String()
}

func escapeHTML(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\x22", "&quot;",
	).Replace(s)
}
