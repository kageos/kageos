package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/google/uuid"
)

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

	source.ArchivedForModel = true
	source.ContextPolicy = ContextPolicyDisplayOnly
	source.ArchiveReason = fmt.Sprintf("已交接到%s，会话仅保留展示历史", workspaceRoleDisplayName(targetRole))
	source.Status = model.ChatSessionStatusDone
	source.UpdatedBy = user

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
		FullCodePath:      fullCodePath,
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
		ArtifactKind:  artifactKind,
		ArtifactJSON:  artifactJSON,
		TargetRole:    targetRole,
		Remark:        req.Remark,
		ContextPolicy: contextPolicy,
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
		SourceSessionID: source.SessionID,
		TargetSessionID: targetSessionID,
		FullCodePath:    fullCodePath,
		TargetRole:      targetRole,
		ArtifactKind:    artifactKind,
		ArtifactJSON:    artifactJSON,
		Remark:          strings.TrimSpace(req.Remark),
		ContextPolicy:   contextPolicy,
		User:            target.User,
	}
	handoffPacket.CreatedBy = user
	handoffPacket.UpdatedBy = user
	if err := s.sessionRepo.TransactionWithMessagesAndHandoffPackets(func(sessionTx *repository.ChatSessionRepository, messageTx *repository.ChatMessageRepository, handoffTx *repository.WorkspaceHandoffPacketRepository) error {
		if err := sessionTx.Update(source); err != nil {
			return fmt.Errorf("归档来源会话失败: %w", err)
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
		return nil, err
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
	}, nil
}

type workspaceHandoffContentInput struct {
	ArtifactKind  string
	ArtifactJSON  string
	TargetRole    string
	Remark        string
	ContextPolicy string
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
		fmt.Sprintf("上下文策略：%s。只以本消息中的结构化产物 JSON 和补充备注为准，不要依赖来源会话的历史讨论。", input.ContextPolicy),
		"不要重复产出已确认的设计文档；除非产物本身缺失关键字段，否则直接执行目标阶段任务。",
	}
	if input.ArtifactKind == "agent_app_prd" && normalizeWorkspaceRole(input.TargetRole) == WorkspaceRoleAppDeveloper {
		lines = append(lines,
			"生成阶段要求：不要重新输出 PRD，不要再次询问确认；先读取 1 到多个匹配案例，再根据 PRD tables/forms/charts/rules 创建目录、写代码文件、注册路由并 build。tables.fields 是业务模型字段，tables.search_fields 是查询请求字段；创建开始时间/创建结束时间/创建人等系统搜索字段不要生成业务列。route、method、widget tag、列表列和预览数据均从 PRD 派生。非常简单的需求才可跳过额外案例。",
		)
	}
	if input.ArtifactKind == workspaceBuildArtifactKind && normalizeWorkspaceRole(input.TargetRole) == WorkspaceRoleQAEngineer {
		lines = append(lines,
			"测试阶段要求：不要修改代码，不要重新 build；先调用 change_role 进入 qa_engineer，再用 search_tools/read_dir 确认当前工作空间函数清单和 schema。按业务操作顺序验证：先主数据/配置表，再 Form 提交，再目标记录表，再 Chart；重点覆盖创建开始时间/创建结束时间和用户筛选。测试失败时判断是测试数据问题、业务 bug 还是构建/schema 问题，并交接给 maintenance_engineer 或 build_engineer。",
		)
	}
	lines = append(lines,
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
