package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/core/agent-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
)

var errWorkspaceHandoffAlreadyProcessed = errors.New("workspace handoff already processed")

// CreateWorkspaceHandoff freezes the source conversation for model context and
// creates a clean target session that starts from one structured artifact.
func (s *WorkspaceChatService) CreateWorkspaceHandoff(ctx context.Context, req *dto.WorkspaceHandoffReq) (*dto.WorkspaceHandoffResp, error) {
	if req == nil {
		return nil, fmt.Errorf("handoff 请求不能为空")
	}
	sourceSessionID := strings.TrimSpace(req.SourceSessionID)
	if sourceSessionID == "" {
		return nil, fmt.Errorf("source_session_id 必填")
	}
	source, err := s.sessionRepo.GetBySessionID(sourceSessionID)
	if err != nil || source == nil {
		return nil, fmt.Errorf("来源会话不存在: %s", sourceSessionID)
	}
	user := contextx.GetRequestUser(ctx)
	if user != "" && source.User != "" && source.User != user {
		return nil, fmt.Errorf("不能交接其他用户的会话")
	}
	targetRole := normalizeWorkspaceRole(req.TargetRole)
	if targetRole == "" || !isKnownWorkspaceRole(targetRole) {
		return nil, fmt.Errorf("target_role 不支持: %s", strings.TrimSpace(req.TargetRole))
	}
	artifactKind := strings.TrimSpace(req.ArtifactKind)
	if artifactKind == "" {
		return nil, fmt.Errorf("artifact_kind 必填")
	}
	artifactJSON := prettyWorkspaceHandoffArtifact(req.Artifact)
	if artifactJSON == "" {
		return nil, fmt.Errorf("artifact 不能为空")
	}
	fullCodePath := strings.TrimSpace(req.FullCodePath)
	if fullCodePath == "" {
		fullCodePath = source.FullCodePath
	}
	if fullCodePath == "" {
		return nil, fmt.Errorf("full_code_path 必填")
	}
	contextPolicy := normalizeWorkspaceHandoffContextPolicy(req.ContextPolicy)
	modeCode := normalizeWorkspaceModeCode(source.ModeCode)
	if modeCode == "" {
		modeCode = "dev"
	}
	sourceMessages := []*model.AgentChatMessage{}
	if s.messageRepo != nil {
		sourceMessages, _ = s.messageRepo.ListBySessionID(source.SessionID)
	}
	handoffContext := buildWorkspaceHandoffContext(workspaceHandoffContextInput{
		Source:        source,
		Messages:      sourceMessages,
		FullCodePath:  fullCodePath,
		TargetRole:    targetRole,
		ArtifactKind:  artifactKind,
		ArtifactJSON:  artifactJSON,
		Remark:        req.Remark,
		ContextPolicy: contextPolicy,
	})
	handoffContextJSON := formatWorkspaceHandoffContextJSON(handoffContext)
	handoffContextMessageJSON := formatWorkspaceHandoffContextJSON(workspaceHandoffContextForMessage(handoffContext))
	executeDirectory := firstNonEmptyString(handoffContext.ExecuteDirectory, fullCodePath)
	targetFullCodePath := workspaceHandoffTargetSessionFullCodePath(fullCodePath, executeDirectory, targetRole, artifactKind)

	targetSessionID := uuid.New().String()
	displayContent := strings.TrimSpace(req.DisplayContent)
	if displayContent == "" {
		displayContent = defaultWorkspaceHandoffDisplayContent(artifactKind, targetRole, req.Remark)
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = displayContent
	}
	if len([]rune(title)) > 50 {
		runes := []rune(title)
		title = string(runes[:50]) + "..."
	}
	target := &model.AgentChatSession{
		TreeID:            source.TreeID,
		FullCodePath:      targetFullCodePath,
		Source:            SourceWorkspace,
		SessionID:         targetSessionID,
		Title:             title,
		ModeCode:          modeCode,
		Status:            model.ChatSessionStatusActive,
		RoleID:            targetRole,
		RoleDisplayName:   workspaceRoleDisplayName(targetRole),
		ParentSessionID:   source.SessionID,
		HandoffKind:       artifactKind,
		HandoffTargetRole: targetRole,
		ContextPolicy:     contextPolicy,
		User:              user,
	}
	if target.User == "" {
		target.User = source.User
	}
	target.CreatedBy = user
	target.UpdatedBy = user
	content := buildWorkspaceHandoffContent(workspaceHandoffContentInput{
		ArtifactKind:         artifactKind,
		ArtifactJSON:         artifactJSON,
		HandoffContextJSON:   handoffContextMessageJSON,
		PRDExecutionMarkdown: handoffContext.PRDExecutionMarkdown,
		ExecuteDirectory:     executeDirectory,
		TargetRole:           targetRole,
		Remark:               req.Remark,
		ContextPolicy:        contextPolicy,
	})
	initialMessage := &model.AgentChatMessage{
		SessionID:      targetSessionID,
		Role:           RoleUser,
		Content:        content,
		DisplayContent: displayContent,
		ContextUsage:   MessageContextArtifact,
		ArtifactKind:   artifactKind,
		User:           target.User,
	}
	initialMessage.CreatedBy = user
	initialMessage.UpdatedBy = user
	handoffPacket := &model.WorkspaceHandoffPacket{
		SourceSessionID:    source.SessionID,
		TargetSessionID:    targetSessionID,
		FullCodePath:       targetFullCodePath,
		TargetRole:         targetRole,
		ArtifactKind:       artifactKind,
		ArtifactJSON:       artifactJSON,
		HandoffContextJSON: handoffContextJSON,
		Remark:             strings.TrimSpace(req.Remark),
		ContextPolicy:      contextPolicy,
		User:               target.User,
	}
	handoffPacket.CreatedBy = user
	handoffPacket.UpdatedBy = user
	var existingResp *dto.WorkspaceHandoffResp
	if err := s.sessionRepo.TransactionWithMessagesAndHandoffPackets(func(sessionTx *repository.ChatSessionRepository, messageTx *repository.ChatMessageRepository, handoffTx *repository.WorkspaceHandoffPacketRepository) error {
		if packet, err := handoffTx.FindLatestBySourceAndTarget(source.SessionID, artifactKind, targetRole, target.User); err != nil {
			return fmt.Errorf("查询已有交接包失败: %w", err)
		} else if packet != nil {
			existingResp = buildExistingWorkspaceHandoffResp(packet, sessionTx, messageTx)
			return nil
		}
		archived, err := sessionTx.ArchiveForModelIfActive(source.SessionID, workspaceRoleDisplayName(targetRole), user)
		if err != nil {
			return fmt.Errorf("归档来源会话失败: %w", err)
		}
		if !archived {
			if packet, err := handoffTx.FindLatestBySourceAndTarget(source.SessionID, artifactKind, targetRole, target.User); err != nil {
				return fmt.Errorf("查询已有交接包失败: %w", err)
			} else if packet != nil {
				existingResp = buildExistingWorkspaceHandoffResp(packet, sessionTx, messageTx)
				return nil
			}
			return errWorkspaceHandoffAlreadyProcessed
		}
		if err := sessionTx.Create(target); err != nil {
			return fmt.Errorf("创建交接会话失败: %w", err)
		}
		if err := messageTx.Create(initialMessage); err != nil {
			return fmt.Errorf("创建交接消息失败: %w", err)
		}
		handoffPacket.InitialMessageID = initialMessage.ID
		if err := handoffTx.Create(handoffPacket); err != nil {
			return fmt.Errorf("创建交接包失败: %w", err)
		}
		return nil
	}); err != nil {
		if errors.Is(err, errWorkspaceHandoffAlreadyProcessed) {
			return nil, fmt.Errorf("该阶段交接已处理，请刷新会话列表后从新的阶段会话继续")
		}
		return nil, err
	}
	if existingResp != nil {
		return existingResp, nil
	}

	return &dto.WorkspaceHandoffResp{
		SessionID:       targetSessionID,
		SourceSessionID: source.SessionID,
		TargetRole:      targetRole,
		ArtifactKind:    artifactKind,
		ContextPolicy:   contextPolicy,
		HandoffPacketID: handoffPacket.ID,
		MessageID:       initialMessage.ID,
		Content:         content,
		DisplayContent:  displayContent,
		HandoffContext:  handoffContextJSON,
	}, nil
}

