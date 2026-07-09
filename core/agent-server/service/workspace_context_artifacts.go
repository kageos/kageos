package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/agent-server/model"
)

const (
	workspaceArtifactReadToolName       = "read_workspace_artifact"
	workspaceArtifactReferenceKind      = "workspace_artifact_ref"
	workspaceArtifactReadResultKind     = "workspace_artifact_read"
	workspaceArtifactContextRefLimit    = 6
	workspaceWorkingStateUserNoteLimit  = 3
	workspaceWorkingStateToolErrorLimit = 3
)

type workspaceArtifactReference struct {
	Kind            string                   `json:"kind"`
	MessageID       int64                    `json:"message_id"`
	SessionID       string                   `json:"session_id,omitempty"`
	Role            string                   `json:"role,omitempty"`
	ContextUsage    string                   `json:"context_usage,omitempty"`
	ArtifactKind    string                   `json:"artifact_kind,omitempty"`
	ToolCallID      string                   `json:"tool_call_id,omitempty"`
	ToolStatus      string                   `json:"tool_status,omitempty"`
	ContentSHA      string                   `json:"content_sha,omitempty"`
	ContentChars    int                      `json:"content_chars,omitempty"`
	ResultDataSHA   string                   `json:"result_data_sha,omitempty"`
	ResultDataChars int                      `json:"result_data_chars,omitempty"`
	Summary         string                   `json:"summary,omitempty"`
	Digest          *workspaceArtifactDigest `json:"digest,omitempty"`
	ReadTool        string                   `json:"read_tool"`
	ReadArgs        map[string]int64         `json:"read_args"`
}

func workspaceMessageArtifactReferenceContent(msg *model.AgentChatMessage) (string, bool) {
	ref, ok := workspaceMessageArtifactReference(msg)
	if !ok {
		return "", false
	}
	raw, err := json.MarshalIndent(ref, "", "  ")
	if err != nil {
		return "", false
	}
	return strings.Join([]string{
		"<workspace_artifact_ref>",
		string(raw),
		"</workspace_artifact_ref>",
		"",
		"注意：这是大型工作台产物的摘要引用，不是全文。需要基于该产物开发、修改、测试、判断字段/规则/PRD JSON/构建诊断时，先调用 read_workspace_artifact 读取对应 message_id。",
	}, "\n"), true
}

func workspaceMessageArtifactReference(msg *model.AgentChatMessage) (workspaceArtifactReference, bool) {
	if msg == nil || msg.ID <= 0 || !workspaceMessageLooksLikeArtifact(msg) {
		return workspaceArtifactReference{}, false
	}
	digest := workspaceTrimArtifactDigest(workspaceMessageArtifactDigest(msg))
	content := strings.TrimSpace(msg.Content)
	resultData := ""
	if msg.ResultData != nil {
		resultData = strings.TrimSpace(*msg.ResultData)
	}
	ref := workspaceArtifactReference{
		Kind:            workspaceArtifactReferenceKind,
		MessageID:       msg.ID,
		SessionID:       strings.TrimSpace(msg.SessionID),
		Role:            strings.TrimSpace(msg.Role),
		ContextUsage:    normalizeMessageContextUsage(msg.ContextUsage),
		ArtifactKind:    workspaceMessageArtifactKind(msg),
		ToolCallID:      strings.TrimSpace(msg.ToolCallID),
		ToolStatus:      strings.TrimSpace(msg.ToolStatus),
		ContentChars:    workspaceRuneLen(content),
		ResultDataChars: workspaceRuneLen(resultData),
		Summary:         workspaceMessageArtifactSummary(msg, digest),
		Digest:          digest,
		ReadTool:        workspaceArtifactReadToolName,
		ReadArgs:        map[string]int64{"message_id": msg.ID},
	}
	if content != "" {
		ref.ContentSHA = fileContentSHA(content)
	}
	if resultData != "" {
		ref.ResultDataSHA = fileContentSHA(resultData)
	}
	if ref.ArtifactKind == "" {
		ref.ArtifactKind = "workspace_artifact"
	}
	return ref, true
}

