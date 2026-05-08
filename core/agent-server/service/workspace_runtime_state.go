package service

import (
	"context"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

const failedRuntimeStateTTL = 5 * time.Minute

func (s *WorkspaceChatService) startWorkspaceRuntimeState(
	ctx context.Context,
	session *model.AgentChatSession,
	fullCodePath string,
	modeCode string,
	user string,
) (string, dto.RuntimeStateItem) {
	if s.runtimeState == nil || session == nil {
		return "", dto.RuntimeStateItem{}
	}
	sourceType := strings.TrimSpace(contextx.GetSourceType(ctx))
	sourceRef := strings.TrimSpace(contextx.GetSourceRef(ctx))
	kind := RuntimeStateKindWorkspaceSession
	if sourceType == ScheduledAgentSourceType {
		kind = RuntimeStateKindScheduledAgentSession
	}
	key := kind + ":" + session.SessionID
	now := time.Now()
	item := dto.RuntimeStateItem{
		Key:          key,
		Kind:         kind,
		Status:       RuntimeStateStatusThinking,
		Stage:        "thinking",
		FullCodePath: fullCodePath,
		Title:        session.Title,
		User:         user,
		ModeCode:     modeCode,
		SessionID:    session.SessionID,
		SourceType:   sourceType,
		SourceRef:    sourceRef,
		StartedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.runtimeState.Upsert(ctx, item); err != nil {
		logger.Warnf(ctx, "[RuntimeState] upsert workspace state failed: %v", err)
	}
	return key, item
}

func (s *WorkspaceChatService) updateWorkspaceRuntimeStateFromEvent(
	ctx context.Context,
	key string,
	base dto.RuntimeStateItem,
	event string,
	data interface{},
) {
	if s.runtimeState == nil || key == "" {
		return
	}
	item := base
	item.Key = key
	item.UpdatedAt = time.Now()

	switch event {
	case EventContent:
		item.Status = RuntimeStateStatusRunning
		item.Stage = "responding"
	case EventToolCall:
		toolCall, _ := data.(StreamEventToolCall)
		item.Stage = strings.TrimSpace(toolCall.Name)
		switch toolCall.Status {
		case ToolCallStatusRunning, ToolCallStatusStreaming:
			item.Status = RuntimeStateStatusToolRunning
		default:
			item.Status = RuntimeStateStatusRunning
		}
	case EventError:
		item.Status = RuntimeStateStatusFailed
		item.Stage = "error"
		expiresAt := time.Now().Add(failedRuntimeStateTTL)
		item.ExpiresAt = &expiresAt
	default:
		return
	}
	if err := s.runtimeState.Upsert(ctx, item); err != nil {
		logger.Warnf(ctx, "[RuntimeState] update workspace state failed: %v", err)
	}
}

func (s *WorkspaceChatService) finishWorkspaceRuntimeState(
	ctx context.Context,
	key string,
	base dto.RuntimeStateItem,
	err error,
	finalStatus string,
) {
	if s.runtimeState == nil || key == "" {
		return
	}
	if err == nil || finalStatus == RuntimeStateStatusCancelled {
		if deleteErr := s.runtimeState.Delete(ctx, key); deleteErr != nil {
			logger.Warnf(ctx, "[RuntimeState] delete workspace state failed: %v", deleteErr)
		}
		return
	}
	item := base
	item.Key = key
	item.Status = RuntimeStateStatusFailed
	item.Stage = "error"
	item.UpdatedAt = time.Now()
	expiresAt := time.Now().Add(failedRuntimeStateTTL)
	item.ExpiresAt = &expiresAt
	if upsertErr := s.runtimeState.Upsert(ctx, item); upsertErr != nil {
		logger.Warnf(ctx, "[RuntimeState] keep failed workspace state failed: %v", upsertErr)
	}
}

func (s *WorkspaceChatService) deleteWorkspaceRuntimeState(ctx context.Context, sessionID string) {
	if s.runtimeState == nil || strings.TrimSpace(sessionID) == "" {
		return
	}
	for _, kind := range []string{RuntimeStateKindWorkspaceSession, RuntimeStateKindScheduledAgentSession} {
		if err := s.runtimeState.Delete(ctx, kind+":"+sessionID); err != nil {
			logger.Warnf(ctx, "[RuntimeState] delete cancelled workspace state failed: %v", err)
		}
	}
}
