package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"gorm.io/gorm"
)

// Plugin 插件模型
// 插件独立管理，可以被多个智能体复用
type Plugin struct {
	models.Base
	Name        string `gorm:"type:varchar(255);not null;comment:插件名称" json:"name"`
	Code        string `gorm:"type:varchar(128);not null;uniqueIndex;comment:插件代码（唯一标识）" json:"code"`
	Description string `gorm:"type:text;comment:插件描述" json:"description"`
	Author      string `gorm:"type:varchar(128);comment:作者" json:"author"`
	Enabled     bool   `gorm:"default:true;index;comment:是否启用" json:"enabled"`

	// NATS 主题（自动生成）
	// 格式：plugins.{user}.{plugin_id}
	Subject string `gorm:"type:varchar(512);uniqueIndex;comment:NATS主题" json:"subject"`

	// 插件配置（JSON，允许为 NULL）
	// 例如：{"timeout": 30, "max_file_size": 10485760}
	Config *string `gorm:"type:json;comment:插件配置" json:"config"`

	User string `gorm:"type:varchar(128);not null;index;comment:创建用户" json:"user"` // 保留用于向后兼容

	// 权限控制
	Visibility int    `gorm:"type:tinyint;default:0;index;comment:可见性(0:公开,1:私有)" json:"visibility"` // 0: 公开, 1: 私有
	Admin      string `gorm:"type:varchar(512);not null;index;comment:管理员列表(逗号分隔)" json:"admin"`      // 管理员列表，逗号分隔，如："user1,user2,user3"
}

// TableName 指定表名
func (Plugin) TableName() string {
	return "plugins"
}

// AfterCreate GORM 钩子：创建后自动生成 NATS 主题和设置默认管理员
func (p *Plugin) AfterCreate(tx *gorm.DB) error {
	// 1. 设置默认管理员（如果为空，设置为创建用户）
	if p.Admin == "" {
		p.Admin = p.CreatedBy
		if err := tx.Model(p).Update("admin", p.Admin).Error; err != nil {
			return err
		}
	}

	// 2. 生成 NATS 主题
	if p.Subject == "" {
		p.Subject = subjects.BuildPluginSubject(p.CreatedBy, p.ID)
		// 更新数据库
		return tx.Model(p).Update("subject", p.Subject).Error
	}
	return nil
}

