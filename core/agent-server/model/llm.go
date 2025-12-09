package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// LLMConfig LLM 配置模型
type LLMConfig struct {
	models.Base
	Name       string `gorm:"type:varchar(255);not null" json:"name"`
	Provider   string `gorm:"type:varchar(32);not null;index" json:"provider"` // openai, claude, local, etc.
	Model      string `gorm:"type:varchar(128);not null" json:"model"`         // gpt-4, claude-3, etc.
	APIKey     string `gorm:"type:varchar(512)" json:"api_key"`                // 加密存储
	APIBase    string `gorm:"type:varchar(512)" json:"api_base"`
	Timeout    int    `gorm:"default:120" json:"timeout"`     // 超时时间（秒）
	MaxTokens  int    `gorm:"default:4000" json:"max_tokens"` // 最大 token 数
	ExtraConfig *string `gorm:"type:json" json:"extra_config"`  // JSON 额外配置
	IsDefault  bool   `gorm:"default:false;index" json:"is_default"`
}

// TableName 指定表名
func (LLMConfig) TableName() string {
	return "llm_configs"
}

