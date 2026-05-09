package model

import "github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"

// WorkspaceHandoffPacket records the structured artifact that moves one workspace role session to the next.
type WorkspaceHandoffPacket struct {
	models.Base
	SourceSessionID  string `gorm:"type:varchar(64);not null;index;comment:来源会话ID" json:"source_session_id"`
	TargetSessionID  string `gorm:"type:varchar(64);not null;uniqueIndex;comment:目标会话ID" json:"target_session_id"`
	FullCodePath     string `gorm:"type:varchar(512);index;comment:工作台目录完整路径" json:"full_code_path"`
	TargetRole       string `gorm:"type:varchar(64);not null;index;comment:目标角色ID" json:"target_role"`
	ArtifactKind     string `gorm:"type:varchar(64);not null;index;comment:交接产物类型" json:"artifact_kind"`
	ArtifactJSON     string `gorm:"type:longtext;comment:结构化交接产物JSON" json:"artifact_json"`
	Remark           string `gorm:"type:longtext;comment:用户确认时的补充备注" json:"remark"`
	ContextPolicy    string `gorm:"type:varchar(64);not null;comment:目标会话上下文策略" json:"context_policy"`
	InitialMessageID int64  `gorm:"type:bigint;index;comment:目标会话首条artifact消息ID" json:"initial_message_id"`
	User             string `gorm:"type:varchar(128);not null;index;comment:创建用户" json:"user"`
}

func (WorkspaceHandoffPacket) TableName() string {
	return "workspace_handoff_packets"
}
