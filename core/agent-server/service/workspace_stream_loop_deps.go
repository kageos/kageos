package service

import (
	"context"

	"github.com/kageos/kageos/core/agent-server/prompt"
	"github.com/kageos/kageos/core/agent-server/streamloop"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/llms"
)

// workspaceStreamLoopDeps 工作台对流式工具对话循环的依赖实现（只认 LLM，单模式）
type workspaceStreamLoopDeps struct {
	ctx                     context.Context
	sendEvent               func(string, interface{})
	sessionID               string
	fullCodePath            string
	llmConfigID             int64
	user                    string
	modeProvider            prompt.WorkspaceModePromptProvider
	toolNames               []string
	systemPromptFragment    string
	files                   string
	service                 *WorkspaceChatService
	currentLLMMeta          messageLLMMetadata
	currentModelContextPlan *dto.WorkspaceModelContextPlan
	modelContextRound       int
}

var _ streamloop.StreamLoopDeps = (*workspaceStreamLoopDeps)(nil)

func (d *workspaceStreamLoopDeps) BuildMessages(ctx context.Context) ([]llms.Message, []llms.ToolDef, error) {
	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, d.fullCodePath, "")
	if err != nil || workspaceCtx == nil {
		return nil, nil, err
	}
	directoryName := workspaceCtx.Directory.Name
	if directoryName == "" {
		directoryName = workspaceCtx.Directory.Code
	}
	msgs, tools, plan, err := d.service.buildLLMMessagesWithPlan(ctx, d.sessionID, d.fullCodePath, directoryName, workspaceCtx, d.modeProvider, d.toolNames, d.systemPromptFragment, d.modelContextRound)
	if err != nil {
		return nil, nil, err
	}
	d.currentModelContextPlan = plan
	d.modelContextRound++
	return msgs, tools, nil
}

func (d *workspaceStreamLoopDeps) PrepareLLM(ctx context.Context, msgs []llms.Message, tools []llms.ToolDef) (llms.LLMClient, *llms.ChatRequest, error) {
	llmConfig, client, chatReq, err := d.service.prepareLLMRequest(ctx, d.llmConfigID, msgs, tools)
	if err != nil {
		return nil, nil, err
	}
	d.currentLLMMeta = buildMessageLLMMetadata(llmConfig, client)
	if d.currentModelContextPlan != nil {
		d.currentModelContextPlan.LLM = &dto.WorkspaceModelContextLLM{
			ConfigID:     d.currentLLMMeta.ConfigID,
			ConfigName:   d.currentLLMMeta.ConfigName,
			Provider:     d.currentLLMMeta.Provider,
			Model:        d.currentLLMMeta.Model,
			RequestModel: chatReq.Model,
			MaxTokens:    chatReq.MaxTokens,
			MessageCount: len(msgs),
			ToolCount:    len(tools),
		}
		d.sendEvent(EventModelContextPlan, d.currentModelContextPlan)
	}
	return client, chatReq, nil
}

func (d *workspaceStreamLoopDeps) SendEvent(event string, data interface{}) {
	d.sendEvent(event, data)
}

func (d *workspaceStreamLoopDeps) SaveAssistantMessage(ctx context.Context, content string, usage *llms.Usage) error {
	d.finalizeCurrentModelContextPlan(usage)
	return d.service.saveAssistantMessage(ctx, d.sessionID, content, d.user, d.currentLLMMeta, d.currentModelContextPlan, usage)
}

func (d *workspaceStreamLoopDeps) SaveAssistantMessageWithToolCalls(ctx context.Context, content string, toolCalls []llms.ToolCall, usage *llms.Usage) error {
	d.finalizeCurrentModelContextPlan(usage)
	return d.service.saveAssistantMessageWithToolCalls(ctx, d.sessionID, content, toolCalls, d.user, d.currentLLMMeta, d.currentModelContextPlan, usage)
}

func (d *workspaceStreamLoopDeps) finalizeCurrentModelContextPlan(usage *llms.Usage) {
	if d.currentModelContextPlan == nil {
		return
	}
	attachLLMUsageToWorkspaceModelContextPlan(d.currentModelContextPlan, usage)
	d.sendEvent(EventModelContextPlan, d.currentModelContextPlan)
}

func (d *workspaceStreamLoopDeps) ExecuteToolCalls(ctx context.Context, allToolCalls []llms.ToolCall, round int, sendEvent func(string, interface{})) ([]streamloop.ToolCallSummary, error) {
	summaries, nextFullCodePath, err := d.service.executeToolCalls(ctx, allToolCalls, d.sessionID, d.fullCodePath, d.user, d.files, round, sendEvent)
	if err != nil {
		return nil, err
	}
	if nextFullCodePath != "" {
		d.fullCodePath = nextFullCodePath
	}
	out := make([]streamloop.ToolCallSummary, len(summaries))
	for i := range summaries {
		out[i] = streamloop.ToolCallSummary{
			ID:        summaries[i].ID,
			Index:     summaries[i].Index,
			Round:     summaries[i].Round,
			Name:      summaries[i].Name,
			Status:    summaries[i].Status,
			Arguments: summaries[i].Arguments, Result: summaries[i].Result, ResultData: summaries[i].ResultData, Metadata: summaries[i].Metadata, Error: summaries[i].Error,
		}
	}
	return out, nil
}

func (d *workspaceStreamLoopDeps) OnDone(summaries []streamloop.ToolCallSummary, usage *llms.Usage) {
	d.service.persistWorkspaceSessionInteractionStatus(d.ctx, d.sessionID, summaries, d.user)
	toolCalls := make([]dto.WorkspaceChatToolCallSummary, len(summaries))
	for i := range summaries {
		toolCalls[i] = dto.WorkspaceChatToolCallSummary{
			ID:        summaries[i].ID,
			Index:     summaries[i].Index,
			Round:     summaries[i].Round,
			Name:      summaries[i].Name,
			Status:    summaries[i].Status,
			Arguments: summaries[i].Arguments, Result: summaries[i].Result, ResultData: summaries[i].ResultData, Metadata: summaries[i].Metadata, Error: summaries[i].Error,
		}
	}
	d.sendEvent(EventDone, StreamEventDone{
		SessionID:        d.sessionID,
		ToolCalls:        toolCalls,
		LLMConfigID:      d.currentLLMMeta.ConfigID,
		LLMConfigName:    d.currentLLMMeta.ConfigName,
		LLMProvider:      d.currentLLMMeta.Provider,
		LLMModel:         d.currentLLMMeta.Model,
		LLMUsage:         llmUsageInfoFromUsage(usage),
		ModelContextPlan: d.currentModelContextPlan,
	})
}
