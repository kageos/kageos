package service

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/streamloop"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/llms"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

// workspaceStreamLoopDeps 工作台对流式工具对话循环的依赖实现
type workspaceStreamLoopDeps struct {
	ctx                  context.Context
	sendEvent            func(string, interface{})
	sessionID            string
	fullCodePath         string
	agentID              int64
	agentIDPtr           *int64
	user                 string
	toolNames            []string
	systemPromptFragment string
	files                *types.Files
	service              *WorkspaceChatService
}

var _ streamloop.StreamLoopDeps = (*workspaceStreamLoopDeps)(nil)

func (d *workspaceStreamLoopDeps) BuildMessages(ctx context.Context) ([]llms.Message, []llms.ToolDef, error) {
	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, d.fullCodePath)
	if err != nil || workspaceCtx == nil {
		return nil, nil, err
	}
	directoryName := workspaceCtx.Directory.Name
	if directoryName == "" {
		directoryName = workspaceCtx.Directory.Code
	}
	return d.service.buildLLMMessages(ctx, d.sessionID, d.fullCodePath, directoryName, d.agentID, workspaceCtx, d.toolNames, d.systemPromptFragment)
}

func (d *workspaceStreamLoopDeps) PrepareLLM(ctx context.Context, msgs []llms.Message, tools []llms.ToolDef) (llms.LLMClient, *llms.ChatRequest, error) {
	_, client, chatReq, err := d.service.prepareLLMRequest(ctx, d.agentID, msgs, tools)
	if err != nil {
		return nil, nil, err
	}
	return client, chatReq, nil
}

func (d *workspaceStreamLoopDeps) SendEvent(event string, data interface{}) {
	d.sendEvent(event, data)
}

func (d *workspaceStreamLoopDeps) SaveAssistantMessage(ctx context.Context, content string) error {
	return d.service.saveAssistantMessage(ctx, d.sessionID, d.agentIDPtr, content, d.user)
}

func (d *workspaceStreamLoopDeps) SaveAssistantMessageWithToolCalls(ctx context.Context, content string, toolCalls []llms.ToolCall) error {
	return d.service.saveAssistantMessageWithToolCalls(ctx, d.sessionID, d.agentIDPtr, content, toolCalls, d.user)
}

func (d *workspaceStreamLoopDeps) ExecuteToolCalls(ctx context.Context, allToolCalls []llms.ToolCall, currentAssistantContent string, sendEvent func(string, interface{})) ([]streamloop.ToolCallSummary, error) {
	summaries := d.service.executeToolCalls(ctx, allToolCalls, currentAssistantContent, d.sessionID, d.fullCodePath, d.agentIDPtr, d.user, d.files, sendEvent)
	out := make([]streamloop.ToolCallSummary, len(summaries))
	for i := range summaries {
		out[i] = streamloop.ToolCallSummary{Name: summaries[i].Name, Status: summaries[i].Status}
	}
	return out, nil
}

func (d *workspaceStreamLoopDeps) OnDone(summaries []streamloop.ToolCallSummary) {
	toolCalls := make([]dto.WorkspaceChatToolCallSummary, len(summaries))
	for i := range summaries {
		toolCalls[i] = dto.WorkspaceChatToolCallSummary{Name: summaries[i].Name, Status: summaries[i].Status}
	}
	d.sendEvent(EventDone, StreamEventDone{SessionID: d.sessionID, AgentID: d.agentID, ToolCalls: toolCalls})
}
