package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// Docs 文档模型
// 文档内容存储在数据库中，通过 ServiceTree 的 RefID 关联
type Docs struct {
	models.Base
	Name         string `json:"name" gorm:"type:varchar(255);not null;comment:文档名称"`                          // 文档名称（显示用）
	Content      string `json:"content" gorm:"type:longtext;comment:文档内容（Markdown格式）"`                        // 文档内容（Markdown）
	Format       string `json:"format" gorm:"type:varchar(32);default:'markdown';comment:文档格式"`               // 文档格式：markdown, text, html
	AppID        int64  `json:"app_id" gorm:"type:bigint;not null;index;comment:所属应用ID"`                      // 所属应用ID
	TreeID       int64  `json:"tree_id" gorm:"type:bigint;not null;index;comment:关联的ServiceTree节点ID"`         // 关联的ServiceTree节点ID
	FullCodePath string `json:"full_code_path" gorm:"type:varchar(500);index;comment:完整路径（与ServiceTree保持一致）"` // 完整路径（如 /system/prompt/sdk/widgets/input.md）
	// 可选字段
	Summary  string `json:"summary" gorm:"type:text;comment:文档摘要"`          // 文档摘要
	Category string `json:"category" gorm:"type:varchar(128);comment:文档分类"` // 文档分类
}

// TableName 指定表名
func (Docs) TableName() string {
	return "docs"
}
