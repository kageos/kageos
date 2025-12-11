package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"gorm.io/gorm"
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
	UseThinking bool   `gorm:"default:false;comment:是否使用思考模式" json:"use_thinking"` // 是否使用思考模式（GLM特有功能）
	IsDefault  bool   `gorm:"default:false;index" json:"is_default"`

	// 权限控制
	Visibility int    `gorm:"type:tinyint;default:0;index;comment:可见性(0:公开,1:私有)" json:"visibility"` // 0: 公开, 1: 私有
	Admin      string `gorm:"type:varchar(512);not null;index;comment:管理员列表(逗号分隔)" json:"admin"`      // 管理员列表，逗号分隔，如："user1,user2,user3"
}

// TableName 指定表名
func (LLMConfig) TableName() string {
	return "llm_configs"
}

// AfterCreate GORM 钩子：设置默认管理员
func (llm *LLMConfig) AfterCreate(tx *gorm.DB) error {
	// 设置默认管理员（如果为空，设置为创建用户）
	if llm.Admin == "" {
		llm.Admin = llm.CreatedBy
		return tx.Model(llm).Update("admin", llm.Admin).Error
	}
	return nil
}

