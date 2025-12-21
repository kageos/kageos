package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"gorm.io/gorm"
)

// QuickLinkRepository 快链仓储
type QuickLinkRepository struct {
	db *gorm.DB
}

// NewQuickLinkRepository 创建快链仓储
func NewQuickLinkRepository(db *gorm.DB) *QuickLinkRepository {
	return &QuickLinkRepository{db: db}
}

// Create 创建快链
func (r *QuickLinkRepository) Create(quickLink *model.QuickLink) error {
	return r.db.Create(quickLink).Error
}

// GetByID 根据ID获取快链
func (r *QuickLinkRepository) GetByID(id int64) (*model.QuickLink, error) {
	var quickLink model.QuickLink
	err := r.db.Where("id = ?", id).First(&quickLink).Error
	if err != nil {
		return nil, err
	}
	return &quickLink, nil
}

// GetByIDAndUser 根据ID和用户获取快链（确保用户只能访问自己的快链）
func (r *QuickLinkRepository) GetByIDAndUser(id int64, username string) (*model.QuickLink, error) {
	var quickLink model.QuickLink
	err := r.db.Where("id = ? AND created_by = ?", id, username).First(&quickLink).Error
	if err != nil {
		return nil, err
	}
	return &quickLink, nil
}

// ListByUser 根据用户获取快链列表
func (r *QuickLinkRepository) ListByUser(username string, functionRouter string, page, pageSize int) ([]*model.QuickLink, int64, error) {
	var quickLinks []*model.QuickLink
	var total int64

	query := r.db.Model(&model.QuickLink{}).Where("created_by = ?", username)
	
	// 如果指定了函数路由，进行过滤
	if functionRouter != "" {
		query = query.Where("function_router = ?", functionRouter)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&quickLinks).Error
	if err != nil {
		return nil, 0, err
	}

	return quickLinks, total, nil
}

// Update 更新快链
func (r *QuickLinkRepository) Update(quickLink *model.QuickLink) error {
	return r.db.Save(quickLink).Error
}

// Delete 删除快链
func (r *QuickLinkRepository) Delete(id int64, username string) error {
	return r.db.Where("id = ? AND created_by = ?", id, username).Delete(&model.QuickLink{}).Error
}

// DeleteByUser 删除用户的所有快链
func (r *QuickLinkRepository) DeleteByUser(username string) error {
	return r.db.Where("created_by = ?", username).Delete(&model.QuickLink{}).Error
}

