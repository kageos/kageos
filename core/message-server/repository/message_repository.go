package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/core/message-server/model"
	"github.com/kageos/kageos/dto"
	"gorm.io/gorm"
)

type MessageRepository struct {
	db *gorm.DB
}

type InboxListFilter struct {
	Status          string
	ThreadKey       string
	SourcePath      string
	IncludeChildren bool
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

func (r *MessageRepository) ListInboxThreads(ctx context.Context, username, status string, offset, limit int) ([]dto.MessageInboxThread, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	threadExpr := inboxThreadKeySQL()
	countQuery := r.inboxQuery(ctx, username, InboxListFilter{Status: status}).
		Select(threadExpr + " AS thread_key").
		Group(threadExpr)
	var total int64
	if err := r.db.WithContext(ctx).Table("(?) AS inbox_threads", countQuery).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []inboxThreadRow
	if err := r.inboxQuery(ctx, username, InboxListFilter{Status: status}).
		Select(strings.Join([]string{
			threadExpr + " AS thread_key",
			"MAX(m.id) AS last_message_id",
			"MAX(m.created_at) AS latest_at",
			"COUNT(*) AS message_count",
			"SUM(CASE WHEN r.read_at IS NULL THEN 1 ELSE 0 END) AS unread_count",
		}, ", ")).
		Group(threadExpr).
		Order("latest_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	if len(rows) == 0 {
		return []dto.MessageInboxThread{}, total, nil
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row.LastMessageID > 0 {
			ids = append(ids, row.LastMessageID)
		}
	}
	items, err := r.listInboxByMessageIDs(ctx, username, ids)
	if err != nil {
		return nil, 0, err
	}
	itemByID := make(map[int64]dto.MessageInboxItem, len(items))
	for _, item := range items {
		itemByID[item.ID] = item
	}

	threads := make([]dto.MessageInboxThread, 0, len(rows))
	for _, row := range rows {
		lastMessage, ok := itemByID[row.LastMessageID]
		if !ok {
			continue
		}
		thread := buildInboxThread(row, lastMessage)
		threads = append(threads, thread)
	}
	return threads, total, nil
}

func (r *MessageRepository) ListSourceCounts(ctx context.Context, username, status string) ([]dto.MessageInboxSourceCount, error) {
	sourceExpr := inboxSourcePathSQL()
	var rows []inboxSourceCountRow
	if err := r.inboxQuery(ctx, username, InboxListFilter{Status: status}).
		Select(strings.Join([]string{
			sourceExpr + " AS source_path",
			"MAX(m.created_at) AS latest_at",
			"COUNT(*) AS message_count",
			"SUM(CASE WHEN r.read_at IS NULL THEN 1 ELSE 0 END) AS unread_count",
		}, ", ")).
		Group(sourceExpr).
		Order("latest_at DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]dto.MessageInboxSourceCount, 0, len(rows))
	for _, row := range rows {
		sourcePath := normalizeMessageSourcePath(row.SourcePath)
		if sourcePath == "" {
			continue
		}
		out = append(out, dto.MessageInboxSourceCount{
			SourcePath:   sourcePath,
			UnreadCount:  row.UnreadCount,
			MessageCount: row.MessageCount,
			LatestAt:     parseInboxLatestAt(row.LatestAt),
		})
	}
	return out, nil
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
		"r.read_at",
		"m.created_at",
	}, ", ")
}

type inboxThreadRow struct {
	ThreadKey     string
	LastMessageID int64
	LatestAt      string
	MessageCount  int64
	UnreadCount   int64
}

type inboxSourceCountRow struct {
	SourcePath   string
	LatestAt     string
	MessageCount int64
	UnreadCount  int64
}

func inboxThreadKeySQL() string {
	return "COALESCE(NULLIF(m.source_parent_path, ''), NULLIF(m.thread_key, ''), NULLIF(m.source_path, ''), NULLIF(m.full_code_path, ''), NULLIF(m.workspace_session_id, ''), NULLIF(m.`from`, ''), 'system')"
}

func inboxSourcePathSQL() string {
	return "COALESCE(NULLIF(m.source_path, ''), NULLIF(m.full_code_path, ''), NULLIF(m.source_parent_path, ''))"
}

func normalizeMessageSourcePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(path, "/")
}

func messageSourcePathVariants(path string) []string {
	path = normalizeMessageSourcePath(path)
	if path == "" {
		return nil
	}
	withoutLeadingSlash := strings.TrimPrefix(path, "/")
	if withoutLeadingSlash == path || withoutLeadingSlash == "" {
		return []string{path}
	}
	return []string{path, withoutLeadingSlash}
}

