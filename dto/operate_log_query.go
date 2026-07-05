package dto

import "encoding/json"

// GetOperateLogsReq 查询通用操作日志请求。
type GetOperateLogsReq struct {
	ID                 int64  `json:"id" form:"id"`
	TenantUser         string `json:"tenant_user" form:"tenant_user"`
	CompanyCode        string `json:"company_code" form:"company_code"`
	ActorUser          string `json:"actor_user" form:"actor_user"`
	TargetUser         string `json:"target_user" form:"target_user"`
	App                string `json:"app" form:"app"`
	ResourceType       string `json:"resource_type" form:"resource_type"`
	ResourcePath       string `json:"resource_path" form:"resource_path"`
	ResourcePathPrefix string `json:"resource_path_prefix" form:"resource_path_prefix"`
	Action             string `json:"action" form:"action"`
	Status             string `json:"status" form:"status"`
	Source             string `json:"source" form:"source"`
	SourceType         string `json:"source_type" form:"source_type"`
	SourceRef          string `json:"source_ref" form:"source_ref"`
	ExecutorType       string `json:"executor_type" form:"executor_type"`
	WorkspaceSessionID string `json:"workspace_session_id" form:"workspace_session_id"`
	InitiatorUser      string `json:"initiator_user" form:"initiator_user"`
	WorkspaceMessageID int64  `json:"workspace_message_id" form:"workspace_message_id"`
	ToolCallID         string `json:"tool_call_id" form:"tool_call_id"`
	ToolName           string `json:"tool_name" form:"tool_name"`
	TraceID            string `json:"trace_id" form:"trace_id"`
	RowID              int64  `json:"row_id" form:"row_id"`
	Keyword            string `json:"keyword" form:"keyword"`
	Page               int    `json:"page" form:"page"`
	PageSize           int    `json:"page_size" form:"page_size"`
	OrderBy            string `json:"order_by" form:"order_by"`
}

// GetOperateLogsResp 查询通用操作日志响应。
type GetOperateLogsResp struct {
	Logs     []OperateLogItem `json:"logs"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

// OperateLogItem 是统一操作日志列表返回项。
type OperateLogItem struct {
	ID                    int64           `json:"id"`
	TenantUser            string          `json:"tenant_user"`
	CompanyCode           string          `json:"company_code"`
	App                   string          `json:"app"`
	ActorUser             string          `json:"actor_user"`
	Action                string          `json:"action"`
	ResourceType          string          `json:"resource_type"`
	ResourcePath          string          `json:"resource_path"`
	ResourceName          string          `json:"resource_name"`
	TargetUser            string          `json:"target_user"`
	TargetID              string          `json:"target_id"`
	Summary               string          `json:"summary"`
	DetailsJSON           json.RawMessage `json:"details_json" swaggertype:"object"`
	OldValuesJSON         json.RawMessage `json:"old_values_json" swaggertype:"object"`
	NewValuesJSON         json.RawMessage `json:"new_values_json" swaggertype:"object"`
	Status                string          `json:"status"`
	Source                string          `json:"source"`
	SourceType            string          `json:"source_type"`
	SourceRef             string          `json:"source_ref"`
	ExecutorType          string          `json:"executor_type"`
	WorkspaceSessionID    string          `json:"workspace_session_id"`
	WorkspaceSessionTitle string          `json:"workspace_session_title"`
	WorkspaceRole         string          `json:"workspace_role"`
	InitiatorUser         string          `json:"initiator_user"`
	WorkspaceMessageID    int64           `json:"workspace_message_id"`
	ToolCallID            string          `json:"tool_call_id"`
	ToolName              string          `json:"tool_name"`
	IPAddress             string          `json:"ip_address"`
	UserAgent             string          `json:"user_agent"`
	TraceID               string          `json:"trace_id"`
	CreatedAt             string          `json:"created_at"`
	UpdatedAt             string          `json:"updated_at"`
}
