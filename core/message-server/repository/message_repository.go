package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/message-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
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
		From:               strings.TrimSpace(meta.From),
		RequestUser:        strings.TrimSpace(meta.RequestUser),
		DepartmentFullPath: strings.TrimSpace(meta.DepartmentFullPath),
		FullCodePath:       strings.TrimSpace(meta.FullCodePath),
		TraceID:            strings.TrimSpace(meta.TraceID),
		ClientSource:       strings.TrimSpace(meta.ClientSource),
		SourceType:         strings.TrimSpace(meta.SourceType),
		SourceRef:          strings.TrimSpace(meta.SourceRef),
		Title:              strings.TrimSpace(payload.Title),
		Content:            strings.TrimSpace(payload.Content),
		ContentType:        strings.TrimSpace(payload.ContentType),
	}
	if entry.ContentType == "" {
		entry.ContentType = "markdown"
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
		"m.id AS id",
		"r.id AS recipient_id",
		"`m`.`from` AS `from`",
		"m.request_user AS request_user",
		"m.department_full_path AS department_full_path",
		"m.full_code_path AS full_code_path",
		"m.trace_id AS trace_id",
		"m.client_source AS client_source",
		"m.source_type AS source_type",
		"m.source_ref AS source_ref",
		"m.title AS title",
		"m.content AS content",
		"m.content_type AS content_type",
		"r.read_at AS read_at",
		"m.created_at AS created_at",
	}, ", ")
}

func normalizeUsernames(usernames []string) []string {
	seen := make(map[string]struct{}, len(usernames))
	out := make([]string, 0, len(usernames))
	for _, username := range usernames {
		username = strings.TrimSpace(username)
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
