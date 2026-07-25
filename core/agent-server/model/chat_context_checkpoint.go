package model

import "github.com/kageos/kageos/pkg/gormx/models"

// AgentChatContextCheckpoint is a reversible model-context compaction record.
// Raw AgentChatMessage rows remain the source of truth; this table only stores
// a compact semantic view plus the exact covered message range.
type AgentChatContextCheckpoint struct {
	models.Base
	SessionID            string  `gorm:"type:varchar(64);not null;index;uniqueIndex:uk_chat_checkpoint_range" json:"session_id"`
	CoveredFromMessageID int64   `gorm:"type:bigint;not null;uniqueIndex:uk_chat_checkpoint_range" json:"covered_from_message_id"`
	CoveredToMessageID   int64   `gorm:"type:bigint;not null;index;uniqueIndex:uk_chat_checkpoint_range" json:"covered_to_message_id"`
	Summary              string  `gorm:"type:longtext;not null" json:"summary"`
	Source               string  `gorm:"type:varchar(32);not null;default:'llm'" json:"source"`
	LLMConfigID          int64   `gorm:"type:bigint;index" json:"llm_config_id"`
	LLMModel             string  `gorm:"type:varchar(128)" json:"llm_model"`
	Usage                *string `gorm:"type:json" json:"usage"`
	User                 string  `gorm:"type:varchar(128);not null;index" json:"user"`
}

func (AgentChatContextCheckpoint) TableName() string {
	return "agent_chat_context_checkpoints"
}
