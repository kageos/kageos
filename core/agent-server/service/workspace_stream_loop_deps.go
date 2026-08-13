package service

import (
	"context"
	"errors"
	"strings"

	"github.com/kageos/kageos/core/agent-server/prompt"
	"github.com/kageos/kageos/core/agent-server/streamloop"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/llms"
)

// workspaceStreamLoopDeps 工作台对流式工具对话循环的依赖实现（只认 LLM，单模式）
type workspaceStreamLoopDeps struct {
	ctx                       context.Context
	sendEvent                 func(string, interface{})
	sessionID                 string
	fullCodePath              string
	llmConfigID               int64
	user                      string
	modeProvider              prompt.WorkspaceModePromptProvider
	toolNames                 []string
	systemPromptFragment      string
	files                     string
	currentMessageID          int64
	service                   *WorkspaceChatService
	currentLLMMeta            messageLLMMetadata
	currentModelContextPlan   *dto.WorkspaceModelContextPlan
	modelContextRound         int
	contextReductionLevel     int
	contextReductionReason    string
	outputLimitRecovery       bool
	outputContinuationContent string
	outputContinuationCount   int
}

var _ streamloop.StreamLoopDeps = (*workspaceStreamLoopDeps)(nil)

func (d *workspaceStreamLoopDeps) BuildMessages(ctx context.Context) ([]llms.Message, []llms.ToolDef, error) {
	llmConfig, err := d.service.resolveWorkspaceLLMConfig(ctx, d.llmConfigID)
	if err != nil {
		return nil, nil, err
	}
	contextWindow, _ := ResolveLLMContextWindow(llmConfig)
	outputReserve := workspaceOutputTokenLimit(workspaceConfiguredMaxTokens(llmConfig), d.contextReductionLevel, d.outputLimitRecovery, contextWindow)
	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, d.fullCodePath, "")
	if err != nil || workspaceCtx == nil {
		return nil, nil, err
	}
	directoryName := workspaceCtx.Directory.Name
	if directoryName == "" {
		directoryName = workspaceCtx.Directory.Code
	}
	msgs, tools, plan, err := d.service.buildLLMMessagesWithPlanAndOptions(ctx, d.sessionID, d.fullCodePath, directoryName, workspaceCtx, d.modeProvider, d.toolNames, d.systemPromptFragment, d.modelContextRound, workspaceLLMContextBuildOptions{
		ReductionLevel:      d.contextReductionLevel,
		ReductionReason:     d.contextReductionReason,
		ContextWindowTokens: contextWindow,
		OutputReserveTokens: outputReserve,
		LLMConfigID:         d.llmConfigID,
	}, d.currentMessageID)
	if err != nil {
		return nil, nil, err
	}
	if d.outputLimitRecovery {
		msgs = applyWorkspaceOutputLimitRecoveryInstruction(msgs)
	}
	if strings.TrimSpace(d.outputContinuationContent) != "" {
		msgs = append(msgs, llms.Message{Role: "assistant", Content: d.outputContinuationContent})
		msgs = applyWorkspaceOutputContinuationInstruction(msgs)
	}
	if d.contextReductionLevel > workspaceContextReductionNone {
		msgs = applyWorkspaceContextRecoveryInstruction(msgs)
	}
	d.currentModelContextPlan = plan
	if plan != nil && plan.Budget != nil {
		d.contextReductionLevel = plan.Budget.ReducerLevel
		d.contextReductionReason = plan.Budget.ReducerReason
	}
	d.modelContextRound++
	return msgs, tools, nil
}

