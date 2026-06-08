package streamloop

import "github.com/kageos/kageos/dto"

// ToolCallSummary 单次工具调用摘要（与 dto.WorkspaceChatToolCallSummary 对齐，供 OnDone 等使用）
type ToolCallSummary struct {
	ID         string                  `json:"id,omitempty"`
	Index      int                     `json:"index"`
	Round      int                     `json:"round"`
	Name       string                  `json:"name"`
	Status     string                  `json:"status"`
	Arguments  string                  `json:"arguments,omitempty"`
	Result     string                  `json:"result,omitempty"`
	ResultData interface{}             `json:"result_data,omitempty"`
	Metadata   *dto.ToolResultMetadata `json:"metadata,omitempty"`
	Error      string                  `json:"error,omitempty"`
}
