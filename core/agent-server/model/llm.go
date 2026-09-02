package model

import (
	"strings"

	"github.com/kageos/kageos/pkg/gormx/models"
	"gorm.io/gorm"
)

const (
	LLMProviderOpenAI    = "openai"
	LLMProviderAnthropic = "anthropic"

	LLMProtocolOpenAIChatCompletions = "openai_chat_completions"
	LLMProtocolOpenAIResponses       = "openai_responses"
	LLMProtocolAnthropicMessages     = "anthropic_messages"
)

// LLMConfig LLM 配置模型
type LLMConfig struct {
	models.Base
	Code                         string  `gorm:"type:varchar(64);index" json:"code"` // 稳定配置编码，用于部署 seed 幂等同步
	Name                         string  `gorm:"type:varchar(255);not null" json:"name"`
	Provider                     string  `gorm:"type:varchar(32);not null;default:'openai';index" json:"provider"`
	Protocol                     string  `gorm:"type:varchar(64);not null;default:'openai_chat_completions';index" json:"protocol"`
	Model                        string  `gorm:"type:varchar(128);not null" json:"model"` // gpt-4.1, claude-sonnet-4-5, etc.
	APIKey                       string  `gorm:"type:varchar(512)" json:"api_key"`        // 加密存储
	APIBase                      string  `gorm:"type:varchar(512)" json:"api_base"`       // 上游 API Base URL；空则使用协议默认值
	EndpointPath                 string  `gorm:"type:varchar(255)" json:"endpoint_path"`  // 可选 endpoint path override，例如 /v1/responses
	APIVersion                   string  `gorm:"type:varchar(64)" json:"api_version"`     // 可选上游 API version，例如 Anthropic 2023-06-01
	AuthScheme                   string  `gorm:"type:varchar(32)" json:"auth_scheme"`     // bearer、x-api-key、none；空则使用协议默认
	Headers                      *string `gorm:"type:json" json:"headers"`                // JSON 额外请求头
	Timeout                      int     `gorm:"default:300" json:"timeout"`              // 超时时间（秒）
	MaxTokens                    int     `gorm:"default:0" json:"max_tokens"`             // 手动最大输出 token 数；0 表示自动
	DetectedMaxOutputTokens      int     `gorm:"default:0" json:"detected_max_output_tokens"`
	DetectedMaxOutputTokenSource string  `gorm:"type:varchar(64)" json:"detected_max_output_token_source"`
	ContextWindow                int     `gorm:"default:0" json:"context_window"`          // 用户手动填写的上下文容量；0 表示自动
	DetectedContextWindow        int     `gorm:"default:0" json:"detected_context_window"` // 探测到的上下文容量
	DetectedContextWindowSource  string  `gorm:"type:varchar(64)" json:"detected_context_window_source"`
	ExtraConfig                  *string `gorm:"type:json" json:"extra_config"` // JSON 额外配置
	Capabilities                 *string `gorm:"type:json" json:"capabilities"` // 探测/人工声明能力，如 stream/tools/reasoning
	IsDefault                    bool    `gorm:"default:false;index" json:"is_default"`

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
