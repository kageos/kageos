package dto

import "encoding/json"

// CreateOperateLoggerReq 创建操作日志请求（企业版）
type CreateOperateLoggerReq struct {
	User       string          `json:"user"`                 // 操作用户
	Action     string          `json:"action"`               // 操作类型（如：OnTableUpdateRow, OnTableDeleteRows, form_submit, request_app 等）
	Resource   string          `json:"resource"`             // 资源类型（如：table, form, app 等）
	ResourceID string          `json:"resource_id"`          // 资源ID（格式：user/app 或 user/app/router）
	IPAddress  string          `json:"ip_address,omitempty"` // IP 地址（可选）
	UserAgent  string          `json:"user_agent,omitempty"` // User Agent（可选）
	TraceID    string          `json:"trace_id,omitempty"`  // 追踪ID（可选）

	// Table 操作相关字段
	RowID     int64           `json:"row_id,omitempty"`     // 记录ID（Table 操作需要）
	Updates   json.RawMessage `json:"updates,omitempty"`     // 更新的字段和值（Table 更新操作需要）
	OldValues json.RawMessage `json:"old_values,omitempty"` // 更新前的值（Table 更新操作需要）

	// Form 操作相关字段
	Router        string          `json:"router,omitempty"`         // 路由路径（Form 操作需要）
	Method        string          `json:"method,omitempty"`         // HTTP 方法（Form 操作需要）
	RequestBody   json.RawMessage `json:"request_body,omitempty"`  // 请求体（Form 操作需要）
	ResponseBody  json.RawMessage `json:"response_body,omitempty"` // 响应体（Form 操作需要）

	// 通用字段
	Version string `json:"version,omitempty"` // 应用版本（可选，如果为空会自动查询）
}

// CreateOperateLoggerResp 创建操作日志响应（企业版）
type CreateOperateLoggerResp struct {
	ID int64 `json:"id,omitempty"` // 日志ID（企业版返回）
}
