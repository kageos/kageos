package model

import (
	"encoding/json"
	"time"
)

type TimerExecution struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	TaskID      int64  `json:"task_id" gorm:"not null;index;comment:任务ID"`
	ExecutorKey string `json:"executor_key" gorm:"size:128;index;comment:执行器路由key"`
	Status      string `json:"status" gorm:"size:20;not null;index;comment:queued/running/success/failed/timeout/cancelled"`

	ExecutorRunID  string     `json:"executor_run_id" gorm:"size:255;comment:业务执行实例ID"`
	ScheduledAt    time.Time  `json:"scheduled_at" gorm:"not null;index;comment:计划触发时间"`
	StartedAt      *time.Time `json:"started_at" gorm:"comment:开始执行时间"`
	FinishedAt     *time.Time `json:"finished_at" gorm:"comment:执行结束时间"`
	WorkerID       string     `json:"worker_id" gorm:"size:128;index;comment:执行worker id"`
	LeaseUntil     *time.Time `json:"lease_until" gorm:"index;comment:执行租约到期时间"`
	HeartbeatAt    *time.Time `json:"heartbeat_at" gorm:"index;comment:worker心跳时间"`
	Attempt        int        `json:"attempt" gorm:"default:0;comment:执行尝试次数"`
	DurationMillis int64      `json:"duration_millis" gorm:"default:0;comment:执行耗时毫秒"`

	OutputSummary string          `json:"output_summary" gorm:"type:text;comment:执行摘要"`
	ResultPayload json.RawMessage `json:"result_payload" gorm:"type:json;comment:执行器私有输出"`
	ErrorMessage  string          `json:"error_message" gorm:"type:text;comment:错误信息"`
	TraceID       string          `json:"trace_id" gorm:"size:100;comment:追踪ID"`
	SourceType    string          `json:"source_type" gorm:"size:64;index;comment:来源类型"`
	SourceRef     string          `json:"source_ref" gorm:"size:512;index;comment:来源引用"`
}

func (TimerExecution) TableName() string {
	return "timer_execution"
}
