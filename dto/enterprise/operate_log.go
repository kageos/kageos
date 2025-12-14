package enterprise

// CreateOperateLoggerReq 创建操作日志请求
type CreateOperateLoggerReq struct {
	User       string                 `json:"user"`        // 操作用户
	Action     string                 `json:"action"`      // 操作类型（如：request_app, create_app, update_app 等）
	Resource   string                 `json:"resource"`   // 资源类型（如：app, function, service_tree 等）
	ResourceID string                 `json:"resource_id"` // 资源ID（如：app code, function id 等）
	Changes    map[string]interface{} `json:"changes,omitempty"` // 变更内容（可选）
	IPAddress  string                 `json:"ip_address,omitempty"` // IP 地址（可选）
	UserAgent  string                 `json:"user_agent,omitempty"` // User Agent（可选）
}

// CreateOperateLoggerResp 创建操作日志响应
type CreateOperateLoggerResp struct {
	ID int64 `json:"id,omitempty"` // 日志ID（企业版返回）
}
