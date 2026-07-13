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

func (r *MessageRepository) ListInbox(ctx context.Context, username string, filter InboxListFilter, offset, limit int) ([]dto.MessageInboxItem, int64, error) {
	var list []dto.MessageInboxItem
	query := r.inboxQuery(ctx, username, filter)

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

func (r *MessageRepository) listInboxByMessageIDs(ctx context.Context, username string, messageIDs []int64) ([]dto.MessageInboxItem, error) {
	if len(messageIDs) == 0 {
		return []dto.MessageInboxItem{}, nil
	}
	var list []dto.MessageInboxItem
	if err := r.inboxBaseQuery(ctx, username).
		Where("m.id IN ?", messageIDs).
		Select(inboxSelectColumns()).
		Scan(&list).Error; err != nil {
		return nil, err
	}
	hydrateMessageSourceDisplays(list)
	return list, nil
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

func (r *MessageRepository) MarkSourceRead(ctx context.Context, username, sourcePath string, includeChildren bool) error {
	sourcePath = normalizeMessageSourcePath(sourcePath)
	if sourcePath == "" {
		return fmt.Errorf("消息来源路径不能为空")
	}
	now := time.Now()
	var recipientIDs []int64
	if err := r.inboxQuery(ctx, username, InboxListFilter{
		SourcePath:      sourcePath,
		IncludeChildren: includeChildren,
	}).Where("r.read_at IS NULL").Pluck("r.id", &recipientIDs).Error; err != nil {
		return err
	}
	for start := 0; start < len(recipientIDs); start += 1000 {
		end := start + 1000
		if end > len(recipientIDs) {
			end = len(recipientIDs)
		}
		if err := r.db.WithContext(ctx).
			Model(&model.MessageRecipient{}).
			Where("id IN ?", recipientIDs[start:end]).
			Update("read_at", now).Error; err != nil {
			return err
		}
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

func (r *MessageRepository) inboxQuery(ctx context.Context, username string, filter InboxListFilter) *gorm.DB {
	query := r.inboxBaseQuery(ctx, username)
	if strings.EqualFold(strings.TrimSpace(filter.Status), "unread") {
		query = query.Where("r.read_at IS NULL")
	}
	if threadKey := strings.TrimSpace(filter.ThreadKey); threadKey != "" {
		query = query.Where("("+inboxThreadKeySQL()+" = ? OR m.thread_key = ?)", threadKey, threadKey)
	}
	if sourcePath := normalizeMessageSourcePath(filter.SourcePath); sourcePath != "" {
		sourceExpr := inboxSourcePathSQL()
		conditions := make([]string, 0, 4)
		args := make([]interface{}, 0, 4)
		for _, variant := range messageSourcePathVariants(sourcePath) {
			conditions = append(conditions, sourceExpr+" = ?")
			args = append(args, variant)
			if filter.IncludeChildren {
				conditions = append(conditions, sourceExpr+" LIKE ?")
				args = append(args, variant+"/%")
			}
		}
		query = query.Where("("+strings.Join(conditions, " OR ")+")", args...)
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
		"m.files",
		"r.read_at",
		"m.created_at",
	}, ", ")
}
