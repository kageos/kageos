package service

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kageos/kageos/core/message-server/model"
	"github.com/kageos/kageos/dto"
)

const (
	NotificationLevelInfo     = "info"
	NotificationLevelWarning  = "warning"
	NotificationLevelCritical = "critical"

	NotificationActionProcess = "process"
	NotificationActionDetail  = "detail"
	NotificationActionSource  = "source"
	NotificationActionSession = "session"
	NotificationActionAsk     = "ask"

	NotificationChannelFeishu   = "feishu"
	NotificationChannelWeCom    = "wecom"
	NotificationChannelDingTalk = "dingtalk"
)

type NotificationCard struct {
	MessageID   int64
	Title       string
	Level       string
	Summary     string
	Content     string
	ContentType string
	Files       string
	FromUser    string
	ToUser      string
	CreatedAt   time.Time
	Source      NotificationCardSource
	Task        NotificationCardTask
	Actions     []NotificationAction
}

type NotificationCardSource struct {
	Type         string
	Ref          string
	Path         string
	Title        string
	ParentPath   string
	ParentTitle  string
	TemplateType string
	Workspace    string
	WorkspaceURL string
}

type NotificationCardTask struct {
	Title         string
	SessionID     string
	SessionTitle  string
	WorkspaceRole string
	ThreadKey     string
}

type NotificationAction struct {
	Kind  string
	Label string
	URL   string
}

type NotificationCardBuildOptions struct {
	BaseURL         string
	MobileActionURL string
	MobileAskURL    string
}

type NotificationCardBuilder interface {
	BuildNotificationCard(ctx context.Context, entry *model.MessageEntry, payload dto.MessageSendPayload, target NotificationTarget, opts NotificationCardBuildOptions) NotificationCard
}

type DefaultNotificationCardBuilder struct{}

func (DefaultNotificationCardBuilder) BuildNotificationCard(_ context.Context, entry *model.MessageEntry, payload dto.MessageSendPayload, target NotificationTarget, opts NotificationCardBuildOptions) NotificationCard {
	if entry == nil {
		entry = &model.MessageEntry{}
	}
	title := firstNonEmptyString(strings.TrimSpace(entry.Title), strings.TrimSpace(payload.Title), "kageos 通知")
	content := firstNonEmptyString(strings.TrimSpace(entry.Content), strings.TrimSpace(payload.Content))
	files := firstNonEmptyString(strings.TrimSpace(entry.Files), strings.TrimSpace(payload.Files))
	if content == "" && files != "" {
		content = "包含附件，请打开详情查看。"
	}
	contentType := strings.TrimSpace(entry.ContentType)
	if contentType == "" {
		contentType = strings.TrimSpace(payload.ContentType)
	}
	if contentType == "" {
		contentType = "markdown"
	}

	sourcePath := normalizeCardFullCodePath(firstNonEmptyString(entry.SourcePath, entry.FullCodePath, entry.SourceParentPath))
	fullCodePath := normalizeCardFullCodePath(firstNonEmptyString(entry.FullCodePath, sourcePath, entry.SourceParentPath))
	sourceTitle := firstNonEmptyString(entry.SourceTitle, entry.SourceParentTitle, baseNameForCardPath(sourcePath), sourcePath)
	workspacePath, workspaceName := workspaceFromCardPath(firstNonEmptyString(sourcePath, fullCodePath, entry.SourceParentPath))

	card := NotificationCard{
		MessageID:   entry.ID,
		Title:       title,
		Level:       inferNotificationLevel(title),
		Summary:     summarizeNotificationContent(content, title, 180),
		Content:     content,
		ContentType: strings.ToLower(strings.TrimSpace(contentType)),
		Files:       files,
		FromUser:    firstNonEmptyString(entry.From, entry.RequestUser, "system"),
		ToUser:      target.Recipient.Username,
		CreatedAt:   entry.CreatedAt,
		Source: NotificationCardSource{
			Type:         entry.SourceType,
			Ref:          entry.SourceRef,
			Path:         sourcePath,
			Title:        sourceTitle,
			ParentPath:   normalizeCardFullCodePath(entry.SourceParentPath),
			ParentTitle:  entry.SourceParentTitle,
			TemplateType: entry.SourceTemplateType,
			Workspace:    workspaceName,
			WorkspaceURL: absoluteCardURL(opts.BaseURL, workspaceRouteForPath(workspacePath, nil)),
		},
		Task: NotificationCardTask{
			Title:         firstNonEmptyString(entry.WorkspaceSessionTitle, entry.SourceTitle, entry.Title),
			SessionID:     strings.TrimSpace(entry.WorkspaceSessionID),
			SessionTitle:  strings.TrimSpace(entry.WorkspaceSessionTitle),
			WorkspaceRole: strings.TrimSpace(entry.WorkspaceRole),
			ThreadKey:     strings.TrimSpace(entry.ThreadKey),
		},
	}
	if card.CreatedAt.IsZero() {
		card.CreatedAt = time.Now()
	}
	card.Actions = buildNotificationActions(opts, fullCodePath, sourcePath, sourceTitle, entry)
	return card
}

