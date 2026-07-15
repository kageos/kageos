package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/kageos/kageos/core/agent-server/model"
)

type workspaceHandoffContentInput struct {
	ArtifactKind         string
	ArtifactJSON         string
	HandoffPacketJSON    string
	HandoffContextJSON   string
	PRDExecutionMarkdown string
	ExecuteDirectory     string
	TargetRole           string
	Remark               string
	ContextPolicy        string
}

type workspaceHandoffContextInput struct {
	Source        *model.AgentChatSession
	Messages      []*model.AgentChatMessage
	FullCodePath  string
	TargetRole    string
	ArtifactKind  string
	ArtifactJSON  string
	Remark        string
	ContextPolicy string
}

type workspaceHandoffContext struct {
	SourceSessionID      string                      `json:"source_session_id,omitempty"`
	SourceTitle          string                      `json:"source_title,omitempty"`
	SourceRole           string                      `json:"source_role,omitempty"`
	FullCodePath         string                      `json:"full_code_path,omitempty"`
	WorkspaceDirectory   string                      `json:"workspace_directory,omitempty"`
	TargetAppDirectory   string                      `json:"target_app_directory,omitempty"`
	ExecuteDirectory     string                      `json:"execute_directory,omitempty"`
	Stage                string                      `json:"stage,omitempty"`
	ArtifactKind         string                      `json:"artifact_kind,omitempty"`
	TargetRole           string                      `json:"target_role,omitempty"`
	ContextPolicy        string                      `json:"context_policy,omitempty"`
	ArtifactIncluded     bool                        `json:"artifact_included,omitempty"`
	StageSummary         string                      `json:"stage_summary,omitempty"`
	UserGoal             string                      `json:"user_goal,omitempty"`
	LatestUserNotes      []string                    `json:"latest_user_notes,omitempty"`
	ConfirmedScope       []string                    `json:"confirmed_scope,omitempty"`
	KeyDecisions         []string                    `json:"key_decisions,omitempty"`
	Constraints          []string                    `json:"constraints,omitempty"`
	NonGoals             []string                    `json:"non_goals,omitempty"`
	UserPreferences      []string                    `json:"user_preferences,omitempty"`
	WorkflowNotes        []string                    `json:"workflow_notes,omitempty"`
	DataModelNotes       []string                    `json:"data_model_notes,omitempty"`
	EdgeCases            []string                    `json:"edge_cases,omitempty"`
	OpenQuestions        []string                    `json:"open_questions,omitempty"`
	ImplementationFocus  []string                    `json:"implementation_focus,omitempty"`
	VerificationFocus    []string                    `json:"verification_focus,omitempty"`
	ReferenceDocs        []string                    `json:"reference_docs,omitempty"`
	ReferenceFiles       []string                    `json:"reference_files,omitempty"`
	Remark               string                      `json:"remark,omitempty"`
	ArtifactDigest       *workspaceArtifactDigest    `json:"artifact_digest,omitempty"`
	BuildDiagnostics     *workspaceBuildDiagnostics  `json:"build_diagnostics,omitempty"`
	PRDExecutionMarkdown string                      `json:"prd_execution_markdown,omitempty"`
	ExecutedHooks        []workspaceExecutedRoleHook `json:"executed_hooks,omitempty"`
	HandoffPacket        *workspaceRoleHandoffPacket `json:"handoff_packet,omitempty"`
}

type workspaceArtifactDigest struct {
	ProjectName string                    `json:"project_name,omitempty"`
	ProjectCode string                    `json:"project_code,omitempty"`
	Summary     string                    `json:"summary,omitempty"`
	Tables      []workspaceResourceDigest `json:"tables,omitempty"`
	Forms       []workspaceResourceDigest `json:"forms,omitempty"`
	Charts      []workspaceResourceDigest `json:"charts,omitempty"`
	Rules       []string                  `json:"rules,omitempty"`
}

