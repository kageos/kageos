package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"gorm.io/gorm"
)

// LLMRepository LLM 配置数据访问层
type LLMRepository struct {
	db *gorm.DB
}

// NewLLMRepository 创建 LLM 配置 Repository
func NewLLMRepository(db *gorm.DB) *LLMRepository {
	return &LLMRepository{db: db}
}

// Create 创建 LLM 配置
func (r *LLMRepository) Create(cfg *model.LLMConfig) error {
	return r.db.Create(cfg).Error
}

// GetByID 根据 ID 获取 LLM 配置
func (r *LLMRepository) GetByID(id int64) (*model.LLMConfig, error) {
	var cfg model.LLMConfig
	if err := r.db.Where("id = ?", id).First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

// GetDefault 获取默认 LLM 配置
func (r *LLMRepository) GetDefault() (*model.LLMConfig, error) {
	var cfg model.LLMConfig
	if err := r.db.Where("is_default = ?", true).First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

// List 获取 LLM 配置列表
func (r *LLMRepository) List(scope string, currentUser string, offset, limit int) ([]*model.LLMConfig, int64, error) {
	var configs []*model.LLMConfig
	var total int64

	query := r.db.Model(&model.LLMConfig{})

	// 权限过滤：根据 scope 参数
	if scope == "mine" {
		// 我的：显示当前用户是管理员的资源（公开+私有）
		query = query.Where("FIND_IN_SET(?, admin) > 0", currentUser)
	} else if scope == "market" {
		// 市场：显示所有公开的资源
		query = query.Where("visibility = ?", 0)
	}
	// 默认：显示所有（向后兼容，或根据需求调整）

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	if err := query.Offset(offset).Limit(limit).Order("id DESC").Find(&configs).Error; err != nil {
		return nil, 0, err
	}

	return configs, total, nil
}

// Update 更新 LLM 配置
func (r *LLMRepository) Update(cfg *model.LLMConfig) error {
	return r.db.Save(cfg).Error
}

// Delete 删除 LLM 配置
func (r *LLMRepository) Delete(id int64) error {
	return r.db.Where("id = ?", id).Delete(&model.LLMConfig{}).Error
}

// SetDefault 设置默认配置
func (r *LLMRepository) SetDefault(id int64) error {
	// 先取消所有默认配置
	if err := r.db.Model(&model.LLMConfig{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
		return err
	}
	// 设置新的默认配置
	return r.db.Model(&model.LLMConfig{}).Where("id = ?", id).Update("is_default", true).Error
}

