package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
)

const (
	defaultReadWorkspaceArtifactMaxChars = 6000
	maxReadWorkspaceArtifactMaxChars     = 12000
)

type ReadWorkspaceArtifactTool struct{}

type readWorkspaceArtifactArgs struct {
	MessageID int64  `json:"message_id" schema_desc:"当前工作台会话里的 artifact/message 引用 ID；优先使用模型上下文 workspace_artifact_ref 中的 message_id" schema_required:"true"`
	Source    string `json:"source,omitempty" schema_desc:"读取来源：auto/artifact_json/content/result_data/display_content/all。默认 auto，优先返回顶层 artifact JSON，其次 result_data，再其次 content"`
	Offset    int    `json:"offset,omitempty" schema_desc:"按字符分片读取的起点，默认 0；内容较长时用上一段返回的 range.next_offset 继续读取"`
	MaxChars  int    `json:"max_chars,omitempty" schema_desc:"本次每段最多返回字符数，默认 6000，最大 12000；需要全文时分多次读取"`
}

type readWorkspaceArtifactResultData struct {
	Kind         string                         `json:"kind" schema_desc:"固定为 workspace_artifact_read" schema_required:"true"`
	MessageID    int64                          `json:"message_id" schema_desc:"读取的消息 ID" schema_required:"true"`
	SessionID    string                         `json:"session_id,omitempty" schema_desc:"消息所属会话"`
	Role         string                         `json:"role,omitempty" schema_desc:"消息角色"`
	ContextUsage string                         `json:"context_usage,omitempty" schema_desc:"消息上下文用途"`
	ArtifactKind string                         `json:"artifact_kind,omitempty" schema_desc:"产物类型"`
	ToolCallID   string                         `json:"tool_call_id,omitempty" schema_desc:"role=tool 时的 tool_call_id"`
	ToolStatus   string                         `json:"tool_status,omitempty" schema_desc:"role=tool 时的执行状态"`
	Source       string                         `json:"source" schema_desc:"实际读取来源" schema_required:"true"`
	Offset       int                            `json:"offset" schema_desc:"本次读取起点" schema_required:"true"`
	MaxChars     int                            `json:"max_chars" schema_desc:"本次读取上限" schema_required:"true"`
	Text         string                         `json:"text,omitempty" schema_desc:"单来源读取结果"`
	TextSHA      string                         `json:"text_sha,omitempty" schema_desc:"该来源完整文本 sha256"`
	Range        *workspaceArtifactTextRange    `json:"range,omitempty" schema_desc:"单来源分片范围"`
	Fields       []workspaceArtifactReadField   `json:"fields,omitempty" schema_desc:"source=all 时按来源返回的多个字段"`
	Digest       *workspaceArtifactDigest       `json:"digest,omitempty" schema_desc:"从产物中提炼的结构摘要"`
	Reference    *workspaceArtifactReferenceDTO `json:"reference,omitempty" schema_desc:"轻量引用元信息"`
}

type workspaceArtifactTextRange struct {
	Start      int  `json:"start"`
	End        int  `json:"end"`
	TotalChars int  `json:"total_chars"`
	Truncated  bool `json:"truncated"`
	HasMore    bool `json:"has_more"`
	NextOffset int  `json:"next_offset,omitempty"`
}

type workspaceArtifactReadField struct {
	Source  string                      `json:"source"`
	Text    string                      `json:"text,omitempty"`
	TextSHA string                      `json:"text_sha,omitempty"`
	Range   *workspaceArtifactTextRange `json:"range,omitempty"`
}

type workspaceArtifactReferenceDTO struct {
	ReadTool        string `json:"read_tool"`
	MessageID       int64  `json:"message_id"`
	ContentSHA      string `json:"content_sha,omitempty"`
	ContentChars    int    `json:"content_chars,omitempty"`
	ResultDataSHA   string `json:"result_data_sha,omitempty"`
	ResultDataChars int    `json:"result_data_chars,omitempty"`
}

var readWorkspaceArtifactToolDef = toolDefinitionWithOutput[readWorkspaceArtifactArgs, structuredToolResultSchema[readWorkspaceArtifactResultData]](
	workspaceArtifactReadToolName,
	"读取当前工作台会话内 `<workspace_artifact_ref>` 指向的完整产物内容或分片。用于模型上下文只保留 PRD、构建产物、大工具输出摘要时，按 message_id 取回精确 JSON/content。只读、无副作用；默认 source=auto 会优先返回顶层 artifact JSON。",
)