func buildExistingWorkspaceHandoffResp(packet *model.WorkspaceHandoffPacket, sessionRepo *repository.ChatSessionRepository, messageRepo *repository.ChatMessageRepository) *dto.WorkspaceHandoffResp {
	if packet == nil {
		return nil
	}
	resp := &dto.WorkspaceHandoffResp{
		SessionID:       packet.TargetSessionID,
		SourceSessionID: packet.SourceSessionID,
		TargetRole:      packet.TargetRole,
		ArtifactKind:    packet.ArtifactKind,
		ContextPolicy:   packet.ContextPolicy,
		HandoffPacketID: packet.ID,
		MessageID:       packet.InitialMessageID,
		HandoffContext:  packet.HandoffContextJSON,
	}
	if sessionRepo != nil {
		if target, err := sessionRepo.GetBySessionID(packet.TargetSessionID); err == nil && target != nil {
			resp.DisplayContent = target.Title
		}
	}
	if messageRepo != nil && packet.InitialMessageID > 0 {
		if msg, err := messageRepo.GetByID(packet.InitialMessageID); err == nil && msg != nil {
			resp.Content = msg.Content
			resp.DisplayContent = firstNonEmptyString(msg.DisplayContent, resp.DisplayContent)
		}
	}
	return resp
}

type workspaceHandoffContentInput struct {
	ArtifactKind         string
	ArtifactJSON         string
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
	PRDExecutionMarkdown string                      `json:"prd_execution_markdown,omitempty"`
	ExecutedHooks        []workspaceExecutedRoleHook `json:"executed_hooks,omitempty"`
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
	return ctx
}

