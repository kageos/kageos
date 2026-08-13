package streamloop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/llms"
	"github.com/kageos/kageos/pkg/logger"
)

const (
	EventContent               = dto.WorkspaceStreamEventContent
	EventThinking              = dto.WorkspaceStreamEventThinking
	EventGenerationAttempt     = dto.WorkspaceStreamEventGenerationAttempt
	EventToolCallsStreamDelta  = dto.WorkspaceStreamEventToolCallsStreamDelta // 增量+节流，节省带宽
	EventError                 = dto.WorkspaceStreamEventError
	MaxToolRounds              = 100 // 最大工具调用轮数，防止无限循环；过小易中断，过大增加耗时与成本
	maxContextReductionRetries = 4
	maxOutputLimitRetries      = 1

	// 节流参数：满足任一条件即 flush
	throttleIntervalMs = 100 // 距上次发送超过 100ms
	throttleSizeChars  = 200 // 累积 delta 超过 200 字
)

// RunStreamLoop 流式工具对话循环：从 BuildMessages 开始，调 LLM 流式，若有 tool_calls 则执行并递归，否则结束
func RunStreamLoop(ctx context.Context, deps StreamLoopDeps) error {
	return runStreamLoopRound(ctx, deps, 0, nil, nil)
}

func runStreamLoopRound(ctx context.Context, deps StreamLoopDeps, round int, previousSummaries []ToolCallSummary, previousUsage *llms.Usage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if round >= MaxToolRounds {
		logger.Warnf(ctx, "[StreamLoop] 达到最大工具调用轮数 %d，停止循环", MaxToolRounds)
		// 发一句提示，避免前端“戛然而止”显得乱
		deps.SendEvent(EventContent, &contentData{Content: "\n\n---\n已达到本轮最大工具调用次数，如需继续请再次发送消息。"})
		deps.OnDone(previousSummaries, previousUsage)
		return nil
	}

	var content string
	var thinkingContent string
	var allToolCalls []llms.ToolCall
	var usage *llms.Usage
	var roundUsage *llms.Usage
	outputLimitRetries := 0
	contextReductionRetries := 0
	committedContentPrefix := ""
	committedThinkingPrefix := ""
	for attempt := 0; ; attempt++ {
		msgs, tools, err := deps.BuildMessages(ctx)
		if err != nil {
			deps.SendEvent(EventError, &errorData{Message: err.Error()})
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		client, chatReq, err := deps.PrepareLLM(ctx, msgs, tools)
		if err != nil {
			if contextReductionRetries < maxContextReductionRetries && requestContextReduction(ctx, deps, err) {
				contextReductionRetries++
				logger.Warnf(ctx, "[StreamLoop] LLM 上下文预检超限，已提高上下文压缩等级后重试 attempt=%d err=%v", attempt+1, err)
				continue
			}
			deps.SendEvent(EventError, &errorData{Message: err.Error()})
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		attemptID := fmt.Sprintf("%d-%d", round, attempt+1)
		deps.SendEvent(EventGenerationAttempt, &dto.WorkspaceStreamGenerationAttempt{AttemptID: attemptID, Round: round, Action: "started"})
		stream, err := client.ChatStream(ctx, chatReq)
		if err != nil {
			if contextReductionRetries < maxContextReductionRetries && requestContextReduction(ctx, deps, err) {
				contextReductionRetries++
				deps.SendEvent(EventGenerationAttempt, &dto.WorkspaceStreamGenerationAttempt{AttemptID: attemptID, Round: round, Action: "discarded", Reason: "context_window_retry"})
				logger.Warnf(ctx, "[StreamLoop] LLM 上下文超限，已提高上下文压缩等级后重试 attempt=%d err=%v", attempt+1, err)
				continue
			}
			deps.SendEvent(EventError, &errorData{Message: "LLM 调用失败: " + err.Error()})
			return err
		}

		var attemptUsage *llms.Usage
		content, thinkingContent, allToolCalls, attemptUsage, err = processStreamChunks(ctx, stream, deps.SendEvent, round)
		roundUsage = addLLMUsage(roundUsage, attemptUsage)
		usage = roundUsage
		if err != nil {
			if outputLimitErr := asOutputLimitError(err); outputLimitErr != nil && outputLimitRetries < maxOutputLimitRetries && requestOutputLimitRecovery(ctx, deps, err) {
				outputLimitRetries++
				deps.SendEvent(EventGenerationAttempt, &dto.WorkspaceStreamGenerationAttempt{AttemptID: attemptID, Round: round, Action: "discarded", Reason: "output_limit_retry"})
				logger.Warnf(ctx, "[StreamLoop] LLM 在可见正文前达到输出上限，已启用精简输出恢复后重试 retry=%d max_tokens=%d", outputLimitRetries, chatReq.MaxTokens)
				continue
			}
			if outputLimitErr := asOutputLimitError(err); outputLimitErr != nil && strings.TrimSpace(content) != "" {
				combinedContent := committedContentPrefix + content
				combinedThinking := joinOutputThinking(committedThinkingPrefix, thinkingContent)
				if requestOutputContinuation(ctx, deps, combinedContent) {
					committedContentPrefix = combinedContent
					committedThinkingPrefix = combinedThinking
					deps.SendEvent(EventGenerationAttempt, &dto.WorkspaceStreamGenerationAttempt{AttemptID: attemptID, Round: round, Action: "committed", Reason: "output_continuation"})
					logger.Warnf(ctx, "[StreamLoop] LLM 可见正文达到输出上限，保留已生成内容并自动续写 chars=%d", len([]rune(combinedContent)))
					continue
				}
				content = combinedContent
				thinkingContent = combinedThinking
				allToolCalls = nil
				deps.SendEvent(EventGenerationAttempt, &dto.WorkspaceStreamGenerationAttempt{AttemptID: attemptID, Round: round, Action: "committed", Reason: "output_continuation_limit"})
				logger.Warnf(ctx, "[StreamLoop] LLM 自动续写达到保护上限，保留并保存已生成正文 chars=%d", len([]rune(content)))
				break
			}
			if contextReductionRetries < maxContextReductionRetries && requestContextReduction(ctx, deps, err) {
				contextReductionRetries++
				deps.SendEvent(EventGenerationAttempt, &dto.WorkspaceStreamGenerationAttempt{AttemptID: attemptID, Round: round, Action: "discarded", Reason: "context_window_retry"})
				logger.Warnf(ctx, "[StreamLoop] LLM 流式上下文超限，已提高上下文压缩等级后重试 attempt=%d err=%v", attempt+1, err)
				continue
			}
			if outputLimitErr := asOutputLimitError(err); outputLimitErr != nil {
				err = actionableOutputLimitError(outputLimitErr, chatReq.MaxTokens, outputLimitRetries > 0)
			}
			deps.SendEvent(EventError, &errorData{Message: err.Error()})
			return err
		}
		content = committedContentPrefix + content
		thinkingContent = joinOutputThinking(committedThinkingPrefix, thinkingContent)
		deps.SendEvent(EventGenerationAttempt, &dto.WorkspaceStreamGenerationAttempt{AttemptID: attemptID, Round: round, Action: "committed"})
		break
	}
	completeOutputRecovery(deps)
	combinedUsage := addLLMUsage(previousUsage, usage)

	if len(allToolCalls) > 0 {
		if err := deps.SaveAssistantMessageWithToolCalls(ctx, content, thinkingContent, allToolCalls, usage); err != nil {
			logger.Warnf(ctx, "[StreamLoop] 保存 assistant 消息失败: %v", err)
			deps.SendEvent(EventError, &errorData{Message: "保存 assistant 消息失败: " + err.Error()})
			return err
		}
		summaries, err := deps.ExecuteToolCalls(ctx, allToolCalls, round, deps.SendEvent)
		if err != nil {
			if ctx.Err() == nil {
				deps.SendEvent(EventError, &errorData{Message: err.Error()})
			}
			return err
		}
		combined := append(previousSummaries, summaries...)
		return runStreamLoopRound(ctx, deps, round+1, combined, combinedUsage)
	}

	if err := deps.SaveAssistantMessage(ctx, content, thinkingContent, usage); err != nil {
		logger.Warnf(ctx, "[StreamLoop] 保存 assistant 消息失败: %v", err)
	}
	deps.OnDone(previousSummaries, combinedUsage)
	return nil
}

func requestContextReduction(ctx context.Context, deps StreamLoopDeps, err error) bool {
	if err == nil || !llms.IsContextWindowError(err) {
		return false
	}
	reducer, ok := deps.(ContextReductionDeps)
	if !ok {
		return false
	}
	return reducer.RequestContextReduction(ctx, err.Error())
}

func requestOutputLimitRecovery(ctx context.Context, deps StreamLoopDeps, err error) bool {
	if asOutputLimitError(err) == nil {
		return false
	}
	recoverer, ok := deps.(OutputLimitRecoveryDeps)
	if !ok {
		return false
	}
	return recoverer.RequestOutputLimitRecovery(ctx, err.Error())
}

func requestOutputContinuation(ctx context.Context, deps StreamLoopDeps, partialContent string) bool {
	recoverer, ok := deps.(OutputContinuationDeps)
	if !ok {
		return false
	}
	return recoverer.RequestOutputContinuation(ctx, partialContent)
}

func completeOutputRecovery(deps StreamLoopDeps) {
	if recoverer, ok := deps.(OutputContinuationDeps); ok {
		recoverer.CompleteOutputRecovery()
	}
}

func joinOutputThinking(prefix, value string) string {
	prefix = strings.TrimSpace(prefix)
	value = strings.TrimSpace(value)
	if prefix == "" {
		return value
	}
	if value == "" {
		return prefix
	}
	return prefix + "\n" + value
}

type outputLimitError struct {
	thinkingOnly bool
}

func (e *outputLimitError) Error() string {
	if e != nil && e.thinkingOnly {
		return "LLM 响应因达到最大输出长度而中断，且可展示内容为空；模型输出可能停在思考阶段"
	}
	return "LLM 响应因达到最大输出长度而中断"
}

func asOutputLimitError(err error) *outputLimitError {
	var target *outputLimitError
	if errors.As(err, &target) {
		return target
	}
	return nil
}

func actionableOutputLimitError(err *outputLimitError, maxTokens int, retried bool) error {
	limit := "当前配置的最大 Token"
	if maxTokens > 0 {
		limit = fmt.Sprintf("当前最大 Token 为 %d", maxTokens)
	}
	prefix := err.Error()
	if retried {
		prefix += "；kageos 已自动精简上下文并重试一次，仍未完成"
	}
	return fmt.Errorf("%s。%s，请到「LLM 管理」调大最大 Token；推理模型还可在额外配置中降低 reasoning_effort 后重试", prefix, limit)
}

type contentData struct {
	Content string `json:"content"`
}

type thinkingData struct {
	Content string `json:"content"`
}

type errorData struct {
	Message string `json:"message"`
}

func processStreamChunks(
	ctx context.Context,
	stream <-chan *llms.StreamChunk,
	sendEvent func(string, interface{}),
	round int,
) (string, string, []llms.ToolCall, *llms.Usage, error) {
	var buf strings.Builder
	var thinkingBuf strings.Builder
	allToolCalls := make([]llms.ToolCall, 0)
	toolCallsIndex := make(map[string]int)
	finalToolCalls := make([]llms.ToolCall, 0)
	finalToolCallsReceived := false
	finishReason := ""
	var usage *llms.Usage
	thinkFilter := newThinkTagFilter()
	sawReasoningContent := false

	// 增量+节流：累积 delta，满足条件时 flush
	pendingDeltas := make(map[int]string)
	pendingNames := make(map[int]string)
	pendingIDs := make(map[int]string)
	var lastSendTime time.Time

	flushToolCallsDelta := func() {
		if len(pendingDeltas) == 0 && len(pendingNames) == 0 && len(pendingIDs) == 0 {
			return
		}
		seen := make(map[int]bool)
		for idx := range pendingDeltas {
			seen[idx] = true
		}
		for idx := range pendingNames {
			seen[idx] = true
		}
		for idx := range pendingIDs {
			seen[idx] = true
		}
		indices := make([]int, 0, len(seen))
		for idx := range seen {
			indices = append(indices, idx)
		}
		sort.Ints(indices)
		updates := make([]dto.WorkspaceStreamToolCallDeltaUpdate, 0, len(indices))
		for _, idx := range indices {
			delta := pendingDeltas[idx]
			name := pendingNames[idx]
			id := pendingIDs[idx]
			updates = append(updates, dto.WorkspaceStreamToolCallDeltaUpdate{Index: idx, Round: round, ID: id, Name: name, Delta: delta})
		}
		if len(updates) > 0 {
			sendEvent(EventToolCallsStreamDelta, &dto.WorkspaceStreamToolCallDeltaData{Updates: updates})
		}
		for k := range pendingDeltas {
			delete(pendingDeltas, k)
		}
		for k := range pendingNames {
			delete(pendingNames, k)
		}
		for k := range pendingIDs {
			delete(pendingIDs, k)
		}
		lastSendTime = time.Now()
	}

	for ch := range stream {
		select {
		case <-ctx.Done():
			logger.Infof(ctx, "[StreamLoop] 上下文已取消，停止处理")
			return "", "", nil, usage, ctx.Err()
		default:
		}
		if ch.Usage != nil {
			usage = ch.Usage
		}
		if ch.FinishReason != "" {
			finishReason = ch.FinishReason
		}
		if ch.ReasoningContent != "" {
			sawReasoningContent = true
			thinkingBuf.WriteString(ch.ReasoningContent)
			sendEvent(EventThinking, &thinkingData{Content: ch.ReasoningContent})
		}
		if ch.Error != "" {
			flushToolCallsDelta()
			return strings.TrimSpace(buf.String()), strings.TrimSpace(thinkingBuf.String()), allToolCalls, usage, fmt.Errorf("LLM 流式错误: %s", ch.Error)
		}
		if ch.Content != "" {
			filtered := thinkFilter.Append(ch.Content)
			if filtered.Thinking != "" {
				thinkingBuf.WriteString(filtered.Thinking)
				sendEvent(EventThinking, &thinkingData{Content: filtered.Thinking})
			}
			if filtered.Content != "" {
				buf.WriteString(filtered.Content)
				sendEvent(EventContent, &contentData{Content: filtered.Content})
			}
		}
		if len(ch.ToolCallDeltas) > 0 {
			prevArgs := make([]string, len(allToolCalls))
			prevNames := make([]string, len(allToolCalls))
			prevIDs := make([]string, len(allToolCalls))
			for i := range allToolCalls {
				prevArgs[i] = allToolCalls[i].Function.Arguments
				prevNames[i] = allToolCalls[i].Function.Name
				prevIDs[i] = allToolCalls[i].ID
			}
			prevLen := len(allToolCalls)

			allToolCalls, toolCallsIndex = mergeToolCallDeltas(ch.ToolCallDeltas, allToolCalls, toolCallsIndex)

			// 计算 delta 并累积
			totalPending := 0
			for i := 0; i < len(allToolCalls); i++ {
				newArgs := allToolCalls[i].Function.Arguments
				delta := ""
				name := ""
				id := ""
				if i < prevLen {
					oldLen := len(prevArgs[i])
					if len(newArgs) > oldLen {
						delta = newArgs[oldLen:]
					}
					if allToolCalls[i].Function.Name != "" && allToolCalls[i].Function.Name != prevNames[i] {
						name = allToolCalls[i].Function.Name
					}
					if allToolCalls[i].ID != "" && allToolCalls[i].ID != prevIDs[i] {
						id = allToolCalls[i].ID
					}
				} else {
					name = allToolCalls[i].Function.Name
					id = allToolCalls[i].ID
					delta = newArgs
				}
				if delta != "" || name != "" || id != "" {
					pendingDeltas[i] += delta
					totalPending += len(delta)
					if name != "" {
						pendingNames[i] = name
					}
					if id != "" {
						pendingIDs[i] = id
					}
				}
			}

			// 节流：时间或字数满足则 flush
			now := time.Now()
			if totalPending >= throttleSizeChars || (!lastSendTime.IsZero() && now.Sub(lastSendTime) >= throttleIntervalMs*time.Millisecond) {
				flushToolCallsDelta()
			} else if lastSendTime.IsZero() && len(pendingDeltas) > 0 {
				lastSendTime = now
			}
		}
		if ch.Done && len(ch.FinalToolCalls) > 0 {
			finalToolCalls = append(finalToolCalls[:0], ch.FinalToolCalls...)
			finalToolCallsReceived = true
		}
	}

	// 流结束，flush 剩余
	flushToolCallsDelta()
	if tail := thinkFilter.Finish(); tail != "" {
		buf.WriteString(tail)
		sendEvent(EventContent, &contentData{Content: tail})
	}

	content := strings.TrimSpace(buf.String())
	switch finishReason {
	case "length", "max_tokens", "max_output_tokens":
		if (thinkFilter.SawThink() || sawReasoningContent) && content == "" {
			return content, strings.TrimSpace(thinkingBuf.String()), nil, usage, &outputLimitError{thinkingOnly: true}
		}
		return content, strings.TrimSpace(thinkingBuf.String()), nil, usage, &outputLimitError{}
	case "content_filter":
		return content, strings.TrimSpace(thinkingBuf.String()), nil, usage, fmt.Errorf("LLM 响应被内容安全策略截断")
	}
	if finalToolCallsReceived {
		allToolCalls = finalToolCalls
	} else {
		allToolCalls = normalizeToolCalls(ctx, allToolCalls)
	}
	if finishReason == "tool_calls" && len(allToolCalls) == 0 {
		return content, strings.TrimSpace(thinkingBuf.String()), nil, usage, fmt.Errorf("LLM 结束原因为 tool_calls，但未返回完整工具调用")
	}
	if err := validateToolCallsForExecution(allToolCalls); err != nil {
		return content, strings.TrimSpace(thinkingBuf.String()), nil, usage, err
	}
	return content, strings.TrimSpace(thinkingBuf.String()), allToolCalls, usage, nil
}

func validateToolCallsForExecution(toolCalls []llms.ToolCall) error {
	for i, tc := range toolCalls {
		if strings.TrimSpace(tc.ID) == "" {
			return fmt.Errorf("LLM 返回的第 %d 个 tool_call 缺少 id", i)
		}
		if strings.TrimSpace(tc.Function.Name) == "" {
			return fmt.Errorf("LLM 返回的第 %d 个 tool_call 缺少 function.name", i)
		}
		args := strings.TrimSpace(tc.Function.Arguments)
		if args != "" && !json.Valid([]byte(args)) {
			return fmt.Errorf("LLM 返回的第 %d 个 tool_call(%s) 参数不是合法 JSON，已中止本轮工具执行", i, tc.Function.Name)
		}
	}
	return nil
}

func addLLMUsage(a, b *llms.Usage) *llms.Usage {
	if a == nil && b == nil {
		return nil
	}
	out := &llms.Usage{}
	if a != nil {
		out.PromptTokens += a.PromptTokens
		out.CompletionTokens += a.CompletionTokens
		out.TotalTokens += a.TotalTokens
		out.CachedTokens += a.CachedTokens
		out.CachedTokensReported = out.CachedTokensReported || a.CachedTokensReported
	}
	if b != nil {
		out.PromptTokens += b.PromptTokens
		out.CompletionTokens += b.CompletionTokens
		out.TotalTokens += b.TotalTokens
		out.CachedTokens += b.CachedTokens
		out.CachedTokensReported = out.CachedTokensReported || b.CachedTokensReported
	}
	return out
}

// appendToolCallArgs 仅当当前 arguments 还不是合法 JSON 时才追加 delta，避免兼容端先发完整 JSON 再发后缀导致重复拼接成无效 JSON。
func appendToolCallArgs(cur, delta string) string {
	if delta == "" {
		return cur
	}
	trimmed := strings.TrimSpace(cur)
	if trimmed != "" && json.Valid([]byte(trimmed)) {
		return trimmed
	}
	return cur + delta
}

func mergeToolCallDeltas(chunkToolCalls []llms.ToolCallDelta, allToolCalls []llms.ToolCall, toolCallsIndex map[string]int) ([]llms.ToolCall, map[string]int) {
	for _, tc := range chunkToolCalls {
		if tc.Index != nil {
			idx := *tc.Index
			if tc.ID != "" {
				if existingIdx, ok := toolCallsIndex[tc.ID]; ok {
					idx = existingIdx
				}
			}
			if idx < 0 {
				idx = len(allToolCalls)
			}
			for len(allToolCalls) <= idx {
				allToolCalls = append(allToolCalls, llms.ToolCall{Type: "function"})
			}
			mergeToolCallFields(&allToolCalls[idx], tc)
			if allToolCalls[idx].ID != "" {
				toolCallsIndex[allToolCalls[idx].ID] = idx
			}
			continue
		}
		if tc.ID != "" {
			if idx, ok := toolCallsIndex[tc.ID]; ok {
				mergeToolCallFields(&allToolCalls[idx], tc)
			} else if lastIdx := lastAnonymousStartedToolCallIndex(allToolCalls); lastIdx >= 0 {
				// 流式先发 name 再发 id（按 index 分片）：最后一条是 id 为空的同一 tool_call，合并到该条，避免出现两条（一条 id 空）导致 API 报 insufficient tool messages
				mergeToolCallFields(&allToolCalls[lastIdx], tc)
				toolCallsIndex[tc.ID] = lastIdx
			} else {
				allToolCalls = append(allToolCalls, toolCallFromDelta(tc))
				toolCallsIndex[tc.ID] = len(allToolCalls) - 1
			}
		} else if tc.Function.Arguments != "" {
			if idx := onlyOpenToolCallIndex(allToolCalls); idx >= 0 {
				allToolCalls[idx].Function.Arguments = appendToolCallArgs(allToolCalls[idx].Function.Arguments, tc.Function.Arguments)
			} else if len(allToolCalls) == 0 && tc.Function.Name != "" {
				allToolCalls = append(allToolCalls, toolCallFromDelta(tc))
			}
		} else if tc.Function.Name != "" {
			if len(allToolCalls) == 0 || allToolCalls[len(allToolCalls)-1].ID != "" {
				allToolCalls = append(allToolCalls, toolCallFromDelta(tc))
			}
		}
	}
	return allToolCalls, toolCallsIndex
}

func toolCallFromDelta(delta llms.ToolCallDelta) llms.ToolCall {
	tc := llms.ToolCall{ID: delta.ID, Type: delta.Type, Function: delta.Function}
	if tc.Type == "" {
		tc.Type = "function"
	}
	return tc
}

func mergeToolCallFields(dst *llms.ToolCall, src llms.ToolCallDelta) {
	if dst.Type == "" {
		dst.Type = "function"
	}
	if src.ID != "" {
		dst.ID = src.ID
	}
	if src.Type != "" {
		dst.Type = src.Type
	}
	if src.Function.Name != "" {
		dst.Function.Name = src.Function.Name
	}
	if src.Function.Arguments != "" {
		dst.Function.Arguments = appendToolCallArgs(dst.Function.Arguments, src.Function.Arguments)
	}
}

func lastAnonymousStartedToolCallIndex(toolCalls []llms.ToolCall) int {
	for i := len(toolCalls) - 1; i >= 0; i-- {
		tc := toolCalls[i]
		if strings.TrimSpace(tc.ID) != "" {
			return -1
		}
		if strings.TrimSpace(tc.Function.Name) != "" || strings.TrimSpace(tc.Function.Arguments) != "" {
			return i
		}
	}
	return -1
}

func onlyOpenToolCallIndex(toolCalls []llms.ToolCall) int {
	found := -1
	for i, tc := range toolCalls {
		if isBlankToolCall(tc) {
			continue
		}
		args := strings.TrimSpace(tc.Function.Arguments)
		if args == "" || !json.Valid([]byte(args)) {
			if found >= 0 {
				return -1
			}
			found = i
		}
	}
	return found
}

func normalizeToolCalls(ctx context.Context, toolCalls []llms.ToolCall) []llms.ToolCall {
	out := make([]llms.ToolCall, 0, len(toolCalls))
	for i, tc := range toolCalls {
		if isBlankToolCall(tc) {
			continue
		}
		if tc.Type == "" {
			tc.Type = "function"
		}
		if strings.TrimSpace(tc.ID) == "" {
			tc.ID = fmt.Sprintf("call_local_%d", i)
			logger.Warnf(ctx, "[StreamLoop] tool_call id 为空，已生成本地 ID: %s, ToolName: %s", tc.ID, tc.Function.Name)
		}
		out = append(out, tc)
	}
	return out
}

func isBlankToolCall(tc llms.ToolCall) bool {
	return strings.TrimSpace(tc.ID) == "" &&
		strings.TrimSpace(tc.Function.Name) == "" &&
		strings.TrimSpace(tc.Function.Arguments) == "" &&
		(strings.TrimSpace(tc.Type) == "" || strings.TrimSpace(tc.Type) == "function")
}
