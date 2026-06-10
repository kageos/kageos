package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type TimerTask struct {
	ID        int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	Title          string          `json:"title" gorm:"size:255"`
	Description    string          `json:"description" gorm:"type:text"`
	Category       string          `json:"category" gorm:"size:100;index"`
	TagsJSON       json.RawMessage `json:"tags_json" gorm:"column:tags;type:json"`
	IdempotencyKey *string         `json:"idempotency_key" gorm:"size:160;uniqueIndex"`

	ExecutorKey     string          `json:"executor_key" gorm:"size:128;not null;index"`
	ExecutorPayload json.RawMessage `json:"executor_payload" gorm:"type:json"`
	MetadataJSON    json.RawMessage `json:"metadata_json" gorm:"column:metadata;type:json"`

	ScheduleType    string     `json:"schedule_type" gorm:"size:20;not null;index"`
	RunAt           *time.Time `json:"run_at"`
	CronExpr        string     `json:"cron_expr" gorm:"size:100"`
	IntervalSeconds int64      `json:"interval_seconds"`
	Timezone        string     `json:"timezone" gorm:"size:64"`
	MaxRuns         int        `json:"max_runs" gorm:"default:0"`
	NextRunAt       *time.Time `json:"next_run_at" gorm:"index"`
	RunCount        int        `json:"run_count" gorm:"default:0"`

	Status              string     `json:"status" gorm:"size:20;not null;index"`
	InflightExecutionID int64      `json:"inflight_execution_id" gorm:"index;default:0"`
	LastExecutionID     int64      `json:"last_execution_id" gorm:"default:0"`
	LastErrorMessage    string     `json:"last_error_message" gorm:"type:text"`
	LeaseOwner          string     `json:"lease_owner" gorm:"size:128;index"`
	LeaseUntil          *time.Time `json:"lease_until" gorm:"index"`

	SourceType      string `json:"source_type" gorm:"size:64;index"`
	SourceRef       string `json:"source_ref" gorm:"size:512;index"`
	ResourceScope   string `json:"resource_scope" gorm:"size:128;index"`
	ResourceKey     string `json:"resource_key" gorm:"size:512;index"`
	RequestUser     string `json:"request_user" gorm:"size:255;index"`
	RequestUserDept string `json:"request_user_dept" gorm:"size:500"`
	CreatedBy       string `json:"created_by" gorm:"size:255;index"`
}

func (TimerTask) TableName() string {
	return "timer_task"
}