type workspaceResourceDigest struct {
	Name           string   `json:"name,omitempty"`
	Code           string   `json:"code,omitempty"`
	Desc           string   `json:"desc,omitempty"`
	Fields         []string `json:"fields,omitempty"`
	SearchFields   []string `json:"search_fields,omitempty"`
	Handlers       []string `json:"handlers,omitempty"`
	TargetTable    string   `json:"target_table,omitempty"`
	RequestFields  []string `json:"request_fields,omitempty"`
	ResponseFields []string `json:"response_fields,omitempty"`
	SourceTable    string   `json:"source_table,omitempty"`
	ChartType      string   `json:"chart_type,omitempty"`
	Dimension      string   `json:"dimension,omitempty"`
	Metrics        []string `json:"metrics,omitempty"`
	Filters        []string `json:"filters,omitempty"`
}

func prettyWorkspaceHandoffArtifact(raw json.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return ""
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(trimmed), "", "  "); err == nil {
		return buf.String()
	}
	return trimmed
}

func buildWorkspaceHandoffContextJSON(input workspaceHandoffContextInput) string {
	return formatWorkspaceHandoffContextJSON(buildWorkspaceHandoffContext(input))
}

func formatWorkspaceHandoffContextJSON(ctx workspaceHandoffContext) string {
	raw, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func workspaceHandoffContextForMessage(ctx workspaceHandoffContext) workspaceHandoffContext {
	ctx.PRDExecutionMarkdown = ""
	ctx.HandoffPacket = nil
	return ctx
}

func buildWorkspaceHandoffContext(input workspaceHandoffContextInput) workspaceHandoffContext {
	artifactMap := workspaceJSONMap(input.ArtifactJSON)
	digest := buildWorkspaceArtifactDigest(artifactMap)
	if digest == nil && input.ArtifactKind == workspaceBuildArtifactKind {
		digest = workspaceHandoffPRDDigestFromMessages(input.Messages)
	}
	buildDiagnostics := workspaceHandoffBuildDiagnostics(input.ArtifactKind, artifactMap, input.ArtifactJSON)
	userGoal, latestNotes := summarizeWorkspaceSourceMessages(input.Messages)
	if userGoal == "" && digest != nil {
		userGoal = firstNonEmptyString(digest.Summary, digest.ProjectName)
	}
	workspaceDirectory := workspaceHandoffWorkspaceDirectory(input.FullCodePath, artifactMap)
	targetAppDirectory := workspaceHandoffTargetAppDirectory(input.FullCodePath, artifactMap, digest, input.Messages)
	executeDirectory := workspaceHandoffExecuteDirectory(input.FullCodePath, input.ArtifactKind, input.TargetRole, workspaceDirectory, targetAppDirectory)
	hookOutput := runWorkspaceRoleHooks(workspaceRoleHookInput{
		Stage:              workspaceRoleHookStageBeforeHandoff,
		SourceRole:         workspaceSessionRoleID(input.Source),
		TargetRole:         input.TargetRole,
		ArtifactKind:       input.ArtifactKind,
		Artifact:           artifactMap,
		FullCodePath:       input.FullCodePath,
		WorkspaceDirectory: workspaceDirectory,
		TargetAppDirectory: targetAppDirectory,
		ExecuteDirectory:   executeDirectory,
		Messages:           input.Messages,
	})

	ctx := workspaceHandoffContext{
		FullCodePath:         strings.TrimSpace(input.FullCodePath),
		WorkspaceDirectory:   workspaceDirectory,
		TargetAppDirectory:   targetAppDirectory,
		ExecuteDirectory:     executeDirectory,
		Stage:                workspaceHandoffStage(input.ArtifactKind, input.TargetRole),
		ArtifactKind:         strings.TrimSpace(input.ArtifactKind),
		TargetRole:           strings.TrimSpace(input.TargetRole),
		ContextPolicy:        strings.TrimSpace(input.ContextPolicy),
		ArtifactIncluded:     strings.TrimSpace(input.ArtifactJSON) != "",
		UserGoal:             compactText(userGoal, 240),
		LatestUserNotes:      latestNotes,
		Remark:               strings.TrimSpace(input.Remark),
		ArtifactDigest:       digest,
		BuildDiagnostics:     buildDiagnostics,
		PRDExecutionMarkdown: hookOutput.PRDExecutionMarkdown,
		ExecutedHooks:        hookOutput.ExecutedHooks,
	}
	if input.Source != nil {
		ctx.SourceSessionID = input.Source.SessionID
		ctx.SourceTitle = input.Source.Title
		ctx.SourceRole = workspaceSessionRoleID(input.Source)
	}
	ctx.StageSummary = workspaceHandoffStageSummary(ctx, digest)
	ctx.ConfirmedScope = workspaceHandoffConfirmedScope(digest)
	ctx.KeyDecisions = workspaceHandoffKeyDecisions(input.ArtifactKind, input.TargetRole, digest, input.Remark)
	if placementDecision := workspaceHandoffDirectoryPlacementDecision(ctx); placementDecision != "" {
		ctx.KeyDecisions = appendUniqueRoleHandoffStrings([]string{placementDecision}, ctx.KeyDecisions...)
	}
	ctx.KeyDecisions = append(ctx.KeyDecisions, workspaceHandoffBuildFailureDecisions(input.ArtifactKind, input.TargetRole, buildDiagnostics)...)
	ctx.Constraints = workspaceHandoffConstraints(input.ArtifactKind, input.TargetRole, digest, latestNotes)
	ctx.Constraints = append(ctx.Constraints, workspaceHandoffBuildFailureConstraints(input.ArtifactKind, input.TargetRole)...)
	ctx.NonGoals = workspaceHandoffFilteredNotes(latestNotes, digestRules(digest), []string{"不", "不要", "无需", "暂不", "只读", "禁止"})
	ctx.UserPreferences = workspaceHandoffFilteredNotes(latestNotes, digestRules(digest), []string{"希望", "优先", "默认", "尽量", "需要", "偏好"})
	ctx.WorkflowNotes = workspaceHandoffWorkflowNotes(digest)
	ctx.DataModelNotes = workspaceHandoffDataModelNotes(digest)
	ctx.EdgeCases = workspaceHandoffFilteredNotes(latestNotes, digestRules(digest), []string{"异常", "边界", "权限", "失败", "为空", "重复", "冲突"})
	ctx.OpenQuestions = workspaceHandoffFilteredNotes(latestNotes, digestRules(digest), []string{"?", "？", "待确认", "不确定", "后续确认"})
	ctx.ImplementationFocus = workspaceHandoffImplementationFocus(input.ArtifactKind, input.TargetRole, digest)
	ctx.ImplementationFocus = append(ctx.ImplementationFocus, workspaceHandoffBuildRepairFocus(input.ArtifactKind, input.TargetRole, buildDiagnostics)...)
	ctx.VerificationFocus = workspaceHandoffVerificationFocus(input.ArtifactKind, input.TargetRole, digest)
	ctx.ReferenceDocs = workspaceHandoffReferenceDocs(input.TargetRole, input.ArtifactKind)
	ctx.ReferenceFiles = workspaceHandoffReferenceFiles(input.FullCodePath, workspaceDirectory, targetAppDirectory, input.ArtifactKind, input.TargetRole, digest)
	ctx.HandoffPacket = buildWorkspaceRoleHandoffPacketFromContext(ctx)
	return ctx
}

func workspaceHandoffWorkspaceDirectory(fullCodePath string, artifact map[string]interface{}) string {
	for _, key := range []string{"workspace_path", "workspace_directory"} {
		if path := normalizeWorkspacePath(workspaceStringField(artifact, key)); path != "" {
			return path
		}
	}
	if root := workspaceRootPath(fullCodePath); root != "" {
		return root
	}
	return normalizeWorkspacePath(fullCodePath)
}

func workspaceHandoffTargetAppDirectory(fullCodePath string, artifact map[string]interface{}, digest *workspaceArtifactDigest, messages []*model.AgentChatMessage) string {
	if target := workspaceTargetDirectoryFromPRD(fullCodePath, digest); target != "" {
		return target
	}
	candidates := []string{}
	for _, key := range []string{"target_app_directory", "target_directory", "execute_directory", "full_code_path"} {
		if path := workspaceStringField(artifact, key); path != "" {
			candidates = append(candidates, path)
		}
	}
	candidates = append(candidates, workspaceHandoffPathCandidatesFromMessages(messages)...)
	return workspaceTargetDirectoryFromCandidates(fullCodePath, candidates)
}

func workspaceHandoffExecuteDirectory(fullCodePath, artifactKind, targetRole, workspaceDirectory, targetAppDirectory string) string {
	role := normalizeWorkspaceRole(targetRole)
	if artifactKind == "agent_app_prd" && role == WorkspaceRoleAppDeveloper {
		return firstNonEmptyString(workspacePathDirectory(fullCodePath), normalizeWorkspacePath(workspaceDirectory), normalizeWorkspacePath(fullCodePath))
	}
	if artifactKind == workspaceBuildArtifactKind && role == WorkspaceRoleQAEngineer && targetAppDirectory != "" {
		return targetAppDirectory
	}
	if artifactKind == workspaceBuildFailureKind && role == WorkspaceRoleBuildEngineer {
		return firstNonEmptyString(normalizeWorkspacePath(targetAppDirectory), normalizeWorkspacePath(workspaceDirectory), normalizeWorkspacePath(fullCodePath))
	}
	return normalizeWorkspacePath(fullCodePath)
}

func workspaceHandoffDirectoryPlacementDecision(ctx workspaceHandoffContext) string {
	if ctx.ArtifactKind != "agent_app_prd" || normalizeWorkspaceRole(ctx.TargetRole) != WorkspaceRoleAppDeveloper {
		return ""
	}
	target := normalizeWorkspacePath(ctx.TargetAppDirectory)
	execute := normalizeWorkspacePath(ctx.ExecuteDirectory)
	if target == "" {
		return ""
	}
	if execute != "" && target != execute {
		return fmt.Sprintf("目录落点决策：需要创建目标应用目录 %s；先在父目录 %s 下 create_directory(code=%q)，后续所有代码、预检和构建都必须限定在 %s。", target, execute, path.Base(target), target)
	}
	return fmt.Sprintf("目录落点决策：无需创建子目录；当前目录 %s 就是目标应用目录。", target)
}

func workspaceHandoffPathCandidatesFromMessages(messages []*model.AgentChatMessage) []string {
	out := []string{}
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		out = appendWorkspacePathCandidates(out, msg.DisplayContent)
		out = appendWorkspacePathCandidates(out, msg.Content)
		if msg.ResultData != nil {
			var data interface{}
			if err := json.Unmarshal([]byte(*msg.ResultData), &data); err == nil {
				out = collectWorkspacePathCandidates(out, data)
			}
		}
		if msg.ToolCalls != nil {
			var calls interface{}
			if err := json.Unmarshal([]byte(*msg.ToolCalls), &calls); err == nil {
				out = collectWorkspacePathCandidates(out, calls)
			}
		}
	}
	return out
}

