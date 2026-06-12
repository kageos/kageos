package service

import (
	"context"
	"fmt"
	"strings"

	msgrepo "github.com/kageos/kageos/core/message-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
)

type MessageConsumerService struct {
	messageRepo *msgrepo.MessageRepository
	providers   []ChannelProvider
}

func NewMessageConsumerService(messageRepo *msgrepo.MessageRepository, providers ...ChannelProvider) *MessageConsumerService {
	if len(providers) == 0 {
		providers = []ChannelProvider{InboxChannelProvider{}}
	}
	return &MessageConsumerService{
		messageRepo: messageRepo,
		providers:   providers,
	}
}

func (s *MessageConsumerService) Consume(ctx context.Context, envelope *dto.MessageSendEnvelope) error {
	if envelope == nil {
		return nil
	}
	if s == nil || s.messageRepo == nil {
		return fmt.Errorf("message consumer is not initialized")
	}
	normalizeMessagePayload(&envelope.Message)
	if errMsg := validateMessagePayload(&envelope.Message); errMsg != "" {
		return fmt.Errorf("%s", errMsg)
	}
	meta := normalizeMessageMeta(envelope.Meta)
	payload := envelope.Message

	logger.Infof(ctx, "[MessageConsumer] Received message from=%s full_code_path=%s to_users=%s title=%s",
		meta.From, meta.FullCodePath, payload.ToUsers, payload.Title)

	recipients := resolveRecipients(&payload)
	if len(recipients) == 0 {
		return fmt.Errorf("没有找到可接收消息的用户")
	}

	usernames := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		usernames = append(usernames, recipient.Username)
	}
	entry, err := s.messageRepo.Create(ctx, meta, payload, usernames)
	if err != nil {
		return err
	}

	for _, provider := range s.providers {
		if provider == nil {
			continue
		}
		for _, recipient := range recipients {
			if err := provider.Deliver(ctx, entry, payload, recipient); err != nil {
				logger.Errorf(ctx, "[MessageConsumer] Deliver channel=%s user=%s failed: %v", provider.Channel(), recipient.Username, err)
			}
		}
	}
	return nil
}

func resolveRecipients(payload *dto.MessageSendPayload) []ResolvedRecipient {
	usernames := splitAndTrim(payload.ToUsers, ",")
	seen := make(map[string]struct{}, len(usernames))
	recipients := make([]ResolvedRecipient, 0, len(usernames))
	for _, username := range usernames {
		if username == "" {
			continue
		}
		if _, ok := seen[username]; ok {
			continue
		}
		seen[username] = struct{}{}
		recipients = append(recipients, ResolvedRecipient{Username: username})
	}
	return recipients
}

func normalizeMessageMeta(meta dto.MessageSendMeta) dto.MessageSendMeta {
	meta.From = strings.TrimSpace(meta.From)
	meta.RequestUser = strings.TrimSpace(meta.RequestUser)
	meta.DepartmentFullPath = strings.TrimSpace(meta.DepartmentFullPath)
	meta.FullCodePath = strings.TrimSpace(meta.FullCodePath)
	meta.TraceID = strings.TrimSpace(meta.TraceID)
	meta.ClientSource = strings.TrimSpace(meta.ClientSource)
	meta.SourceType = strings.TrimSpace(meta.SourceType)
	meta.SourceRef = strings.TrimSpace(meta.SourceRef)
	meta.SourcePath = strings.TrimSpace(meta.SourcePath)
	meta.SourceTitle = strings.TrimSpace(meta.SourceTitle)
	meta.SourceParentPath = strings.TrimSpace(meta.SourceParentPath)
	meta.SourceParentTitle = strings.TrimSpace(meta.SourceParentTitle)
	meta.SourceTemplateType = strings.TrimSpace(meta.SourceTemplateType)
	meta.WorkspaceSessionID = strings.TrimSpace(meta.WorkspaceSessionID)
	meta.WorkspaceSessionTitle = strings.TrimSpace(meta.WorkspaceSessionTitle)
	meta.WorkspaceRole = strings.TrimSpace(meta.WorkspaceRole)
	meta.ThreadKey = strings.TrimSpace(meta.ThreadKey)
	if meta.SourcePath == "" {
		meta.SourcePath = meta.FullCodePath
	}
	if meta.From == "" {
		meta.From = meta.RequestUser
	}
	if meta.From == "" {
		meta.From = "system"
	}
	if meta.RequestUser == "" {
		meta.RequestUser = meta.From
	}
	return meta
}

func normalizeMessagePayload(payload *dto.MessageSendPayload) {
	payload.ToUsers = strings.TrimSpace(payload.ToUsers)
	payload.Title = strings.TrimSpace(payload.Title)
	payload.Content = strings.TrimSpace(payload.Content)
	payload.ContentType = strings.ToLower(strings.TrimSpace(payload.ContentType))
	if payload.ContentType == "" {
		payload.ContentType = "markdown"
	}
}

func validateMessagePayload(payload *dto.MessageSendPayload) string {
	if payload == nil {
		return "消息内容不能为空"
	}
	if payload.ToUsers == "" {
		return "to_users 不能为空"
	}
	if payload.Content == "" {
		return "content 不能为空"
	}
	switch payload.ContentType {
	case "markdown", "html", "text":
		return ""
	default:
		return "content_type 仅支持 markdown、html、text"
	}
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
