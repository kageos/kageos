package dto

import (
	"encoding/json"
	"time"
)

// RecordTableActionLogReq 记录 Table 操作日志请求
type RecordTableActionLogReq struct {
	TenantUser     string          `json:"tenant_user"`                                     // 租户用户（app 的所有者，从路径解析）
	RequestUser    string          `json:"request_user"`                                    // 请求用户（实际执行操作的用户）
	App            string          `json:"app"`                                             // 应用名
	Router         string          `json:"router"`                                          // 路由路径（如：crm/crm_ticket）
	Action         string          `json:"action"`                                          // 操作类型：OnTableAddRow, OnTableUpdateRow, OnTableDeleteRows
	Source         string          `json:"source"`                                          // 来源（如 browser、agent、openapi）
	RowID          int64           `json:"row_id"`                                          // 记录ID（OnTableUpdateRow 和 OnTableDeleteRows 需要）
	RowIDs         []int64         `json:"row_ids"`                                         // 记录ID列表（OnTableDeleteRows 需要，批量删除）
	Body           json.RawMessage `json:"body" swaggertype:"string" example:"{}"`          // 请求体（OnTableAddRow 需要）
	Updates        json.RawMessage `json:"updates" swaggertype:"string" example:"{}"`       // 更新的字段和值（OnTableUpdateRow 需要）
	OldValues      json.RawMessage `json:"old_values" swaggertype:"string" example:"{}"`    // 更新前的值（OnTableUpdateRow 需要）
	ResponseBody   json.RawMessage `json:"response_body" swaggertype:"string" example:"{}"` // 响应体或错误信息
	IPAddress      string          `json:"ip_address"`                                      // IP地址
	UserAgent      string          `json:"user_agent"`                                      // User Agent
	TraceID        string          `json:"trace_id"`                                        // 追踪ID
	Version        string          `json:"version"`                                         // 应用版本（可选）
	DurationMillis int64           `json:"duration_millis"`                                 // 执行耗时
	Status         string          `json:"status"`                                          // success/failed
	Summary        string          `json:"summary"`                                         // 人类可读摘要
}

// RecordFormOperateLogReq 记录 Form 提交操作日志请求。
type RecordFormOperateLogReq struct {
	TenantUser     string          `json:"tenant_user"`                                     // 租户用户（app 的所有者，从路径解析）
	RequestUser    string          `json:"request_user"`                                    // 请求用户（实际执行操作的用户）
	App            string          `json:"app"`                                             // 应用名
	Router         string          `json:"router"`                                          // 路由路径（如：tools/pdf.form）
	Action         string          `json:"action"`                                          // 操作类型：form_submit
	FunctionMethod string          `json:"function_method"`                                 // HTTP 方法
	RequestBody    json.RawMessage `json:"request_body" swaggertype:"string" example:"{}"`  // 请求体
	ResponseBody   json.RawMessage `json:"response_body" swaggertype:"string" example:"{}"` // 响应体
	IPAddress      string          `json:"ip_address"`                                      // IP地址
	UserAgent      string          `json:"user_agent"`                                      // User Agent
	TraceID        string          `json:"trace_id"`                                        // 追踪ID
	Version        string          `json:"version"`                                         // 应用版本（可选）
	DurationMillis int64           `json:"duration_millis"`                                 // 执行耗时
	Status         string          `json:"status"`                                          // success/failed
	Summary        string          `json:"summary"`                                         // 人类可读摘要
}

// TableActionLogDetails 是 operate_logs.details_json 中 Table 操作的固定结构。
type TableActionLogDetails struct {
	RowID          int64       `json:"row_id"`
	RowIDs         []int64     `json:"row_ids,omitempty"`
	Version        string      `json:"version,omitempty"`
	DurationMillis int64       `json:"duration_millis,omitempty"`
	SourceType     string      `json:"source_type,omitempty"`
	SourceRef      string      `json:"source_ref,omitempty"`
	ResponseBody   interface{} `json:"response_body,omitempty"`
}

// FormOperateLogDetails 是 operate_logs.details_json 中 Form 提交的固定结构。
type FormOperateLogDetails struct {
	Router         string      `json:"router"`
	Method         string      `json:"method"`
	Version        string      `json:"version,omitempty"`
	DurationMillis int64       `json:"duration_millis,omitempty"`
	SourceType     string      `json:"source_type,omitempty"`
	SourceRef      string      `json:"source_ref,omitempty"`
	RequestBody    interface{} `json:"request_body,omitempty"`
	ResponseBody   interface{} `json:"response_body,omitempty"`
}

// FunctionExecutionLogDetails 是函数级执行日志的固定结构。
type FunctionExecutionLogDetails struct {
	Router          string      `json:"router"`
	Method          string      `json:"method"`
	TemplateType    string      `json:"template_type,omitempty"`
	ScheduledAction string      `json:"scheduled_action,omitempty"`
	DurationMillis  int64       `json:"duration_millis,omitempty"`
	SourceType      string      `json:"source_type,omitempty"`
	SourceRef       string      `json:"source_ref,omitempty"`
	RequestPayload  interface{} `json:"request_payload,omitempty"`
	ResponseBody    interface{} `json:"response_body,omitempty"`
}

// AppCallLogResponseBody 是 Form/Table 审计中记录应用回调响应的固定结构。
type AppCallLogResponseBody struct {
	Code          int         `json:"code"`
	ErrCode       int         `json:"err_code,omitempty"`
	TraceID       string      `json:"trace_id,omitempty"`
	Version       string      `json:"version,omitempty"`
	Result        interface{} `json:"result,omitempty"`
	Message       string      `json:"msg,omitempty"`
	Error         string      `json:"error,omitempty"`
	TotalCostMill int64       `json:"total_cost_mill"`
}

// TeamRoleAssignedValues 是权限赋权日志的新值结构。
type TeamRoleAssignedValues struct {
	RoleCode  string     `json:"role_code"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// TeamRoleRemovedDetails 是权限移除日志的详情结构。
type TeamRoleRemovedDetails struct {
	RoleCode     string `json:"role_code"`
	RowsAffected int64  `json:"rows_affected"`
}

// ServiceTreeNodeLogValues 是目录/函数/文档节点审计的结构化快照。
type ServiceTreeNodeLogValues struct {
	ID           int64  `json:"id"`
	Type         string `json:"type"`
	Name         string `json:"name"`
	Code         string `json:"code"`
	Description  string `json:"description,omitempty"`
	Tags         string `json:"tags,omitempty"`
	Admins       string `json:"admins,omitempty"`
	AppID        int64  `json:"app_id"`
	RefID        int64  `json:"ref_id,omitempty"`
	FullCodePath string `json:"full_code_path"`
	TemplateType string `json:"template_type,omitempty"`
	Version      string `json:"version,omitempty"`
	VersionNum   int    `json:"version_num,omitempty"`
}

// ServiceTreeNodeLogDetails 是目录/函数/文档节点审计的固定详情。
type ServiceTreeNodeLogDetails struct {
	NodeID       int64  `json:"node_id"`
	NodeType     string `json:"node_type"`
	FullCodePath string `json:"full_code_path"`
}