func workspaceMessageLooksLikeArtifact(msg *model.AgentChatMessage) bool {
	if msg == nil {
		return false
	}
	if workspaceMessageIsArtifactReadResult(msg) {
		return false
	}
	if normalizeMessageContextUsage(msg.ContextUsage) == MessageContextArtifact {
		return true
	}
	if strings.TrimSpace(msg.ArtifactKind) != "" {
		return true
	}
	if kind := workspaceMessageArtifactKind(msg); kind != "" {
		return true
	}
	return false
}

func workspaceMessageIsArtifactReadResult(msg *model.AgentChatMessage) bool {
	if msg == nil || msg.ResultData == nil || strings.TrimSpace(*msg.ResultData) == "" {
		return false
	}
	var value interface{}
	if err := json.Unmarshal([]byte(*msg.ResultData), &value); err != nil {
		return false
	}
	return workspaceArtifactKindFromValue(value) == workspaceArtifactReadResultKind
}

func workspaceMessageArtifactKind(msg *model.AgentChatMessage) string {
	if msg == nil {
		return ""
	}
	if kind := strings.TrimSpace(msg.ArtifactKind); kind != "" {
		return kind
	}
	if msg.ResultData != nil {
		if kind := workspaceArtifactKindFromJSON(*msg.ResultData); kind != "" {
			return kind
		}
	}
	for _, raw := range workspaceJSONPayloadsFromText(msg.Content) {
		if kind := workspaceArtifactKindFromJSON(raw); kind != "" {
			return kind
		}
	}
	for _, raw := range workspaceJSONPayloadsFromText(msg.DisplayContent) {
		if kind := workspaceArtifactKindFromJSON(raw); kind != "" {
			return kind
		}
	}
	return ""
}

func workspaceArtifactKindFromJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var value interface{}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return ""
	}
	return workspaceArtifactKindFromValue(value)
}

func workspaceArtifactKindFromValue(value interface{}) string {
	switch v := value.(type) {
	case map[string]interface{}:
		if kind := strings.TrimSpace(workspaceStringField(v, "kind")); kind != "" && workspaceKindLooksLikeArtifact(kind) {
			return kind
		}
		for _, item := range v {
			if kind := workspaceArtifactKindFromValue(item); kind != "" {
				return kind
			}
		}
	case []interface{}:
		for _, item := range v {
			if kind := workspaceArtifactKindFromValue(item); kind != "" {
				return kind
			}
		}
	}
	return ""
}

func workspaceKindLooksLikeArtifact(kind string) bool {
	kind = strings.TrimSpace(kind)
	if kind == "" || kind == workspaceArtifactReadResultKind || kind == workspaceArtifactReferenceKind {
		return false
	}
	return strings.HasPrefix(kind, "agent_app_") || strings.HasPrefix(kind, "workspace_")
}

func workspaceMessageArtifactDigest(msg *model.AgentChatMessage) *workspaceArtifactDigest {
	if msg == nil {
		return nil
	}
	if msg.ResultData != nil {
		if digest := workspaceArtifactDigestFromJSON(*msg.ResultData); digest != nil {
			return digest
		}
	}
	for _, raw := range workspaceJSONPayloadsFromText(msg.Content) {
		if digest := workspaceArtifactDigestFromJSON(raw); digest != nil {
			return digest
		}
	}
	for _, raw := range workspaceJSONPayloadsFromText(msg.DisplayContent) {
		if digest := workspaceArtifactDigestFromJSON(raw); digest != nil {
			return digest
		}
	}
	return nil
}

func workspaceArtifactDigestFromJSON(raw string) *workspaceArtifactDigest {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var value interface{}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil
	}
	return workspaceArtifactDigestFromValue(value)
}

