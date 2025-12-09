package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// CodeGenConfig 代码生成配置模型
type CodeGenConfig struct {
	models.Base
	Name        string `gorm:"type:varchar(255);not null" json:"name"`
	TemplatePath string `gorm:"type:varchar(512)" json:"template_path"`
	OutputPath   string `gorm:"type:varchar(512)" json:"output_path"`
	SDKVersion  string `gorm:"type:varchar(32);not null" json:"sdk_version"`
	ExtraConfig string `gorm:"type:json" json:"extra_config"` // JSON 额外配置
	IsDefault   bool   `gorm:"default:false;index" json:"is_default"`
}

// TableName 指定表名
func (CodeGenConfig) TableName() string {
	return "code_gen_configs"
}

