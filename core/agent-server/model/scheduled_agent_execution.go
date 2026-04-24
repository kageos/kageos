package model

import (
	"encoding/json"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

const (
	ScheduledAgentExecutionStatusPending   = "pending"
	ScheduledAgentExecutionStatusRunning   = "running"
	ScheduledAgentExecutionStatusSuccess   = "success"
	ScheduledAgentExecutionStatusFailed    = "failed"
	ScheduledAgentExecutionStatusTimeout   = "timeout"
	ScheduledAgentExecutionStatusCancelled = "cancelled"
)

// ScheduledAgentExecution 定时 Agent 会话每次执行记录。
type ScheduledAgentExecution struct {
	models.Base

	TaskID         int64           `json:"task_id" gorm:"not null;index;comment:任务 ID"`
	SessionID      string          `json:"session_id" gorm:"type:varchar(64);index;comment:工作台会话 ID"`
	ScheduledAt    time.Time       `json:"scheduled_at" gorm:"not null;index;comment:计划执行时间"`
	StartedAt      *time.Time      `json:"started_at" gorm:"index;comment:开始时间"`
	FinishedAt     *time.Time      `json:"finished_at" gorm:"index;comment:结束时间"`
	Status         string          `json:"status" gorm:"type:varchar(20);not null;index;comment:pending/running/success/failed/timeout/cancelled"`
	DurationMillis int64           `json:"duration_millis" gorm:"not null;default:0;comment:执行耗时毫秒"`
	InputGoal      string          `json:"input_goal" gorm:"type:longtext;comment:当次目标"`
	OutputSummary  string          `json:"output_summary" gorm:"type:longtext;comment:输出摘要"`
	ToolCallCount  int             `json:"tool_call_count" gorm:"default:0;comment:工具调用次数"`
	TokenUsage     json.RawMessage `json:"token_usage" gorm:"type:json;comment:token 使用统计，MVP 可为空"`
	ErrorMessage   string          `json:"error_message" gorm:"type:text;comment:失败信息"`
	TraceID        string          `json:"trace_id" gorm:"type:varchar(100);index;comment:追踪 ID"`
	SourceType     string          `json:"source_type" gorm:"type:varchar(64);comment:来源类型"`
	SourceRef      string          `json:"source_ref" gorm:"type:varchar(255);index;comment:来源引用"`
}

func (ScheduledAgentExecution) TableName() string {
	return "scheduled_agent_execution"
}
