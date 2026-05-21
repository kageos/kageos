package streamloop

import (
	"context"

	"github.com/kageos/kageos/pkg/llms"
)

// StreamLoopDeps 流式工具对话循环的依赖：由调用方（如工作台）实现并注入
type StreamLoopDeps interface {
	// BuildMessages 返回当前完整对话（含历史 + 刚保存的 user/assistant/tool）及本轮工具定义
	BuildMessages(ctx context.Context) ([]llms.Message, []llms.ToolDef, error)
	// PrepareLLM 根据当前上下文构造 LLM 客户端和本次请求
	PrepareLLM(ctx context.Context, msgs []llms.Message, tools []llms.ToolDef) (llms.LLMClient, *llms.ChatRequest, error)
	// SendEvent 向 SSE 发送事件（content / tool_calls_stream_delta / tool_call / error）
	SendEvent(event string, data interface{})
	// SaveAssistantMessage 保存纯文本 assistant 消息
	SaveAssistantMessage(ctx context.Context, content string) error
	// SaveAssistantMessageWithToolCalls 保存带 tool_calls 的 assistant 消息
	SaveAssistantMessageWithToolCalls(ctx context.Context, content string, toolCalls []llms.ToolCall) error
	// ExecuteToolCalls 按顺序执行工具、发 tool_call 事件、把每条 tool 结果写入 impl 的 store，返回摘要列表。
	ExecuteToolCalls(ctx context.Context, allToolCalls []llms.ToolCall, sendEvent func(string, interface{})) ([]ToolCallSummary, error)
	// OnDone 发送 EventDone（payload 含 session_id、tool_calls 等，由实现方决定）
	OnDone(summaries []ToolCallSummary)
}