func (d *workspaceStreamLoopDeps) PrepareLLM(ctx context.Context, msgs []llms.Message, tools []llms.ToolDef) (llms.LLMClient, *llms.ChatRequest, error) {
	llmConfig, client, chatReq, err := d.service.prepareLLMRequest(ctx, d.llmConfigID, msgs, tools)
	if err != nil {
		return nil, nil, err
	}
	d.currentLLMMeta = buildMessageLLMMetadata(llmConfig, client)
	contextWindow, contextWindowSource := ResolveLLMContextWindow(llmConfig)
	chatReq.MaxTokens = workspaceOutputTokenLimit(chatReq.MaxTokens, d.contextReductionLevel, d.outputLimitRecovery, contextWindow)
	if d.outputLimitRecovery {
		chatReq.ReasoningEffort = lowerWorkspaceRecoveryReasoningEffort(chatReq.ReasoningEffort)
		if strings.TrimSpace(chatReq.Verbosity) != "" {
			chatReq.Verbosity = "low"
		}
	}
	if workspaceLLMConfigSupportsPromptCache(llmConfig) && chatReq.PromptCacheKey == "" {
		chatReq.PromptCacheKey = workspacePromptCacheKey(d.currentModelContextPlan)
	}
	if chatReq.PromptCacheRetention == "" {
		chatReq.PromptCacheRetention = workspaceDefaultPromptCacheRetention(llmConfig, chatReq.Model)
	}
	if d.currentModelContextPlan != nil {
		d.currentModelContextPlan.Budget = buildWorkspaceModelContextBudget(msgs, tools, chatReq.MaxTokens, contextWindow, d.contextReductionLevel, d.contextReductionReason)
		d.currentModelContextPlan.CachePlan.PromptCacheKey = chatReq.PromptCacheKey
		d.currentModelContextPlan.CachePlan.PromptCacheRetention = chatReq.PromptCacheRetention
		d.currentModelContextPlan.LLM = &dto.WorkspaceModelContextLLM{
			ConfigID:            d.currentLLMMeta.ConfigID,
			ConfigName:          d.currentLLMMeta.ConfigName,
			Provider:            d.currentLLMMeta.Provider,
			Model:               d.currentLLMMeta.Model,
			RequestModel:        chatReq.Model,
			MaxTokens:           chatReq.MaxTokens,
			ContextWindow:       contextWindow,
			ContextWindowSource: contextWindowSource,
			MessageCount:        len(msgs),
			ToolCount:           len(tools),
		}
		d.sendEvent(EventModelContextPlan, d.currentModelContextPlan)
		if d.currentModelContextPlan.Budget.Status == "over_soft_limit" {
			return nil, nil, &llms.ContextWindowError{Message: "kageos 上下文预检超过模型安全容量，需进一步压缩后重试"}
		}
	}
	return client, chatReq, nil
}

func workspaceOutputTokenLimit(maxTokens int, reductionLevel int, outputLimitRecovery bool, contextWindow int) int {
	if maxTokens <= 0 {
		maxTokens = workspaceContextDefaultOutputReserve
	}
	if contextWindow <= 0 {
		contextWindow = DefaultLLMContextWindow
	}
	contextShare := contextWindow / 4
	if contextShare < 1024 {
		contextShare = 1024
	}
	if maxTokens > contextShare {
		maxTokens = contextShare
	}
	if outputLimitRecovery {
		return maxTokens
	}
	return reduceWorkspaceOutputReserve(maxTokens, reductionLevel)
}

func (d *workspaceStreamLoopDeps) RequestContextReduction(ctx context.Context, reason string) bool {
	if learnedLimit := llms.ContextWindowLimitFromError(errors.New(reason)); learnedLimit >= 1024 && learnedLimit <= 10000000 {
		if cfg, err := d.service.resolveWorkspaceLLMConfig(ctx, d.llmConfigID); err == nil && cfg.ContextWindow <= 0 {
			_ = d.service.llmRepo.UpdateDetectedContextWindow(cfg.ID, learnedLimit, "provider_error")
		}
	}
	if d.contextReductionLevel >= workspaceContextReductionCritical {
		return false
	}
	d.contextReductionLevel++
	d.contextReductionReason = strings.TrimSpace(reason)
	if d.contextReductionReason == "" {
		d.contextReductionReason = "context_window_retry"
	}
	return true
}

func (d *workspaceStreamLoopDeps) RequestOutputLimitRecovery(ctx context.Context, reason string) bool {
	if d.outputLimitRecovery {
		return false
	}
	d.outputLimitRecovery = true
	if d.contextReductionLevel < workspaceContextReductionLight {
		d.contextReductionLevel = workspaceContextReductionLight
	}
	d.contextReductionReason = "output_limit_retry"
	return true
}

func (d *workspaceStreamLoopDeps) RequestOutputContinuation(ctx context.Context, partialContent string) bool {
	if strings.TrimSpace(partialContent) == "" || d.outputContinuationCount >= 4 {
		return false
	}
	d.outputContinuationCount++
	d.outputContinuationContent = partialContent
	d.outputLimitRecovery = true
	return true
}

func (d *workspaceStreamLoopDeps) CompleteOutputRecovery() {
	d.outputLimitRecovery = false
	d.outputContinuationContent = ""
	d.outputContinuationCount = 0
}

