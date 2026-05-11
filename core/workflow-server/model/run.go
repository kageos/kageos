package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type WorkflowRun struct {
	ID        int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	WorkflowID int64  `json:"workflow_id" gorm:"not null;index;comment:工作流ID"`
	VersionID  int64  `json:"version_id" gorm:"not null;index;comment:定义版本ID"`
	Status     string `json:"status" gorm:"size:20;not null;index;comment:pending/running/waiting/success/failed/cancelled/timeout"`

	InputJSON    json.RawMessage `json:"input_json" gorm:"column:input;type:json;comment:本次输入"`
	OutputJSON   json.RawMessage `json:"output_json" gorm:"column:output;type:json;comment:最终输出"`
	ErrorMessage string          `json:"error_message" gorm:"type:text;comment:失败信息"`

	RequestUser     string `json:"request_user" gorm:"size:255;index;comment:触发用户"`
	RequestUserDept string `json:"request_user_dept" gorm:"size:500;comment:触发用户部门"`
	TraceID         string `json:"trace_id" gorm:"size:128;index;comment:链路追踪ID"`

	StartedAt      *time.Time `json:"started_at" gorm:"index;comment:开始时间"`
	FinishedAt     *time.Time `json:"finished_at" gorm:"index;comment:结束时间"`
	DurationMillis int64      `json:"duration_millis" gorm:"default:0;comment:耗时毫秒"`
}

func (WorkflowRun) TableName() string {
	return "workflow_run"
}

type WorkflowStepRun struct {
	ID        int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	RunID    int64  `json:"run_id" gorm:"not null;index;comment:工作流运行ID"`
	StepID   string `json:"step_id" gorm:"size:128;not null;index;comment:节点ID"`
	StepName string `json:"step_name" gorm:"size:255;comment:节点名称"`
	NodeType string `json:"node_type" gorm:"size:128;not null;index;comment:节点类型"`
	NodeRef  string `json:"node_ref" gorm:"size:500;comment:被调用资源"`
	Status   string `json:"status" gorm:"size:20;not null;index;comment:pending/running/waiting/success/failed/skipped/cancelled"`

	InputJSON    json.RawMessage `json:"input_json" gorm:"column:input;type:json;comment:节点输入"`
	OutputJSON   json.RawMessage `json:"output_json" gorm:"column:output;type:json;comment:节点输出"`
	ErrorMessage string          `json:"error_message" gorm:"type:text;comment:失败信息"`
	TraceID      string          `json:"trace_id" gorm:"size:128;index;comment:链路追踪ID"`
	Attempt      int             `json:"attempt" gorm:"default:1;comment:第几次尝试"`

	StartedAt      *time.Time `json:"started_at" gorm:"index;comment:开始时间"`
	FinishedAt     *time.Time `json:"finished_at" gorm:"index;comment:结束时间"`
	DurationMillis int64      `json:"duration_millis" gorm:"default:0;comment:耗时毫秒"`
}

func (WorkflowStepRun) TableName() string {
	return "workflow_step_run"
}

type WorkflowRunEvent struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`

	RunID       int64           `json:"run_id" gorm:"not null;index;comment:工作流运行ID"`
	StepRunID   int64           `json:"step_run_id" gorm:"index;default:0;comment:步骤运行ID"`
	EventType   string          `json:"event_type" gorm:"size:128;not null;index;comment:事件类型"`
	Message     string          `json:"message" gorm:"type:text;comment:可读消息"`
	PayloadJSON json.RawMessage `json:"payload_json" gorm:"column:payload;type:json;comment:结构化上下文"`
}

func (WorkflowRunEvent) TableName() string {
	return "workflow_run_event"
}
