package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"gorm.io/gorm"
)

// BoardPostRepository 版块帖子数据访问层
type BoardPostRepository struct {
	db *gorm.DB
}

// NewBoardPostRepository 创建版块帖子 Repository
func NewBoardPostRepository(db *gorm.DB) *BoardPostRepository {
	return &BoardPostRepository{db: db}
}

// Create 创建帖子
func (r *BoardPostRepository) Create(post *model.BoardPost) error {
	return r.db.Create(post).Error
}

// GetByID 根据 ID 获取帖子
func (r *BoardPostRepository) GetByID(id int64) (*model.BoardPost, error) {
	var post model.BoardPost
	if err := r.db.Where("id = ?", id).First(&post).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

// ListByFullCodePath 按版块路径分页列表（按创建时间倒序）
func (r *BoardPostRepository) ListByFullCodePath(fullCodePath string, offset, limit int) ([]*model.BoardPost, int64, error) {
	var list []*model.BoardPost
	var total int64
	query := r.db.Model(&model.BoardPost{}).Where("full_code_path = ?", fullCodePath)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// Update 更新帖子
func (r *BoardPostRepository) Update(post *model.BoardPost) error {
	return r.db.Save(post).Error
}

// Delete 删除帖子
func (r *BoardPostRepository) Delete(id int64) error {
	return r.db.Where("id = ?", id).Delete(&model.BoardPost{}).Error
}

// DeleteByTreeID 删除某版块下全部帖子（删版块时用）
func (r *BoardPostRepository) DeleteByTreeID(treeID int64) error {
	return r.db.Where("tree_id = ?", treeID).Delete(&model.BoardPost{}).Error
}
