package service

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/core/agent-server/prompt"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/llms"
)

const (
	workspaceModelContextPlanVersion = "workspace_model_context.v1"
	workspaceModelContextRefLimit    = 80
)

type workspaceModelContextPlanInput struct {
	SessionID                   string
	Round                       int
	FullCodePath                string
	DirectoryName               string
	WorkspaceCtx                *dto.GetWorkspaceContextResp
	Session                     *model.AgentChatSession
	ModeProvider                prompt.WorkspaceModePromptProvider
	ContextPolicy               string
	ParentSessionID             string
	ModelContextAnchorMessageID int64
	AllMessages                 []*model.AgentChatMessage
	ScopedMessages              []*model.AgentChatMessage
	IncludedMessages            []*model.AgentChatMessage
	ExcludedUnsupported         []*model.AgentChatMessage
	RequestedToolNames          []string
	LLMToolNames                []string
	LLMMessageCount             int
	LLMToolCount                int
}

func (s *WorkspaceChatService) buildWorkspaceModelContextPlan(ctx context.Context, input workspaceModelContextPlanInput) *dto.WorkspaceModelContextPlan {
	roleID, roleSource := workspaceModelContextRole(input.Session, input.AllMessages)
	role := dto.WorkspaceModelContextRole{
		ID:     roleID,
		Source: roleSource,
	}
	var docs dto.WorkspaceModelContextDocs
	if definition, ok := workspaceRoleDefinitionFor(roleID); ok {
		role.DisplayName = definition.DisplayName
		role.Responsibility = definition.Responsibility
		role.HandoffRequired = append([]string(nil), definition.HandoffRequired...)
		role.AllowedTools = append([]string(nil), definition.AllowedTools...)
		role.ForbiddenTools = append([]string(nil), definition.ForbiddenTools...)
		role.AllowedTransitions = workspaceModelContextTransitionIDs(definition.AllowedTransitions)
		docs.DocumentPackage = append([]string(nil), definition.DocumentPackage...)
		docs.RequiredDocs = append([]string(nil), definition.RequiredDocs...)
		docs.OptionalDocs = append([]string(nil), definition.OptionalDocs...)
	}
	loadedDocs := loadedGuideDocsFromMessages(ctx, input.ScopedMessages)
	docs.LoadedDocs = sortedWorkspaceModelContextSet(loadedDocs)
	docs.MissingDocs = missingWorkspaceModelContextDocs(docs.DocumentPackage, loadedDocs)

	handoff := latestWorkspaceModelContextHandoff(input.IncludedMessages)
	if handoff != nil && input.FullCodePath == "" {
		input.FullCodePath = handoff.ExecuteDirectory
	}

	modeCode := ""
	if input.ModeProvider != nil {
		modeCode = input.ModeProvider.Code()
	}
	includedRefs, includedTruncated := workspaceModelContextMessageRefs(input.IncludedMessages, "included", workspaceModelContextRefLimit)
	excludedRefs, excludedTruncated := workspaceModelContextMessageRefs(input.ExcludedUnsupported, "unsupported_role", workspaceModelContextRefLimit)
	excludedStored := len(input.ExcludedUnsupported)

	return &dto.WorkspaceModelContextPlan{
		ProtocolVersion: workspaceModelContextPlanVersion,
		SessionID:       strings.TrimSpace(input.SessionID),
		Round:           input.Round,
		ModeCode:        modeCode,
		Role:            role,
		Execution: dto.WorkspaceModelContextExecution{
			FullCodePath:  strings.TrimSpace(input.FullCodePath),
			DirectoryName: firstNonEmptyString(input.DirectoryName, workspaceModelContextDirectoryName(input.WorkspaceCtx)),
			DirectoryCode: workspaceModelContextDirectoryCode(input.WorkspaceCtx),
			DirectoryType: workspaceModelContextDirectoryType(input.WorkspaceCtx),
			ChildrenCount: workspaceModelContextChildrenCount(input.WorkspaceCtx),
			FilesCount:    workspaceModelContextFilesCount(input.WorkspaceCtx),
			ScopePolicy:   workspaceModelContextScopePolicy(roleID, input.FullCodePath, handoff),
		},
		Messages: dto.WorkspaceModelContextMessages{
			ContextPolicy:               firstNonEmptyString(input.ContextPolicy, ContextPolicyFull),
			ModelContextAnchorMessageID: input.ModelContextAnchorMessageID,
			ParentSessionID:             strings.TrimSpace(input.ParentSessionID),
			SourceHistoryPolicy:         workspaceModelContextSourceHistoryPolicy(input.ParentSessionID, input.ModelContextAnchorMessageID),
			SystemMessages:              1,
			LLMMessages:                 input.LLMMessageCount,
			TotalStoredMessages:         len(input.AllMessages),
			IncludedStoredMessages:      len(input.IncludedMessages),
			ExcludedStoredMessages:      excludedStored,
			ExcludedByAnchor:            0,
			ExcludedDisplayOnly:         0,
			Included:                    includedRefs,
			Excluded:                    excludedRefs,
			Truncated:                   includedTruncated || excludedTruncated,
		},
		Handoff: handoff,
		Docs:    docs,
		Tools: dto.WorkspaceModelContextTools{
			RequestedNames: append([]string(nil), input.RequestedToolNames...),
			LLMTools:       append([]string(nil), input.LLMToolNames...),
			LLMToolCount:   input.LLMToolCount,
			Policy:         workspaceModelContextToolPolicy(roleID),
		},
		CachePlan: dto.WorkspaceModelContextCachePlan{
			StablePrefixStrategy: "system_env_role_protocol_handoff_first",
			StablePrefixItems:    workspaceModelContextStablePrefixItems(roleID, input.FullCodePath, handoff),
			ActualUsageField:     "assistant.llm_usage.cached_tokens",
		},
	}
}

