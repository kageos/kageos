package dto

// GetTableOperateLogsReq 查询 Table 操作日志请求
type GetTableOperateLogsReq struct {
	TenantUser  string `json:"tenant_user" form:"tenant_user"`   // 租户用户（app 的所有者）
	RequestUser string `json:"request_user" form:"request_user"` // 请求用户（实际执行操作的用户）
	App         string `json:"app" form:"app"`                   // 应用名
	FullCodePath string `json:"full_code_path" form:"full_code_path"` // 完整代码路径
	RowID       int64  `json:"row_id" form:"row_id"`             // 记录ID
	Action      string `json:"action" form:"action"`              // 操作类型：OnTableAddRow, OnTableUpdateRow, OnTableDeleteRows
	Page        int    `json:"page" form:"page"`                 // 页码（从1开始）
	PageSize    int    `json:"page_size" form:"page_size"`       // 每页数量
	OrderBy     string `json:"order_by" form:"order_by"`         // 排序字段（默认：created_at DESC）
}

// GetTableOperateLogsResp 查询 Table 操作日志响应
type GetTableOperateLogsResp struct {
	Logs     interface{} `json:"logs"`      // 日志列表
	Total    int64       `json:"total"`     // 总数
	Page     int         `json:"page"`      // 当前页码
	PageSize int         `json:"page_size"` // 每页数量
}

// GetFormOperateLogsReq 查询 Form 操作日志请求
type GetFormOperateLogsReq struct {
	TenantUser  string `json:"tenant_user" form:"tenant_user"`   // 租户用户（app 的所有者）
	RequestUser string `json:"request_user" form:"request_user"` // 请求用户（实际执行操作的用户）
	App         string `json:"app" form:"app"`                    // 应用名
	FullCodePath string `json:"full_code_path" form:"full_code_path"` // 完整代码路径
	Action      string `json:"action" form:"action"`              // 操作类型：request_app, form_submit
	Page        int    `json:"page" form:"page"`                 // 页码（从1开始）
	PageSize    int    `json:"page_size" form:"page_size"`       // 每页数量
	OrderBy     string `json:"order_by" form:"order_by"`         // 排序字段（默认：created_at DESC）
}

// GetFormOperateLogsResp 查询 Form 操作日志响应
type GetFormOperateLogsResp struct {
	Logs     interface{} `json:"logs"`      // 日志列表
	Total    int64       `json:"total"`     // 总数
	Page     int         `json:"page"`      // 当前页码
	PageSize int         `json:"page_size"` // 每页数量
}

