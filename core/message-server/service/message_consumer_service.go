package service

import (
	"bytes"
	"context"
	"strings"

	hrrepo "github.com/ai-agent-os/ai-agent-os/core/hr-server/repository"
	msgrepo "github.com/ai-agent-os/ai-agent-os/core/message-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/emailx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/yuin/goldmark"
)

// MessageConsumerService 消费系统通知/业务消息，并按渠道投递。
// 当前首个渠道为邮件；站内消息、Webhook、IM 通道后续都应挂在这里，而不是放在 hr-server。
type MessageConsumerService struct {
	userRepo    *hrrepo.UserRepository
	messageRepo *msgrepo.MessageRepository
	sender      *emailx.Sender
}

func NewMessageConsumerService(userRepo *hrrepo.UserRepository, messageRepo *msgrepo.MessageRepository, emailCfg config.EmailConfig) *MessageConsumerService {
	return &MessageConsumerService{
		userRepo:    userRepo,
		messageRepo: messageRepo,
		sender:      emailx.NewSender(emailCfg.SMTP),
	}
}

func (s *MessageConsumerService) Consume(ctx context.Context, envelope *dto.MessageSendEnvelope) error {
	if envelope == nil {
		return nil
	}
	meta := envelope.Meta
	payload := envelope.Message
	logger.Infof(ctx, "[MessageConsumer] Received message from=%s full_code_path=%s to_users=%s to_depts=%s title=%s",
		meta.From, meta.FullCodePath, payload.ToUsers, payload.ToDepartments, payload.Title)

	recipients, err := s.resolveRecipients(ctx, &payload)
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		logger.Warnf(ctx, "[MessageConsumer] No recipients resolved (to_users=%s to_depts=%s), from=%s", payload.ToUsers, payload.ToDepartments, meta.From)
		return nil
	}

	usernames := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		usernames = append(usernames, recipient.Username)
	}
	if s.messageRepo != nil {
		if _, err := s.messageRepo.Create(ctx, meta, payload, usernames); err != nil {
			return err
		}
	}

	subject := strings.TrimSpace(payload.Title)
	if subject == "" {
		subject = "通知"
	}
	body := wrapBody(payload.ContentType, payload.Content)
	for _, recipient := range recipients {
		email := strings.TrimSpace(recipient.Email)
		if email == "" {
			continue
		}
		if err := s.sender.SendHTML(email, subject, body); err != nil {
			logger.Errorf(ctx, "[MessageConsumer] Send email to %s failed: %v", email, err)
			continue
		}
		logger.Infof(ctx, "[MessageConsumer] Send email to %s ok, title=%s", email, subject)
	}
	return nil
}

type resolvedRecipient struct {
	Username string
	Email    string
}

func (s *MessageConsumerService) resolveRecipients(ctx context.Context, payload *dto.MessageSendPayload) ([]resolvedRecipient, error) {
	recipientSet := make(map[string]resolvedRecipient)
	if payload.ToUsers != "" {
		usernames := splitAndTrim(payload.ToUsers, ",")
		if len(usernames) > 0 {
			users, err := s.userRepo.GetUsersByUsernames(usernames)
			if err != nil {
				return nil, err
			}
			for _, user := range users {
				username := strings.TrimSpace(user.Username)
				if username != "" {
					recipientSet[username] = resolvedRecipient{Username: username, Email: strings.TrimSpace(user.Email)}
				}
			}
		}
	}

	if payload.ToDepartments != "" {
		deptPaths := splitAndTrim(payload.ToDepartments, ",")
		for _, path := range deptPaths {
			users, err := s.userRepo.GetUsersByDepartmentFullPath(path)
			if err != nil {
				logger.Warnf(ctx, "[MessageConsumer] GetUsersByDepartmentFullPath %s: %v", path, err)
				continue
			}
			for _, user := range users {
				username := strings.TrimSpace(user.Username)
				if username != "" {
					recipientSet[username] = resolvedRecipient{Username: username, Email: strings.TrimSpace(user.Email)}
				}
			}
		}
	}

	recipients := make([]resolvedRecipient, 0, len(recipientSet))
	for _, recipient := range recipientSet {
		recipients = append(recipients, recipient)
	}
	return recipients, nil
}

func (s *MessageConsumerService) resolveToEmails(ctx context.Context, payload *dto.MessageSendPayload) ([]string, error) {
	emailSet := make(map[string]struct{})
	if payload.ToUsers != "" {
		usernames := splitAndTrim(payload.ToUsers, ",")
		if len(usernames) > 0 {
			users, err := s.userRepo.GetUsersByUsernames(usernames)
			if err != nil {
				return nil, err
			}
			for _, user := range users {
				if strings.TrimSpace(user.Email) != "" {
					emailSet[user.Email] = struct{}{}
				}
			}
		}
	}

	if payload.ToDepartments != "" {
		deptPaths := splitAndTrim(payload.ToDepartments, ",")
		for _, path := range deptPaths {
			users, err := s.userRepo.GetUsersByDepartmentFullPath(path)
			if err != nil {
				logger.Warnf(ctx, "[MessageConsumer] GetUsersByDepartmentFullPath %s: %v", path, err)
				continue
			}
			for _, user := range users {
				if strings.TrimSpace(user.Email) != "" {
					emailSet[user.Email] = struct{}{}
				}
			}
		}
	}

	emails := make([]string, 0, len(emailSet))
	for email := range emailSet {
		emails = append(emails, email)
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

func wrapBody(contentType, content string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "html":
		return content
	case "markdown", "":
		return markdownToHTML(content)
	case "text":
		return `<div style="white-space: pre-wrap;">` + escapeHTML(content) + `</div>`
	default:
		return `<div style="white-space: pre-wrap;">` + escapeHTML(content) + `</div>`
	}
}

func markdownToHTML(md string) string {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(md), &buf); err != nil {
		return `<div style="white-space: pre-wrap;">` + escapeHTML(md) + `</div>`
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
