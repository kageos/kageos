package model

import (
	"encoding/json"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

const (
	ScheduledAgentTaskStatusPending   = "pending"
	ScheduledAgentTaskStatusPaused    = "paused"
	ScheduledAgentTaskStatusDone      = "done"
	ScheduledAgentTaskStatusFailed    = "failed"
	ScheduledAgentTaskStatusCancelled = "cancelled"
)

// ScheduledAgentTask 定时 Agent 会话任务。
type ScheduledAgentTask struct {
	models.Base

	Name         string `json:"name" gorm:"type:varchar(255);not null;comment:任务名称"`
	FullCodePath string `json:"full_code_path" gorm:"type:varchar(512);not null;index;comment:工作台目录完整路径"`
	Goal         string `json:"goal" gorm:"type:longtext;comment:每次执行目标"`
	ModeCode     string `json:"mode_code" gorm:"type:varchar(32);not null;default:'dev';index;comment:工作台模式代码"`
	Files        string `json:"files" gorm:"type:longtext;comment:文件引用，逗号分隔"`
	LLMConfigID  int64  `json:"llm_config_id" gorm:"default:0;comment:LLM 配置 ID，0 使用默认"`

	ContextPolicy  json.RawMessage `json:"context_policy" gorm:"type:json;comment:上下文策略，MVP 不启用"`
	ToolPolicy     json.RawMessage `json:"tool_policy" gorm:"type:json;comment:工具策略，MVP 不启用；NULL 表示不限制"`
	ApprovalPolicy json.RawMessage `json:"approval_policy" gorm:"type:json;comment:审批策略，MVP 不启用"`
	BudgetPolicy   json.RawMessage `json:"budget_policy" gorm:"type:json;comment:预算策略，MVP 仅使用 max_duration_seconds"`

	ScheduleType    string     `json:"schedule_type" gorm:"type:varchar(20);not null;index;comment:atime/cron/every"`
	RunAt           time.Time  `json:"run_at" gorm:"comment:atime 的执行时间；cron/every 的创建生效时间"`
	NextRunAt       *time.Time `json:"next_run_at" gorm:"index;comment:下次执行时间"`
	CronExpr        string     `json:"cron_expr" gorm:"type:varchar(100);comment:cron 表达式"`
	IntervalSeconds int64      `json:"interval_seconds" gorm:"comment:every 时间间隔秒"`
	MaxRuns         int        `json:"max_runs" gorm:"default:0;comment:最多自动执行次数，0 不限制"`
	Timezone        string     `json:"timezone" gorm:"type:varchar(64);comment:时区"`

	Status            string     `json:"status" gorm:"type:varchar(20);not null;index;comment:pending/paused/done/failed/cancelled"`
	RunCount          int        `json:"run_count" gorm:"default:0;comment:已自动执行次数"`
	LastSessionID     string     `json:"last_session_id" gorm:"type:varchar(64);index;comment:最近一次工作台会话 ID"`
	LastExecutionID   int64      `json:"last_execution_id" gorm:"index;comment:最近一次执行记录 ID"`
	LastErrorMessage  string     `json:"last_error_message" gorm:"type:text;comment:最近一次失败信息"`
	RequestUser       string     `json:"request_user" gorm:"type:varchar(255);comment:以谁的身份执行"`
	RequestUserDept   string     `json:"request_user_dept" gorm:"type:varchar(500);comment:请求用户部门路径"`
	NotifyUsers       string     `json:"notify_users" gorm:"type:text;comment:通知用户，逗号分隔"`
	NotifyDepartments string     `json:"notify_departments" gorm:"type:text;comment:通知部门 full_code_path，逗号分隔"`
	NotifyOn          string     `json:"notify_on" gorm:"type:varchar(20);default:none;comment:通知触发条件:none/all/success/failed"`
	SourceType        string     `json:"source_type" gorm:"type:varchar(64);comment:来源类型"`
	SourceRef         string     `json:"source_ref" gorm:"type:varchar(255);index;comment:来源引用，供后续工具白名单使用"`
	LeaseOwner        string     `json:"lease_owner" gorm:"type:varchar(128);index;comment:执行租约持有者"`
	LeaseUntil        *time.Time `json:"lease_until" gorm:"index;comment:执行租约到期时间"`
}

func (ScheduledAgentTask) TableName() string {
	return "scheduled_agent_task"
}
