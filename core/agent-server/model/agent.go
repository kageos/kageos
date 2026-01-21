package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"gorm.io/gorm"
)

// Agent 智能体模型
// 智能体分为两种类型：
// 1. 纯知识库类型（knowledge_only）：只需要用户调用然后查询对应知识库直接生成代码即可
// 2. 插件调用类型（plugin）：配置有消息主题，需要调用外部插件处理，处理完然后再调用知识库
type Agent struct {
	models.Base
	Name        string `gorm:"type:varchar(255);not null" json:"name"`
	AgentType   string `gorm:"type:varchar(32);not null;index" json:"agent_type"` // knowledge_only: 纯知识库类型, plugin: 插件调用类型
	ChatType    string `gorm:"type:varchar(32);not null;index" json:"chat_type"`  // function_gen 函数生成任务，请求和响应的参数基于不同的ChatType来区分不同的请求响应类型
	Enabled     bool   `gorm:"default:true;index" json:"enabled"`
	Author      string `gorm:"type:varchar(128)" json:"author"`
	Description string `gorm:"type:text" json:"description"`
	Timeout     int    `gorm:"default:30" json:"timeout"` // 超时时间（秒）
	
	// 插件函数路径（仅 plugin 类型需要）
	// 例如：/system/official/agent/plugin/excel_or_csv/table_parse
	PluginFunctionPath string `gorm:"type:varchar(512);index;comment:插件函数路径(full-code-path)" json:"plugin_function_path"`
	
	// 知识库关联（两种类型都需要）
	KnowledgeBaseID int64        `gorm:"type:bigint;not null;index;comment:知识库ID" json:"knowledge_base_id"`
	KnowledgeBase   KnowledgeBase `gorm:"foreignKey:KnowledgeBaseID;references:ID" json:"knowledge_base,omitempty"` // 预加载关联

	// LLM 配置关联（如果为空，则使用默认 LLM 配置）
	LLMConfigID int64    `gorm:"type:bigint;index;comment:LLM配置ID" json:"llm_config_id"`
	LLMConfig   LLMConfig `gorm:"foreignKey:LLMConfigID;references:ID" json:"llm_config,omitempty"` // 预加载关联

	// System Prompt 模板（支持 {knowledge} 变量，会被替换为知识库内容）
	// 如果为空，使用默认模板："你是一个专业的代码生成助手。以下是相关的知识库内容，请参考这些内容来生成代码：\n{knowledge}"
	SystemPromptTemplate string `gorm:"type:text;comment:System Prompt模板" json:"system_prompt_template"`

	// 元数据（JSON，允许为 NULL）
	Metadata *string `gorm:"type:json" json:"metadata"`

	// 权限控制
	Visibility int    `gorm:"type:tinyint;default:0;index;comment:可见性(0:公开,1:私有)" json:"visibility"` // 0: 公开, 1: 私有
	Admin      string `gorm:"type:varchar(512);not null;index;comment:管理员列表(逗号分隔)" json:"admin"`      // 管理员列表，逗号分隔，如："user1,user2,user3"

	// Logo（可选，如果为空则使用默认生成的 Logo）
	Logo string `gorm:"type:varchar(512);comment:智能体Logo URL" json:"logo"`

	// 开场白（可选，用于显示智能体的使用教程等）
	Greeting     string `gorm:"type:text;comment:开场白内容" json:"greeting"`                    // 开场白内容
	GreetingType string `gorm:"type:varchar(32);default:'text';comment:开场白格式类型" json:"greeting_type"` // 格式类型：text, md, html

	// 使用统计
	GenerationCount int64 `gorm:"type:bigint;default:0;index;comment:生成次数统计" json:"generation_count"` // 生成次数统计
}

// TableName 指定表名
func (Agent) TableName() string {
	return "agents"
}

// AfterCreate GORM 钩子：创建后设置默认管理员
func (a *Agent) AfterCreate(tx *gorm.DB) error {
	// 设置默认管理员（如果为空，设置为创建用户）
	if a.Admin == "" {
		a.Admin = a.CreatedBy
		if err := tx.Model(a).Update("admin", a.Admin).Error; err != nil {
			return err
		}
	}
	return nil
}