func (t *ReadWorkspaceArtifactTool) Definition() dto.ToolDef {
	return readWorkspaceArtifactToolDef
}

func (t *ReadWorkspaceArtifactTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	_ = ctx
	_ = call
	return toolResult("read_workspace_artifact 只能在工作台会话中由 WorkspaceChatService 读取当前会话消息。", true)
}

func (s *WorkspaceChatService) readWorkspaceArtifactTool(ctx context.Context, rawArgs map[string]interface{}) ToolResult {
	if s == nil || s.messageRepo == nil {
		return toolResult("read_workspace_artifact 失败：消息仓储未初始化。", true)
	}
	args, err := decodeToolArgs[readWorkspaceArtifactArgs](rawArgs)
	if err != nil {
		return toolResult("read_workspace_artifact 参数解析失败: "+err.Error(), true)
	}
	if args.MessageID <= 0 {
		return toolResult("read_workspace_artifact 参数缺失：message_id 必须是当前会话中的消息 ID。", true)
	}
	sessionID := strings.TrimSpace(contextx.GetWorkspaceSessionID(ctx))
	if sessionID == "" {
		return toolResult("read_workspace_artifact 只能在工作台会话中调用：缺少 workspace session 上下文。", true)
	}
	msg, err := s.messageRepo.GetByID(ctx, args.MessageID)
	if err != nil || msg == nil {
		return toolResult(fmt.Sprintf("read_workspace_artifact 找不到消息：message_id=%d", args.MessageID), true)
	}
	if strings.TrimSpace(msg.SessionID) != sessionID {
		return toolResult("read_workspace_artifact 被拒绝：只能读取当前工作台会话内的消息引用。", true)
	}
	if s.sessionRepo != nil {
		if session, err := s.sessionRepo.GetBySessionID(ctx, sessionID); err == nil && session != nil {
			if err := ensureWorkspaceSessionOwner(ctx, session); err != nil {
				return toolResult("read_workspace_artifact 被拒绝："+err.Error(), true)
			}
		}
	}

	maxChars := args.MaxChars
	if maxChars <= 0 {
		maxChars = defaultReadWorkspaceArtifactMaxChars
	}
	if maxChars > maxReadWorkspaceArtifactMaxChars {
		maxChars = maxReadWorkspaceArtifactMaxChars
	}
	offset := args.Offset
	if offset < 0 {
		offset = 0
	}
	source := normalizeReadWorkspaceArtifactSource(args.Source)
	fields := workspaceArtifactReadableFields(msg, source, offset, maxChars)
	if len(fields) == 0 {
		return toolResult("read_workspace_artifact 没有可读取内容：该消息 content/display_content/result_data 均为空。", true)
	}
	data := readWorkspaceArtifactResultData{
		Kind:         workspaceArtifactReadResultKind,
		MessageID:    msg.ID,
		SessionID:    strings.TrimSpace(msg.SessionID),
		Role:         strings.TrimSpace(msg.Role),
		ContextUsage: normalizeMessageContextUsage(msg.ContextUsage),
		ArtifactKind: workspaceMessageArtifactKind(msg),
		ToolCallID:   strings.TrimSpace(msg.ToolCallID),
		ToolStatus:   strings.TrimSpace(msg.ToolStatus),
		Source:       source,
		Offset:       offset,
		MaxChars:     maxChars,
		Digest:       workspaceTrimArtifactDigest(workspaceMessageArtifactDigest(msg)),
		Fields:       fields,
	}
	if source != "all" && len(fields) == 1 {
		data.Source = fields[0].Source
		data.Text = fields[0].Text
		data.TextSHA = fields[0].TextSHA
		data.Range = fields[0].Range
		data.Fields = nil
	}
	if ref, ok := workspaceMessageArtifactReference(msg); ok {
		data.Reference = &workspaceArtifactReferenceDTO{
			ReadTool:        ref.ReadTool,
			MessageID:       ref.MessageID,
			ContentSHA:      ref.ContentSHA,
			ContentChars:    ref.ContentChars,
			ResultDataSHA:   ref.ResultDataSHA,
			ResultDataChars: ref.ResultDataChars,
		}
	}
	return toolResultWithStructuredData(data, false)
}