func collectWorkspacePathCandidates(out []string, value interface{}) []string {
	switch v := value.(type) {
	case string:
		return appendWorkspacePathCandidates(out, v)
	case []interface{}:
		for _, item := range v {
			out = collectWorkspacePathCandidates(out, item)
		}
	case map[string]interface{}:
		for _, item := range v {
			out = collectWorkspacePathCandidates(out, item)
		}
	}
	return out
}

func appendWorkspacePathCandidates(out []string, text string) []string {
	for _, path := range workspacePathsFromText(text) {
		out = appendUniqueWorkspaceString(out, path, 0)
	}
	return out
}

func workspaceHandoffTargetSessionFullCodePath(fallbackFullCodePath, executeDirectory, targetRole, artifactKind string) string {
	executeDirectory = normalizeWorkspacePath(executeDirectory)
	fallbackFullCodePath = normalizeWorkspacePath(fallbackFullCodePath)
	if artifactKind == "agent_app_prd" && normalizeWorkspaceRole(targetRole) == WorkspaceRoleAppDeveloper && executeDirectory != "" {
		return executeDirectory
	}
	if artifactKind == workspaceBuildArtifactKind && normalizeWorkspaceRole(targetRole) == WorkspaceRoleQAEngineer && executeDirectory != "" {
		return executeDirectory
	}
	if artifactKind == workspaceBuildFailureKind && normalizeWorkspaceRole(targetRole) == WorkspaceRoleBuildEngineer && executeDirectory != "" {
		return executeDirectory
	}
	return firstNonEmptyString(fallbackFullCodePath, executeDirectory)
}