func parseInboxLatestAt(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func buildMessageThreadKey(sourceParentPath, sourcePath, fullCodePath, sessionID string) string {
	if sourceParentPath = strings.TrimSpace(sourceParentPath); sourceParentPath != "" {
		return "directory:" + sourceParentPath
	}
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

func buildInboxThread(row inboxThreadRow, lastMessage dto.MessageInboxItem) dto.MessageInboxThread {
	key := strings.TrimSpace(row.ThreadKey)
	if key == "" {
		key = strings.TrimSpace(lastMessage.ThreadKey)
	}
	if key == "" {
		key = buildMessageThreadKey(lastMessage.SourceParentPath, lastMessage.SourcePath, lastMessage.FullCodePath, lastMessage.WorkspaceSessionID)
	}
	title := threadTitle(lastMessage)
	subtitle := threadSubtitle(lastMessage, row.MessageCount)
	path := threadPath(lastMessage)
	return dto.MessageInboxThread{
		Key:                  key,
		Kind:                 threadKind(lastMessage),
		Title:                title,
		Subtitle:             subtitle,
		Path:                 path,
		UnreadCount:          row.UnreadCount,
		MessageCount:         row.MessageCount,
		LatestAt:             lastMessage.CreatedAt,
		LastMessage:          lastMessage,
		ScheduledTaskID:      lastMessage.ScheduledTaskID,
		ScheduledExecutionID: lastMessage.ScheduledExecutionID,
	}
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
	item.ScheduledTaskID, item.ScheduledExecutionID = parseScheduledSourceRef(item.SourceRef)
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

func threadTitle(item dto.MessageInboxItem) string {
	display := item.SourceDisplay
	var displayParentName, displayName string
	if display != nil {
		displayParentName = display.ParentName
		displayName = display.Name
	}
	for _, value := range []string{
		displayParentName,
		item.SourceParentTitle,
		displayName,
		item.SourceTitle,
		item.From,
		"system",
	} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "system"
}

func threadSubtitle(item dto.MessageInboxItem, count int64) string {
	sourceName := sourceSecondaryText(item)
	if count > 1 {
		return fmt.Sprintf("%s · %d 条消息", sourceName, count)
	}
	return sourceName
}

func sourceSecondaryText(item dto.MessageInboxItem) string {
	var displayName, displayParentName string
	if item.SourceDisplay != nil {
		displayName = item.SourceDisplay.Name
		displayParentName = item.SourceDisplay.ParentName
	}
	functionName := strings.TrimSpace(displayName)
	if functionName == "" {
		functionName = strings.TrimSpace(item.SourceTitle)
	}
	parentName := strings.TrimSpace(displayParentName)
	if parentName == "" {
		parentName = strings.TrimSpace(item.SourceParentTitle)
	}
	if functionName != "" && functionName != parentName {
		return functionName
	}
	if strings.TrimSpace(item.WorkspaceSessionTitle) != "" {
		return strings.TrimSpace(item.WorkspaceSessionTitle)
	}
	for _, value := range []string{item.SourcePath, item.FullCodePath, item.From, "-"} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "-"
}

func threadPath(item dto.MessageInboxItem) string {
	if item.SourceDisplay != nil {
		if path := strings.TrimSpace(item.SourceDisplay.ParentFullCodePath); path != "" {
			return path
		}
		if path := strings.TrimSpace(item.SourceDisplay.FullCodePath); path != "" {
			return path
		}
	}
	if path := strings.TrimSpace(item.SourceParentPath); path != "" {
		return path
	}
	if path := strings.TrimSpace(item.SourcePath); path != "" {
		return path
	}
	return strings.TrimSpace(item.FullCodePath)
}

func threadKind(item dto.MessageInboxItem) string {
	displayParentPath := ""
	if item.SourceDisplay != nil {
		displayParentPath = item.SourceDisplay.ParentFullCodePath
	}
	if strings.TrimSpace(item.SourceParentPath) != "" || strings.TrimSpace(displayParentPath) != "" {
		return "directory"
	}
	if strings.TrimSpace(item.WorkspaceSessionID) != "" {
		return "session"
	}
	if strings.TrimSpace(item.SourcePath) != "" || strings.TrimSpace(item.FullCodePath) != "" {
		return "function"
	}
	return "sender"
}

func parseScheduledSourceRef(sourceRef string) (int64, int64) {
	parts := strings.Split(strings.TrimSpace(sourceRef), ":")
	var taskID int64
	var executionID int64
	for i := 0; i < len(parts)-1; i++ {
		switch parts[i] {
		case "timer_task":
			taskID, _ = strconv.ParseInt(parts[i+1], 10, 64)
		case "execution", "timer_execution":
			executionID, _ = strconv.ParseInt(parts[i+1], 10, 64)
		}
	}
	return taskID, executionID
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
