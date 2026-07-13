package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/message-server/model"
	"github.com/kageos/kageos/dto"
	"gorm.io/gorm"
)

func (r *MessageRepository) Create(ctx context.Context, meta dto.MessageSendMeta, payload dto.MessageSendPayload, usernames []string) (*model.MessageEntry, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("message repository is nil")
	}
	usernames = normalizeUsernames(usernames)
	if len(usernames) == 0 {
		return nil, fmt.Errorf("message recipients is empty")
	}

	entry := &model.MessageEntry{
		From:                  strings.TrimSpace(meta.From),
		RequestUser:           strings.TrimSpace(meta.RequestUser),
		DepartmentFullPath:    strings.TrimSpace(meta.DepartmentFullPath),
		FullCodePath:          strings.TrimSpace(meta.FullCodePath),
		TraceID:               strings.TrimSpace(meta.TraceID),
		ClientSource:          strings.TrimSpace(meta.ClientSource),
		SourceType:            strings.TrimSpace(meta.SourceType),
		SourceRef:             strings.TrimSpace(meta.SourceRef),
		SourcePath:            strings.TrimSpace(meta.SourcePath),
		SourceTitle:           strings.TrimSpace(meta.SourceTitle),
		SourceParentPath:      strings.TrimSpace(meta.SourceParentPath),
		SourceParentTitle:     strings.TrimSpace(meta.SourceParentTitle),
		SourceTemplateType:    strings.TrimSpace(meta.SourceTemplateType),
		WorkspaceSessionID:    strings.TrimSpace(meta.WorkspaceSessionID),
		WorkspaceSessionTitle: strings.TrimSpace(meta.WorkspaceSessionTitle),
		WorkspaceRole:         strings.TrimSpace(meta.WorkspaceRole),
		ThreadKey:             strings.TrimSpace(meta.ThreadKey),
		Title:                 strings.TrimSpace(payload.Title),
		Content:               strings.TrimSpace(payload.Content),
		ContentType:           strings.TrimSpace(payload.ContentType),
		Files:                 strings.TrimSpace(payload.Files),
	}
	if entry.ContentType == "" {
		entry.ContentType = "markdown"
	}
	if entry.SourcePath == "" {
		entry.SourcePath = entry.FullCodePath
	}
	if entry.ThreadKey == "" {
		entry.ThreadKey = buildMessageThreadKey(entry.SourceParentPath, entry.SourcePath, entry.FullCodePath, entry.WorkspaceSessionID)
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(entry).Error; err != nil {
			return err
		}
		recipients := make([]*model.MessageRecipient, 0, len(usernames))
		for _, username := range usernames {
			recipients = append(recipients, &model.MessageRecipient{
				MessageID: entry.ID,
				Username:  username,
			})
		}
		return tx.Create(&recipients).Error
	})
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func normalizeUsernames(usernames []string) []string {
	seen := make(map[string]struct{}, len(usernames))
	normalized := make([]string, 0, len(usernames))
	for _, username := range usernames {
		username = strings.TrimSpace(username)
		if username == "" {
			continue
		}
		if _, ok := seen[username]; ok {
			continue
		}
		seen[username] = struct{}{}
		normalized = append(normalized, username)
	}
	return normalized
}
