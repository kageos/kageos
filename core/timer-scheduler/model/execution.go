package model

import (
	"encoding/json"
	"time"
)

type TimerExecution struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	TaskID      int64  `json:"task_id" gorm:"not null;index"`
	ExecutorKey string `json:"executor_key" gorm:"size:128;not null;index"`
	Status      string `json:"status" gorm:"size:20;not null;index"`
	TriggerType string `json:"trigger_type" gorm:"size:20;not null;index;default:scheduled"`

	ExecutorRunID    string     `json:"executor_run_id" gorm:"size:255"`
	ScheduledAt      time.Time  `json:"scheduled_at" gorm:"not null;index"`
	StartedAt        *time.Time `json:"started_at"`
	FinishedAt       *time.Time `json:"finished_at"`
	WorkerID         string     `json:"worker_id" gorm:"size:128;index"`
	LeaseUntil       *time.Time `json:"lease_until" gorm:"index"`
	HeartbeatAt      *time.Time `json:"heartbeat_at" gorm:"index"`
	HeartbeatMisses  int        `json:"heartbeat_misses" gorm:"default:0"`
	Attempt          int        `json:"attempt" gorm:"default:0"`
	LastDispatchedAt *time.Time `json:"last_dispatched_at" gorm:"index"`
	DurationMillis   int64      `json:"duration_millis" gorm:"default:0"`

	OutputSummary string          `json:"output_summary" gorm:"type:text"`
	ResultPayload json.RawMessage `json:"result_payload" gorm:"type:json"`
	ErrorMessage  string          `json:"error_message" gorm:"type:text"`
	TraceID       string          `json:"trace_id" gorm:"size:100;index"`

	SourceType      string `json:"source_type" gorm:"size:64;index"`
	SourceRef       string `json:"source_ref" gorm:"size:512;index"`
	ResourceScope   string `json:"resource_scope" gorm:"size:128;index"`
	ResourceKey     string `json:"resource_key" gorm:"size:512;index"`
	RequestUser     string `json:"request_user" gorm:"size:255;index"`
	RequestUserDept string `json:"request_user_dept" gorm:"size:500"`
}

func (TimerExecution) TableName() string {
	return "timer_execution"
}
