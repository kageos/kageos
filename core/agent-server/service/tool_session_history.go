package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
)

const (
	searchSessionHistoryToolName = "search_session_history"
	readSessionMessagesToolName  = "read_session_messages"
	maxSessionHistoryResults     = 20
	maxSessionMessageReadCount   = 10
	defaultSessionReadMaxChars   = 6000
	maxSessionReadMaxChars       = 12000
)

type SearchSessionHistoryTool struct{}
type ReadSessionMessagesTool struct{}

type searchSessionHistoryArgs struct {
	Query         string   `json:"query,omitempty" schema_desc:"要查找的关键词；不确定 message_id 时使用。空关键词可配合消息 ID 范围浏览索引"`
	Roles         []string `json:"roles,omitempty" schema_desc:"可选角色过滤：user/assistant/tool/system"`
	FromMessageID int64    `json:"from_message_id,omitempty" schema_desc:"可选起始消息 ID（包含）"`
	ToMessageID   int64    `json:"to_message_id,omitempty" schema_desc:"可选结束消息 ID（包含）"`
	Limit         int      `json:"limit,omitempty" schema_desc:"最多返回条数，默认 10，最大 20"`
}

type sessionHistoryHit struct {
	MessageID    int64          `json:"message_id" schema_desc:"原始消息 ID" schema_required:"true"`
	Role         string         `json:"role" schema_desc:"消息角色" schema_required:"true"`
	CreatedAt    string         `json:"created_at,omitempty"`
	ContextUsage string         `json:"context_usage,omitempty"`
	ArtifactKind string         `json:"artifact_kind,omitempty"`
	ToolCallID   string         `json:"tool_call_id,omitempty"`
	ToolStatus   string         `json:"tool_status,omitempty"`
	Excerpt      string         `json:"excerpt" schema_desc:"命中附近的简短片段，不代替原文" schema_required:"true"`
	ReadArgs     map[string]any `json:"read_args" schema_desc:"传给 read_session_messages 的参数" schema_required:"true"`
}

type searchSessionHistoryData struct {
	Kind      string              `json:"kind" schema_required:"true"`
	SessionID string              `json:"session_id" schema_required:"true"`
	Query     string              `json:"query,omitempty"`
	Hits      []sessionHistoryHit `json:"hits" schema_required:"true"`
	Count     int                 `json:"count" schema_required:"true"`
}

type readSessionMessagesArgs struct {
	MessageIDs    []int64 `json:"message_ids,omitempty" schema_desc:"要精确读取的消息 ID，最多 10 个；优先使用摘要或 search_session_history 返回的 message_id"`
	FromMessageID int64   `json:"from_message_id,omitempty" schema_desc:"按范围读取时的起始消息 ID（包含）"`
	ToMessageID   int64   `json:"to_message_id,omitempty" schema_desc:"按范围读取时的结束消息 ID（包含）"`
	MaxChars      int     `json:"max_chars,omitempty" schema_desc:"本次所有消息合计最多返回字符数，默认 6000，最大 12000；需要更多时拆分 message_ids 或范围重复读取"`
	OffsetChars   int     `json:"offset_chars,omitempty" schema_desc:"读取单条 message_id 时从第几个字符继续，默认 0；响应 truncated=true 时使用 next_offset_chars 分页读取完整原文"`
}

type sessionMessageRead struct {
	MessageID      int64  `json:"message_id" schema_required:"true"`
	Role           string `json:"role" schema_required:"true"`
	CreatedAt      string `json:"created_at,omitempty"`
	ContextUsage   string `json:"context_usage,omitempty"`
	ArtifactKind   string `json:"artifact_kind,omitempty"`
	ToolCallID     string `json:"tool_call_id,omitempty"`
	ToolStatus     string `json:"tool_status,omitempty"`
	Content        string `json:"content,omitempty"`
	DisplayContent string `json:"display_content,omitempty"`
	ToolCalls      string `json:"tool_calls,omitempty"`
	ResultData     string `json:"result_data,omitempty"`
}

type readSessionMessagesData struct {
	Kind          string               `json:"kind" schema_required:"true"`
	SessionID     string               `json:"session_id" schema_required:"true"`
	Messages      []sessionMessageRead `json:"messages" schema_required:"true"`
	Count         int                  `json:"count" schema_required:"true"`
	Truncated     bool                 `json:"truncated"`
	NextMessageID int64                `json:"next_message_id,omitempty"`
	NextOffset    int                  `json:"next_offset_chars,omitempty"`
}

var searchSessionHistoryToolDef = toolDefinitionWithOutput[searchSessionHistoryArgs, structuredToolResultSchema[searchSessionHistoryData]](
	searchSessionHistoryToolName,
	"搜索当前工作台会话的原始历史消息索引。上下文摘要不够精确、不知道 message_id、需要找回旧决策/需求/错误/工具结果时先调用；只返回片段和 message_id，再用 read_session_messages 精确读取。只能访问当前用户的当前会话，只读无副作用。",
)

