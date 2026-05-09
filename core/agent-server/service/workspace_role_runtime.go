package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/pkg/llms"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

func (s *WorkspaceChatService) currentWorkspaceRoleForSession(ctx context.Context, sessionID string) string {
	if s == nil || strings.TrimSpace(sessionID) == "" {
		return ""
	}
	if s.sessionRepo != nil {
		if session, err := s.sessionRepo.GetBySessionID(sessionID); err == nil && session != nil {
			if roleID := normalizeWorkspaceRole(session.RoleID); roleID != "" {
				return roleID
			}
			if roleID := normalizeWorkspaceRole(session.HandoffTargetRole); roleID != "" {
				return roleID
			}
		}
	}
	if s.messageRepo != nil {
		if messages, err := s.messageRepo.ListBySessionID(sessionID); err == nil {
			if roleID := latestWorkspaceRoleFromMessages(messages); roleID != "" {
				return roleID
			}
		}
	}
	_ = ctx
	return ""
}

func (s *WorkspaceChatService) updateWorkspaceSessionRole(ctx context.Context, sessionID string, roleID string, user string) {
	if s == nil || s.sessionRepo == nil || strings.TrimSpace(sessionID) == "" {
		return
	}
	roleID = normalizeWorkspaceRole(roleID)
	if roleID == "" {
		return
	}
	session, err := s.sessionRepo.GetBySessionID(sessionID)
	if err != nil || session == nil {
		logger.Warnf(ctx, "[WorkspaceRole] 查询会话角色失败 SessionID=%s: %v", sessionID, err)
		return
	}
	if session.RoleID == roleID && session.RoleDisplayName == workspaceRoleDisplayName(roleID) {
		return
	}
	session.RoleID = roleID
	session.RoleDisplayName = workspaceRoleDisplayName(roleID)
	if user != "" {
		session.UpdatedBy = user
	}
	if err := s.sessionRepo.Update(session); err != nil {
		logger.Warnf(ctx, "[WorkspaceRole] 更新会话角色失败 SessionID=%s RoleID=%s: %v", sessionID, roleID, err)
	}
}

func workspaceSessionRoleDisplayName(session *model.AgentChatSession) string {
	if session == nil {
		return ""
	}
	if strings.TrimSpace(session.RoleDisplayName) != "" {
		return strings.TrimSpace(session.RoleDisplayName)
	}
	if roleID := normalizeWorkspaceRole(session.RoleID); roleID != "" {
		return workspaceRoleDisplayName(roleID)
	}
	if roleID := normalizeWorkspaceRole(session.HandoffTargetRole); roleID != "" {
		return workspaceRoleDisplayName(roleID)
	}
	return ""
}

func workspaceSessionRoleID(session *model.AgentChatSession) string {
	if session == nil {
		return ""
	}
	if roleID := normalizeWorkspaceRole(session.RoleID); roleID != "" {
		return roleID
	}
	return normalizeWorkspaceRole(session.HandoffTargetRole)
}

func latestWorkspaceRoleFromMessages(messages []*model.AgentChatMessage) string {
	toolNamesByCallID := make(map[string]string)
	latestRole := ""
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		switch msg.Role {
		case RoleAssistant:
			if msg.ToolCalls == nil || strings.TrimSpace(*msg.ToolCalls) == "" {
				continue
			}
			var toolCalls []llms.ToolCall
			if err := json.Unmarshal([]byte(*msg.ToolCalls), &toolCalls); err != nil {
				continue
			}
			for _, tc := range toolCalls {
				if strings.TrimSpace(tc.ID) == "" {
					continue
				}
				toolNamesByCallID[tc.ID] = tc.Function.Name
			}
		case RoleTool:
			if msg.ToolStatus != ToolCallStatusOK {
				continue
			}
			if toolNamesByCallID[msg.ToolCallID] != "change_role" {
				continue
			}
			if roleID := workspaceRoleFromToolMessage(msg); roleID != "" {
				latestRole = roleID
			}
		}
	}
	return latestRole
}

func workspaceRoleFromToolMessage(msg *model.AgentChatMessage) string {
	if msg == nil {
		return ""
	}
	if msg.ResultData != nil && strings.TrimSpace(*msg.ResultData) != "" {
		if roleID := workspaceRoleFromJSON([]byte(*msg.ResultData)); roleID != "" {
			return roleID
		}
	}
	if content := strings.TrimSpace(msg.Content); content != "" {
		return workspaceRoleFromJSON([]byte(content))
	}
	return ""
}

func workspaceRoleFromJSON(raw []byte) string {
	var data struct {
		RoleID      string `json:"role_id"`
		CurrentRole string `json:"current_role"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return ""
	}
	if data.RoleID != "" {
		return normalizeWorkspaceRole(data.RoleID)
	}
	return normalizeWorkspaceRole(data.CurrentRole)
}