func buildWorkspaceHandoffContext(input workspaceHandoffContextInput) workspaceHandoffContext {
	artifactMap := workspaceJSONMap(input.ArtifactJSON)
	digest := buildWorkspaceArtifactDigest(artifactMap)
	if digest == nil && input.ArtifactKind == workspaceBuildArtifactKind {
		digest = workspaceHandoffPRDDigestFromMessages(input.Messages)
	}
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
		PRDExecutionMarkdown: hookOutput.PRDExecutionMarkdown,
		ExecutedHooks:        hookOutput.ExecutedHooks,
	}
	if input.Source != nil {
		ctx.SourceSessionID = input.Source.SessionID
		ctx.SourceTitle = input.Source.Title
	}
	ctx.StageSummary = workspaceHandoffStageSummary(ctx, digest)
	ctx.ConfirmedScope = workspaceHandoffConfirmedScope(digest)
	ctx.KeyDecisions = workspaceHandoffKeyDecisions(input.ArtifactKind, input.TargetRole, digest, input.Remark)
	ctx.Constraints = workspaceHandoffConstraints(input.ArtifactKind, input.TargetRole, digest, latestNotes)
	ctx.NonGoals = workspaceHandoffFilteredNotes(latestNotes, digestRules(digest), []string{"不", "不要", "无需", "暂不", "只读", "禁止"})
	ctx.UserPreferences = workspaceHandoffFilteredNotes(latestNotes, digestRules(digest), []string{"希望", "优先", "默认", "尽量", "需要", "偏好"})
	ctx.WorkflowNotes = workspaceHandoffWorkflowNotes(digest)
	ctx.DataModelNotes = workspaceHandoffDataModelNotes(digest)
	ctx.EdgeCases = workspaceHandoffFilteredNotes(latestNotes, digestRules(digest), []string{"异常", "边界", "权限", "失败", "为空", "重复", "冲突"})
	ctx.OpenQuestions = workspaceHandoffFilteredNotes(latestNotes, digestRules(digest), []string{"?", "？", "待确认", "不确定", "后续确认"})
	ctx.ImplementationFocus = workspaceHandoffImplementationFocus(input.ArtifactKind, input.TargetRole, digest)
	ctx.VerificationFocus = workspaceHandoffVerificationFocus(input.ArtifactKind, input.TargetRole, digest)
	ctx.ReferenceDocs = workspaceHandoffReferenceDocs(input.TargetRole, input.ArtifactKind)
	ctx.ReferenceFiles = workspaceHandoffReferenceFiles(input.FullCodePath, workspaceDirectory, targetAppDirectory, input.ArtifactKind, input.TargetRole, digest)
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
		return firstNonEmptyString(normalizeWorkspacePath(workspaceDirectory), normalizeWorkspacePath(fullCodePath))
	}
	if artifactKind == workspaceBuildArtifactKind && role == WorkspaceRoleQAEngineer && targetAppDirectory != "" {
		return targetAppDirectory
	}
	return normalizeWorkspacePath(fullCodePath)
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
	return firstNonEmptyString(fallbackFullCodePath, executeDirectory)
}

