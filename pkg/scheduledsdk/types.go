package scheduledsdk

import (
	"encoding/json"
	"time"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusPaused    TaskStatus = "paused"
	TaskStatusDone      TaskStatus = "done"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

type ExecutionStatus string

const (
	ExecutionStatusQueued    ExecutionStatus = "queued"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusSuccess   ExecutionStatus = "success"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusTimeout   ExecutionStatus = "timeout"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

type Task struct {
	ID                  int64             `json:"id"`
	Title               string            `json:"title,omitempty"`
	Description         string            `json:"description,omitempty"`
	Category            string            `json:"category,omitempty"`
	Tags                []string          `json:"tags,omitempty"`
	IdempotencyKey      string            `json:"idempotency_key,omitempty"`
	ExecutorKey         string            `json:"executor_key"`
	ExecutorPayload     json.RawMessage   `json:"executor_payload,omitempty"`
	Metadata            map[string]string `json:"metadata,omitempty"`
	Status              TaskStatus        `json:"status"`
	Schedule            Schedule          `json:"schedule"`
	NextRunAt           *time.Time        `json:"next_run_at,omitempty"`
	RunCount            int               `json:"run_count"`
	InflightExecutionID int64             `json:"inflight_execution_id,omitempty"`
	LastExecutionID     int64             `json:"last_execution_id,omitempty"`
	LastErrorMessage    string            `json:"last_error_message,omitempty"`
	SourceType          string            `json:"source_type,omitempty"`
	SourceRef           string            `json:"source_ref,omitempty"`
	ResourceScope       string            `json:"resource_scope,omitempty"`
	ResourceKey         string            `json:"resource_key,omitempty"`
	RequestUser         string            `json:"request_user,omitempty"`
	RequestUserDept     string            `json:"request_user_dept,omitempty"`
	CreatedBy           string            `json:"created_by,omitempty"`
	CreatedAt           time.Time         `json:"created_at,omitempty"`
	UpdatedAt           time.Time         `json:"updated_at,omitempty"`
}

type Execution struct {
	ID               int64           `json:"id"`
	TaskID           int64           `json:"task_id"`
	ExecutorKey      string          `json:"executor_key"`
	Status           ExecutionStatus `json:"status"`
	TriggerType      string          `json:"trigger_type,omitempty"`
	ExecutorRunID    string          `json:"executor_run_id,omitempty"`
	ScheduledAt      time.Time       `json:"scheduled_at"`
	StartedAt        *time.Time      `json:"started_at,omitempty"`
	FinishedAt       *time.Time      `json:"finished_at,omitempty"`
	WorkerID         string          `json:"worker_id,omitempty"`
	LeaseUntil       *time.Time      `json:"lease_until,omitempty"`
	HeartbeatAt      *time.Time      `json:"heartbeat_at,omitempty"`
	HeartbeatMisses  int             `json:"heartbeat_misses,omitempty"`
	Attempt          int             `json:"attempt,omitempty"`
	DurationMillis   int64           `json:"duration_millis,omitempty"`
	OutputSummary    string          `json:"output_summary,omitempty"`
	ResultPayload    json.RawMessage `json:"result_payload,omitempty"`
	ErrorMessage     string          `json:"error_message,omitempty"`
	TraceID          string          `json:"trace_id,omitempty"`
	SourceType       string          `json:"source_type,omitempty"`
	SourceRef        string          `json:"source_ref,omitempty"`
	ResourceScope    string          `json:"resource_scope,omitempty"`
	ResourceKey      string          `json:"resource_key,omitempty"`
	RequestUser      string          `json:"request_user,omitempty"`
	RequestUserDept  string          `json:"request_user_dept,omitempty"`
	LastDispatchedAt *time.Time      `json:"last_dispatched_at,omitempty"`
	CreatedAt        time.Time       `json:"created_at,omitempty"`
	UpdatedAt        time.Time       `json:"updated_at,omitempty"`
}

type CreateTaskRequest struct {
	Title           string            `json:"title,omitempty"`
	Description     string            `json:"description,omitempty"`
	Category        string            `json:"category,omitempty"`
	Tags            []string          `json:"tags,omitempty"`
	IdempotencyKey  string            `json:"idempotency_key,omitempty"`
	ExecutorKey     string            `json:"executor_key"`
	ExecutorPayload json.RawMessage   `json:"executor_payload,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	Schedule        Schedule          `json:"schedule"`
	SourceType      string            `json:"source_type,omitempty"`
	SourceRef       string            `json:"source_ref,omitempty"`
	ResourceScope   string            `json:"resource_scope,omitempty"`
	ResourceKey     string            `json:"resource_key,omitempty"`
	RequestUser     string            `json:"request_user,omitempty"`
	RequestUserDept string            `json:"request_user_dept,omitempty"`
	CreatedBy       string            `json:"created_by,omitempty"`
}

type UpdateTaskRequest struct {
	Title           *string            `json:"title,omitempty"`
	Description     *string            `json:"description,omitempty"`
	Category        *string            `json:"category,omitempty"`
	Tags            *[]string          `json:"tags,omitempty"`
	ExecutorPayload json.RawMessage    `json:"executor_payload,omitempty"`
	Metadata        *map[string]string `json:"metadata,omitempty"`
	Schedule        *Schedule          `json:"schedule,omitempty"`
	SourceType      *string            `json:"source_type,omitempty"`
	SourceRef       *string            `json:"source_ref,omitempty"`
	ResourceScope   *string            `json:"resource_scope,omitempty"`
	ResourceKey     *string            `json:"resource_key,omitempty"`
	RequestUser     *string            `json:"request_user,omitempty"`
	RequestUserDept *string            `json:"request_user_dept,omitempty"`
}

type ListTasksRequest struct {
	ExecutorKey   string `json:"executor_key,omitempty"`
	Status        string `json:"status,omitempty"`
	Category      string `json:"category,omitempty"`
	SourceType    string `json:"source_type,omitempty"`
	SourceRef     string `json:"source_ref,omitempty"`
	ResourceScope string `json:"resource_scope,omitempty"`
	ResourceKey   string `json:"resource_key,omitempty"`
	CreatedBy     string `json:"created_by,omitempty"`
	Page          int    `json:"page,omitempty"`
	PageSize      int    `json:"page_size,omitempty"`
}

type ListTasksResponse struct {
	List  []*Task `json:"list"`
	Total int64   `json:"total"`
}

type ListExecutionsRequest struct {
	Status   string `json:"status,omitempty"`
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"page_size,omitempty"`
}

type ListExecutionsResponse struct {
	List  []*Execution `json:"list"`
	Total int64        `json:"total"`
}

type ExecutionRequestedEvent struct {
	EventID         string            `json:"event_id"`
	TaskID          int64             `json:"task_id"`
	ExecutionID     int64             `json:"execution_id"`
	ExecutorKey     string            `json:"executor_key"`
	ScheduledAt     time.Time         `json:"scheduled_at"`
	TraceID         string            `json:"trace_id,omitempty"`
	Attempt         int               `json:"attempt,omitempty"`
	SourceType      string            `json:"source_type,omitempty"`
	SourceRef       string            `json:"source_ref,omitempty"`
	ResourceScope   string            `json:"resource_scope,omitempty"`
	ResourceKey     string            `json:"resource_key,omitempty"`
	Token           string            `json:"token,omitempty"`
	RequestUser     string            `json:"request_user,omitempty"`
	RequestUserDept string            `json:"request_user_dept,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	ExecutorPayload json.RawMessage   `json:"executor_payload,omitempty"`
}

type MarkExecutionStartedRequest struct {
	TaskID        int64     `json:"task_id"`
	ExecutionID   int64     `json:"execution_id"`
	WorkerID      string    `json:"worker_id,omitempty"`
	StartedAt     time.Time `json:"started_at,omitempty"`
	ExecutorRunID string    `json:"executor_run_id,omitempty"`
}

type MarkExecutionHeartbeatRequest struct {
	TaskID      int64     `json:"task_id"`
	ExecutionID int64     `json:"execution_id"`
	WorkerID    string    `json:"worker_id,omitempty"`
	HeartbeatAt time.Time `json:"heartbeat_at,omitempty"`
}

type MarkExecutionFinishedRequest struct {
	TaskID         int64           `json:"task_id"`
	ExecutionID    int64           `json:"execution_id"`
	Status         ExecutionStatus `json:"status"`
	ExecutorRunID  string          `json:"executor_run_id,omitempty"`
	FinishedAt     time.Time       `json:"finished_at,omitempty"`
	DurationMillis int64           `json:"duration_millis,omitempty"`
	OutputSummary  string          `json:"output_summary,omitempty"`
	ResultPayload  json.RawMessage `json:"result_payload,omitempty"`
	ErrorMessage   string          `json:"error_message,omitempty"`
}

type ExecutionResult struct {
	Status         ExecutionStatus `json:"status,omitempty"`
	ExecutorRunID  string          `json:"executor_run_id,omitempty"`
	OutputSummary  string          `json:"output_summary,omitempty"`
	ResultPayload  json.RawMessage `json:"result_payload,omitempty"`
	ErrorMessage   string          `json:"error_message,omitempty"`
	DurationMillis int64           `json:"duration_millis,omitempty"`
}
