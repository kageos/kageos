package streamloop

import "github.com/ai-agent-os/ai-agent-os/dto"

// ToolCallSummary 单次工具调用摘要（与 dto.WorkspaceChatToolCallSummary 对齐，供 OnDone 等使用）
type ToolCallSummary struct {
	Name       string                  `json:"name"`
	Status     string                  `json:"status"`
	Arguments  string                  `json:"arguments,omitempty"`
	Result     string                  `json:"result,omitempty"`
	ResultData interface{}             `json:"result_data,omitempty"`
	Metadata   *dto.ToolResultMetadata `json:"metadata,omitempty"`
	Error      string                  `json:"error,omitempty"`
}