func workspaceJSONPayloadsFromText(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	out := workspaceJSONBlocksFromText(text)
	if strings.HasPrefix(text, "{") {
		out = append(out, text)
	}
	if idx := strings.Index(text, "{"); idx >= 0 {
		candidate := strings.TrimSpace(text[idx:])
		if candidate != text {
			var tmp map[string]interface{}
			if err := json.Unmarshal([]byte(candidate), &tmp); err == nil {
				out = append(out, candidate)
			}
		}
	}
	return out
}

func workspaceMessageArtifactSummary(msg *model.AgentChatMessage, digest *workspaceArtifactDigest) string {
	if summary := workspaceArtifactDigestSummary(digest); summary != "" {
		return compactText(summary, 1200)
	}
	if msg != nil && msg.ResultData != nil {
		if summary := workspaceArtifactGenericSummaryFromJSON(*msg.ResultData); summary != "" {
			return compactText(summary, 800)
		}
	}
	if msg != nil {
		for _, raw := range workspaceJSONPayloadsFromText(msg.Content) {
			if summary := workspaceArtifactGenericSummaryFromJSON(raw); summary != "" {
				return compactText(summary, 800)
			}
		}
		return compactText(firstNonEmptyString(msg.DisplayContent, msg.Content), 600)
	}
	return ""
}

func workspaceArtifactDigestSummary(digest *workspaceArtifactDigest) string {
	if digest == nil {
		return ""
	}
	parts := []string{}
	project := firstNonEmptyString(digest.ProjectName, digest.ProjectCode)
	if project != "" {
		if digest.ProjectName != "" && digest.ProjectCode != "" && digest.ProjectName != digest.ProjectCode {
			project = fmt.Sprintf("%s(%s)", digest.ProjectName, digest.ProjectCode)
		}
		if digest.Summary != "" {
			project += ": " + digest.Summary
		}
		parts = append(parts, "项目="+project)
	} else if digest.Summary != "" {
		parts = append(parts, "摘要="+digest.Summary)
	}
	if labels := workspaceResourceDigestLabels(digest.Tables, 6); len(labels) > 0 {
		parts = append(parts, "Tables="+strings.Join(labels, "，"))
	}
	if labels := workspaceResourceDigestLabels(digest.Forms, 6); len(labels) > 0 {
		parts = append(parts, "Forms="+strings.Join(labels, "，"))
	}
	if labels := workspaceResourceDigestLabels(digest.Charts, 6); len(labels) > 0 {
		parts = append(parts, "Charts="+strings.Join(labels, "，"))
	}
	if len(digest.Rules) > 0 {
		parts = append(parts, "Rules="+strings.Join(trimWorkspaceStrings(digest.Rules, 4), "；"))
	}
	return strings.Join(parts, "\n")
}

func workspaceResourceDigestLabels(items []workspaceResourceDigest, limit int) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		label := firstNonEmptyString(item.Name, item.Code)
		if label == "" {
			continue
		}
		if item.Code != "" && item.Name != "" && item.Code != item.Name {
			label = fmt.Sprintf("%s(%s)", item.Name, item.Code)
		}
		out = append(out, label)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func workspaceArtifactGenericSummaryFromJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var value interface{}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return ""
	}
	return workspaceArtifactGenericSummaryFromValue(value)
}

func workspaceArtifactGenericSummaryFromValue(value interface{}) string {
	switch v := value.(type) {
	case map[string]interface{}:
		kind := workspaceStringField(v, "kind")
		if kind != "" && workspaceKindLooksLikeArtifact(kind) {
			parts := []string{"kind=" + kind}
			for _, key := range []string{"status", "workspace_path", "app", "new_version", "git_commit_hash", "summary", "error"} {
				if item := workspaceStringField(v, key); item != "" {
					parts = append(parts, key+"="+compactText(item, 220))
				}
			}
			if diagnostics := workspaceMapField(v, "build_diagnostics"); len(diagnostics) > 0 {
				if item := workspaceStringField(diagnostics, "error_summary"); item != "" {
					parts = append(parts, "build_error="+compactText(item, 260))
				}
			}
			return strings.Join(parts, "；")
		}
		for _, item := range v {
			if summary := workspaceArtifactGenericSummaryFromValue(item); summary != "" {
				return summary
			}
		}
	case []interface{}:
		for _, item := range v {
			if summary := workspaceArtifactGenericSummaryFromValue(item); summary != "" {
				return summary
			}
		}
	}
	return ""
}

