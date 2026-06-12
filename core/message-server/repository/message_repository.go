package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/message-server/model"
	"github.com/kageos/kageos/dto"
	"gorm.io/gorm"
)

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

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
	}
	if entry.ContentType == "" {
		entry.ContentType = "markdown"
	}
	if entry.SourcePath == "" {
		entry.SourcePath = entry.FullCodePath
	}
	if entry.ThreadKey == "" {
		entry.ThreadKey = buildMessageThreadKey(entry.SourcePath, entry.FullCodePath, entry.WorkspaceSessionID)
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

func (r *MessageRepository) ListInbox(ctx context.Context, username, status string, offset, limit int) ([]dto.MessageInboxItem, int64, error) {
	var list []dto.MessageInboxItem
	query := r.inboxQuery(ctx, username, status)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}
	if err := query.
		Select(inboxSelectColumns()).
		Order("m.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&list).Error; err != nil {
		return nil, 0, err
	}
	hydrateMessageSourceDisplays(list)
	return list, total, nil
}

func (r *MessageRepository) GetInboxMessage(ctx context.Context, username string, messageID int64) (*dto.MessageInboxItem, error) {
	var item dto.MessageInboxItem
	result := r.inboxBaseQuery(ctx, username).
		Where("m.id = ?", messageID).
		Select(inboxSelectColumns()).
		Limit(1).
		Scan(&item)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	hydrateMessageSourceDisplay(&item)
	return &item, nil
}

func (r *MessageRepository) CountUnread(ctx context.Context, username string) (int64, error) {
	var count int64
	err := r.inboxBaseQuery(ctx, username).Where("r.read_at IS NULL").Count(&count).Error
	return count, err
}

func (r *MessageRepository) MarkRead(ctx context.Context, username string, messageID int64) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.MessageRecipient{}).
		Where("username = ? AND message_id = ?", username, messageID).
		Updates(map[string]interface{}{"read_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("消息不存在")
	}
	return nil
}

func (r *MessageRepository) MarkAllRead(ctx context.Context, username string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.MessageRecipient{}).
		Where("username = ? AND read_at IS NULL", username).
		Update("read_at", now).Error
}

func (r *MessageRepository) inboxQuery(ctx context.Context, username, status string) *gorm.DB {
	query := r.inboxBaseQuery(ctx, username)
	if strings.EqualFold(strings.TrimSpace(status), "unread") {
		query = query.Where("r.read_at IS NULL")
	}
	return query
}

func (r *MessageRepository) inboxBaseQuery(ctx context.Context, username string) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("message_recipient AS r").
		Joins("JOIN message_entry AS m ON m.id = r.message_id").
		Where("r.username = ? AND r.deleted_at IS NULL AND m.deleted_at IS NULL", username)
}

func inboxSelectColumns() string {
	return strings.Join([]string{
		"m.id",
		"r.id AS recipient_id",
		"m.`from`",
		"m.request_user",
		"m.department_full_path",
		"m.full_code_path",
		"m.trace_id",
		"m.client_source",
		"m.source_type",
		"m.source_ref",
		"m.source_path",
		"m.source_title",
		"m.source_parent_path",
		"m.source_parent_title",
		"m.source_template_type",
		"m.workspace_session_id",
		"m.workspace_session_title",
		"m.workspace_role",
		"m.thread_key",
		"m.title",
		"m.content",
		"m.content_type",
		"r.read_at",
		"m.created_at",
	}, ", ")
}

func buildMessageThreadKey(sourcePath, fullCodePath, sessionID string) string {
	if sourcePath = strings.TrimSpace(sourcePath); sourcePath != "" {
		return "source:" + sourcePath
	}
	if fullCodePath = strings.TrimSpace(fullCodePath); fullCodePath != "" {
		return "source:" + fullCodePath
	}
	if sessionID = strings.TrimSpace(sessionID); sessionID != "" {
		return "session:" + sessionID
	}
	return ""
}

func hydrateMessageSourceDisplays(items []dto.MessageInboxItem) {
	for i := range items {
		hydrateMessageSourceDisplay(&items[i])
	}
}

func hydrateMessageSourceDisplay(item *dto.MessageInboxItem) {
	if item == nil {
		return
	}
	sourcePath := strings.TrimSpace(item.SourcePath)
	if sourcePath == "" {
		sourcePath = strings.TrimSpace(item.FullCodePath)
	}
	name := strings.TrimSpace(item.SourceTitle)
	if name == "" {
		name = pathBaseName(sourcePath)
	}
	if sourcePath == "" && name == "" {
		return
	}
	item.SourceDisplay = &dto.MessageSourceDisplay{
		Name:               name,
		Type:               strings.TrimSpace(item.SourceType),
		TemplateType:       strings.TrimSpace(item.SourceTemplateType),
		FullCodePath:       sourcePath,
		ParentName:         strings.TrimSpace(item.SourceParentTitle),
		ParentFullCodePath: strings.TrimSpace(item.SourceParentPath),
		ThreadKey:          strings.TrimSpace(item.ThreadKey),
	}
}

func pathBaseName(path string) string {
	path = strings.Trim(strings.TrimSpace(path), "/")
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
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
