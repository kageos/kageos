package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"gorm.io/gorm"
)

// WorkspaceModeRepository 工作台模式数据访问层
type WorkspaceModeRepository struct {
	db *gorm.DB
}

// NewWorkspaceModeRepository 创建 WorkspaceMode Repository
func NewWorkspaceModeRepository(db *gorm.DB) *WorkspaceModeRepository {
	return &WorkspaceModeRepository{db: db}
}

// GetByCode 根据 code 获取模式
func (r *WorkspaceModeRepository) GetByCode(code string) (*model.WorkspaceMode, error) {
	var m model.WorkspaceMode
	if err := r.db.Where("code = ?", code).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// GetByID 根据 ID 获取模式
func (r *WorkspaceModeRepository) GetByID(id int64) (*model.WorkspaceMode, error) {
	var m model.WorkspaceMode
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// List 分页列表，按 sort_order、id 排序
func (r *WorkspaceModeRepository) List(offset, limit int) ([]*model.WorkspaceMode, int64, error) {
	var list []*model.WorkspaceMode
	var total int64
	if err := r.db.Model(&model.WorkspaceMode{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}
	if err := r.db.Order("sort_order ASC, id ASC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// Create 创建
func (r *WorkspaceModeRepository) Create(m *model.WorkspaceMode) error {
	return r.db.Create(m).Error
}

// Update 更新
func (r *WorkspaceModeRepository) Update(m *model.WorkspaceMode) error {
	return r.db.Save(m).Error
}

// Delete 删除；若 is_builtin=true 调用方应禁止删除
func (r *WorkspaceModeRepository) Delete(id int64) error {
	return r.db.Delete(&model.WorkspaceMode{}, id).Error
}