func workspaceHandoffStage(artifactKind, targetRole string) string {
	switch {
	case artifactKind == "agent_app_prd" && normalizeWorkspaceRole(targetRole) == WorkspaceRoleAppDeveloper:
		return "product_manager_to_app_developer"
	case artifactKind == workspaceBuildArtifactKind && normalizeWorkspaceRole(targetRole) == WorkspaceRoleQAEngineer:
		return "build_engineer_to_qa_engineer"
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
	for _, prefix := range []string{"已确认阶段交接产物", "已确认prd", "确认prd", "开始测试", "已构建成功"} {
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

func renderWorkspacePRDExecutionMarkdown(artifact map[string]interface{}, executeDirectory, targetAppDirectory string) string {
	if len(artifact) == 0 {
		return ""
	}
	project := workspaceMapField(artifact, "project")
	projectName := firstNonEmptyString(workspaceStringField(project, "name"), workspaceStringField(artifact, "project_name"))
	projectCode := firstNonEmptyString(workspaceStringField(project, "code"), workspaceStringField(artifact, "project_code"))
	projectSummary := firstNonEmptyString(workspaceStringField(project, "summary"), workspaceStringField(artifact, "summary"))

	var b strings.Builder
	title := firstNonEmptyString(projectName, projectCode, "未命名应用")
	b.WriteString("# 已确认 PRD：")
	b.WriteString(workspaceMarkdownHeading(title))
	b.WriteString("\n\n")
	workspaceWriteMarkdownTable(&b, []string{"项", "内容"}, [][]string{
		{"项目名称", projectName},
		{"项目 code", projectCode},
		{"业务目标", projectSummary},
		{"执行目录", executeDirectory},
		{"目标应用目录", targetAppDirectory},
	})

	tables := workspaceMapsFromSlice(workspaceSliceField(artifact, "tables"))
	forms := workspaceMapsFromSlice(workspaceSliceField(artifact, "forms"))
	charts := workspaceMapsFromSlice(workspaceSliceField(artifact, "charts"))
	maintainTables, readonlyTables := workspaceSplitPRDTables(tables)

	resourceRows := [][]string{}
	for _, table := range maintainTables {
		resourceRows = append(resourceRows, []string{"可维护 Table", workspacePRDResourceName(table), workspaceStringField(table, "code"), workspaceJoinOrDash(workspacePRDFieldNames(table, "fields")), workspaceJoinOrDash(workspaceStringItems(table, "handlers"))})
	}
	for _, form := range forms {
		resourceRows = append(resourceRows, []string{"Form", workspacePRDResourceName(form), workspaceStringField(form, "code"), workspaceJoinOrDash(workspacePRDFieldNames(form, "request_fields")), firstNonEmptyString(workspaceStringField(form, "target_table"), "-")})
	}
	for _, table := range readonlyTables {
		resourceRows = append(resourceRows, []string{"只读 Table", workspacePRDResourceName(table), workspaceStringField(table, "code"), workspaceJoinOrDash(workspacePRDFieldNames(table, "fields")), "只读查询"})
	}
	for _, chart := range charts {
		resourceRows = append(resourceRows, []string{"Chart", workspacePRDResourceName(chart), workspaceStringField(chart, "code"), workspaceJoinOrDash(workspaceNamedItems(chart, "metrics")), firstNonEmptyString(workspaceStringField(chart, "source_table"), "-")})
	}
	if len(resourceRows) > 0 {
		b.WriteString("\n\n## 资源总览\n\n")
		b.WriteString("生成顺序：可维护 Table -> Form -> 只读 Table -> Chart。\n\n")
		workspaceWriteMarkdownTable(&b, []string{"类型", "名称", "code", "核心字段/指标", "操作/目标"}, resourceRows)
	}

	for _, table := range maintainTables {
		workspaceWritePRDTableSection(&b, table)
	}
	for _, form := range forms {
		workspaceWritePRDFormSection(&b, form)
	}
	for _, table := range readonlyTables {
		workspaceWritePRDTableSection(&b, table)
	}
	for _, chart := range charts {
		workspaceWritePRDChartSection(&b, chart)
	}
	workspaceWritePRDRulesSection(&b, workspaceRules(artifact))
	return strings.TrimSpace(b.String())
}

func workspaceMapsFromSlice(items []interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		m := workspaceAsMap(item)
		if len(m) > 0 {
			out = append(out, m)
		}
	}
	return out
}

func workspaceSplitPRDTables(tables []map[string]interface{}) ([]map[string]interface{}, []map[string]interface{}) {
	maintainTables := []map[string]interface{}{}
	readonlyTables := []map[string]interface{}{}
	for _, table := range tables {
		if len(workspaceStringItems(table, "handlers")) > 0 {
			maintainTables = append(maintainTables, table)
		} else {
			readonlyTables = append(readonlyTables, table)
		}
	}
	return maintainTables, readonlyTables
}

func workspaceWritePRDTableSection(b *strings.Builder, table map[string]interface{}) {
	name := workspacePRDResourceName(table)
	b.WriteString("\n\n## Table：")
	b.WriteString(workspaceMarkdownHeading(name))
	b.WriteString("\n\n")
	handlers := workspaceStringItems(table, "handlers")
	operation := "只读查询"
	if len(handlers) > 0 {
		operation = strings.Join(handlers, "、")
	}
	workspaceWriteMarkdownTable(b, []string{"项", "内容"}, [][]string{
		{"标题", firstNonEmptyString(workspaceStringField(table, "title"), name)},
		{"说明", workspaceStringField(table, "desc")},
		{"操作能力", operation},
		{"搜索字段", workspaceJoinOrDash(workspacePRDFieldNames(table, "search_fields"))},
	})
	workspaceWritePRDFieldTable(b, "字段", workspaceSliceField(table, "fields"))
	workspaceWritePRDFieldTable(b, "搜索字段", workspaceSliceField(table, "search_fields"))
	workspaceWritePRDExamplesTable(b, "示例数据", workspaceSliceField(table, "examples"))
}

func workspaceWritePRDFormSection(b *strings.Builder, form map[string]interface{}) {
	name := workspacePRDResourceName(form)
	b.WriteString("\n\n## Form：")
	b.WriteString(workspaceMarkdownHeading(name))
	b.WriteString("\n\n")
	workspaceWriteMarkdownTable(b, []string{"项", "内容"}, [][]string{
		{"说明", workspaceStringField(form, "desc")},
		{"目标表", firstNonEmptyString(workspaceStringField(form, "target_table"), "纯处理型 Form")},
		{"请求字段", workspaceJoinOrDash(workspacePRDFieldNames(form, "request_fields"))},
		{"响应字段", workspaceJoinOrDash(workspacePRDFieldNames(form, "response_fields"))},
	})
	workspaceWritePRDFieldTable(b, "请求字段", workspaceSliceField(form, "request_fields"))
	workspaceWritePRDFieldTable(b, "响应字段", workspaceSliceField(form, "response_fields"))
	workspaceWritePRDExamplesTable(b, "提交示例", []interface{}{form["example"]})
}

func workspaceWritePRDChartSection(b *strings.Builder, chart map[string]interface{}) {
	name := workspacePRDResourceName(chart)
	b.WriteString("\n\n## Chart：")
	b.WriteString(workspaceMarkdownHeading(name))
	b.WriteString("\n\n")
	workspaceWriteMarkdownTable(b, []string{"项", "内容"}, [][]string{
		{"说明", workspaceStringField(chart, "desc")},
		{"来源表", workspaceStringField(chart, "source_table")},
		{"图表类型", workspaceStringField(chart, "chart_type")},
		{"维度", workspaceStringField(chart, "dimension")},
		{"指标", workspaceJoinOrDash(workspaceNamedItems(chart, "metrics"))},
	})
	workspaceWritePRDFieldTable(b, "筛选字段", workspaceSliceField(chart, "filters"))
	workspaceWritePRDExamplesTable(b, "图表示例", workspaceSliceField(chart, "examples"))
}

func workspaceWritePRDRulesSection(b *strings.Builder, rules []string) {
	b.WriteString("\n\n## 业务规则与复杂逻辑\n\n")
	rows := [][]string{}
	for i, rule := range rules {
		rows = append(rows, []string{fmt.Sprintf("R%d", i+1), rule, "必须实现并测试"})
	}
	if len(rows) == 0 {
		rows = append(rows, []string{"R0", "PRD 未写明复杂逻辑", "开发前若发现状态计算、重复提交、权限、只读、跨表写入、统计口径或异常边界不明确，先补充确认。"})
	}
	workspaceWriteMarkdownTable(b, []string{"编号", "规则", "开发要求"}, rows)
}

func workspaceWritePRDFieldTable(b *strings.Builder, title string, items []interface{}) {
	if len(items) == 0 {
		return
	}
	rows := [][]string{}
	for _, item := range items {
		field := workspaceAsMap(item)
		if len(field) == 0 {
			if name := workspaceStringValue(item); name != "" {
				rows = append(rows, []string{name, "-", "-", "-", "-"})
			}
			continue
		}
		rows = append(rows, []string{
			workspaceStringField(field, "name"),
			workspaceStringField(field, "widget"),
			workspaceRequiredText(field["required"]),
			workspaceStringField(field, "desc"),
			workspaceStringField(field, "hide"),
		})
	}
	if len(rows) == 0 {
		return
	}
	b.WriteString("\n\n### ")
	b.WriteString(workspaceMarkdownHeading(title))
	b.WriteString("\n\n")
	workspaceWriteMarkdownTable(b, []string{"字段", "组件", "必填", "说明", "展示限制"}, rows)
}

func workspaceWritePRDExamplesTable(b *strings.Builder, title string, items []interface{}) {
	rows := [][]string{}
	for _, item := range items {
		if item == nil {
			continue
		}
		if text := workspaceCompactJSON(item, 500); text != "" && text != "null" && text != "{}" {
			rows = append(rows, []string{fmt.Sprintf("E%d", len(rows)+1), text})
		}
	}
	if len(rows) == 0 {
		return
	}
	b.WriteString("\n\n### ")
	b.WriteString(workspaceMarkdownHeading(title))
	b.WriteString("\n\n")
	workspaceWriteMarkdownTable(b, []string{"序号", "内容"}, rows)
}

func workspacePRDResourceName(m map[string]interface{}) string {
	return firstNonEmptyString(workspaceStringField(m, "name"), workspaceStringField(m, "title"), workspaceStringField(m, "code"), "未命名资源")
}

func workspacePRDFieldNames(m map[string]interface{}, key string) []string {
	items := workspaceSliceField(m, key)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s := workspaceStringValue(item); s != "" {
			out = appendUniqueWorkspaceString(out, s, 18)
		}
	}
	return out
}

func workspaceRequiredText(value interface{}) string {
	switch v := value.(type) {
	case bool:
		if v {
			return "是"
		}
		return "否"
	case string:
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return "否"
}

func workspaceJoinOrDash(items []string) string {
	if len(items) == 0 {
		return "-"
	}
	return strings.Join(items, "、")
}

func workspaceCompactJSON(value interface{}, maxRunes int) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return compactText(workspaceStringValue(value), maxRunes)
	}
	return compactText(string(raw), maxRunes)
}

