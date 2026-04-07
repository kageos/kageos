package dto

import "encoding/json"

// RecordTableOperateLogReq 记录 Table 操作日志请求
type RecordTableOperateLogReq struct {
	TenantUser  string          `json:"tenant_user"`                                  // 租户用户（app 的所有者，从路径解析）
	RequestUser string          `json:"request_user"`                                 // 请求用户（实际执行操作的用户）
	App         string          `json:"app"`                                          // 应用名
	Router      string          `json:"router"`                                       // 路由路径（如：crm/crm_ticket）
	Action      string          `json:"action"`                                       // 操作类型：OnTableAddRow, OnTableUpdateRow, OnTableDeleteRows
	RowID       int64           `json:"row_id"`                                       // 记录ID（OnTableUpdateRow 和 OnTableDeleteRows 需要）
	RowIDs      []int64         `json:"row_ids"`                                      // 记录ID列表（OnTableDeleteRows 需要，批量删除）
	Body        json.RawMessage `json:"body" swaggertype:"string" example:"{}"`       // 请求体（OnTableAddRow 需要）
	Updates     json.RawMessage `json:"updates" swaggertype:"string" example:"{}"`    // 更新的字段和值（OnTableUpdateRow 需要）
	OldValues   json.RawMessage `json:"old_values" swaggertype:"string" example:"{}"` // 更新前的值（OnTableUpdateRow 需要）
	IPAddress   string          `json:"ip_address"`                                   // IP地址
	UserAgent   string          `json:"user_agent"`                                   // User Agent
	TraceID     string          `json:"trace_id"`                                     // 追踪ID
}

// RecordFormOperateLogReq 记录 Form 操作日志请求
type RecordFormOperateLogReq struct {
	TenantUser     string          `json:"tenant_user"`                                     // 租户用户（app 的所有者，从路径解析）
	RequestUser    string          `json:"request_user"`                                    // 请求用户（实际执行操作的用户）
	App            string          `json:"app"`                                             // 应用名
	Router         string          `json:"router"`                                          // 路由路径（如：plugins/cashier_desk）
	Source         string          `json:"source"`                                          // 来源（如：browser、scheduled_task、agent、api）
	Action         string          `json:"action"`                                          // 操作类型：form_submit 或 request_app
	FunctionMethod string          `json:"function_method"`                                 // HTTP 方法
	RequestBody    json.RawMessage `json:"request_body" swaggertype:"string" example:"{}"`  // 请求体
	ResponseBody   json.RawMessage `json:"response_body" swaggertype:"string" example:"{}"` // 响应体
	IPAddress      string          `json:"ip_address"`                                      // IP地址
	UserAgent      string          `json:"user_agent"`                                      // User Agent
	TraceID        string          `json:"trace_id"`                                        // 追踪ID
	Version        string          `json:"version"`                                         // 应用版本（可选）
}
