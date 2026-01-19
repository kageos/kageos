package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"gorm.io/gorm"
)

// DocRepository 文档数据访问层
type DocRepository struct {
	db *gorm.DB
}

// NewDocRepository 创建文档 Repository
func NewDocRepository(db *gorm.DB) *DocRepository {
	return &DocRepository{db: db}
}

// Create 创建文档
func (r *DocRepository) Create(doc *model.Doc) error {
	return r.db.Create(doc).Error
}

// GetByID 根据 ID 获取文档
func (r *DocRepository) GetByID(id int64) (*model.Doc, error) {
	var doc model.Doc
	if err := r.db.Where("id = ?", id).First(&doc).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

// GetByTreeID 根据 TreeID 获取文档
func (r *DocRepository) GetByTreeID(treeID int64) (*model.Doc, error) {
	var doc model.Doc
	if err := r.db.Where("tree_id = ?", treeID).First(&doc).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

// Update 更新文档
func (r *DocRepository) Update(doc *model.Doc) error {
	return r.db.Save(doc).Error
}

// Delete 删除文档
func (r *DocRepository) Delete(id int64) error {
	return r.db.Where("id = ?", id).Delete(&model.Doc{}).Error
}

// DeleteByTreeID 根据 TreeID 删除文档
func (r *DocRepository) DeleteByTreeID(treeID int64) error {
	return r.db.Where("tree_id = ?", treeID).Delete(&model.Doc{}).Error
}

// ListByAppID 根据 AppID 获取文档列表
func (r *DocRepository) ListByAppID(appID int64, offset, limit int) ([]*model.Doc, int64, error) {
	var docs []*model.Doc
	var total int64

	query := r.db.Model(&model.Doc{}).Where("app_id = ?", appID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	if err := query.Offset(offset).Limit(limit).Order("id DESC").Find(&docs).Error; err != nil {
		return nil, 0, err
	}

	return docs, total, nil
}

// ListByTreeIDs 根据 TreeID 列表批量获取文档
func (r *DocRepository) ListByTreeIDs(treeIDs []int64) ([]*model.Doc, error) {
	var docs []*model.Doc
	if err := r.db.Where("tree_id IN ?", treeIDs).Find(&docs).Error; err != nil {
		return nil, err
	}
	return docs, nil
}
