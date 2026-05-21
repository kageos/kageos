package model

import (
	"github.com/kageos/kageos/pkg/gormx/models"
)

// AgentChatSession 工作台聊天会话模型。
// 类型名与表名保留历史命名，以兼容已有数据库。
type AgentChatSession struct {
	models.Base
	TreeID            int64  `gorm:"type:bigint;not null;index;comment:服务目录ID" json:"tree_id"`
	FullCodePath      string `gorm:"type:varchar(512);index;comment:服务目录完整路径（workspace 用，有语意）" json:"full_code_path"`
	Source            string `gorm:"type:varchar(32);index;comment:来源(workspace=工作台,空值为历史数据)" json:"source"`
	SessionID         string `gorm:"type:varchar(64);not null;uniqueIndex;comment:会话ID（UUID）" json:"session_id"`
	Title             string `gorm:"type:varchar(255);comment:会话标题" json:"title"`
	ModeCode          string `gorm:"type:varchar(32);not null;default:'dev';index;comment:工作台模式代码" json:"mode_code"`
	Status            string `gorm:"type:varchar(32);not null;default:'active';index;comment:会话状态(active/generating/output/pending_confirmation/pending_test/done)" json:"status"`
	RoleID            string `gorm:"type:varchar(64);index;comment:当前工作台角色ID，如 product_manager/app_developer" json:"role_id"`
	RoleDisplayName   string `gorm:"type:varchar(64);comment:当前工作台角色展示名称，如 产品经理/应用开发工程师" json:"role_display_name"`
	ParentSessionID   string `gorm:"type:varchar(64);index;comment:阶段交接来源会话ID" json:"parent_session_id"`
	HandoffKind       string `gorm:"type:varchar(64);index;comment:阶段交接产物类型" json:"handoff_kind"`
	HandoffTargetRole string `gorm:"type:varchar(64);comment:阶段交接目标身份" json:"handoff_target_role"`
	ContextPolicy     string `gorm:"type:varchar(64);not null;default:'full';index;comment:模型上下文策略(full/artifact_only/display_only)" json:"context_policy"`
	ArchivedForModel  bool   `gorm:"not null;default:false;index;comment:是否已归档且不再进入模型上下文" json:"archived_for_model"`
	ArchiveReason     string `gorm:"type:varchar(255);comment:会话归档原因" json:"archive_reason"`
	User              string `gorm:"type:varchar(128);not null;index;comment:创建用户" json:"user"`
}

// 会话状态常量
const (
	ChatSessionStatusActive              = "active"               // 活跃状态，可以继续输入
	ChatSessionStatusGenerating          = "generating"           // 生成中（后台执行），前端可轮询消息
	ChatSessionStatusOutput              = "output"               // 已生成产物/新文件，可以继续输入
	ChatSessionStatusDone                = "done"                 // 已完成，会话结束，不能再输入
	ChatSessionStatusCancelled           = "cancelled"            // 已取消（用户手动停止）
	ChatSessionStatusPendingConfirmation = "pending_confirmation" // 阶段产物等待用户确认（如 PRD）
	ChatSessionStatusPendingTest         = "pending_test"         // 构建产物等待用户确认是否进入测试
)

// TableName 指定表名
func (AgentChatSession) TableName() string {
	return "agent_chat_sessions"
}
