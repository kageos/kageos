package dto

import "encoding/json"

// CreateScheduledTaskReq 创建定时任务请求
type CreateScheduledTaskReq struct {
	Name            string          `json:"name" binding:"required"`           // 任务名称
	FullCodePath    string          `json:"full_code_path" binding:"required"` // 如 /system/official/message/send_message_to_users.form
	Action          string          `json:"action"`                            // 执行动作：空/execute/form(普通函数)、table_create、table_update、table_delete
	Method          string          `json:"method"`                            // GET/POST，默认 POST
	Payload         json.RawMessage `json:"payload"`                           // 请求体 JSON
	RequestUser     string          `json:"request_user"`                      // 以谁执行，空则用当前用户
	RequestUserDept string          `json:"request_user_dept"`                 // 请求用户部门路径（透传，空则用创建人部门）
	ScheduleType    string          `json:"schedule_type" binding:"required"`  // atime / cron / every
	RunAt           string          `json:"run_at"`                            // 仅 schedule_type=atime 时必填；cron/every 由服务端按创建时间自动补
	CronExpr        string          `json:"cron_expr"`                         // schedule_type=cron 时必填
	IntervalSeconds int64           `json:"interval_seconds"`                  // schedule_type=every 时必填
	MaxRuns         int             `json:"max_runs"`                          // schedule_type=every 时可选，0=不限制
	Timezone        string          `json:"timezone"`                          // 可选
}

// ScheduledTaskItem 定时任务列表项
type ScheduledTaskItem struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	User            string  `json:"user"`
	App             string  `json:"app"`
	FullCodePath    string  `json:"full_code_path"`
	Action          string  `json:"action"`
	Method          string  `json:"method"`
	Payload         string  `json:"payload"`
	RequestUser     string  `json:"request_user"`
	RequestUserDept string  `json:"request_user_dept"`
	CreatedBy       string  `json:"created_by"`
	ScheduleType    string  `json:"schedule_type"`
	RunAt           string  `json:"run_at"` // atime 的执行时间；cron/every 的生效时间
	NextRunAt       *string `json:"next_run_at,omitempty"`
	CronExpr        string  `json:"cron_expr,omitempty"`
	IntervalSeconds int64   `json:"interval_seconds,omitempty"`
	MaxRuns         int     `json:"max_runs,omitempty"`
	Timezone        string  `json:"timezone,omitempty"`
	Status          string  `json:"status"`
	RunCount        int     `json:"run_count"`
	ErrorMessage    string  `json:"error_message,omitempty"`
	CreatedAt       string  `json:"created_at"`
}

// ScheduledTaskExecutionItem 执行记录列表项
type ScheduledTaskExecutionItem struct {
	ID              int64  `json:"id"`
	TaskID          int64  `json:"task_id"`
	ExecutedAt      string `json:"executed_at"`
	Status          string `json:"status"`
	DurationMillis  int64  `json:"duration_millis,omitempty"`
	RequestPayload  string `json:"request_payload"`  // JSON 字符串
	ResponsePayload string `json:"response_payload"` // JSON 字符串
	ErrorMessage    string `json:"error_message,omitempty"`
	TraceID         string `json:"trace_id,omitempty"`
	CreatedAt       string `json:"created_at"`
}

// ListScheduledTasksResp 定时任务列表响应
type ListScheduledTasksResp struct {
	List  []ScheduledTaskItem `json:"list"`
	Total int64               `json:"total"`
}

// ListScheduledTaskExecutionsResp 定时任务执行记录列表响应
type ListScheduledTaskExecutionsResp struct {
	List  []ScheduledTaskExecutionItem `json:"list"`
	Total int64                        `json:"total"`
}
