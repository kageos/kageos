package service

import (
	"context"
	"encoding/json"
	"strings"

	msgmodel "github.com/kageos/kageos/core/message-server/model"
	msgrepo "github.com/kageos/kageos/core/message-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
)

type UserNotificationTargetResolver struct {
	repo  *msgrepo.MessageRepository
	vault *NotificationSecretVault
}

func NewUserNotificationTargetResolver(repo *msgrepo.MessageRepository, vault *NotificationSecretVault) *UserNotificationTargetResolver {
	return &UserNotificationTargetResolver{repo: repo, vault: vault}
}

func (r *UserNotificationTargetResolver) ResolveNotificationTargets(ctx context.Context, recipients []ResolvedRecipient, entry *msgmodel.MessageEntry, payload dto.MessageSendPayload) ([]NotificationTarget, error) {
	if r == nil || r.repo == nil {
		return nil, nil
	}
	recipientByUsername := make(map[string]ResolvedRecipient, len(recipients))
	usernames := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		username := strings.TrimSpace(recipient.Username)
		if username == "" {
			continue
		}
		recipient.Username = username
		recipientByUsername[username] = recipient
		usernames = append(usernames, username)
	}
	rows, err := r.repo.ListEnabledNotificationChannels(ctx, usernames)
	if err != nil {
		return nil, err
	}
	targets := make([]NotificationTarget, 0, len(rows))
	for _, row := range rows {
		if row == nil || strings.TrimSpace(row.WebhookURLCipher) == "" {
			continue
		}
		recipient, ok := recipientByUsername[strings.TrimSpace(row.OwnerUsername)]
		if !ok {
			continue
		}
		webhookURL, err := r.vault.Open(row.WebhookURLCipher)
		if err != nil {
			logger.Errorf(ctx, "[NotificationTargetResolver] open webhook url failed user=%s channel=%s: %v", row.OwnerUsername, row.Channel, err)
			continue
		}
		webhookURL = strings.TrimSpace(webhookURL)
		if webhookURL == "" {
			continue
		}
		secret, err := r.vault.Open(row.SecretCipher)
		if err != nil {
			logger.Errorf(ctx, "[NotificationTargetResolver] open secret failed user=%s channel=%s: %v", row.OwnerUsername, row.Channel, err)
			continue
		}
		targets = append(targets, NotificationTarget{
			Recipient:  recipient,
			Channel:    normalizeNotificationChannel(row.Channel),
			WebhookURL: webhookURL,
			Secret:     strings.TrimSpace(secret),
			Metadata:   parseNotificationMetadata(row.Metadata),
		})
	}
	return targets, nil
}

func parseNotificationMetadata(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var metadata map[string]string
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return nil
	}
	return metadata
}

func marshalNotificationMetadata(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}
	clean := make(map[string]string, len(metadata))
	for key, value := range metadata {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		clean[key] = strings.TrimSpace(value)
	}
	if len(clean) == 0 {
		return ""
	}
	raw, err := json.Marshal(clean)
	if err != nil {
		return ""
	}
	return string(raw)
}
