package streamloop

// ToolCallSummary 单次工具调用摘要（与 dto.WorkspaceChatToolCallSummary 对齐，供 OnDone 等使用）
type ToolCallSummary struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Arguments string `json:"arguments,omitempty"`
	Result    string `json:"result,omitempty"`
	Error     string `json:"error,omitempty"`
}