func workspaceWriteMarkdownTable(b *strings.Builder, headers []string, rows [][]string) {
	if len(headers) == 0 || len(rows) == 0 {
		return
	}
	b.WriteString("| ")
	for i, header := range headers {
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(workspaceMarkdownCell(header))
	}
	b.WriteString(" |\n| ")
	for i := range headers {
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString("---")
	}
	b.WriteString(" |\n")
	for _, row := range rows {
		b.WriteString("| ")
		for i := range headers {
			if i > 0 {
				b.WriteString(" | ")
			}
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			b.WriteString(workspaceMarkdownCell(cell))
		}
		b.WriteString(" |\n")
	}
}

func workspaceMarkdownHeading(s string) string {
	s = compactText(s, 120)
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func workspaceMarkdownCell(s string) string {
	s = compactText(s, 360)
	if s == "" {
		return "-"
	}
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", "<br>")
	return s
}

func workspaceHandoffConfirmedScope(digest *workspaceArtifactDigest) []string {
	if digest == nil {
		return nil
	}
	out := []string{}
	if project := firstNonEmptyString(digest.ProjectName, digest.Summary); project != "" {
		out = append(out, "项目："+project)
	}
	for _, table := range digest.Tables {
		out = append(out, compactText("表格："+resourceName(table)+"；字段："+strings.Join(table.Fields, "、"), 220))
	}
	for _, form := range digest.Forms {
		note := "表单：" + resourceName(form)
		if form.TargetTable != "" {
			note += "；写入：" + form.TargetTable
		}
		out = append(out, compactText(note, 220))
	}
	for _, chart := range digest.Charts {
		note := "图表：" + resourceName(chart)
		if chart.SourceTable != "" {
			note += "；来源：" + chart.SourceTable
		}
		if chart.ChartType != "" {
			note += "；类型：" + chart.ChartType
		}
		out = append(out, compactText(note, 220))
	}
	return trimWorkspaceStrings(out, 12)
}

func workspaceHandoffKeyDecisions(artifactKind, targetRole string, digest *workspaceArtifactDigest, remark string) []string {
	role := normalizeWorkspaceRole(targetRole)
	out := []string{workspaceHandoffBaseDecision(artifactKind, role)}
	switch {
	case artifactKind == "agent_app_prd" && role == WorkspaceRoleAppDeveloper:
		out = appendUniqueWorkspaceString(out, "完整 PRD JSON 和 PRD_EXECUTION_MARKDOWN 已随本次 agent_app_prd artifact 传入；JSON 是唯一精确需求源，Markdown 是开发执行视图。", 12)
	case artifactKind == workspaceBuildArtifactKind && role == WorkspaceRoleQAEngineer:
		out = appendUniqueWorkspaceString(out, "构建产物 JSON 已随本次 agent_app_build artifact 传入；测试以目标应用目录和函数 schema 为准。", 12)
	}
	for _, rule := range digestRules(digest) {
		out = appendUniqueWorkspaceString(out, rule, 12)
	}
	if remark = strings.TrimSpace(remark); remark != "" {
		out = appendUniqueWorkspaceString(out, "用户确认备注："+compactText(remark, 220), 12)
	}
	return out
}

func workspaceHandoffBaseDecision(artifactKind, role string) string {
	switch {
	case artifactKind == "agent_app_prd" && role == WorkspaceRoleAppDeveloper:
		return "已确认 PRD，开发阶段不重新设计 PRD、不再次询问确认，除非结构化产物缺失关键字段。"
	case artifactKind == workspaceBuildArtifactKind && role == WorkspaceRoleQAEngineer:
		return "已确认构建产物，测试阶段不修改代码、不重新 build；按交接产物和当前工作区函数清单验证。"
	default:
		target := workspaceHandoffRoleLabel(role)
		return fmt.Sprintf("已确认%s，进入%s阶段；下一角色只基于交接摘要、结构化产物、用户补充备注和参考资料推进。", workspaceHandoffConfirmedArtifactText(artifactKind), target)
	}
}

func workspaceHandoffConstraints(artifactKind, targetRole string, digest *workspaceArtifactDigest, notes []string) []string {
	role := normalizeWorkspaceRole(targetRole)
	out := workspaceHandoffBaseConstraints(artifactKind, role)
	if artifactKind == "agent_app_prd" && digest != nil && len(digest.Tables) > 0 {
		out = append(out, "tables.fields 是业务模型字段；tables.search_fields 是查询请求字段，不自动生成业务列。")
	}
	out = append(out, workspaceHandoffFilteredNotes(notes, digestRules(digest), []string{"必须", "不能", "不要", "只允许", "限制"})...)
	return trimWorkspaceStrings(out, 12)
}

func workspaceHandoffBaseConstraints(artifactKind, role string) []string {
	switch {
	case artifactKind == "agent_app_prd" && role == WorkspaceRoleAppDeveloper:
		return []string{
			"业务能力必须落地为 Kageos SDK Go 应用，不生成独立 HTML/CSS/JS 页面。",
			"PRD v2 只消费 project/tables/forms/charts/rules；不要回退到旧 models/functions/workflow。",
		}
	case artifactKind == workspaceBuildArtifactKind && role == WorkspaceRoleQAEngineer:
		return []string{
			"测试阶段只验证现有构建产物和工作区函数，不直接修改业务代码。",
			"测试结论必须区分测试数据问题、业务缺陷、构建/schema 问题和环境问题。",
		}
	default:
		return []string{
			"不要携带来源会话完整历史，只依据交接摘要、结构化产物、用户补充备注和参考资料推进。",
		}
	}
}

func workspaceHandoffArtifactLabel(artifactKind string) string {
	switch strings.TrimSpace(artifactKind) {
	case "agent_app_prd":
		return "PRD"
	case workspaceBuildArtifactKind:
		return "构建产物"
	case "":
		return "阶段产物"
	default:
		return artifactKind
	}
}

func workspaceHandoffConfirmedArtifactText(artifactKind string) string {
	label := workspaceHandoffArtifactLabel(artifactKind)
	if label == "PRD" || strings.Contains(label, "_") {
		return " " + label
	}
	return label
}

func workspaceHandoffRoleLabel(role string) string {
	role = normalizeWorkspaceRole(role)
	if label := strings.TrimSpace(workspaceRoleDisplayName(role)); label != "" {
		return label
	}
	if role != "" {
		return role
	}
	return "下一角色"
}

func workspaceHandoffWorkflowNotes(digest *workspaceArtifactDigest) []string {
	if digest == nil {
		return nil
	}
	out := []string{}
	if len(digest.Tables) > 0 {
		out = append(out, "先生成基础/配置表和可维护 Table。")
	}
	for _, form := range digest.Forms {
		if form.TargetTable != "" {
			out = append(out, compactText("Form "+resourceName(form)+" 提交后应写入目标表 "+form.TargetTable+"，再通过对应记录表查询验证。", 220))
		}
	}
	for _, chart := range digest.Charts {
		if chart.SourceTable != "" {
			out = append(out, compactText("Chart "+resourceName(chart)+" 基于 "+chart.SourceTable+" 统计，不能只返回静态示例。", 220))
		}
	}
	return trimWorkspaceStrings(out, 12)
}

func workspaceHandoffDataModelNotes(digest *workspaceArtifactDigest) []string {
	if digest == nil {
		return nil
	}
	out := []string{}
	for _, table := range digest.Tables {
		if len(table.Fields) > 0 {
			out = append(out, compactText(resourceName(table)+" 业务字段："+strings.Join(table.Fields, "、"), 220))
		}
		if len(table.SearchFields) > 0 {
			out = append(out, compactText(resourceName(table)+" 查询字段："+strings.Join(table.SearchFields, "、"), 220))
		}
		if len(table.Handlers) == 0 {
			out = append(out, resourceName(table)+" 为只读查询表，不要补新增、编辑、删除。")
		}
	}
	return trimWorkspaceStrings(out, 16)
}

func workspaceHandoffImplementationFocus(artifactKind, targetRole string, digest *workspaceArtifactDigest) []string {
	role := normalizeWorkspaceRole(targetRole)
	if artifactKind != "agent_app_prd" || role != WorkspaceRoleAppDeveloper {
		return nil
	}
	out := []string{
		"先使用 change_role 返回的 app_developer 角色文档包和 SDK 主文档；再读取 1 到多个匹配案例。",
		"按 PRD 派生目录、Go 文件、Table/Form/Chart 路由和 build，不重新输出 PRD。",
		"严格区分业务字段和搜索字段；创建开始时间/创建结束时间/创建人默认映射系统字段。",
	}
	if digest != nil {
		if len(digest.Forms) > 0 {
			out = append(out, "Form 必须按 target_table 产生可查询记录；目标记录表默认只读，除非 PRD 明确维护能力。")
		}
		if len(digest.Charts) > 0 {
			out = append(out, "Chart 必须基于 source_table、filters 和 examples 实现真实统计。")
		}
	}
	return out
}

func workspaceHandoffVerificationFocus(artifactKind, targetRole string, digest *workspaceArtifactDigest) []string {
	out := []string{}
	if artifactKind == "agent_app_prd" && normalizeWorkspaceRole(targetRole) == WorkspaceRoleAppDeveloper {
		out = append(out, "build 成功后进入测试阶段，按基础表 -> Form -> 记录表 -> Chart 顺序验证。")
	}
	if digest != nil {
		for _, table := range digest.Tables {
			if len(table.SearchFields) > 0 {
				out = append(out, compactText("验证 "+resourceName(table)+" 的核心筛选："+strings.Join(table.SearchFields, "、"), 220))
			}
		}
		for _, form := range digest.Forms {
			if form.TargetTable != "" {
				out = append(out, compactText("验证 "+resourceName(form)+" 提交后在 "+form.TargetTable+" 可查到记录。", 220))
			}
		}
	}
	return trimWorkspaceStrings(out, 12)
}

func workspaceHandoffReferenceDocs(targetRole, artifactKind string) []string {
	role := normalizeWorkspaceRole(targetRole)
	spec, _ := workspaceRoleSpecFor(role)
	docs := roleDocumentPackage(role, spec)
	if artifactKind == "agent_app_prd" && role == WorkspaceRoleAppDeveloper {
		docs = appendUniqueWorkspaceString(docs, "/system/prompt/case_catalog", 0)
		docs = appendUniqueWorkspaceString(docs, "/system/prompt/sdk/agent-app-sdk-readme", 0)
	}
	if artifactKind == workspaceBuildArtifactKind && role == WorkspaceRoleQAEngineer {
		docs = appendUniqueWorkspaceString(docs, "/system/prompt/roles/qa-engineer", 0)
	}
	return trimWorkspaceStrings(docs, 12)
}

func workspaceHandoffReferenceFiles(fullCodePath, workspaceDirectory, targetAppDirectory, artifactKind, targetRole string, digest *workspaceArtifactDigest) []string {
	out := []string{}
	role := normalizeWorkspaceRole(targetRole)
	if artifactKind == "agent_app_prd" && role == WorkspaceRoleAppDeveloper {
		if path := strings.TrimSpace(workspaceDirectory); path != "" {
			out = appendUniqueWorkspaceString(out, path, 16)
		}
	} else if path := strings.TrimSpace(fullCodePath); path != "" {
		out = appendUniqueWorkspaceString(out, path, 16)
	}
	if path := strings.TrimSpace(targetAppDirectory); path != "" {
		out = appendUniqueWorkspaceString(out, path, 16)
	}
	if digest != nil {
		for _, table := range digest.Tables {
			if table.Code != "" {
				out = appendUniqueWorkspaceString(out, workspaceHandoffReferenceFilePath(targetAppDirectory, table.Code+".go"), 16)
			}
		}
		for _, form := range digest.Forms {
			if form.Code != "" {
				out = appendUniqueWorkspaceString(out, workspaceHandoffReferenceFilePath(targetAppDirectory, form.Code+".go"), 16)
			}
		}
		for _, chart := range digest.Charts {
			if chart.Code != "" {
				out = appendUniqueWorkspaceString(out, workspaceHandoffReferenceFilePath(targetAppDirectory, chart.Code+".go"), 16)
			}
		}
	}
	return out
}

func workspaceHandoffReferenceFilePath(targetAppDirectory, fileName string) string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return ""
	}
	if target := normalizeWorkspacePath(targetAppDirectory); target != "" {
		return target + "/" + fileName
	}
	return fileName
}