func workspaceHandoffStage(artifactKind, targetRole string) string {
	switch {
	case artifactKind == "agent_app_prd" && normalizeWorkspaceRole(targetRole) == WorkspaceRoleAppDeveloper:
		return "product_manager_to_app_developer"
	case artifactKind == workspaceBuildArtifactKind && normalizeWorkspaceRole(targetRole) == WorkspaceRoleQAEngineer:
		return "build_engineer_to_qa_engineer"
	case artifactKind == workspaceBuildFailureKind && normalizeWorkspaceRole(targetRole) == WorkspaceRoleBuildEngineer:
		return "build_failure_to_build_engineer"
	default:
		return "workspace_stage_handoff"
	}
}

func workspaceHandoffStageSummary(ctx workspaceHandoffContext, digest *workspaceArtifactDigest) string {
	target := workspaceHandoffRoleLabel(ctx.TargetRole)
	name := ""
	if digest != nil {
		name = firstNonEmptyString(digest.ProjectName, digest.ProjectCode)
	}
	if name == "" {
		name = workspaceHandoffArtifactLabel(ctx.ArtifactKind)
	}
	return fmt.Sprintf("%s 已确认，进入%s阶段；目标模型只接收本交接摘要和结构化产物，不接收来源会话完整历史。", name, target)
}