func buildNotificationActions(opts NotificationCardBuildOptions, fullCodePath, sourcePath, sourceTitle string, entry *model.MessageEntry) []NotificationAction {
	if entry == nil {
		return nil
	}
	baseURL := opts.BaseURL
	routePath := firstNonEmptyString(fullCodePath, sourcePath, entry.SourceParentPath)
	sourceRoutePath := firstNonEmptyString(sourcePath, fullCodePath, entry.SourceParentPath)
	actions := make([]NotificationAction, 0, 5)
	if strings.TrimSpace(opts.MobileActionURL) != "" {
		actions = append(actions, NotificationAction{
			Kind:  NotificationActionProcess,
			Label: "处理消息",
			URL:   opts.MobileActionURL,
		})
	}
	detailQuery := url.Values{}
	detailQuery.Set("_open", "inbox")
	detailQuery.Set("_focus", "message")
	if entry.ID > 0 {
		detailQuery.Set("_message_id", strconv.FormatInt(entry.ID, 10))
	}
	if sourceRoutePath != "" {
		detailQuery.Set("_source_path", sourceRoutePath)
	}
	if strings.TrimSpace(entry.TraceID) != "" {
		detailQuery.Set("_trace_id", strings.TrimSpace(entry.TraceID))
	}
	if route := workspaceRouteForPath(routePath, detailQuery); route != "" {
		actions = append(actions, NotificationAction{
			Kind:  NotificationActionDetail,
			Label: "查看详情",
			URL:   absoluteCardURL(baseURL, route),
		})
	}
	if route := workspaceRouteForPath(sourceRoutePath, nil); route != "" {
		actions = append(actions, NotificationAction{
			Kind:  NotificationActionSource,
			Label: "打开目录",
			URL:   absoluteCardURL(baseURL, route),
		})
	}
	if strings.TrimSpace(entry.WorkspaceSessionID) != "" {
		sessionQuery := url.Values{}
		sessionQuery.Set("_open", "session")
		sessionQuery.Set("_focus", "workspace_session")
		sessionQuery.Set("_session_id", strings.TrimSpace(entry.WorkspaceSessionID))
		if sourceRoutePath != "" {
			sessionQuery.Set("_source_path", sourceRoutePath)
		}
		if strings.TrimSpace(entry.TraceID) != "" {
			sessionQuery.Set("_trace_id", strings.TrimSpace(entry.TraceID))
		}
		sessionQuery.Set("_mws", "open")
		sessionQuery.Set("_mws_sid", strings.TrimSpace(entry.WorkspaceSessionID))
		sessionQuery.Set("_mws_path", routePath)
		if sourceTitle != "" {
			sessionQuery.Set("_mws_name", sourceTitle)
		}
		sessionQuery.Set("_mws_expanded", "1")
		sessionQuery.Set("_mws_maximized", "1")
		if route := workspaceRouteForPath(routePath, sessionQuery); route != "" {
			actions = append(actions, NotificationAction{
				Kind:  NotificationActionSession,
				Label: "打开会话",
				URL:   absoluteCardURL(baseURL, route),
			})
		}
	}
	if strings.TrimSpace(opts.MobileAskURL) != "" {
		actions = append(actions, NotificationAction{
			Kind:  NotificationActionAsk,
			Label: "主动问 kageos",
			URL:   opts.MobileAskURL,
		})
	}
	return dedupeNotificationActions(actions)
}