func workspaceHandoffFilteredNotes(notes []string, rules []string, keywords []string) []string {
	out := []string{}
	for _, item := range append(notes, rules...) {
		if workspaceContainsAny(item, keywords) {
			out = appendUniqueWorkspaceString(out, compactText(item, 220), 8)
		}
	}
	return out
}

func workspaceContainsAny(s string, keywords []string) bool {
	for _, keyword := range keywords {
		if keyword != "" && strings.Contains(s, keyword) {
			return true
		}
	}
	return false
}

func digestRules(digest *workspaceArtifactDigest) []string {
	if digest == nil {
		return nil
	}
	return digest.Rules
}

func resourceName(r workspaceResourceDigest) string {
	return firstNonEmptyString(r.Name, r.Code, "未命名资源")
}

func workspaceJSONMap(raw string) map[string]interface{} {
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &out); err != nil {
		return map[string]interface{}{}
	}
	return out
}

func workspaceMapField(m map[string]interface{}, key string) map[string]interface{} {
	return workspaceAsMap(m[key])
}

func workspaceSliceField(m map[string]interface{}, key string) []interface{} {
	if v, ok := m[key].([]interface{}); ok {
		return v
	}
	return nil
}

func workspaceAsMap(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

func workspaceStringField(m map[string]interface{}, key string) string {
	return workspaceStringValue(m[key])
}

func workspaceStringValue(v interface{}) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case fmt.Stringer:
		return strings.TrimSpace(x.String())
	case map[string]interface{}:
		for _, key := range []string{"name", "field", "metric", "label", "title", "desc", "summary", "value", "code"} {
			if s := workspaceStringField(x, key); s != "" {
				return s
			}
		}
	}
	return ""
}