var readSessionMessagesToolDef = toolDefinitionWithOutput[readSessionMessagesArgs, structuredToolResultSchema[readSessionMessagesData]](
	readSessionMessagesToolName,
	"按 message_id 或消息 ID 范围精确读取当前工作台会话的原始记录。用于从 `<conversation_checkpoint>` 摘要恢复旧需求、决策、回复或工具结果；原始记录是事实源，摘要有歧义时以本工具返回内容为准。返回 truncated=true 时按 next_message_id/next_offset_chars 继续分页，直到读完。只能访问当前用户的当前会话，只读无副作用。",
)

func (t *SearchSessionHistoryTool) Definition() dto.ToolDef { return searchSessionHistoryToolDef }
func (t *ReadSessionMessagesTool) Definition() dto.ToolDef  { return readSessionMessagesToolDef }

func (t *SearchSessionHistoryTool) Execute(context.Context, ToolCall) ToolResult {
	return toolResult("search_session_history 只能由 WorkspaceChatService 在当前会话中执行。", true)
}

func (t *ReadSessionMessagesTool) Execute(context.Context, ToolCall) ToolResult {
	return toolResult("read_session_messages 只能由 WorkspaceChatService 在当前会话中执行。", true)
}

func (s *WorkspaceChatService) authorizeSessionHistoryRead(ctx context.Context) (string, ToolResult, bool) {
	if s == nil || s.messageRepo == nil || s.sessionRepo == nil {
		return "", toolResult("会话历史消息或会话仓储未初始化。", true), false
	}
	sessionID := strings.TrimSpace(contextx.GetWorkspaceSessionID(ctx))
	if sessionID == "" {
		return "", toolResult("缺少当前 workspace session 上下文。", true), false
	}
	session, err := s.sessionRepo.GetBySessionID(sessionID)
	if err != nil || session == nil {
		return "", toolResult("当前工作台会话不存在。", true), false
	}
	if err := ensureWorkspaceSessionOwner(ctx, session); err != nil {
		return "", toolResult("会话历史读取被拒绝："+err.Error(), true), false
	}
	return sessionID, ToolResult{}, true
}

func (s *WorkspaceChatService) searchSessionHistoryTool(ctx context.Context, rawArgs map[string]interface{}) ToolResult {
	sessionID, rejected, ok := s.authorizeSessionHistoryRead(ctx)
	if !ok {
		return rejected
	}
	args, err := decodeToolArgs[searchSessionHistoryArgs](rawArgs)
	if err != nil {
		return toolResult("search_session_history 参数解析失败: "+err.Error(), true)
	}
	limit := args.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > maxSessionHistoryResults {
		limit = maxSessionHistoryResults
	}
	roleSet := map[string]struct{}{}
	for _, role := range args.Roles {
		role = strings.ToLower(strings.TrimSpace(role))
		if role != "" {
			roleSet[role] = struct{}{}
		}
	}
	query := strings.ToLower(strings.TrimSpace(args.Query))
	messages, err := s.messageRepo.ListBySessionID(sessionID)
	if err != nil {
		return toolResult("search_session_history 查询失败: "+err.Error(), true)
	}
	hits := make([]sessionHistoryHit, 0, limit)
	for i := len(messages) - 1; i >= 0 && len(hits) < limit; i-- {
		msg := messages[i]
		if msg == nil || (args.FromMessageID > 0 && msg.ID < args.FromMessageID) || (args.ToMessageID > 0 && msg.ID > args.ToMessageID) {
			continue
		}
		if len(roleSet) > 0 {
			if _, exists := roleSet[strings.ToLower(strings.TrimSpace(msg.Role))]; !exists {
				continue
			}
		}
		searchable := strings.Join([]string{msg.Content, msg.DisplayContent, workspaceMessageResultDataText(msg), stringValue(msg.ToolCalls)}, "\n")
		if query != "" && !strings.Contains(strings.ToLower(searchable), query) {
			continue
		}
		hits = append(hits, sessionHistoryHit{
			MessageID:    msg.ID,
			Role:         strings.TrimSpace(msg.Role),
			CreatedAt:    time.Time(msg.CreatedAt).Format(time.RFC3339),
			ContextUsage: normalizeMessageContextUsage(msg.ContextUsage),
			ArtifactKind: workspaceMessageArtifactKind(msg),
			ToolCallID:   strings.TrimSpace(msg.ToolCallID),
			ToolStatus:   strings.TrimSpace(msg.ToolStatus),
			Excerpt:      sessionHistoryExcerpt(searchable, query, 260),
			ReadArgs:     map[string]any{"message_ids": []int64{msg.ID}},
		})
	}
	return toolResultWithStructuredData(searchSessionHistoryData{Kind: "session_history_search", SessionID: sessionID, Query: args.Query, Hits: hits, Count: len(hits)}, false)
}

