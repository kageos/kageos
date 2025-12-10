package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"gorm.io/gorm"
)

// PluginRepository 插件数据访问层
type PluginRepository struct {
	db *gorm.DB
}

// NewPluginRepository 创建插件 Repository
func NewPluginRepository(db *gorm.DB) *PluginRepository {
	return &PluginRepository{db: db}
}

// Create 创建插件
func (r *PluginRepository) Create(plugin *model.Plugin) error {
	return r.db.Create(plugin).Error
}

// GetByID 根据 ID 获取插件
func (r *PluginRepository) GetByID(id int64) (*model.Plugin, error) {
	var plugin model.Plugin
	if err := r.db.Where("id = ?", id).First(&plugin).Error; err != nil {
		return nil, err
	}
	return &plugin, nil
}

// GetByCode 根据 Code 获取插件
func (r *PluginRepository) GetByCode(code string) (*model.Plugin, error) {
	var plugin model.Plugin
	if err := r.db.Where("code = ?", code).First(&plugin).Error; err != nil {
		return nil, err
	}
	return &plugin, nil
}

// List 获取插件列表（支持筛选）
func (r *PluginRepository) List(scope string, currentUser string, enabled *bool, offset, limit int) ([]*model.Plugin, int64, error) {
	var plugins []*model.Plugin
	var total int64

	query := r.db.Model(&model.Plugin{})

	// 权限过滤：根据 scope 参数
	if scope == "mine" {
		// 我的：显示当前用户是管理员的资源（公开+私有）
		query = query.Where("FIND_IN_SET(?, admin) > 0", currentUser)
	} else if scope == "market" {
		// 市场：显示所有公开的资源
		query = query.Where("visibility = ?", 0)
	}
	// 默认：显示所有（向后兼容，或根据需求调整）

	// 如果指定了 enabled，添加筛选条件
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	if err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&plugins).Error; err != nil {
		return nil, 0, err
	}

	return plugins, total, nil
}

// Update 更新插件
func (r *PluginRepository) Update(plugin *model.Plugin) error {
	return r.db.Save(plugin).Error
}

// Delete 删除插件（软删除）
func (r *PluginRepository) Delete(id int64) error {
	return r.db.Delete(&model.Plugin{}, id).Error
}

// Enable 启用插件
func (r *PluginRepository) Enable(id int64) error {
	return r.db.Model(&model.Plugin{}).Where("id = ?", id).Update("enabled", true).Error
}

// Disable 禁用插件
func (r *PluginRepository) Disable(id int64) error {
	return r.db.Model(&model.Plugin{}).Where("id = ?", id).Update("enabled", false).Error
}