func workspaceNamedItems(m map[string]interface{}, key string) []string {
	items := workspaceSliceField(m, key)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s := workspaceStringValue(item); s != "" {
			out = appendUniqueWorkspaceString(out, s, 24)
		}
	}
	return out
}

func workspaceStringItems(m map[string]interface{}, key string) []string {
	items := workspaceSliceField(m, key)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s := workspaceStringValue(item); s != "" {
			out = appendUniqueWorkspaceString(out, s, 24)
		}
	}
	return out
}

func appendUniqueWorkspaceString(items []string, item string, limit int) []string {
	item = strings.TrimSpace(item)
	if item == "" {
		return items
	}
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	items = append(items, item)
	if limit > 0 && len(items) > limit {
		return items[:limit]
	}
	return items
}

func trimWorkspaceStrings(items []string, limit int) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = appendUniqueWorkspaceString(out, strings.TrimSpace(item), limit)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func firstNonEmptyString(items ...string) string {
	for _, item := range items {
		if s := strings.TrimSpace(item); s != "" {
			return s
		}
	}
	return ""
}

func normalizeWorkspaceHandoffContextPolicy(policy string) string {
	switch strings.TrimSpace(policy) {
	case ContextPolicyFull:
		return ContextPolicyFull
	case ContextPolicyDisplayOnly:
		return ContextPolicyDisplayOnly
	default:
		return ContextPolicyArtifactOnly
	}
}