func workspaceTrimArtifactDigest(digest *workspaceArtifactDigest) *workspaceArtifactDigest {
	if digest == nil {
		return nil
	}
	out := &workspaceArtifactDigest{
		ProjectName: compactText(digest.ProjectName, 120),
		ProjectCode: compactText(digest.ProjectCode, 80),
		Summary:     compactText(digest.Summary, 240),
		Tables:      workspaceTrimResourceDigests(digest.Tables, 8),
		Forms:       workspaceTrimResourceDigests(digest.Forms, 8),
		Charts:      workspaceTrimResourceDigests(digest.Charts, 8),
		Rules:       trimWorkspaceStrings(digest.Rules, 8),
	}
	for i := range out.Rules {
		out.Rules[i] = compactText(out.Rules[i], 160)
	}
	return out
}

func workspaceTrimResourceDigests(items []workspaceResourceDigest, limit int) []workspaceResourceDigest {
	if len(items) == 0 {
		return nil
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	out := make([]workspaceResourceDigest, 0, len(items))
	for _, item := range items {
		out = append(out, workspaceResourceDigest{
			Name:           compactText(item.Name, 100),
			Code:           compactText(item.Code, 80),
			Desc:           compactText(item.Desc, 160),
			Fields:         trimWorkspaceStrings(item.Fields, 10),
			SearchFields:   trimWorkspaceStrings(item.SearchFields, 8),
			Handlers:       trimWorkspaceStrings(item.Handlers, 6),
			TargetTable:    compactText(item.TargetTable, 100),
			RequestFields:  trimWorkspaceStrings(item.RequestFields, 10),
			ResponseFields: trimWorkspaceStrings(item.ResponseFields, 10),
			SourceTable:    compactText(item.SourceTable, 100),
			ChartType:      compactText(item.ChartType, 40),
			Dimension:      compactText(item.Dimension, 80),
			Metrics:        trimWorkspaceStrings(item.Metrics, 8),
			Filters:        trimWorkspaceStrings(item.Filters, 8),
		})
	}
	return out
}

func buildWorkspaceWorkingStateBlock(messages []*model.AgentChatMessage, session *model.AgentChatSession, currentTurnMessageID int64) string {
	if len(messages) == 0 && session == nil {
		return ""
	}
	lines := []string{"## 当前会话工作状态（自动压缩）"}
	if session != nil {
		sessionParts := []string{}
		if session.SessionID != "" {
			sessionParts = append(sessionParts, "session_id="+session.SessionID)
		}
		if role := workspaceSessionRoleID(session); role != "" {
			sessionParts = append(sessionParts, "role="+role)
		}
		if session.FullCodePath != "" {
			sessionParts = append(sessionParts, "directory="+session.FullCodePath)
		}
		if len(sessionParts) > 0 {
			lines = append(lines, "- 会话："+strings.Join(sessionParts, "；"))
		}
	}
	if summary := workspaceLatestTaskStateSummary(messages); summary != "" {
		lines = append(lines, "- 最近阶段摘要："+compactText(summary, 700))
	}
	if notes := workspaceLatestUserNotes(messages, currentTurnMessageID, workspaceWorkingStateUserNoteLimit); len(notes) > 0 {
		lines = append(lines, "- 最近用户意图：")
		for _, note := range notes {
			lines = append(lines, "  - "+compactText(note, 240))
		}
	}
	if refs := workspaceLatestArtifactReferences(messages, workspaceArtifactContextRefLimit); len(refs) > 0 {
		lines = append(lines, "- 产物引用（摘要索引，不含全文）：")
		for _, ref := range refs {
			summary := compactText(ref.Summary, 260)
			line := fmt.Sprintf("  - message_id=%d kind=%s", ref.MessageID, firstNonEmptyString(ref.ArtifactKind, "workspace_artifact"))
			if ref.ContentSHA != "" {
				line += " content_sha=" + ref.ContentSHA
			} else if ref.ResultDataSHA != "" {
				line += " result_data_sha=" + ref.ResultDataSHA
			}
			if ref.ContentChars > 0 || ref.ResultDataChars > 0 {
				line += fmt.Sprintf(" chars=%d/%d", ref.ContentChars, ref.ResultDataChars)
			}
			if summary != "" {
				line += " summary=" + summary
			}
			lines = append(lines, line)
		}
	}
	if errors := workspaceLatestToolErrors(messages, workspaceWorkingStateToolErrorLimit); len(errors) > 0 {
		lines = append(lines, "- 最近工具/构建问题：")
		for _, item := range errors {
			lines = append(lines, "  - "+compactText(item, 260))
		}
	}
	lines = append(lines,
		"- 使用规则：`<workspace_artifact_ref>` 和本段产物引用只是索引；需要精确 PRD 字段、业务规则、构建诊断或大工具输出时，先调用 `read_workspace_artifact` 按 message_id 读取原文，再执行开发/修改/测试判断。",
	)
	if len(lines) <= 2 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func workspaceLatestTaskStateSummary(messages []*model.AgentChatMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg == nil || msg.ResultData == nil || strings.TrimSpace(*msg.ResultData) == "" {
			continue
		}
		var value map[string]interface{}
		if err := json.Unmarshal([]byte(*msg.ResultData), &value); err != nil {
			continue
		}
		if _, ok := value["handoff"]; !ok {
			continue
		}
		if summary := workspaceStringField(value, "summary"); summary != "" {
			return summary
		}
	}
	return ""
}

func workspaceLatestUserNotes(messages []*model.AgentChatMessage, currentTurnMessageID int64, limit int) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg == nil || msg.Role != RoleUser {
			continue
		}
		usage := normalizeMessageContextUsage(msg.ContextUsage)
		if usage == MessageContextDisplayOnly || usage == MessageContextArtifact {
			continue
		}
		if usage == MessageContextCurrentTurn && (currentTurnMessageID == 0 || msg.ID != currentTurnMessageID) {
			continue
		}
		text := compactText(firstNonEmptyString(msg.DisplayContent, msg.Content), 260)
		if text == "" || workspaceHandoffLooksLikeInternalMessage(text) {
			continue
		}
		if _, ok := seen[text]; ok {
			continue
		}
		seen[text] = struct{}{}
		out = append([]string{text}, out...)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func workspaceLatestArtifactReferences(messages []*model.AgentChatMessage, limit int) []workspaceArtifactReference {
	out := []workspaceArtifactReference{}
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		ref, ok := workspaceMessageArtifactReference(msg)
		if !ok {
			continue
		}
		out = append([]workspaceArtifactReference{ref}, out...)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func workspaceLatestToolErrors(messages []*model.AgentChatMessage, limit int) []string {
	out := []string{}
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg == nil || msg.Role != RoleTool {
			continue
		}
		kind := workspaceMessageArtifactKind(msg)
		if msg.ToolStatus != ToolCallStatusError && kind != workspaceBuildFailureKind {
			continue
		}
		text := workspaceMessageArtifactSummary(msg, workspaceMessageArtifactDigest(msg))
		if text == "" {
			text = firstNonEmptyString(msg.Content, msg.DisplayContent)
		}
		if text == "" {
			continue
		}
		if msg.ToolCallID != "" {
			text = "tool_call_id=" + msg.ToolCallID + " " + text
		}
		out = append([]string{text}, out...)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}