func summarizeWorkspaceSourceMessages(messages []*model.AgentChatMessage) (string, []string) {
	notes := make([]string, 0, 6)
	seen := map[string]struct{}{}
	firstGoal := ""
	for _, msg := range messages {
		if msg == nil || msg.Role != RoleUser || normalizeMessageContextUsage(msg.ContextUsage) == MessageContextArtifact {
			continue
		}
		text := strings.TrimSpace(msg.DisplayContent)
		if text == "" {
			text = strings.TrimSpace(msg.Content)
		}
		text = compactText(text, 220)
		if text == "" || workspaceHandoffLooksLikeInternalMessage(text) {
			continue
		}
		if firstGoal == "" {
			firstGoal = text
		}
		if _, ok := seen[text]; ok {
			continue
		}
		seen[text] = struct{}{}
		notes = append(notes, text)
		if len(notes) > 8 {
			notes = notes[len(notes)-8:]
		}
	}
	return firstGoal, notes
}

func workspaceHandoffLooksLikeInternalMessage(text string) bool {
	compact := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(text)), " ", "")
	for _, prefix := range []string{"已确认阶段交接产物", "已确认prd", "确认prd", "已构建成功"} {
		if strings.HasPrefix(compact, strings.ReplaceAll(strings.ToLower(prefix), " ", "")) {
			return true
		}
	}
	return false
}

