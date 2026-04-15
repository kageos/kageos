package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// ScheduledTask 定时任务
// schedule_type: atime=指定时间执行一次, cron=cron表达式重复, every=每N秒执行一次
type ScheduledTask struct {
	ID        int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	User            string          `json:"user" gorm:"size:255;not null;index;comment:归属用户"`
	App             string          `json:"app" gorm:"size:255;not null;index;comment:归属应用"`
	Name            string          `json:"name" gorm:"size:255;comment:任务名称（用户可编辑）"`
	FullCodePath    string          `json:"full_code_path" gorm:"size:500;not null;comment:要执行的表单路径"`
	Action          string          `json:"action" gorm:"size:32;default:execute;comment:执行动作：execute/table_create/table_update/table_delete"`
	Method          string          `json:"method" gorm:"size:10;default:POST;comment:GET/POST"`
	Payload         json.RawMessage `json:"payload" gorm:"type:json;comment:请求体"`
	RequestUser     string          `json:"request_user" gorm:"size:255;comment:以谁的身份执行"`
	RequestUserDept string          `json:"request_user_dept" gorm:"size:500;comment:请求用户部门路径（透传用）"`
	CreatedBy       string          `json:"created_by" gorm:"size:255;index;comment:创建人（用于列表筛选）"`

	ScheduleType    string     `json:"schedule_type" gorm:"size:20;not null;index;comment:atime/cron/every"`
	RunAt           time.Time  `json:"run_at" gorm:"comment:atime 的执行时间；cron/every 的创建生效时间"`
	NextRunAt       *time.Time `json:"next_run_at" gorm:"index;comment:下次执行时间"`
	LeaseOwner      string     `json:"lease_owner" gorm:"size:128;index;comment:执行租约持有者"`
	LeaseUntil      *time.Time `json:"lease_until" gorm:"index;comment:执行租约到期时间"`
	CronExpr        string     `json:"cron_expr" gorm:"size:100;comment:cron表达式"`
	IntervalSeconds int64      `json:"interval_seconds" gorm:"comment:every 时间间隔秒"`
	MaxRuns         int        `json:"max_runs" gorm:"default:0;comment:every 最多执行次数,0不限制"`
	RunCount        int        `json:"run_count" gorm:"default:0;comment:已执行次数"`

	Status       string `json:"status" gorm:"size:20;not null;index;comment:pending/done/failed/cancelled"`
	Timezone     string `json:"timezone" gorm:"size:64;comment:时区"`
	ErrorMessage string `json:"error_message" gorm:"type:text;comment:最近一次失败信息"`
}

func (ScheduledTask) TableName() string {
	return "scheduled_task"
}