func normalizeReadWorkspaceArtifactSource(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	switch source {
	case "artifact_json", "content", "result_data", "display_content", "all":
		return source
	default:
		return "auto"
	}
}

func workspaceArtifactReadableFields(msg *model.AgentChatMessage, source string, offset int, maxChars int) []workspaceArtifactReadField {
	if msg == nil {
		return nil
	}
	field := func(source string, text string) (workspaceArtifactReadField, bool) {
		text = strings.TrimSpace(text)
		if text == "" {
			return workspaceArtifactReadField{}, false
		}
		slice, rng := workspaceArtifactTextSlice(text, offset, maxChars)
		return workspaceArtifactReadField{
			Source:  source,
			Text:    slice,
			TextSHA: fileContentSHA(text),
			Range:   rng,
		}, true
	}
	switch source {
	case "artifact_json":
		if item, ok := field("artifact_json", workspaceMessagePrimaryArtifactJSON(msg)); ok {
			return []workspaceArtifactReadField{item}
		}
	case "content":
		if item, ok := field("content", msg.Content); ok {
			return []workspaceArtifactReadField{item}
		}
	case "result_data":
		if msg.ResultData != nil {
			if item, ok := field("result_data", *msg.ResultData); ok {
				return []workspaceArtifactReadField{item}
			}
		}
	case "display_content":
		if item, ok := field("display_content", msg.DisplayContent); ok {
			return []workspaceArtifactReadField{item}
		}
	case "all":
		out := []workspaceArtifactReadField{}
		for _, pair := range []struct {
			source string
			text   string
		}{
			{"artifact_json", workspaceMessagePrimaryArtifactJSON(msg)},
			{"result_data", workspaceMessageResultDataText(msg)},
			{"content", msg.Content},
			{"display_content", msg.DisplayContent},
		} {
			if item, ok := field(pair.source, pair.text); ok {
				out = append(out, item)
			}
		}
		return out
	default:
		for _, pair := range []struct {
			source string
			text   string
		}{
			{"artifact_json", workspaceMessagePrimaryArtifactJSON(msg)},
			{"result_data", workspaceMessageResultDataText(msg)},
			{"content", msg.Content},
			{"display_content", msg.DisplayContent},
		} {
			if item, ok := field(pair.source, pair.text); ok {
				return []workspaceArtifactReadField{item}
			}
		}
	}
	return nil
}

func workspaceMessageResultDataText(msg *model.AgentChatMessage) string {
	if msg == nil || msg.ResultData == nil {
		return ""
	}
	return strings.TrimSpace(*msg.ResultData)
}

func workspaceMessagePrimaryArtifactJSON(msg *model.AgentChatMessage) string {
	if msg == nil {
		return ""
	}
	artifactKind := workspaceMessageArtifactKind(msg)
	if msg.ResultData != nil {
		if workspaceJSONTopLevelKind(*msg.ResultData) != "" {
			return strings.TrimSpace(*msg.ResultData)
		}
	}
	for _, text := range []string{msg.Content, msg.DisplayContent} {
		for _, raw := range workspaceJSONPayloadsFromText(text) {
			topKind := workspaceJSONTopLevelKind(raw)
			if topKind == "" {
				continue
			}
			if artifactKind == "" || topKind == artifactKind {
				return strings.TrimSpace(raw)
			}
		}
	}
	return ""
}

func workspaceJSONTopLevelKind(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return ""
	}
	kind := workspaceStringField(payload, "kind")
	if !workspaceKindLooksLikeArtifact(kind) {
		return ""
	}
	return kind
}

func workspaceArtifactTextSlice(text string, offset int, maxChars int) (string, *workspaceArtifactTextRange) {
	runes := []rune(text)
	total := len(runes)
	if offset > total {
		offset = total
	}
	end := offset + maxChars
	if end > total {
		end = total
	}
	if end < offset {
		end = offset
	}
	rng := &workspaceArtifactTextRange{
		Start:      offset,
		End:        end,
		TotalChars: total,
		Truncated:  offset > 0 || end < total,
		HasMore:    end < total,
	}
	if rng.HasMore {
		rng.NextOffset = end
	}
	return string(runes[offset:end]), rng
}
