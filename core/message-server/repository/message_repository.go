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
	if err := r.enrichInboxSourceDisplays(ctx, list); err != nil {
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
	list := []dto.MessageInboxItem{item}
	if err := r.enrichInboxSourceDisplays(ctx, list); err != nil {
		return nil, err
	}
	item = list[0]
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
		"m.from AS `from`",
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

type messageSourceTreeRow struct {
	Name         string `gorm:"column:name"`
	Type         string `gorm:"column:type"`
	TemplateType string `gorm:"column:template_type"`
	FullCodePath string `gorm:"column:full_code_path"`
}

func (r *MessageRepository) enrichInboxSourceDisplays(ctx context.Context, list []dto.MessageInboxItem) error {
	if len(list) == 0 {
		return nil
	}

	pathSet := make(map[string]struct{})
	paths := make([]string, 0, len(list))
	for _, item := range list {
		for _, path := range messageSourcePathCandidates(item) {
			if _, ok := pathSet[path]; ok {
				continue
			}
			pathSet[path] = struct{}{}
			paths = append(paths, path)
		}
	}

	sourceByPath := make(map[string]messageSourceTreeRow)
	if len(paths) > 0 {
		var rows []messageSourceTreeRow
		if err := r.db.WithContext(ctx).
			Table("service_tree").
			Select("name, type, template_type, full_code_path").
			Where("full_code_path IN ? AND deleted_at IS NULL", paths).
			Scan(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			if row.FullCodePath == "" {
				continue
			}
			sourceByPath[normalizeMessageSourceFullCodePath(row.FullCodePath)] = row
		}
	}

	for index := range list {
		item := &list[index]
		for _, path := range messageSourcePathCandidates(*item) {
			row, ok := sourceByPath[path]
			if !ok {
				continue
			}
			item.SourceDisplay = &dto.MessageSourceDisplay{
				Name:         strings.TrimSpace(row.Name),
				Type:         strings.TrimSpace(row.Type),
				TemplateType: strings.TrimSpace(row.TemplateType),
				FullCodePath: strings.TrimSpace(row.FullCodePath),
			}
			break
		}
		if item.SourceDisplay == nil {
			item.SourceDisplay = fallbackMessageSourceDisplay(*item)
		}
	}

	return nil
}

func messageSourcePathCandidates(item dto.MessageInboxItem) []string {
	candidates := make([]string, 0, 4)
	appendCandidate := func(raw string) {
		path := normalizeMessageSourceFullCodePath(raw)
		if path == "" {
			return
		}
		for _, existing := range candidates {
			if existing == path {
				return
			}
		}
		candidates = append(candidates, path)
	}

	appendCandidate(item.FullCodePath)
	if strings.HasPrefix(strings.TrimSpace(item.SourceRef), "/") {
		appendCandidate(item.SourceRef)
	}

	initialCount := len(candidates)
	for i := 0; i < initialCount; i++ {
		appendCandidate(stripMessageFunctionPathSuffix(candidates[i]))
	}
	return candidates
}

func fallbackMessageSourceDisplay(item dto.MessageInboxItem) *dto.MessageSourceDisplay {
	path := ""
	candidates := messageSourcePathCandidates(item)
	if len(candidates) > 0 {
		path = candidates[0]
	}
	sourceType := strings.TrimSpace(item.SourceType)
	templateType := inferTemplateTypeFromMessagePath(path)
	if sourceType == "" && templateType != "" {
		sourceType = "function"
	}
	if sourceType == "" && path == "" {
		return nil
	}

	name := messageSourceTypeDisplayName(sourceType, templateType)
	if name == "" {
		name = "服务目录"
	}
	return &dto.MessageSourceDisplay{
		Name:         name,
		Type:         sourceType,
		TemplateType: templateType,
		FullCodePath: path,
	}
}

func messageSourceTypeDisplayName(sourceType, templateType string) string {
	switch strings.ToLower(strings.TrimSpace(sourceType)) {
	case "function":
		switch strings.ToLower(strings.TrimSpace(templateType)) {
		case "form":
			return "表单"
		case "table":
			return "表格"
		case "chart":
			return "报表"
		default:
			return "函数"
		}
	case "package", "service", "directory", "catalog":
		return "服务目录"
	case "docs":
		return "文档"
	case "board":
		return "讨论区"
	case "scheduled_task":
		return "定时任务"
	case "scheduled_agent_task":
		return "定时会话"
	case "agent_tool":
		return "智能体工具"
	case "system":
		return "系统"
	case "user":
		return "用户"
	default:
		return ""
	}
}

func normalizeMessageSourceFullCodePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimSuffix(path, "/")
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func stripMessageFunctionPathSuffix(path string) string {
	path = normalizeMessageSourceFullCodePath(path)
	lower := strings.ToLower(path)
	for _, suffix := range []string{".form", ".table", ".chart"} {
		if strings.HasSuffix(lower, suffix) {
			return path[:len(path)-len(suffix)]
		}
	}
	return path
}

func inferTemplateTypeFromMessagePath(path string) string {
	lower := strings.ToLower(strings.TrimSpace(path))
	switch {
	case strings.HasSuffix(lower, ".form"):
		return "form"
	case strings.HasSuffix(lower, ".table"):
		return "table"
	case strings.HasSuffix(lower, ".chart"):
		return "chart"
	default:
		return ""
	}
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
