package streamloop

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/llms"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

const (
	EventContent            = "content"
	EventToolCallsStream    = "tool_calls_stream"    // 保留兼容，新协议用 delta
	EventToolCallsStreamDelta = "tool_calls_stream_delta" // 增量+节流，节省带宽
	EventError              = "error"
	MaxToolRounds           = 100 // 最大工具调用轮数，防止无限循环；过小易中断，过大增加耗时与成本

	// 节流参数：满足任一条件即 flush
	throttleIntervalMs = 100  // 距上次发送超过 100ms
	throttleSizeChars  = 200 // 累积 delta 超过 200 字
)

// RunStreamLoop 流式工具对话循环：从 BuildMessages 开始，调 LLM 流式，若有 tool_calls 则执行并递归，否则结束
func RunStreamLoop(ctx context.Context, deps StreamLoopDeps) error {
	return runStreamLoopRound(ctx, deps, 0, nil)
}

func runStreamLoopRound(ctx context.Context, deps StreamLoopDeps, round int, previousSummaries []ToolCallSummary) error {
	if round >= MaxToolRounds {
		logger.Warnf(ctx, "[StreamLoop] 达到最大工具调用轮数 %d，停止循环", MaxToolRounds)
		// 发一句提示，避免前端“戛然而止”显得乱
		deps.SendEvent(EventContent, &contentData{Content: "\n\n---\n已达到本轮最大工具调用次数，如需继续请再次发送消息。"})
		deps.OnDone(previousSummaries)
		return nil
	}

	msgs, tools, err := deps.BuildMessages(ctx)
	if err != nil {
		deps.SendEvent(EventError, &errorData{Message: err.Error()})
		return err
	}
	client, chatReq, err := deps.PrepareLLM(ctx, msgs, tools)
	if err != nil {
		deps.SendEvent(EventError, &errorData{Message: err.Error()})
		return err
	}
	stream, err := client.ChatStream(ctx, chatReq)
	if err != nil {
		deps.SendEvent(EventError, &errorData{Message: "LLM 调用失败: " + err.Error()})
		return err
	}

	content, allToolCalls, err := processStreamChunks(ctx, stream, deps.SendEvent)
	if err != nil {
		deps.SendEvent(EventError, &errorData{Message: err.Error()})
		return err
	}

	if len(allToolCalls) > 0 {
		if err := deps.SaveAssistantMessageWithToolCalls(ctx, content, allToolCalls); err != nil {
			logger.Warnf(ctx, "[StreamLoop] 保存 assistant 消息失败: %v", err)
			deps.SendEvent(EventError, &errorData{Message: "保存 assistant 消息失败: " + err.Error()})
			return err
		}
		summaries, err := deps.ExecuteToolCalls(ctx, allToolCalls, content, deps.SendEvent)
		if err != nil {
			deps.SendEvent(EventError, &errorData{Message: err.Error()})
			return err
		}
		combined := append(previousSummaries, summaries...)
		return runStreamLoopRound(ctx, deps, round+1, combined)
	}

	if err := deps.SaveAssistantMessage(ctx, content); err != nil {
		logger.Warnf(ctx, "[StreamLoop] 保存 assistant 消息失败: %v", err)
	}
	deps.OnDone(previousSummaries)
	return nil
}

type contentData struct {
	Content string `json:"content"`
}

type toolCallsStreamItem struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type toolCallsStreamData struct {
	ToolCalls []toolCallsStreamItem `json:"tool_calls"`
}

// toolCallsStreamDeltaUpdate 增量更新项（index + 可选 name + delta）
type toolCallsStreamDeltaUpdate struct {
	Index int    `json:"index"`
	Name  string `json:"name,omitempty"` // 仅新 tool_call 首次出现时
	Delta string `json:"delta"`
}

type toolCallsStreamDeltaData struct {
	Updates []toolCallsStreamDeltaUpdate `json:"updates"`
}

type errorData struct {
	Message string `json:"message"`
}