func (s *WorkspaceChatService) readSessionMessagesTool(ctx context.Context, rawArgs map[string]interface{}) ToolResult {
	sessionID, rejected, ok := s.authorizeSessionHistoryRead(ctx)
	if !ok {
		return rejected
	}
	args, err := decodeToolArgs[readSessionMessagesArgs](rawArgs)
	if err != nil {
		return toolResult("read_session_messages 参数解析失败: "+err.Error(), true)
	}
	maxChars := args.MaxChars
	if maxChars <= 0 {
		maxChars = defaultSessionReadMaxChars
	}
	if maxChars > maxSessionReadMaxChars {
		maxChars = maxSessionReadMaxChars
	}
	wanted := map[int64]struct{}{}
	for _, id := range args.MessageIDs {
		if id > 0 && len(wanted) < maxSessionMessageReadCount {
			wanted[id] = struct{}{}
		}
	}
	if len(wanted) == 0 && args.FromMessageID <= 0 && args.ToMessageID <= 0 {
		return toolResult("read_session_messages 需要 message_ids 或消息 ID 范围。", true)
	}
	messages, err := s.messageRepo.ListBySessionID(sessionID)
	if err != nil {
		return toolResult("read_session_messages 查询失败: "+err.Error(), true)
	}
	selected := make([]*model.AgentChatMessage, 0, maxSessionMessageReadCount)
	var nextRangeMessageID int64
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		_, explicit := wanted[msg.ID]
		inRange := len(wanted) == 0 && (args.FromMessageID <= 0 || msg.ID >= args.FromMessageID) && (args.ToMessageID <= 0 || msg.ID <= args.ToMessageID)
		if explicit || inRange {
			if len(selected) >= maxSessionMessageReadCount {
				if len(wanted) == 0 {
					nextRangeMessageID = msg.ID
				}
				break
			}
			selected = append(selected, msg)
		}
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].ID < selected[j].ID })
	remaining := maxChars
	reads := make([]sessionMessageRead, 0, len(selected))
	truncated := false
	var nextMessageID int64
	nextOffset := 0
	offset := 0
	if len(selected) == 1 && args.OffsetChars > 0 {
		offset = args.OffsetChars
	}
	for i, msg := range selected {
		if remaining <= 0 {
			truncated = true
			nextMessageID = msg.ID
			break
		}
		read, used, cut := sessionMessageReadFromModel(msg, remaining, offset)
		reads = append(reads, read)
		remaining -= used
		truncated = truncated || cut
		if cut {
			nextMessageID = msg.ID
			nextOffset = offset + used
			break
		}
		if remaining <= 0 && i+1 < len(selected) {
			nextMessageID = selected[i+1].ID
		}
	}
	if !truncated && nextRangeMessageID > 0 {
		truncated = true
		nextMessageID = nextRangeMessageID
	}
	return toolResultWithStructuredData(readSessionMessagesData{Kind: "session_messages_read", SessionID: sessionID, Messages: reads, Count: len(reads), Truncated: truncated, NextMessageID: nextMessageID, NextOffset: nextOffset}, false)
}

func sessionMessageReadFromModel(msg *model.AgentChatMessage, maxChars int, offsetChars int) (sessionMessageRead, int, bool) {
	read := sessionMessageRead{
		MessageID:    msg.ID,
		Role:         strings.TrimSpace(msg.Role),
		CreatedAt:    time.Time(msg.CreatedAt).Format(time.RFC3339),
		ContextUsage: normalizeMessageContextUsage(msg.ContextUsage),
		ArtifactKind: workspaceMessageArtifactKind(msg),
		ToolCallID:   strings.TrimSpace(msg.ToolCallID),
		ToolStatus:   strings.TrimSpace(msg.ToolStatus),
	}
	fields := []*string{&read.Content, &read.DisplayContent, &read.ToolCalls, &read.ResultData}
	values := []string{msg.Content, msg.DisplayContent, stringValue(msg.ToolCalls), workspaceMessageResultDataText(msg)}
	used := 0
	truncated := false
	if offsetChars < 0 {
		offsetChars = 0
	}
	skip := offsetChars
	for i, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || used >= maxChars {
			continue
		}
		runes := []rune(value)
		if skip >= len(runes) {
			skip -= len(runes)
			continue
		}
		if skip > 0 {
			runes = runes[skip:]
			skip = 0
		}
		allowed := maxChars - used
		if len(runes) > allowed {
			runes = runes[:allowed]
			truncated = true
		}
		*fields[i] = string(runes)
		used += len(runes)
	}
	if !truncated && used >= maxChars {
		total := 0
		for _, value := range values {
			total += len([]rune(strings.TrimSpace(value)))
		}
		truncated = offsetChars+used < total
	}
	return read, used, truncated
}

func sessionHistoryExcerpt(text, query string, maxRunes int) string {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if text == "" {
		return "（空消息）"
	}
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	if query != "" {
		lowerRunes := []rune(strings.ToLower(text))
		queryRunes := []rune(query)
		for i := 0; i+len(queryRunes) <= len(lowerRunes); i++ {
			if string(lowerRunes[i:i+len(queryRunes)]) == query {
				start := i - maxRunes/3
				if start < 0 {
					start = 0
				}
				end := start + maxRunes
				if end > len(runes) {
					end = len(runes)
				}
				return fmt.Sprintf("…%s…", string(runes[start:end]))
			}
		}
	}
	return string(runes[:maxRunes]) + "…"
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
