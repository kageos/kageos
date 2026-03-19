package model

import (
	"encoding/json"
	"time"
)

// ScheduledTaskExecution 定时任务执行记录（每次执行一条）
type ScheduledTaskExecution struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	TaskID      int64     `json:"task_id" gorm:"not null;index;comment:任务ID"`
	ExecutedAt time.Time  `json:"executed_at" gorm:"not null;index;comment:实际执行时间"`
	Status          string          `json:"status" gorm:"size:20;not null;comment:success/failed"`
	RequestPayload  json.RawMessage `json:"request_payload" gorm:"type:json;comment:当次请求参数"`
	ResponsePayload json.RawMessage `json:"response_payload" gorm:"type:json;comment:当次响应参数"`
	ErrorMessage    string          `json:"error_message" gorm:"type:text;comment:失败信息"`
	TraceID     string   `json:"trace_id" gorm:"size:100;comment:追踪ID"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (ScheduledTaskExecution) TableName() string {
	return "scheduled_task_execution"
}