func workspaceModelContextRole(session *model.AgentChatSession, messages []*model.AgentChatMessage) (string, string) {
	if roleID := workspaceSessionRoleID(session); roleID != "" {
		return roleID, "session"
	}
	if roleID := latestWorkspaceRoleFromMessages(messages); roleID != "" {
		return roleID, "messages"
	}
	return WorkspaceRoleRouter, "default_router"
}

func workspaceModelContextTransitionIDs(transitions []nextWorkspaceRole) []string {
	out := make([]string, 0, len(transitions))
	for _, item := range transitions {
		roleID := normalizeWorkspaceRole(item.RoleID)
		if roleID != "" && !containsWorkspaceRoleString(out, roleID) {
			out = append(out, roleID)
		}
	}
	return out
}

func toolNamesFromWorkspaceToolDefs(tools []dto.ToolDef) []string {
	out := make([]string, 0, len(tools))
	for _, tool := range tools {
		if name := strings.TrimSpace(tool.Name); name != "" {
			out = append(out, name)
		}
	}
	return out
}

func workspaceModelContextMessageRefs(messages []*model.AgentChatMessage, reason string, limit int) ([]dto.WorkspaceModelContextMessageRef, bool) {
	if limit <= 0 {
		limit = workspaceModelContextRefLimit
	}
	out := make([]dto.WorkspaceModelContextMessageRef, 0, min(len(messages), limit))
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		if len(out) >= limit {
			return out, true
		}
		out = append(out, dto.WorkspaceModelContextMessageRef{
			ID:           msg.ID,
			Role:         msg.Role,
			ContextUsage: normalizeMessageContextUsage(msg.ContextUsage),
			ArtifactKind: strings.TrimSpace(msg.ArtifactKind),
			Reason:       reason,
		})
	}
	return out, len(messages) > len(out)
}

func latestWorkspaceModelContextHandoff(messages []*model.AgentChatMessage) *dto.WorkspaceModelContextHandoff {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg == nil || strings.TrimSpace(msg.Content) == "" {
			continue
		}
		packet := extractWorkspaceRoleHandoffPacketFromText(msg.Content)
		if packet == nil {
			continue
		}
		out := &dto.WorkspaceModelContextHandoff{
			PacketVersion:      packet.Version,
			SourceSessionID:    packet.SourceSessionID,
			SourceRole:         packet.SourceRole,
			TargetRole:         packet.TargetRole,
			ArtifactKind:       packet.ArtifactKind,
			ExecuteDirectory:   packet.ExecuteDirectory,
			WorkspaceDirectory: packet.WorkspaceDirectory,
			TargetAppDirectory: packet.TargetAppDirectory,
			TaskContext:        append([]string(nil), packet.TaskContext...),
			KeyInformation:     append([]string(nil), packet.KeyInformation...),
			References:         append([]string(nil), packet.References...),
			ValidationStatus:   packet.Validation.Status,
		}
		return out
	}
	return nil
}

func extractWorkspaceRoleHandoffPacketFromText(content string) *workspaceRoleHandoffPacket {
	idx := strings.LastIndex(content, "HANDOFF_PACKET JSON:")
	if idx < 0 {
		return nil
	}
	rest := content[idx+len("HANDOFF_PACKET JSON:"):]
	start := strings.Index(rest, "```json")
	if start < 0 {
		return nil
	}
	rest = rest[start+len("```json"):]
	end := strings.Index(rest, "```")
	if end < 0 {
		return nil
	}
	raw := strings.TrimSpace(rest[:end])
	if raw == "" || raw == "{}" {
		return nil
	}
	var packet workspaceRoleHandoffPacket
	if err := json.Unmarshal([]byte(raw), &packet); err != nil {
		return nil
	}
	normalizeAndValidateWorkspaceRoleHandoffPacket(&packet)
	return &packet
}

func sortedWorkspaceModelContextSet(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for item := range set {
		if s := strings.TrimSpace(item); s != "" {
			out = append(out, s)
		}
	}
	slices.Sort(out)
	return out
}