func inferNotificationLevel(title string) string {
	title = strings.TrimSpace(title)
	switch {
	case strings.Contains(title, "高优先级") || strings.Contains(title, "紧急") || strings.Contains(strings.ToLower(title), "critical"):
		return NotificationLevelCritical
	case strings.Contains(title, "提醒") || strings.Contains(title, "注意") || strings.Contains(strings.ToLower(title), "warning"):
		return NotificationLevelWarning
	default:
		return NotificationLevelInfo
	}
}

func summarizeNotificationContent(content, fallback string, maxRunes int) string {
	text := strings.TrimSpace(stripNotificationMarkup(content))
	if text == "" {
		text = strings.TrimSpace(fallback)
	}
	return truncateRunes(text, maxRunes)
}

func stripNotificationMarkup(s string) string {
	replacer := strings.NewReplacer(
		"\r\n", "\n",
		"\r", "\n",
		"**", "",
		"__", "",
		"`", "",
		"#", "",
		">", "",
		"* ", "",
		"- ", "",
		"<br>", "\n",
		"<br/>", "\n",
		"<br />", "\n",
	)
	s = replacer.Replace(s)
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

func truncateRunes(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if maxRunes <= 0 || utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return strings.TrimSpace(string(runes[:maxRunes])) + "..."
}

func normalizeCardFullCodePath(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "/workspace/") {
		s = strings.TrimPrefix(s, "/workspace")
	}
	if !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	return strings.TrimRight(s, "/")
}

func workspaceFromCardPath(fullCodePath string) (string, string) {
	fullCodePath = normalizeCardFullCodePath(fullCodePath)
	parts := strings.Split(strings.Trim(fullCodePath, "/"), "/")
	if len(parts) < 2 {
		return fullCodePath, strings.Trim(fullCodePath, "/")
	}
	workspacePath := "/" + parts[0] + "/" + parts[1]
	return workspacePath, parts[0] + "/" + parts[1]
}

func workspaceRouteForPath(fullCodePath string, query url.Values) string {
	fullCodePath = normalizeCardFullCodePath(fullCodePath)
	route := "/workspace"
	if fullCodePath != "" {
		route += fullCodePath
	}
	if query != nil && len(query) > 0 {
		route += "?" + query.Encode()
	}
	return route
}

func absoluteCardURL(baseURL, route string) string {
	route = strings.TrimSpace(route)
	if route == "" {
		return ""
	}
	if strings.HasPrefix(route, "http://") || strings.HasPrefix(route, "https://") {
		return route
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return route
	}
	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}
	return baseURL + route
}

func baseNameForCardPath(fullCodePath string) string {
	fullCodePath = normalizeCardFullCodePath(fullCodePath)
	if fullCodePath == "" || fullCodePath == "/" {
		return ""
	}
	name := path.Base(fullCodePath)
	if name == "." || name == "/" {
		return ""
	}
	return name
}

func dedupeNotificationActions(actions []NotificationAction) []NotificationAction {
	seen := make(map[string]struct{}, len(actions))
	out := make([]NotificationAction, 0, len(actions))
	for _, action := range actions {
		if strings.TrimSpace(action.URL) == "" {
			continue
		}
		key := fmt.Sprintf("%s\x00%s", action.Kind, action.URL)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, action)
	}
	return out
}

func normalizeNotificationChannel(channel string) string {
	return strings.ToLower(strings.TrimSpace(channel))
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
