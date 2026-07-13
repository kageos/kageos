package repository

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/kageos/kageos/dto"
)

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

func (r *MessageRepository) ListWorkspaceCounts(ctx context.Context, username, status string) ([]dto.MessageInboxWorkspaceCount, error) {
	var rows []inboxWorkspaceCountSourceRow
	if err := r.inboxQuery(ctx, username, InboxListFilter{Status: status}).
		Select(strings.Join([]string{
			"m.source_path",
			"m.full_code_path",
			"m.source_parent_path",
			"m.source_title",
			"m.source_parent_title",
			"r.read_at",
			"m.created_at",
		}, ", ")).
		Order("m.created_at DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	groups := make(map[string]*dto.MessageInboxWorkspaceCount)
	for _, row := range rows {
		workspace := messageWorkspaceFromPaths(row.SourcePath, row.FullCodePath, row.SourceParentPath)
		group := groups[workspace.WorkspaceKey]
		if group == nil {
			group = &workspace
			groups[workspace.WorkspaceKey] = group
		}
		group.MessageCount += 1
		if row.ReadAt == nil {
			group.UnreadCount += 1
		}
		if group.LatestAt.IsZero() || row.CreatedAt.After(group.LatestAt) {
			group.LatestAt = row.CreatedAt
			group.LatestSourcePath = firstNonEmptyStringForRepository(row.SourcePath, row.FullCodePath, row.SourceParentPath)
			group.LatestSourceTitle = firstNonEmptyStringForRepository(row.SourceTitle, row.SourceParentTitle, group.Title)
		}
	}

	out := make([]dto.MessageInboxWorkspaceCount, 0, len(groups))
	for _, group := range groups {
		out = append(out, *group)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].LatestAt.After(out[j].LatestAt)
	})
	return out, nil
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

type inboxWorkspaceCountSourceRow struct {
	SourcePath        string
	FullCodePath      string
	SourceParentPath  string
	SourceTitle       string
	SourceParentTitle string
	ReadAt            *time.Time
	CreatedAt         time.Time
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

func messageWorkspaceFromPaths(paths ...string) dto.MessageInboxWorkspaceCount {
	for _, path := range paths {
		workspacePath, workspaceUser, workspaceCode := messageWorkspacePath(path)
		if workspacePath == "" {
			continue
		}
		return dto.MessageInboxWorkspaceCount{
			WorkspaceKey:  workspacePath,
			WorkspaceUser: workspaceUser,
			WorkspaceCode: workspaceCode,
			WorkspacePath: workspacePath,
			Title:         workspaceUser + "/" + workspaceCode,
		}
	}
	return dto.MessageInboxWorkspaceCount{
		WorkspaceKey: "global",
		Title:        "全局消息",
	}
}

func messageWorkspacePath(path string) (string, string, string) {
	path = normalizeMessageSourcePath(path)
	if path == "" {
		return "", "", ""
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		return "", "", ""
	}
	user := strings.TrimSpace(parts[0])
	code := strings.TrimSpace(parts[1])
	if user == "" || code == "" {
		return "", "", ""
	}
	return "/" + user + "/" + code, user, code
}

func firstNonEmptyStringForRepository(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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