func missingWorkspaceModelContextDocs(documentPackage []string, loaded map[string]struct{}) []string {
	out := make([]string, 0)
	for _, doc := range documentPackage {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		if _, ok := loaded[doc]; !ok && !containsWorkspaceRoleString(out, doc) {
			out = append(out, doc)
		}
	}
	return out
}

func workspaceModelContextDirectoryName(workspaceCtx *dto.GetWorkspaceContextResp) string {
	if workspaceCtx == nil {
		return ""
	}
	return firstNonEmptyString(workspaceCtx.Directory.Name, workspaceCtx.Directory.Code)
}

func workspaceModelContextDirectoryCode(workspaceCtx *dto.GetWorkspaceContextResp) string {
	if workspaceCtx == nil {
		return ""
	}
	return strings.TrimSpace(workspaceCtx.Directory.Code)
}

func workspaceModelContextDirectoryType(workspaceCtx *dto.GetWorkspaceContextResp) string {
	if workspaceCtx == nil {
		return ""
	}
	return strings.TrimSpace(workspaceCtx.Directory.Type)
}

func workspaceModelContextChildrenCount(workspaceCtx *dto.GetWorkspaceContextResp) int {
	if workspaceCtx == nil {
		return 0
	}
	return len(workspaceCtx.Children)
}

func workspaceModelContextFilesCount(workspaceCtx *dto.GetWorkspaceContextResp) int {
	if workspaceCtx == nil {
		return 0
	}
	return len(workspaceCtx.Files)
}

func workspaceModelContextSourceHistoryPolicy(parentSessionID string, anchorID int64) string {
	if strings.TrimSpace(parentSessionID) != "" {
		return "same_session_full_with_parent_reference"
	}
	return "same_session_full"
}

func workspaceModelContextScopePolicy(roleID string, fullCodePath string, handoff *dto.WorkspaceModelContextHandoff) string {
	roleID = normalizeWorkspaceRole(roleID)
	dir := strings.TrimSpace(fullCodePath)
	if handoff != nil && strings.TrimSpace(handoff.ExecuteDirectory) != "" {
		dir = strings.TrimSpace(handoff.ExecuteDirectory)
	}
	switch roleID {
	case WorkspaceRoleQAEngineer, WorkspaceRoleAppOperator:
		return "runtime_tools_default_scope_execute_directory_or_current_app:" + dir
	case WorkspaceRoleAppDeveloper, WorkspaceRoleMaintenanceEngineer, WorkspaceRoleBuildEngineer:
		return "read_write_build_default_scope_execute_directory:" + dir
	default:
		return "route_or_read_only_scope_current_directory:" + dir
	}
}

func workspaceModelContextToolPolicy(roleID string) string {
	if normalizeWorkspaceRole(roleID) == WorkspaceRoleRouter {
		return "mode_tools_with_router_read_only_intent_policy"
	}
	return "mode_tools_with_role_runtime_gate_and_scope_gate"
}

func workspaceModelContextStablePrefixItems(roleID, fullCodePath string, handoff *dto.WorkspaceModelContextHandoff) []string {
	items := []string{
		"workspace_system_prompt",
		"workspace_env:" + strings.TrimSpace(fullCodePath),
	}
	if roleID = normalizeWorkspaceRole(roleID); roleID != "" {
		items = append(items, "role_definition:"+roleID+":"+workspaceRoleDefinitionProtocolVersion)
	}
	if handoff != nil {
		items = append(items, "handoff_packet:"+firstNonEmptyString(handoff.PacketVersion, workspaceRoleHandoffPacketVersion)+":"+handoff.ArtifactKind+":"+handoff.ExecuteDirectory)
	}
	return items
}

func attachLLMUsageToWorkspaceModelContextPlan(plan *dto.WorkspaceModelContextPlan, usage *llms.Usage) {
	if plan == nil {
		return
	}
	result := &dto.WorkspaceModelContextCacheResult{
		Status: "usage_unavailable",
		Source: "llm_usage",
	}
	if usage != nil {
		result.PromptTokens = usage.PromptTokens
		result.CompletionTokens = usage.CompletionTokens
		result.TotalTokens = usage.TotalTokens
		result.CachedTokens = usage.CachedTokens
		result.CachedTokensReported = usage.CachedTokensReported
		if usage.PromptTokens > 0 && usage.CachedTokens > 0 {
			result.CacheHitRatePercent = int(float64(usage.CachedTokens)/float64(usage.PromptTokens)*100 + 0.5)
		}
		switch {
		case !usage.CachedTokensReported:
			result.Status = "not_reported"
		case usage.PromptTokens <= 0:
			result.Status = "usage_unavailable"
		case usage.CachedTokens > 0:
			result.Status = "hit"
		default:
			result.Status = "miss"
		}
	}
	plan.CachePlan.Result = result
}
