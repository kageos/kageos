package dto

import (
	"encoding/json"
	"time"
)

type CreateScheduledAgentTaskReq struct {
	Name              string          `json:"name" binding:"required"`
	FullCodePath      string          `json:"full_code_path" binding:"required"`
	Goal              string          `json:"goal" binding:"required"`
	ModeCode          string          `json:"mode_code"`
	Files             string          `json:"files,omitempty"`
	LLMConfigID       int64           `json:"llm_config_id"`
	ContextPolicy     json.RawMessage `json:"context_policy,omitempty"`
	ToolPolicy        json.RawMessage `json:"tool_policy,omitempty"`
	ApprovalPolicy    json.RawMessage `json:"approval_policy,omitempty"`
	BudgetPolicy      json.RawMessage `json:"budget_policy,omitempty"`
	ScheduleType      string          `json:"schedule_type" binding:"required"`
	RunAt             string          `json:"run_at,omitempty"`
	CronExpr          string          `json:"cron_expr,omitempty"`
	IntervalSeconds   int64           `json:"interval_seconds,omitempty"`
	MaxRuns           int             `json:"max_runs,omitempty"`
	Timezone          string          `json:"timezone,omitempty"`
	RequestUser       string          `json:"request_user,omitempty"`
	RequestUserDept   string          `json:"request_user_dept,omitempty"`
	NotifyUsers       []string        `json:"notify_users,omitempty"`
	NotifyDepartments []string        `json:"notify_departments,omitempty"`
	NotifyOn          string          `json:"notify_on,omitempty"`
}

type UpdateScheduledAgentTaskReq struct {
	Name              string          `json:"name,omitempty"`
	FullCodePath      string          `json:"full_code_path,omitempty"`
	Goal              string          `json:"goal,omitempty"`
	ModeCode          string          `json:"mode_code,omitempty"`
	Files             string          `json:"files,omitempty"`
	LLMConfigID       *int64          `json:"llm_config_id,omitempty"`
	ContextPolicy     json.RawMessage `json:"context_policy,omitempty"`
	ToolPolicy        json.RawMessage `json:"tool_policy,omitempty"`
	ApprovalPolicy    json.RawMessage `json:"approval_policy,omitempty"`
	BudgetPolicy      json.RawMessage `json:"budget_policy,omitempty"`
	ScheduleType      string          `json:"schedule_type,omitempty"`
	RunAt             string          `json:"run_at,omitempty"`
	CronExpr          string          `json:"cron_expr,omitempty"`
	IntervalSeconds   *int64          `json:"interval_seconds,omitempty"`
	MaxRuns           *int            `json:"max_runs,omitempty"`
	Timezone          string          `json:"timezone,omitempty"`
	RequestUser       string          `json:"request_user,omitempty"`
	RequestUserDept   string          `json:"request_user_dept,omitempty"`
	NotifyUsers       []string        `json:"notify_users,omitempty"`
	NotifyDepartments []string        `json:"notify_departments,omitempty"`
	NotifyOn          string          `json:"notify_on,omitempty"`
}

type ScheduledAgentTaskItem struct {
	ID                int64           `json:"id"`
	Name              string          `json:"name"`
	FullCodePath      string          `json:"full_code_path"`
	Goal              string          `json:"goal"`
	ModeCode          string          `json:"mode_code"`
	Files             string          `json:"files,omitempty"`
	LLMConfigID       int64           `json:"llm_config_id"`
	ContextPolicy     json.RawMessage `json:"context_policy,omitempty"`
	ToolPolicy        json.RawMessage `json:"tool_policy,omitempty"`
	ApprovalPolicy    json.RawMessage `json:"approval_policy,omitempty"`
	BudgetPolicy      json.RawMessage `json:"budget_policy,omitempty"`
	ScheduleType      string          `json:"schedule_type"`
	RunAt             time.Time       `json:"run_at"`
	NextRunAt         *time.Time      `json:"next_run_at,omitempty"`
	CronExpr          string          `json:"cron_expr,omitempty"`
	IntervalSeconds   int64           `json:"interval_seconds,omitempty"`
	MaxRuns           int             `json:"max_runs,omitempty"`
	Timezone          string          `json:"timezone,omitempty"`
	Status            string          `json:"status"`
	RunCount          int             `json:"run_count"`
	LastSessionID     string          `json:"last_session_id,omitempty"`
	LastExecutionID   int64           `json:"last_execution_id,omitempty"`
	LastErrorMessage  string          `json:"last_error_message,omitempty"`
	RequestUser       string          `json:"request_user,omitempty"`
	RequestUserDept   string          `json:"request_user_dept,omitempty"`
	NotifyUsers       []string        `json:"notify_users,omitempty"`
	NotifyDepartments []string        `json:"notify_departments,omitempty"`
	NotifyOn          string          `json:"notify_on,omitempty"`
	SourceType        string          `json:"source_type,omitempty"`
	SourceRef         string          `json:"source_ref,omitempty"`
	CreatedBy         string          `json:"created_by"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type ScheduledAgentExecutionItem struct {
	ID             int64           `json:"id"`
	TaskID         int64           `json:"task_id"`
	SessionID      string          `json:"session_id,omitempty"`
	ScheduledAt    time.Time       `json:"scheduled_at"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	FinishedAt     *time.Time      `json:"finished_at,omitempty"`
	Status         string          `json:"status"`
	DurationMillis int64           `json:"duration_millis"`
	InputGoal      string          `json:"input_goal,omitempty"`
	OutputSummary  string          `json:"output_summary,omitempty"`
	ToolCallCount  int             `json:"tool_call_count"`
	TokenUsage     json.RawMessage `json:"token_usage,omitempty"`
	ErrorMessage   string          `json:"error_message,omitempty"`
	TraceID        string          `json:"trace_id,omitempty"`
	SourceType     string          `json:"source_type,omitempty"`
	SourceRef      string          `json:"source_ref,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