func buildWorkspaceArtifactDigest(artifact map[string]interface{}) *workspaceArtifactDigest {
	if len(artifact) == 0 {
		return nil
	}
	project := workspaceMapField(artifact, "project")
	digest := &workspaceArtifactDigest{
		ProjectName: firstNonEmptyString(workspaceStringField(project, "name"), workspaceStringField(artifact, "project_name")),
		ProjectCode: firstNonEmptyString(workspaceStringField(project, "code"), workspaceStringField(artifact, "project_code")),
		Summary:     firstNonEmptyString(workspaceStringField(project, "summary"), workspaceStringField(artifact, "summary")),
		Tables:      workspaceResourceDigests(artifact, "tables", "table"),
		Forms:       workspaceResourceDigests(artifact, "forms", "form"),
		Charts:      workspaceResourceDigests(artifact, "charts", "chart"),
		Rules:       workspaceRules(artifact),
	}
	if digest.ProjectName == "" && digest.ProjectCode == "" && digest.Summary == "" && len(digest.Tables) == 0 && len(digest.Forms) == 0 && len(digest.Charts) == 0 && len(digest.Rules) == 0 {
		return nil
	}
	return digest
}

func workspaceHandoffPRDDigestFromMessages(messages []*model.AgentChatMessage) *workspaceArtifactDigest {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg == nil {
			continue
		}
		if msg.ResultData != nil {
			var data interface{}
			if err := json.Unmarshal([]byte(*msg.ResultData), &data); err == nil {
				if digest := workspaceArtifactDigestFromValue(data); digest != nil {
					return digest
				}
			}
		}
		for _, text := range []string{msg.Content, msg.DisplayContent} {
			for _, raw := range workspaceJSONBlocksFromText(text) {
				if digest := buildWorkspaceArtifactDigest(workspaceJSONMap(raw)); digest != nil {
					return digest
				}
			}
		}
	}
	return nil
}

func workspaceArtifactDigestFromValue(value interface{}) *workspaceArtifactDigest {
	switch v := value.(type) {
	case map[string]interface{}:
		if digest := buildWorkspaceArtifactDigest(v); digest != nil {
			return digest
		}
		for _, item := range v {
			if digest := workspaceArtifactDigestFromValue(item); digest != nil {
				return digest
			}
		}
	case []interface{}:
		for _, item := range v {
			if digest := workspaceArtifactDigestFromValue(item); digest != nil {
				return digest
			}
		}
	}
	return nil
}

func workspaceJSONBlocksFromText(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	blocks := []string{}
	parts := strings.Split(text, "```json")
	for _, part := range parts[1:] {
		body, _, ok := strings.Cut(part, "```")
		if ok && strings.TrimSpace(body) != "" {
			blocks = append(blocks, strings.TrimSpace(body))
		}
	}
	return blocks
}

func workspaceResourceDigests(artifact map[string]interface{}, key, kind string) []workspaceResourceDigest {
	items := workspaceSliceField(artifact, key)
	out := make([]workspaceResourceDigest, 0, len(items))
	for _, item := range items {
		m := workspaceAsMap(item)
		if len(m) == 0 {
			continue
		}
		d := workspaceResourceDigest{
			Name:           workspaceStringField(m, "name"),
			Code:           workspaceStringField(m, "code"),
			Desc:           workspaceStringField(m, "desc"),
			Fields:         workspaceNamedItems(m, "fields"),
			SearchFields:   workspaceNamedItems(m, "search_fields"),
			Handlers:       workspaceStringItems(m, "handlers"),
			TargetTable:    workspaceStringField(m, "target_table"),
			RequestFields:  workspaceNamedItems(m, "request_fields"),
			ResponseFields: workspaceNamedItems(m, "response_fields"),
			SourceTable:    workspaceStringField(m, "source_table"),
			ChartType:      workspaceStringField(m, "chart_type"),
			Dimension:      workspaceStringField(m, "dimension"),
			Metrics:        workspaceNamedItems(m, "metrics"),
			Filters:        workspaceNamedItems(m, "filters"),
		}
		if kind == "form" && len(d.ResponseFields) == 0 {
			d.ResponseFields = workspaceNamedItems(m, "response")
		}
		out = append(out, d)
	}
	return out
}

func workspaceRules(artifact map[string]interface{}) []string {
	items := workspaceSliceField(artifact, "rules")
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s := compactText(workspaceStringValue(item), 220); s != "" {
			out = appendUniqueWorkspaceString(out, s, 12)
		}
	}
	return out
}