func processStreamChunks(
	ctx context.Context,
	stream <-chan *llms.StreamChunk,
	sendEvent func(string, interface{}),
) (string, []llms.ToolCall, error) {
	var buf strings.Builder
	allToolCalls := make([]llms.ToolCall, 0)
	toolCallsIndex := make(map[string]int)

	// 增量+节流：累积 delta，满足条件时 flush
	pendingDeltas := make(map[int]string)
	pendingNames := make(map[int]string)
	var lastSendTime time.Time

	flushToolCallsDelta := func() {
		if len(pendingDeltas) == 0 && len(pendingNames) == 0 {
			return
		}
		seen := make(map[int]bool)
		for idx := range pendingDeltas {
			seen[idx] = true
		}
		for idx := range pendingNames {
			seen[idx] = true
		}
		indices := make([]int, 0, len(seen))
		for idx := range seen {
			indices = append(indices, idx)
		}
		sort.Ints(indices)
		updates := make([]toolCallsStreamDeltaUpdate, 0, len(indices))
		for _, idx := range indices {
			delta := pendingDeltas[idx]
			name := pendingNames[idx]
			updates = append(updates, toolCallsStreamDeltaUpdate{Index: idx, Name: name, Delta: delta})
		}
		if len(updates) > 0 {
			sendEvent(EventToolCallsStreamDelta, &toolCallsStreamDeltaData{Updates: updates})
		}
		for k := range pendingDeltas {
			delete(pendingDeltas, k)
		}
		for k := range pendingNames {
			delete(pendingNames, k)
		}
		lastSendTime = time.Now()
	}

	for ch := range stream {
		select {
		case <-ctx.Done():
			logger.Infof(ctx, "[StreamLoop] 上下文已取消，停止处理")
			return "", nil, ctx.Err()
		default:
		}
		if ch.Error != "" {
			flushToolCallsDelta()
			sendEvent(EventError, &errorData{Message: "LLM 流式错误: " + ch.Error})
			return "", nil, fmt.Errorf("LLM 流式错误: %s", ch.Error)
		}
		if ch.Content != "" {
			buf.WriteString(ch.Content)
			sendEvent(EventContent, &contentData{Content: ch.Content})
		}
		if len(ch.ToolCalls) > 0 {
			prevArgs := make([]string, len(allToolCalls))
			for i := range allToolCalls {
				prevArgs[i] = allToolCalls[i].Function.Arguments
			}
			prevLen := len(allToolCalls)

			allToolCalls, toolCallsIndex = mergeToolCalls(ch.ToolCalls, allToolCalls, toolCallsIndex)

			// 计算 delta 并累积
			totalPending := 0
			for i := 0; i < len(allToolCalls); i++ {
				newArgs := allToolCalls[i].Function.Arguments
				delta := ""
				name := ""
				if i < prevLen {
					oldLen := len(prevArgs[i])
					if len(newArgs) > oldLen {
						delta = newArgs[oldLen:]
					}
				} else {
					name = allToolCalls[i].Function.Name
					delta = newArgs
				}
				if delta != "" || name != "" {
					pendingDeltas[i] += delta
					totalPending += len(delta)
					if name != "" {
						pendingNames[i] = name
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
	}

	// 流结束，flush 剩余
	flushToolCallsDelta()

	content := strings.TrimSpace(buf.String())
	for _, tc := range allToolCalls {
		if tc.Function.Arguments == "" {
			logger.Warnf(ctx, "[StreamLoop] tool_call arguments 为空，ToolCallID: %s, ToolName: %s", tc.ID, tc.Function.Name)
		}
	}
	return content, allToolCalls, nil
}

// appendToolCallArgs 仅当当前 arguments 还不是合法 JSON 时才追加 delta，避免 MiniMax 等先发完整 JSON 再发后缀导致重复拼接成无效 JSON（2013）
func appendToolCallArgs(cur, delta string) string {
	if delta == "" {
		return cur
	}
	cur = strings.TrimSpace(cur)
	if cur != "" && json.Valid([]byte(cur)) {
		return cur
	}
	return cur + delta
}

func mergeToolCalls(chunkToolCalls []llms.ToolCall, allToolCalls []llms.ToolCall, toolCallsIndex map[string]int) ([]llms.ToolCall, map[string]int) {
	for _, tc := range chunkToolCalls {
		if tc.ID != "" {
			if idx, ok := toolCallsIndex[tc.ID]; ok {
				if tc.Function.Name != "" {
					allToolCalls[idx].Function.Name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					allToolCalls[idx].Function.Arguments = appendToolCallArgs(allToolCalls[idx].Function.Arguments, tc.Function.Arguments)
				}
			} else if len(allToolCalls) > 0 && allToolCalls[len(allToolCalls)-1].ID == "" {
				// 流式先发 name 再发 id（按 index 分片）：最后一条是 id 为空的同一 tool_call，合并到该条，避免出现两条（一条 id 空）导致 API 报 insufficient tool messages
				lastIdx := len(allToolCalls) - 1
				allToolCalls[lastIdx].ID = tc.ID
				if tc.Function.Name != "" {
					allToolCalls[lastIdx].Function.Name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					allToolCalls[lastIdx].Function.Arguments = appendToolCallArgs(allToolCalls[lastIdx].Function.Arguments, tc.Function.Arguments)
				}
				toolCallsIndex[tc.ID] = lastIdx
			} else {
				allToolCalls = append(allToolCalls, tc)
				toolCallsIndex[tc.ID] = len(allToolCalls) - 1
			}
		} else if tc.Function.Arguments != "" {
			if len(allToolCalls) > 0 {
				lastIdx := len(allToolCalls) - 1
				allToolCalls[lastIdx].Function.Arguments = appendToolCallArgs(allToolCalls[lastIdx].Function.Arguments, tc.Function.Arguments)
			}
			if len(allToolCalls) == 0 && tc.Function.Name != "" {
				allToolCalls = append(allToolCalls, tc)
			}
		} else if tc.Function.Name != "" {
			if len(allToolCalls) == 0 || allToolCalls[len(allToolCalls)-1].ID != "" {
				allToolCalls = append(allToolCalls, tc)
			}
		}
	}
	return allToolCalls, toolCallsIndex
}
