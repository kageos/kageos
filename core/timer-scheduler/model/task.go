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

	Title       string          `json:"title" gorm:"size:255;not null;comment:任务标题"`
	Description string          `json:"description" gorm:"type:text;comment:任务描述"`
	Category    string          `json:"category" gorm:"size:100;index;comment:任务分类"`
	TagsJSON    json.RawMessage `json:"tags_json" gorm:"column:tags;type:json;comment:标签JSON数组"`

	ExecutorKey     string          `json:"executor_key" gorm:"size:128;not null;index;comment:执行器路由key"`
	ExecutorPayload json.RawMessage `json:"executor_payload" gorm:"type:json;comment:执行器私有输入"`
	MetadataJSON    json.RawMessage `json:"metadata_json" gorm:"column:metadata;type:json;comment:调度侧可读描述元数据"`

	ScheduleType    string     `json:"schedule_type" gorm:"size:20;not null;index;comment:atime/cron/every"`
	RunAt           *time.Time `json:"run_at" gorm:"comment:atime执行时间"`
	CronExpr        string     `json:"cron_expr" gorm:"size:100;comment:cron表达式"`
	IntervalSeconds int64      `json:"interval_seconds" gorm:"comment:every时间间隔秒"`
	MaxRuns         int        `json:"max_runs" gorm:"default:0;comment:最多执行次数,0不限制"`
	Timezone        string     `json:"timezone" gorm:"size:64;comment:时区"`
	NextRunAt       *time.Time `json:"next_run_at" gorm:"index;comment:下次执行时间"`
	RunCount        int        `json:"run_count" gorm:"default:0;comment:已执行次数"`

	Status              string     `json:"status" gorm:"size:20;not null;index;comment:pending/paused/done/failed/cancelled"`
	InflightExecutionID int64      `json:"inflight_execution_id" gorm:"index;default:0;comment:当前执行中的execution id"`
	LastExecutionID     int64      `json:"last_execution_id" gorm:"default:0;comment:最近一次execution id"`
	LastErrorMessage    string     `json:"last_error_message" gorm:"type:text;comment:最近一次错误"`
	LeaseOwner          string     `json:"lease_owner" gorm:"size:128;index;comment:调度派发租约持有者"`
	LeaseUntil          *time.Time `json:"lease_until" gorm:"index;comment:调度派发租约到期时间"`

	SourceType        string `json:"source_type" gorm:"size:64;index;comment:来源类型"`
	SourceRef         string `json:"source_ref" gorm:"size:512;index;comment:来源引用"`
	RequestUser       string `json:"request_user" gorm:"size:255;comment:执行身份"`
	RequestUserDept   string `json:"request_user_dept" gorm:"size:500;comment:执行身份部门"`
	NotifyUsers       string `json:"notify_users" gorm:"type:text;comment:通知用户，逗号分隔"`
	NotifyDepartments string `json:"notify_departments" gorm:"type:text;comment:通知部门，逗号分隔"`
	NotifyOn          string `json:"notify_on" gorm:"size:20;default:none;comment:none/all/success/failed"`
	CreatedBy         string `json:"created_by" gorm:"size:255;index;comment:创建人"`
}

func (TimerTask) TableName() string {
	return "timer_task"
}
