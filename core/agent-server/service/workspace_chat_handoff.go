package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/core/agent-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
)

var errWorkspaceHandoffAlreadyProcessed = errors.New("workspace handoff already processed")

// CreateWorkspaceHandoff injects a stage-transition artifact into the current
// conversation. It preserves prior history and updates the active role in-place.
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
	if workspaceRoleHandoffPacketHasValidationErrors(handoffContext.HandoffPacket) {
		return nil, fmt.Errorf("阶段交接校验失败: %s", workspaceRoleHandoffPacketValidationSummary(handoffContext.HandoffPacket))
	}
	handoffContextJSON := formatWorkspaceHandoffContextJSON(handoffContext)
	handoffContextMessageJSON := formatWorkspaceHandoffContextJSON(workspaceHandoffContextForMessage(handoffContext))
	handoffPacketJSON := formatWorkspaceRoleHandoffPacketJSON(handoffContext.HandoffPacket)
	executeDirectory := firstNonEmptyString(handoffContext.ExecuteDirectory, fullCodePath)
	targetFullCodePath := workspaceHandoffTargetSessionFullCodePath(fullCodePath, executeDirectory, targetRole, artifactKind)

	targetSessionID := source.SessionID
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
	content := buildWorkspaceHandoffContent(workspaceHandoffContentInput{
		ArtifactKind:         artifactKind,
		ArtifactJSON:         artifactJSON,
		HandoffPacketJSON:    handoffPacketJSON,
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
		User:           firstNonEmptyString(user, source.User),
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
		User:               firstNonEmptyString(user, source.User),
	}
	handoffPacket.CreatedBy = user
	handoffPacket.UpdatedBy = user
	var existingResp *dto.WorkspaceHandoffResp
	if err := s.sessionRepo.TransactionWithMessagesAndHandoffPackets(func(sessionTx *repository.ChatSessionRepository, messageTx *repository.ChatMessageRepository, handoffTx *repository.WorkspaceHandoffPacketRepository) error {
		if packet, err := handoffTx.FindLatestBySourceAndTarget(source.SessionID, artifactKind, targetRole, firstNonEmptyString(user, source.User)); err != nil {
			return fmt.Errorf("查询已有交接包失败: %w", err)
		} else if packet != nil {
			existingResp = buildExistingWorkspaceHandoffResp(packet, sessionTx, messageTx)
			return nil
		}
		source.FullCodePath = firstNonEmptyString(normalizeWorkspacePath(targetFullCodePath), fullCodePath, source.FullCodePath)
		source.Status = model.ChatSessionStatusActive
		source.RoleID = targetRole
		source.RoleDisplayName = workspaceRoleDisplayName(targetRole)
		source.HandoffKind = artifactKind
		source.HandoffTargetRole = targetRole
		source.ContextPolicy = contextPolicy
		source.ModelContextAnchorMessageID = 0
		source.ArchivedForModel = false
		source.ArchiveReason = ""
		source.UpdatedBy = user
		if strings.TrimSpace(source.Title) == "" {
			source.Title = title
		}
		if err := sessionTx.Update(source); err != nil {
			return fmt.Errorf("更新当前会话阶段信息失败: %w", err)
		}
		if err := messageTx.Create(initialMessage); err != nil {
			return fmt.Errorf("创建阶段注入消息失败: %w", err)
		}
		handoffPacket.InitialMessageID = initialMessage.ID
		if err := handoffTx.Create(handoffPacket); err != nil {
			return fmt.Errorf("创建阶段注入包失败: %w", err)
		}
		return nil
	}); err != nil {
		if errors.Is(err, errWorkspaceHandoffAlreadyProcessed) {
			return nil, fmt.Errorf("该阶段注入已处理，请刷新当前会话后继续")
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
