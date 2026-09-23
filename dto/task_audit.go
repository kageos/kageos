package dto

import (
	"encoding/json"
	"time"
)

// Internal trusted-service subject. Consumers acknowledge only after persistence.
const TaskAuditSubject = "platform.timer.audit.record"

type TaskAuditEvent struct {
	EventID      string          `json:"event_id"`
	OccurredAt   time.Time       `json:"occurred_at"`
	Action       string          `json:"action"`
	TaskID       int64           `json:"task_id"`
	ResourcePath string          `json:"resource_path"`
	ResourceName string          `json:"resource_name"`
	Actor        string          `json:"actor"`
	Initiator    string          `json:"initiator,omitempty"`
	Source       string          `json:"source"`
	SourceType   string          `json:"source_type,omitempty"`
	SourceRef    string          `json:"source_ref,omitempty"`
	SessionID    string          `json:"session_id,omitempty"`
	ToolCallID   string          `json:"tool_call_id,omitempty"`
	TraceID      string          `json:"trace_id,omitempty"`
	Status       string          `json:"status"`
	Error        string          `json:"error,omitempty"`
	ExecutionID  int64           `json:"execution_id,omitempty"`
	Before       json.RawMessage `json:"before,omitempty"`
	After        json.RawMessage `json:"after,omitempty"`
}
