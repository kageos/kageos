package service

import (
	"context"
	"fmt"
	"strings"

	msgmodel "github.com/kageos/kageos/core/message-server/model"
	msgrepo "github.com/kageos/kageos/core/message-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
)

type MessageConsumerService struct {
	messageRepo             *msgrepo.MessageRepository
	providers               map[string]ChannelProvider
	targetResolver          NotificationTargetResolver
	cardBuilder             NotificationCardBuilder
	notificationCardBaseURL string
}

type MessageConsumerOption func(*MessageConsumerService)

type NotificationTargetResolver interface {
	ResolveNotificationTargets(ctx context.Context, recipients []ResolvedRecipient, entry *msgmodel.MessageEntry, payload dto.MessageSendPayload) ([]NotificationTarget, error)
}

type NotificationTargetResolverFunc func(ctx context.Context, recipients []ResolvedRecipient, entry *msgmodel.MessageEntry, payload dto.MessageSendPayload) ([]NotificationTarget, error)

func (f NotificationTargetResolverFunc) ResolveNotificationTargets(ctx context.Context, recipients []ResolvedRecipient, entry *msgmodel.MessageEntry, payload dto.MessageSendPayload) ([]NotificationTarget, error) {
	if f == nil {
		return nil, nil
	}
	return f(ctx, recipients, entry, payload)
}

type noopNotificationTargetResolver struct{}

func (noopNotificationTargetResolver) ResolveNotificationTargets(context.Context, []ResolvedRecipient, *msgmodel.MessageEntry, dto.MessageSendPayload) ([]NotificationTarget, error) {
	return nil, nil
}

func WithChannelProviders(providers ...ChannelProvider) MessageConsumerOption {
	return func(s *MessageConsumerService) {
		if s == nil {
			return
		}
		for _, provider := range providers {
			if provider == nil {
				continue
			}
			channel := normalizeNotificationChannel(provider.Channel())
			if channel == "" || channel == "inbox" {
				continue
			}
			s.providers[channel] = provider
		}
	}
}

func WithNotificationTargetResolver(resolver NotificationTargetResolver) MessageConsumerOption {
	return func(s *MessageConsumerService) {
		if s == nil || resolver == nil {
			return
		}
		s.targetResolver = resolver
	}
}

func WithNotificationCardBuilder(builder NotificationCardBuilder) MessageConsumerOption {
	return func(s *MessageConsumerService) {
		if s == nil || builder == nil {
			return
		}
		s.cardBuilder = builder
	}
}

func WithNotificationCardBaseURL(baseURL string) MessageConsumerOption {
	return func(s *MessageConsumerService) {
		if s == nil {
			return
		}
		s.notificationCardBaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	}
}

func NewMessageConsumerService(messageRepo *msgrepo.MessageRepository, opts ...MessageConsumerOption) *MessageConsumerService {
	s := &MessageConsumerService{
		messageRepo:    messageRepo,
		providers:      make(map[string]ChannelProvider),
		targetResolver: noopNotificationTargetResolver{},
		cardBuilder:    DefaultNotificationCardBuilder{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	return s
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

	s.deliverExternalNotifications(ctx, entry, payload, recipients)
	return nil
}

func (s *MessageConsumerService) deliverExternalNotifications(ctx context.Context, entry *msgmodel.MessageEntry, payload dto.MessageSendPayload, recipients []ResolvedRecipient) {
	if s == nil || s.targetResolver == nil || len(s.providers) == 0 {
		return
	}
	targets, err := s.targetResolver.ResolveNotificationTargets(ctx, recipients, entry, payload)
	if err != nil {
		logger.Errorf(ctx, "[MessageConsumer] Resolve notification targets failed: %v", err)
		return
	}
	if len(targets) == 0 {
		return
	}
	if entry == nil {
		logger.Errorf(ctx, "[MessageConsumer] Build notification card failed: message entry missing")
		return
	}
	for _, target := range targets {
		channel := normalizeNotificationChannel(target.Channel)
		if channel == "" {
			continue
		}
		provider := s.providers[channel]
		if provider == nil {
			logger.Warnf(ctx, "[MessageConsumer] Notification channel=%s user=%s has no provider", channel, target.Recipient.Username)
			s.recordNotificationDeliveryFailure(ctx, target, fmt.Sprintf("通知渠道 %s 未注册 provider", channel), false)
			continue
		}
		builder := s.cardBuilder
		if builder == nil {
			builder = DefaultNotificationCardBuilder{}
		}
		card := builder.BuildNotificationCard(ctx, entry, payload, target, NotificationCardBuildOptions{BaseURL: s.notificationCardBaseURL})
		if err := provider.Deliver(ctx, target, card); err != nil {
			logger.Errorf(ctx, "[MessageConsumer] Deliver channel=%s user=%s failed: %v", provider.Channel(), target.Recipient.Username, err)
			s.recordNotificationDeliveryFailure(ctx, target, err.Error(), false)
			continue
		}
		s.recordNotificationDeliverySuccess(ctx, target, false)
	}
}

func (s *MessageConsumerService) recordNotificationDeliverySuccess(ctx context.Context, target NotificationTarget, isTest bool) {
	if s == nil || s.messageRepo == nil {
		return
	}
	username := strings.TrimSpace(target.Recipient.Username)
	channel := normalizeNotificationChannel(target.Channel)
	if username == "" || channel == "" {
		return
	}
	if err := s.messageRepo.RecordNotificationChannelDeliverySuccess(ctx, username, channel, isTest); err != nil {
		logger.Warnf(ctx, "[MessageConsumer] record notification delivery success failed user=%s channel=%s: %v", username, channel, err)
	}
}

func (s *MessageConsumerService) recordNotificationDeliveryFailure(ctx context.Context, target NotificationTarget, message string, isTest bool) {
	if s == nil || s.messageRepo == nil {
		return
	}
	username := strings.TrimSpace(target.Recipient.Username)
	channel := normalizeNotificationChannel(target.Channel)
	if username == "" || channel == "" {
		return
	}
	if err := s.messageRepo.RecordNotificationChannelDeliveryFailure(ctx, username, channel, message, isTest); err != nil {
		logger.Warnf(ctx, "[MessageConsumer] record notification delivery failure failed user=%s channel=%s: %v", username, channel, err)
	}
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