func defaultWorkspaceHandoffDisplayContent(artifactKind, targetRole, remark string) string {
	switch artifactKind {
	case "agent_app_prd":
		if strings.TrimSpace(remark) != "" {
			return "已确认 PRD，开始创建目录和生成代码。\n\n补充备注：\n" + strings.TrimSpace(remark)
		}
		return "已确认 PRD，开始创建目录和生成代码。"
	case workspaceBuildArtifactKind:
		if strings.TrimSpace(remark) != "" {
			return "已构建成功，开始测试验证。\n\n补充备注：\n" + strings.TrimSpace(remark)
		}
		return "已构建成功，开始测试验证。"
	default:
		label := strings.TrimSpace(artifactKind)
		if label == "" {
			label = "阶段产物"
		}
		return fmt.Sprintf("已确认 %s，进入 %s 阶段。", label, workspaceRoleDisplayName(targetRole))
	}
}

func buildWorkspaceHandoffContent(input workspaceHandoffContentInput) string {
	artifactLabel := input.ArtifactKind
	if artifactLabel == "" {
		artifactLabel = "artifact"
	}
	lines := []string{
		"已确认阶段交接产物，进入下一阶段。",
		"",
		fmt.Sprintf("这是阶段交接后的执行会话。请先调用 change_role，target_role 固定为 %s。", input.TargetRole),
		fmt.Sprintf("change_role.execute_directory 必须固定为 %s；后续读取、构建、测试、运行只能围绕该目录或该目录下函数。", firstNonEmptyString(input.ExecuteDirectory, "当前工作台目录")),
		"change_role 只携带四块交接信息：execute_directory、task_context、key_information、references。",
		fmt.Sprintf("上下文策略：%s。只以本消息中的 HANDOFF_CONTEXT JSON、结构化产物 JSON 和补充备注为准，不要依赖来源会话的完整历史讨论。", input.ContextPolicy),
		"不要重复产出已确认的设计文档；除非产物本身缺失关键字段，否则直接执行目标阶段任务。",
	}
	if input.ArtifactKind == "agent_app_prd" && normalizeWorkspaceRole(input.TargetRole) == WorkspaceRoleAppDeveloper {
		lines = append(lines,
			"生成阶段要求：不要重新输出 PRD，不要再次询问确认；change_role.references 必须包含 agent_app_prd JSON（本消息）、/system/prompt/roles/app-developer、/system/prompt/sdk/agent-app-sdk-readme、/system/prompt/case_catalog 和匹配案例路径；先读取 1 到多个匹配案例，再根据 PRD tables/forms/charts/rules 创建目录、写代码文件、注册路由并 build。tables.fields 是业务模型字段，tables.search_fields 是查询请求字段；创建开始时间/创建结束时间/创建人等系统搜索字段不要生成业务列。route、method、widget tag、列表列和预览数据均从 PRD 派生。非常简单的需求才可跳过额外案例。",
		)
	}
	if input.ArtifactKind == workspaceBuildArtifactKind && normalizeWorkspaceRole(input.TargetRole) == WorkspaceRoleQAEngineer {
		lines = append(lines,
			"测试阶段要求：不要修改代码，不要重新 build；先调用 change_role 进入 qa_engineer，并把 execute_directory 固定为本目录；read_dir/search_tools/search_resources 必须显式使用该目录，禁止测试整个空间。按业务操作顺序验证：先主数据/配置表，再 Form 提交，再目标记录表，再 Chart；重点覆盖创建开始时间/创建结束时间和用户筛选。测试失败时判断是测试数据问题、业务 bug 还是构建/schema 问题，并交接给 maintenance_engineer 或 build_engineer。",
		)
	}
	if input.ArtifactKind == "agent_app_prd" && strings.TrimSpace(input.PRDExecutionMarkdown) != "" {
		lines = append(lines,
			"",
			"PRD_EXECUTION_MARKDOWN:",
			"```markdown",
			strings.TrimSpace(input.PRDExecutionMarkdown),
			"```",
		)
	}
	lines = append(lines,
		"",
		"HANDOFF_CONTEXT JSON:",
		"```json",
		nonEmptyWorkspaceHandoffJSON(input.HandoffContextJSON),
		"```",
		"",
		strings.ToUpper(artifactLabel)+" JSON:",
		"```json",
		input.ArtifactJSON,
		"```",
	)
	if remark := strings.TrimSpace(input.Remark); remark != "" {
		lines = append(lines, "", "补充备注：", remark)
	}
	return strings.Join(lines, "\n")
}

func nonEmptyWorkspaceHandoffJSON(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "{}"
	}
	return s
}