const workspaceOutputLimitRecoveryInstruction = "上一次模型响应达到了输出上限。本次必须优先产出可见且完整的结果：缩短内部推理，先给结论或先完成一个可验证步骤，不要复述分析过程；任务较大时明确分段，但本次至少返回可继续使用的正文。"
const workspaceContextRecoveryInstruction = "系统刚刚为避免上下文溢出，把较早对话整理为可回溯检查点；原始消息仍完整保存在当前会话。请结合检查点、当前工作状态和最近原文直接继续任务；需要旧需求、决策或工具结果的精确细节时调用 search_session_history/read_session_messages，不能要求用户重新发送仍在会话中的内容。"
const workspaceOutputContinuationInstruction = "上一条 assistant 消息是本次回答已成功生成并展示的前半段。请从它的末尾直接续写尚未完成的部分，不要复述开头、不要重新起标题；优先把当前用户任务完整收尾。"

func applyWorkspaceOutputLimitRecoveryInstruction(msgs []llms.Message) []llms.Message {
	out := append([]llms.Message(nil), msgs...)
	for i := range out {
		if strings.EqualFold(strings.TrimSpace(out[i].Role), "system") {
			out[i].Content = strings.TrimSpace(out[i].Content) + "\n\n" + workspaceOutputLimitRecoveryInstruction
			return out
		}
	}
	return append([]llms.Message{{Role: "system", Content: workspaceOutputLimitRecoveryInstruction}}, out...)
}

func applyWorkspaceContextRecoveryInstruction(msgs []llms.Message) []llms.Message {
	out := append([]llms.Message(nil), msgs...)
	for i := range out {
		if strings.EqualFold(strings.TrimSpace(out[i].Role), "system") {
			out[i].Content = strings.TrimSpace(out[i].Content) + "\n\n" + workspaceContextRecoveryInstruction
			return out
		}
	}
	return append([]llms.Message{{Role: "system", Content: workspaceContextRecoveryInstruction}}, out...)
}

func applyWorkspaceOutputContinuationInstruction(msgs []llms.Message) []llms.Message {
	out := append([]llms.Message(nil), msgs...)
	for i := range out {
		if strings.EqualFold(strings.TrimSpace(out[i].Role), "system") {
			out[i].Content = strings.TrimSpace(out[i].Content) + "\n\n" + workspaceOutputContinuationInstruction
			return out
		}
	}
	return append([]llms.Message{{Role: "system", Content: workspaceOutputContinuationInstruction}}, out...)
}

func lowerWorkspaceRecoveryReasoningEffort(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "xhigh", "high", "medium":
		return "low"
	default:
		return strings.TrimSpace(value)
	}
}

func (d *workspaceStreamLoopDeps) SendEvent(event string, data interface{}) {
	d.sendEvent(event, data)
}

func (d *workspaceStreamLoopDeps) SaveAssistantMessage(ctx context.Context, content string, thinkingContent string, usage *llms.Usage) error {
	d.finalizeCurrentModelContextPlan(usage)
	return d.service.saveAssistantMessage(ctx, d.sessionID, content, thinkingContent, d.user, d.currentLLMMeta, d.currentModelContextPlan, usage)
}

func (d *workspaceStreamLoopDeps) SaveAssistantMessageWithToolCalls(ctx context.Context, content string, thinkingContent string, toolCalls []llms.ToolCall, usage *llms.Usage) error {
	d.finalizeCurrentModelContextPlan(usage)
	return d.service.saveAssistantMessageWithToolCalls(ctx, d.sessionID, content, thinkingContent, toolCalls, d.user, d.currentLLMMeta, d.currentModelContextPlan, usage)
}

func (d *workspaceStreamLoopDeps) finalizeCurrentModelContextPlan(usage *llms.Usage) {
	if d.currentModelContextPlan == nil {
		return
	}
	attachLLMUsageToWorkspaceModelContextPlan(d.currentModelContextPlan, usage)
	d.sendEvent(EventModelContextPlan, d.currentModelContextPlan)
}

func (d *workspaceStreamLoopDeps) ExecuteToolCalls(ctx context.Context, allToolCalls []llms.ToolCall, round int, sendEvent func(string, interface{})) ([]streamloop.ToolCallSummary, error) {
	auditCtx := contextx.WithInitiatorUser(ctx, d.user)
	auditCtx = contextx.WithWorkspaceMessageID(auditCtx, d.currentMessageID)
	summaries, nextFullCodePath, err := d.service.executeToolCalls(auditCtx, allToolCalls, d.sessionID, d.fullCodePath, d.user, d.files, round, sendEvent)
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
