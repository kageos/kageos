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
	if routeTargets, routeMatched, err := r.resolveRouteTargets(ctx, recipients, entry); err != nil {
		return nil, err
	} else if routeMatched {
		return routeTargets, nil
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
			Kind:            NotificationTargetKindUser,
			Recipient:       recipient,
			AuthorizedUsers: []string{recipient.Username},
			Channel:         normalizeNotificationChannel(row.Channel),
			WebhookURL:      webhookURL,
			Secret:          strings.TrimSpace(secret),
			Metadata:        parseNotificationMetadata(row.Metadata),
		})
	}
	return targets, nil
}

func (r *UserNotificationTargetResolver) resolveRouteTargets(ctx context.Context, recipients []ResolvedRecipient, entry *msgmodel.MessageEntry) ([]NotificationTarget, bool, error) {
	if entry == nil {
		return nil, false, nil
	}
	candidates := msgrepo.NotificationRouteCandidatePaths(entry.SourcePath, entry.FullCodePath, entry.SourceParentPath)
	if len(candidates) == 0 {
		return nil, false, nil
	}
	rows, err := r.repo.ListEnabledNotificationRoutesByPaths(ctx, candidates)
	if err != nil {
		return nil, false, err
	}
	if len(rows) == 0 {
		return nil, false, nil
	}
	candidateRank := make(map[string]int, len(candidates))
	for idx, candidate := range candidates {
		candidateRank[candidate] = idx
	}
	bestRank := len(candidates)
	for _, row := range rows {
		if row == nil {
			continue
		}
		if rank, ok := candidateRank[msgrepo.NormalizeNotificationScopePath(row.ScopePath)]; ok && rank < bestRank {
			bestRank = rank
		}
	}
	if bestRank == len(candidates) {
		return nil, false, nil
	}
	bestScope := candidates[bestRank]
	authorizedUsers := notificationTargetAuthorizedUsers(recipients)
	representative := ResolvedRecipient{}
	if len(recipients) > 0 {
		representative = recipients[0]
	}
	targets := make([]NotificationTarget, 0, len(rows))
	for _, row := range rows {
		if row == nil || msgrepo.NormalizeNotificationScopePath(row.ScopePath) != bestScope {
			continue
		}
		if strings.TrimSpace(row.WebhookURLCipher) == "" {
			continue
		}
		webhookURL, err := r.vault.Open(row.WebhookURLCipher)
		if err != nil {
			logger.Errorf(ctx, "[NotificationTargetResolver] open route webhook url failed route_id=%d scope=%s channel=%s: %v", row.ID, row.ScopePath, row.Channel, err)
			continue
		}
		webhookURL = strings.TrimSpace(webhookURL)
		if webhookURL == "" {
			continue
		}
		secret, err := r.vault.Open(row.SecretCipher)
		if err != nil {
			logger.Errorf(ctx, "[NotificationTargetResolver] open route secret failed route_id=%d scope=%s channel=%s: %v", row.ID, row.ScopePath, row.Channel, err)
			continue
		}
		targets = append(targets, NotificationTarget{
			Kind:            NotificationTargetKindRoute,
			Recipient:       representative,
			AuthorizedUsers: authorizedUsers,
			Channel:         normalizeNotificationChannel(row.Channel),
			WebhookURL:      webhookURL,
			Secret:          strings.TrimSpace(secret),
			Metadata:        parseNotificationMetadata(row.Metadata),
			RouteID:         row.ID,
			ScopePath:       msgrepo.NormalizeNotificationScopePath(row.ScopePath),
			ScopeType:       strings.TrimSpace(row.ScopeType),
			RequireAuth:     row.RequireAuth,
		})
	}
	return targets, true, nil
}

func notificationTargetAuthorizedUsers(recipients []ResolvedRecipient) []string {
	out := make([]string, 0, len(recipients))
	seen := map[string]struct{}{}
	for _, recipient := range recipients {
		username := strings.TrimSpace(recipient.Username)
		if username == "" {
			continue
		}
		if _, ok := seen[username]; ok {
			continue
		}
		seen[username] = struct{}{}
		out = append(out, username)
	}
	return out
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
