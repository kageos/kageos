package model

import (
	"strings"

	"github.com/kageos/kageos/pkg/gormx/models"
	"gorm.io/gorm"
)

// LLMProviderOpenAI is the only supported LLM protocol marker.
//
// The provider column is kept for persisted metadata and older rows, but the
// runtime does not branch on multiple providers.
const LLMProviderOpenAI = "openai"

// LLMConfig LLM 配置模型
type LLMConfig struct {
	models.Base
	Code        string  `gorm:"type:varchar(64);index" json:"code"` // 稳定配置编码，用于部署 seed 幂等同步
	Name        string  `gorm:"type:varchar(255);not null" json:"name"`
	Provider    string  `gorm:"type:varchar(32);not null;index" json:"provider"` // 固定为 openai，仅作持久化元数据
	Model       string  `gorm:"type:varchar(128);not null" json:"model"`         // gpt-4.1, gpt-4o-mini, etc.
	APIKey      string  `gorm:"type:varchar(512)" json:"api_key"`                // 加密存储
	APIBase     string  `gorm:"type:varchar(512)" json:"api_base"`               // OpenAI API base URL override，空则使用 SDK 默认
	Timeout     int     `gorm:"default:300" json:"timeout"`                      // 超时时间（秒）
	MaxTokens   int     `gorm:"default:8196" json:"max_tokens"`                  // 最大 token 数
	ExtraConfig *string `gorm:"type:json" json:"extra_config"`                   // JSON 额外配置
	IsDefault   bool    `gorm:"default:false;index" json:"is_default"`

	// 权限控制
	Visibility int    `gorm:"type:tinyint;default:0;index;comment:可见性(0:公开,1:私有)" json:"visibility"` // 0: 公开, 1: 私有
	Admin      string `gorm:"type:varchar(512);not null;index;comment:管理员列表(逗号分隔)" json:"admin"`     // 管理员列表，逗号分隔，如："user1,user2,user3"
}

// TableName 指定表名
func (LLMConfig) TableName() string {
	return "llm_configs"
}

// IsAdminUser reports whether username can manage this LLM config.
func (llm *LLMConfig) IsAdminUser(username string) bool {
	if llm == nil {
		return false
	}

	username = strings.TrimSpace(username)
	if username == "" {
		return false
	}

	if strings.TrimSpace(llm.CreatedBy) == username {
		return true
	}

	for _, admin := range strings.Split(llm.Admin, ",") {
		if strings.TrimSpace(admin) == username {
			return true
		}
	}

	return false
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
